package sourceaccess

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

const (
	DefaultPreviewRows    = 20
	HardMaxPreviewRows    = 50
	DefaultInspectTimeout = 5 * time.Second
)

type CatalogActor struct {
	TenantID    string
	PrincipalID string
}

type CreateConnectionDraftInput struct {
	Code                 string          `json:"code"`
	Name                 string          `json:"name"`
	AdapterKind          AdapterKind     `json:"adapter_kind"`
	AdapterVersion       string          `json:"adapter_version"`
	SecretRef            string          `json:"secret_ref,omitempty"`
	Definition           json.RawMessage `json:"definition,omitempty"`
	DeclaredCapabilities []Capability    `json:"declared_capabilities,omitempty"`
	OwnerPrincipalID     string          `json:"owner_principal_id,omitempty"`
}

type CreateViewDraftInput struct {
	ConnectionVersion int64           `json:"connection_version,omitempty"`
	Code              string          `json:"code"`
	Name              string          `json:"name"`
	Definition        json.RawMessage `json:"definition"`
	OutputKind        OutputKind      `json:"output_kind,omitempty"`
	StableKeys        []string        `json:"stable_keys,omitempty"`
}

type InspectViewDraftInput struct {
	StableKeys []string `json:"stable_keys,omitempty"`
}

type InspectedViewDraft struct {
	View   ViewRevision `json:"view"`
	Schema SchemaResult `json:"schema"`
}

type CreateBindingDraftInput struct {
	ViewVersion              int64                   `json:"view_version,omitempty"`
	Code                     string                  `json:"code"`
	Name                     string                  `json:"name"`
	Purpose                  string                  `json:"purpose"`
	Operations               []Operation             `json:"operations"`
	SelectedFields           []string                `json:"selected_fields"`
	KeyFields                []string                `json:"key_fields,omitempty"`
	Limits                   ResourceLimits          `json:"limits"`
	Mapping                  json.RawMessage         `json:"mapping,omitempty"`
	ParameterSchema          json.RawMessage         `json:"parameter_schema,omitempty"`
	OutputSchema             json.RawMessage         `json:"output_schema,omitempty"`
	RequiredFreshnessMinutes int                     `json:"required_freshness_minutes,omitempty"`
	Completeness             CompletenessRequirement `json:"completeness,omitempty"`
	SensitivityHandling      json.RawMessage         `json:"sensitivity_handling,omitempty"`
}

type CatalogUsageKind string

const (
	UsageConnection CatalogUsageKind = "CONNECTION"
	UsageView       CatalogUsageKind = "VIEW"
	UsageBinding    CatalogUsageKind = "BINDING"
)

type CatalogUsageReference struct {
	Kind    CatalogUsageKind `json:"kind"`
	ID      string           `json:"id"`
	Version int64            `json:"version"`
	Code    string           `json:"code"`
	Name    string           `json:"name"`
}

type CatalogUsageReport struct {
	Kind            CatalogUsageKind        `json:"kind"`
	ID              string                  `json:"id"`
	Children        []CatalogUsageReference `json:"children"`
	Consumers       []CatalogUsageReference `json:"consumers"`
	ConsumerDomains []string                `json:"consumer_domains"`
	Complete        bool                    `json:"complete"`
	Scope           string                  `json:"scope"`
}

type CatalogService struct {
	repo     CatalogRepository
	adapters map[AdapterKind]Adapter
	secrets  SecretResolver
	now      func() time.Time
	newID    func() (string, error)
}

func NewCatalogService(repo CatalogRepository, secrets SecretResolver, adapters map[AdapterKind]Adapter) *CatalogService {
	registered := make(map[AdapterKind]Adapter, len(adapters))
	for kind, adapter := range adapters {
		if adapter != nil {
			registered[kind] = adapter
		}
	}
	return &CatalogService{repo: repo, secrets: secrets, adapters: registered, now: time.Now, newID: id.NewUUIDv7}
}

func DefaultCatalogAdapters() map[AdapterKind]Adapter {
	return map[AdapterKind]Adapter{
		AdapterPostgres: NewPostgresAdapter(DefaultPostgresOptions()),
		AdapterRESTJSON: NewRESTJSONAdapter(DefaultRESTJSONOptions()),
	}
}

// EnvironmentSecretResolver keeps credential values out of catalog records.
// A SecretRef must be env://NAME; only NAME is persisted, while its value is
// resolved inside the adapter process boundary.
type EnvironmentSecretResolver struct{}

func (EnvironmentSecretResolver) Resolve(ctx context.Context, secretRef string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	name, ok := environmentSecretName(secretRef)
	if !ok {
		return "", ErrCredentials
	}
	value, exists := os.LookupEnv(name)
	if !exists || strings.TrimSpace(value) == "" {
		return "", ErrCredentials
	}
	return value, nil
}

func environmentSecretName(secretRef string) (string, bool) {
	trimmed := strings.TrimSpace(secretRef)
	name := strings.TrimPrefix(trimmed, "env://")
	if name == trimmed || !validEnvironmentSecretName(name) {
		return "", false
	}
	return name, true
}

func validEnvironmentSecretName(value string) bool {
	if len(value) < 1 || len(value) > 128 {
		return false
	}
	for index, character := range value {
		if (character >= 'A' && character <= 'Z') || character == '_' || (index > 0 && character >= '0' && character <= '9') {
			continue
		}
		return false
	}
	return true
}

func (s *CatalogService) CreateConnectionDraft(ctx context.Context, actor CatalogActor, sourceID string, input CreateConnectionDraftInput) (ConnectionRevision, error) {
	if err := validateCatalogActor(actor); err != nil || strings.TrimSpace(sourceID) == "" {
		return ConnectionRevision{}, ErrCatalogInvalid
	}
	if err := s.validateDraftAdapter(input); err != nil {
		return ConnectionRevision{}, err
	}
	revisionID, connectionID, err := s.twoIDs()
	if err != nil {
		return ConnectionRevision{}, err
	}
	now := s.now().UTC()
	owner := strings.TrimSpace(input.OwnerPrincipalID)
	if owner != "" && owner != actor.PrincipalID {
		return ConnectionRevision{}, fmt.Errorf("%w: owner assignment requires a governed principal-selection command", ErrCatalogInvalid)
	}
	owner = actor.PrincipalID
	value := ConnectionRevision{
		RevisionID: revisionID, ConnectionID: connectionID, TenantID: actor.TenantID, SourceID: strings.TrimSpace(sourceID),
		Code: input.Code, Name: input.Name, AdapterKind: input.AdapterKind, AdapterVersion: input.AdapterVersion,
		SecretRef: input.SecretRef, Definition: defaultJSONObject(input.Definition),
		DeclaredCapabilities: append([]Capability(nil), input.DeclaredCapabilities...), OwnerPrincipalID: owner,
		RevisionLifecycle: draftLifecycle(actor.PrincipalID, now),
	}
	return s.repoOrError().CreateConnectionRevision(ctx, value)
}

func (s *CatalogService) CreateViewDraft(ctx context.Context, actor CatalogActor, connectionID string, input CreateViewDraftInput) (ViewRevision, error) {
	if err := validateCatalogActor(actor); err != nil {
		return ViewRevision{}, err
	}
	if len(input.StableKeys) != 0 {
		return ViewRevision{}, fmt.Errorf("%w: stable keys are selected from the inspected native schema", ErrCatalogInvalid)
	}
	connection, err := s.connection(ctx, actor.TenantID, connectionID, input.ConnectionVersion)
	if err != nil {
		return ViewRevision{}, err
	}
	if connection.AdapterKind == AdapterReference || !revisionUsableAsParent(connection.Status) {
		return ViewRevision{}, ErrCatalogInvalid
	}
	if _, err := s.adapterFor(connection.AdapterKind, connection.AdapterVersion); err != nil {
		return ViewRevision{}, err
	}
	revisionID, viewID, err := s.twoIDs()
	if err != nil {
		return ViewRevision{}, err
	}
	outputKind := input.OutputKind
	if outputKind == "" {
		outputKind = OutputRecords
	}
	definition, err := normalizeDraftViewDefinition(connection, viewID, outputKind, input.Definition)
	if err != nil {
		return ViewRevision{}, err
	}
	now := s.now().UTC()
	value := ViewRevision{
		RevisionID: revisionID, ViewID: viewID, TenantID: actor.TenantID, SourceID: connection.SourceID,
		ConnectionID: connection.ConnectionID, ConnectionVersion: connection.Version,
		Code: input.Code, Name: input.Name, Definition: definition, OutputKind: outputKind,
		RevisionLifecycle: draftLifecycle(actor.PrincipalID, now),
	}
	return s.repoOrError().CreateViewRevision(ctx, value)
}

// InspectViewDraft executes the exact View revision and persists the observed
// native schema as a new immutable draft revision. Stable keys are therefore
// validated against observed fields rather than accepted as an assertion.
func (s *CatalogService) InspectViewDraft(ctx context.Context, actor CatalogActor, viewID string, version int64, input InspectViewDraftInput) (InspectedViewDraft, error) {
	if err := validateCatalogActor(actor); err != nil {
		return InspectedViewDraft{}, err
	}
	viewRevision, err := s.view(ctx, actor.TenantID, viewID, version)
	if err != nil {
		return InspectedViewDraft{}, err
	}
	if !revisionUsableAsParent(viewRevision.Status) || viewRevision.Version == math.MaxInt64 {
		return InspectedViewDraft{}, ErrCatalogInvalid
	}
	connectionRevision, err := s.repoOrError().ConnectionRevision(ctx, actor.TenantID, viewRevision.ConnectionID, viewRevision.ConnectionVersion)
	if err != nil {
		return InspectedViewDraft{}, err
	}
	connection, view, adapter, err := s.executionContracts(connectionRevision, viewRevision)
	if err != nil {
		return InspectedViewDraft{}, err
	}
	operationCtx, cancel := context.WithTimeout(ctx, DefaultInspectTimeout)
	defer cancel()
	session, err := adapter.Open(operationCtx, connection, s.secrets)
	if err != nil {
		return InspectedViewDraft{}, err
	}
	defer session.Close()
	if !session.Capabilities().Has(CapabilityInspect) {
		return InspectedViewDraft{}, ErrCapabilityUnavailable
	}
	reader, ok := session.(SchemaReader)
	if !ok {
		return InspectedViewDraft{}, ErrCapabilityUnavailable
	}
	result, err := reader.Inspect(operationCtx, view)
	if err != nil {
		return InspectedViewDraft{}, err
	}
	if err := validateCatalogReceipt(result.Receipt, connection, view, Binding{}, OperationInspect, int64(len(result.Fields))); err != nil {
		return InspectedViewDraft{}, err
	}
	if len(result.Fields) == 0 || len(result.Fields) > HardMaxSchemaFields {
		return InspectedViewDraft{}, ErrLimitExceeded
	}
	fingerprint := strings.TrimSpace(result.Receipt.SchemaFingerprint)
	if !isLowerHex(fingerprint, 64) {
		fingerprint, err = nativeSchemaFingerprint(result.Fields)
		if err != nil {
			return InspectedViewDraft{}, err
		}
	}
	revisionID, err := s.oneID()
	if err != nil {
		return InspectedViewDraft{}, err
	}
	now := s.now().UTC()
	inspected := ViewRevision{
		RevisionID: revisionID, ViewID: viewRevision.ViewID, TenantID: viewRevision.TenantID, SourceID: viewRevision.SourceID,
		ConnectionID: viewRevision.ConnectionID, ConnectionVersion: viewRevision.ConnectionVersion,
		Code: viewRevision.Code, Name: viewRevision.Name, Definition: cloneRawMessage(viewRevision.Definition), OutputKind: viewRevision.OutputKind,
		StableKeys: append([]string(nil), input.StableKeys...), NativeSchema: append([]NativeField(nil), result.Fields...), SchemaFingerprint: fingerprint,
		RevisionLifecycle: RevisionLifecycle{Status: RevisionDraft, IsCurrent: false, Version: viewRevision.Version + 1, CreatedBy: actor.PrincipalID, CreatedAt: now, UpdatedAt: now},
	}
	created, err := s.repoOrError().CreateViewRevision(ctx, inspected)
	if err != nil {
		return InspectedViewDraft{}, err
	}
	return InspectedViewDraft{View: created, Schema: result}, nil
}

func (s *CatalogService) CreateBindingDraft(ctx context.Context, actor CatalogActor, viewID string, input CreateBindingDraftInput) (BindingRevision, error) {
	if err := validateCatalogActor(actor); err != nil {
		return BindingRevision{}, err
	}
	view, err := s.view(ctx, actor.TenantID, viewID, input.ViewVersion)
	if err != nil {
		return BindingRevision{}, err
	}
	if !revisionUsableAsParent(view.Status) || len(view.NativeSchema) == 0 || view.SchemaFingerprint == "" {
		return BindingRevision{}, fmt.Errorf("%w: binding parent requires an inspected schema", ErrCatalogInvalid)
	}
	revisionID, bindingID, err := s.twoIDs()
	if err != nil {
		return BindingRevision{}, err
	}
	now := s.now().UTC()
	value := BindingRevision{
		RevisionID: revisionID, BindingID: bindingID, TenantID: actor.TenantID, SourceID: view.SourceID,
		ViewID: view.ViewID, ViewVersion: view.Version, Code: input.Code, Name: input.Name, Purpose: input.Purpose,
		Operations: append([]Operation(nil), input.Operations...), SelectedFields: append([]string(nil), input.SelectedFields...),
		KeyFields: append([]string(nil), input.KeyFields...), Limits: input.Limits, Mapping: defaultJSONObject(input.Mapping),
		ParameterSchema: defaultJSONObject(input.ParameterSchema), OutputSchema: defaultJSONObject(input.OutputSchema),
		RequiredFreshnessMinutes: input.RequiredFreshnessMinutes, Completeness: input.Completeness,
		SensitivityHandling: defaultJSONObject(input.SensitivityHandling), RevisionLifecycle: draftLifecycle(actor.PrincipalID, now),
	}
	return s.repoOrError().CreateBindingRevision(ctx, value)
}

func (s *CatalogService) Connections(ctx context.Context, tenantID, sourceID string, limit int) ([]ConnectionRevision, error) {
	return s.repoOrError().ListConnectionRevisions(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(sourceID), limit)
}

func (s *CatalogService) Connection(ctx context.Context, tenantID, connectionID string, version int64) (ConnectionRevision, error) {
	return s.connection(ctx, tenantID, connectionID, version)
}

func (s *CatalogService) Views(ctx context.Context, tenantID, connectionID string, limit int) ([]ViewRevision, error) {
	return s.repoOrError().ListViewRevisions(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(connectionID), limit)
}

func (s *CatalogService) View(ctx context.Context, tenantID, viewID string, version int64) (ViewRevision, error) {
	return s.view(ctx, tenantID, viewID, version)
}

func (s *CatalogService) Bindings(ctx context.Context, tenantID, viewID string, limit int) ([]BindingRevision, error) {
	return s.repoOrError().ListBindingRevisions(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(viewID), limit)
}

func (s *CatalogService) Binding(ctx context.Context, tenantID, bindingID string, version int64) (BindingRevision, error) {
	return s.binding(ctx, tenantID, bindingID, version)
}

func (s *CatalogService) PreviewBinding(ctx context.Context, tenantID, bindingID string, version int64, request PageRequest) (RecordPage, error) {
	bindingRevision, err := s.binding(ctx, tenantID, bindingID, version)
	if err != nil {
		return RecordPage{}, err
	}
	if !revisionExecutable(bindingRevision.Status) {
		return RecordPage{}, ErrCatalogInvalid
	}
	viewRevision, err := s.repoOrError().ViewRevision(ctx, tenantID, bindingRevision.ViewID, bindingRevision.ViewVersion)
	if err != nil {
		return RecordPage{}, err
	}
	connectionRevision, err := s.repoOrError().ConnectionRevision(ctx, tenantID, viewRevision.ConnectionID, viewRevision.ConnectionVersion)
	if err != nil {
		return RecordPage{}, err
	}
	connection, view, adapter, err := s.executionContracts(connectionRevision, viewRevision)
	if err != nil {
		return RecordPage{}, err
	}
	binding, err := bindingRevision.Contract(viewRevision)
	if err != nil {
		return RecordPage{}, err
	}
	if !binding.Allows(OperationPage) {
		return RecordPage{}, ErrCapabilityUnavailable
	}
	limits, err := binding.NormalizedLimits()
	if err != nil {
		return RecordPage{}, err
	}
	request.Limit = boundedPreviewRows(request.Limit, limits.PageRows)
	operationCtx, cancel := context.WithTimeout(ctx, limits.Timeout)
	defer cancel()
	session, err := adapter.Open(operationCtx, connection, s.secrets)
	if err != nil {
		return RecordPage{}, err
	}
	defer session.Close()
	if !session.Capabilities().Has(CapabilityPage) {
		return RecordPage{}, ErrCapabilityUnavailable
	}
	reader, ok := session.(PageReader)
	if !ok {
		return RecordPage{}, ErrCapabilityUnavailable
	}
	page, err := reader.ReadPage(operationCtx, view, binding, request)
	if err != nil {
		return RecordPage{}, err
	}
	if err := validateCatalogReceipt(page.Receipt, connection, view, binding, OperationPage, int64(len(page.Records))); err != nil {
		return RecordPage{}, err
	}
	if err := validateExpectedSchema(view, page.Receipt.SchemaFingerprint); err != nil {
		return RecordPage{}, err
	}
	if len(page.Records) > request.Limit || page.Receipt.Bytes > limits.ResponseBytes {
		return RecordPage{}, ErrLimitExceeded
	}
	return page, nil
}

// LookupBinding resolves an exact governed Binding through its registered adapter.
// It is an internal reusable operation for forms/evidence/workflow consumers; no
// connector configuration or query material is copied into those domains.
func (s *CatalogService) LookupBinding(ctx context.Context, tenantID, bindingID string, version int64, request LookupRequest) (LookupResult, error) {
	bindingRevision, err := s.binding(ctx, tenantID, bindingID, version)
	if err != nil {
		return LookupResult{}, err
	}
	if !revisionExecutable(bindingRevision.Status) {
		return LookupResult{}, ErrCatalogInvalid
	}
	viewRevision, err := s.repoOrError().ViewRevision(ctx, tenantID, bindingRevision.ViewID, bindingRevision.ViewVersion)
	if err != nil {
		return LookupResult{}, err
	}
	connectionRevision, err := s.repoOrError().ConnectionRevision(ctx, tenantID, viewRevision.ConnectionID, viewRevision.ConnectionVersion)
	if err != nil {
		return LookupResult{}, err
	}
	connection, view, adapter, err := s.executionContracts(connectionRevision, viewRevision)
	if err != nil {
		return LookupResult{}, err
	}
	binding, err := bindingRevision.Contract(viewRevision)
	if err != nil {
		return LookupResult{}, err
	}
	if !binding.Allows(OperationLookup) {
		return LookupResult{}, ErrCapabilityUnavailable
	}
	limits, err := binding.NormalizedLimits()
	if err != nil {
		return LookupResult{}, err
	}
	operationCtx, cancel := context.WithTimeout(ctx, limits.Timeout)
	defer cancel()
	session, err := adapter.Open(operationCtx, connection, s.secrets)
	if err != nil {
		return LookupResult{}, err
	}
	defer session.Close()
	if !session.Capabilities().Has(CapabilityLookup) {
		return LookupResult{}, ErrCapabilityUnavailable
	}
	reader, ok := session.(LookupReader)
	if !ok {
		return LookupResult{}, ErrCapabilityUnavailable
	}
	result, err := reader.Lookup(operationCtx, view, binding, request)
	if err != nil {
		return LookupResult{}, err
	}
	if err := validateCatalogReceipt(result.Receipt, connection, view, binding, OperationLookup, int64(len(result.Records))); err != nil {
		return LookupResult{}, err
	}
	if err := validateExpectedSchema(view, result.Receipt.SchemaFingerprint); err != nil {
		return LookupResult{}, err
	}
	if len(result.Records) > limits.LookupValues || result.Receipt.Bytes > limits.ResponseBytes {
		return LookupResult{}, ErrLimitExceeded
	}
	return result, nil
}

func (s *CatalogService) WhereUsed(ctx context.Context, tenantID string, kind CatalogUsageKind, resourceID string, limit int) (CatalogUsageReport, error) {
	report := CatalogUsageReport{
		Kind: kind, ID: strings.TrimSpace(resourceID), Children: []CatalogUsageReference{}, Consumers: []CatalogUsageReference{},
		ConsumerDomains: []string{"ASSURANCE", "EVIDENCE", "FORMS", "WORKFLOW", "AI_GATEWAY"}, Complete: true,
		Scope: "CURRENT_CATALOG_AND_IMPLEMENTED_BINDING_REFERENCES",
	}
	switch kind {
	case UsageConnection:
		if _, err := s.connection(ctx, tenantID, resourceID, 0); err != nil {
			return CatalogUsageReport{}, err
		}
		values, err := s.repoOrError().ListCurrentViews(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(resourceID), limit)
		if err != nil {
			return CatalogUsageReport{}, err
		}
		for _, value := range values {
			report.Children = append(report.Children, CatalogUsageReference{Kind: UsageView, ID: value.ViewID, Version: value.Version, Code: value.Code, Name: value.Name})
		}
	case UsageView:
		if _, err := s.view(ctx, tenantID, resourceID, 0); err != nil {
			return CatalogUsageReport{}, err
		}
		values, err := s.repoOrError().ListCurrentBindings(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(resourceID), limit)
		if err != nil {
			return CatalogUsageReport{}, err
		}
		for _, value := range values {
			report.Children = append(report.Children, CatalogUsageReference{Kind: UsageBinding, ID: value.BindingID, Version: value.Version, Code: value.Code, Name: value.Name})
		}
	case UsageBinding:
		if _, err := s.binding(ctx, tenantID, resourceID, 0); err != nil {
			return CatalogUsageReport{}, err
		}
	default:
		return CatalogUsageReport{}, ErrCatalogInvalid
	}
	return report, nil
}

func validateCatalogReceipt(receipt OperationReceipt, connection Connection, view View, binding Binding, operation Operation, expectedCount int64) error {
	if receipt.SourceID != connection.SourceID ||
		receipt.ConnectionID != connection.ID ||
		receipt.ConnectionVersion != connection.Version ||
		receipt.AdapterKind != connection.AdapterKind ||
		receipt.AdapterVersion != connection.AdapterVersion ||
		receipt.ViewID != view.ID ||
		receipt.ViewVersion != view.Version ||
		receipt.Operation != operation ||
		receipt.Count != expectedCount ||
		receipt.Bytes < 0 {
		return ErrExecution
	}
	switch operation {
	case OperationInspect:
		if receipt.BindingID != "" || receipt.BindingVersion != "" || receipt.Completeness != CompletenessComplete {
			return ErrExecution
		}
	case OperationPage:
		if receipt.BindingID != binding.ID || receipt.BindingVersion != binding.Version ||
			(receipt.Completeness != CompletenessComplete && receipt.Completeness != CompletenessPartial) {
			return ErrExecution
		}
	case OperationLookup:
		if receipt.BindingID != binding.ID || receipt.BindingVersion != binding.Version || (receipt.Completeness != CompletenessComplete && receipt.Completeness != CompletenessPartial) {
			return ErrExecution
		}
	case OperationChanges:
		if receipt.BindingID != binding.ID || receipt.BindingVersion != binding.Version || receipt.Completeness != CompletenessComplete || receipt.Position == nil {
			return ErrExecution
		}
		if err := validateCheckpointPosition(*receipt.Position); err != nil {
			return ErrExecution
		}
	default:
		return ErrExecution
	}
	if connection.AdapterKind == AdapterTabularArtifact {
		if strings.TrimSpace(receipt.ArtifactID) == "" || !isLowerHex(receipt.ArtifactSHA256, 64) || strings.TrimSpace(receipt.ParserVersion) == "" || len(receipt.ParserVersion) > hardMaxAdapterVersionLen || containsControl(receipt.ParserVersion) {
			return ErrExecution
		}
	}
	return nil
}

func (s *CatalogService) validateDraftAdapter(input CreateConnectionDraftInput) error {
	if input.AdapterKind == AdapterReference {
		return fmt.Errorf("%w: reference connections are created only by legacy migration", ErrCatalogInvalid)
	}
	if _, err := s.adapterFor(input.AdapterKind, input.AdapterVersion); err != nil {
		return err
	}
	switch input.AdapterKind {
	case AdapterPostgres:
		secretRef := strings.TrimSpace(input.SecretRef)
		if secretRef == "" || secretRef != input.SecretRef || len(secretRef) > HardMaxIdentifierBytes || containsControl(secretRef) {
			return fmt.Errorf("%w: PostgreSQL connections require a bounded opaque secret reference", ErrCatalogInvalid)
		}
		definition, err := normalizeJSONObject(defaultJSONObject(input.Definition), HardMaxDefinitionBytes, "connection definition")
		if err != nil {
			return err
		}
		if string(definition) != "{}" {
			return fmt.Errorf("%w: PostgreSQL connection definitions are not supported", ErrCatalogInvalid)
		}
	case AdapterRESTJSON:
		if _, err := normalizeRESTJSONConnectionDefinition(defaultJSONObject(input.Definition), strings.TrimSpace(input.SecretRef)); err != nil {
			return errors.Join(ErrCatalogInvalid, err)
		}
	case AdapterTabularArtifact:
		if input.SecretRef != "" {
			return fmt.Errorf("%w: tabular artifact connections do not carry credentials", ErrCatalogInvalid)
		}
		definition, err := normalizeJSONObject(defaultJSONObject(input.Definition), HardMaxDefinitionBytes, "connection definition")
		if err != nil {
			return err
		}
		if string(definition) != "{}" {
			return fmt.Errorf("%w: tabular artifact connection definitions are not supported", ErrCatalogInvalid)
		}
	case AdapterWebhookEvent:
		if input.SecretRef != "" {
			return fmt.Errorf("%w: webhook event connections use verified service identity rather than connector credentials", ErrCatalogInvalid)
		}
		definition, err := normalizeJSONObject(defaultJSONObject(input.Definition), HardMaxDefinitionBytes, "connection definition")
		if err != nil {
			return err
		}
		if string(definition) != "{}" {
			return fmt.Errorf("%w: webhook event connection definitions are not supported", ErrCatalogInvalid)
		}
	}
	return nil
}

func (s *CatalogService) adapterFor(kind AdapterKind, version string) (Adapter, error) {
	if s == nil || s.adapters == nil || s.adapters[kind] == nil {
		return nil, fmt.Errorf("%w: source adapter is not registered", ErrCatalogInvalid)
	}
	switch kind {
	case AdapterPostgres:
		if version != PostgresAdapterVersion {
			return nil, fmt.Errorf("%w: unsupported PostgreSQL adapter version", ErrCatalogInvalid)
		}
	case AdapterRESTJSON:
		if version != RESTJSONAdapterVersion {
			return nil, fmt.Errorf("%w: unsupported REST/JSON adapter version", ErrCatalogInvalid)
		}
	case AdapterTabularArtifact:
		if version != TabularArtifactAdapterVersion {
			return nil, fmt.Errorf("%w: unsupported tabular artifact adapter version", ErrCatalogInvalid)
		}
	case AdapterWebhookEvent:
		if version != WebhookEventAdapterVersion {
			return nil, fmt.Errorf("%w: unsupported webhook event adapter version", ErrCatalogInvalid)
		}
	}
	return s.adapters[kind], nil
}

func normalizeDraftViewDefinition(connection ConnectionRevision, viewID string, outputKind OutputKind, raw json.RawMessage) (json.RawMessage, error) {
	candidate := View{ID: viewID, ConnectionID: connection.ConnectionID, Version: "1", OutputKind: outputKind, Definition: cloneRawMessage(raw)}
	connectionContract, err := connection.Contract()
	if err != nil {
		return nil, err
	}
	if err := candidate.Validate(connectionContract); err != nil {
		return nil, errors.Join(ErrCatalogInvalid, err)
	}
	if connection.AdapterKind == AdapterPostgres {
		definition, err := decodePostgresView(candidate)
		if err != nil {
			return nil, errors.Join(ErrCatalogInvalid, err)
		}
		encoded, err := json.Marshal(definition)
		if err != nil {
			return nil, errors.Join(ErrCatalogInvalid, err)
		}
		return encoded, nil
	}
	if connection.AdapterKind == AdapterRESTJSON {
		definition, err := normalizeRESTJSONViewDefinition(raw)
		if err != nil {
			return nil, errors.Join(ErrCatalogInvalid, err)
		}
		return definition, nil
	}
	if connection.AdapterKind == AdapterTabularArtifact {
		definition, err := NormalizeTabularArtifactViewDefinition(raw)
		if err != nil {
			return nil, errors.Join(ErrCatalogInvalid, err)
		}
		return definition, nil
	}
	if connection.AdapterKind == AdapterWebhookEvent {
		definition, err := NormalizeWebhookEventViewDefinition(raw)
		if err != nil {
			return nil, errors.Join(ErrCatalogInvalid, err)
		}
		return definition, nil
	}
	return normalizeJSONObject(raw, HardMaxDefinitionBytes, "view definition")
}

func (s *CatalogService) executionContracts(connectionRevision ConnectionRevision, viewRevision ViewRevision) (Connection, View, Adapter, error) {
	if !revisionExecutable(connectionRevision.Status) || !revisionExecutable(viewRevision.Status) || connectionRevision.AdapterKind == AdapterReference {
		return Connection{}, View{}, nil, ErrCatalogInvalid
	}
	connection, err := connectionRevision.Contract()
	if err != nil {
		return Connection{}, View{}, nil, err
	}
	view, err := viewRevision.Contract(connectionRevision)
	if err != nil {
		return Connection{}, View{}, nil, err
	}
	adapter, err := s.adapterFor(connection.AdapterKind, connection.AdapterVersion)
	if err != nil || s.secrets == nil {
		return Connection{}, View{}, nil, ErrCapabilityUnavailable
	}
	return connection, view, adapter, nil
}

func (s *CatalogService) connection(ctx context.Context, tenantID, connectionID string, version int64) (ConnectionRevision, error) {
	repo := s.repoOrError()
	if version > 0 {
		return repo.ConnectionRevision(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(connectionID), version)
	}
	return repo.CurrentConnection(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(connectionID))
}

func (s *CatalogService) view(ctx context.Context, tenantID, viewID string, version int64) (ViewRevision, error) {
	repo := s.repoOrError()
	if version > 0 {
		return repo.ViewRevision(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(viewID), version)
	}
	return repo.CurrentView(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(viewID))
}

func (s *CatalogService) binding(ctx context.Context, tenantID, bindingID string, version int64) (BindingRevision, error) {
	repo := s.repoOrError()
	if version > 0 {
		return repo.BindingRevision(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(bindingID), version)
	}
	return repo.CurrentBinding(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(bindingID))
}

func (s *CatalogService) repoOrError() CatalogRepository {
	if s == nil || s.repo == nil {
		return unavailableCatalogRepository{}
	}
	return s.repo
}

func (s *CatalogService) oneID() (string, error) {
	if s == nil || s.newID == nil || s.now == nil {
		return "", ErrCatalogStorage
	}
	value, err := s.newID()
	if err != nil {
		return "", errors.Join(ErrCatalogStorage, err)
	}
	return value, nil
}

func (s *CatalogService) twoIDs() (string, string, error) {
	first, err := s.oneID()
	if err != nil {
		return "", "", err
	}
	second, err := s.oneID()
	if err != nil {
		return "", "", err
	}
	return first, second, nil
}

func draftLifecycle(actorID string, now time.Time) RevisionLifecycle {
	return RevisionLifecycle{Status: RevisionDraft, IsCurrent: false, Version: 1, CreatedBy: actorID, CreatedAt: now, UpdatedAt: now}
}

func revisionUsableAsParent(status RevisionStatus) bool {
	return status == RevisionDraft || status == RevisionPendingApproval || status == RevisionActive || status == RevisionPaused
}

func revisionExecutable(status RevisionStatus) bool {
	return revisionUsableAsParent(status)
}

func boundedPreviewRows(requested, bindingLimit int) int {
	if requested <= 0 {
		requested = DefaultPreviewRows
	}
	if requested > HardMaxPreviewRows {
		requested = HardMaxPreviewRows
	}
	if bindingLimit > 0 && requested > bindingLimit {
		requested = bindingLimit
	}
	return requested
}

func validateCatalogActor(actor CatalogActor) error {
	if strings.TrimSpace(actor.TenantID) == "" || strings.TrimSpace(actor.PrincipalID) == "" {
		return ErrCatalogInvalid
	}
	return nil
}

type unavailableCatalogRepository struct{}

func (unavailableCatalogRepository) CreateConnectionRevision(context.Context, ConnectionRevision) (ConnectionRevision, error) {
	return ConnectionRevision{}, ErrCatalogStorage
}
func (unavailableCatalogRepository) ConnectionRevision(context.Context, string, string, int64) (ConnectionRevision, error) {
	return ConnectionRevision{}, ErrCatalogStorage
}
func (unavailableCatalogRepository) CurrentConnection(context.Context, string, string) (ConnectionRevision, error) {
	return ConnectionRevision{}, ErrCatalogStorage
}
func (unavailableCatalogRepository) ListCurrentConnections(context.Context, string, string, int) ([]ConnectionRevision, error) {
	return nil, ErrCatalogStorage
}
func (unavailableCatalogRepository) ListConnectionRevisions(context.Context, string, string, int) ([]ConnectionRevision, error) {
	return nil, ErrCatalogStorage
}
func (unavailableCatalogRepository) CreateViewRevision(context.Context, ViewRevision) (ViewRevision, error) {
	return ViewRevision{}, ErrCatalogStorage
}
func (unavailableCatalogRepository) ViewRevision(context.Context, string, string, int64) (ViewRevision, error) {
	return ViewRevision{}, ErrCatalogStorage
}
func (unavailableCatalogRepository) CurrentView(context.Context, string, string) (ViewRevision, error) {
	return ViewRevision{}, ErrCatalogStorage
}
func (unavailableCatalogRepository) ListCurrentViews(context.Context, string, string, int) ([]ViewRevision, error) {
	return nil, ErrCatalogStorage
}
func (unavailableCatalogRepository) ListViewRevisions(context.Context, string, string, int) ([]ViewRevision, error) {
	return nil, ErrCatalogStorage
}
func (unavailableCatalogRepository) CreateBindingRevision(context.Context, BindingRevision) (BindingRevision, error) {
	return BindingRevision{}, ErrCatalogStorage
}
func (unavailableCatalogRepository) BindingRevision(context.Context, string, string, int64) (BindingRevision, error) {
	return BindingRevision{}, ErrCatalogStorage
}
func (unavailableCatalogRepository) CurrentBinding(context.Context, string, string) (BindingRevision, error) {
	return BindingRevision{}, ErrCatalogStorage
}
func (unavailableCatalogRepository) ListCurrentBindings(context.Context, string, string, int) ([]BindingRevision, error) {
	return nil, ErrCatalogStorage
}
func (unavailableCatalogRepository) ListBindingRevisions(context.Context, string, string, int) ([]BindingRevision, error) {
	return nil, ErrCatalogStorage
}
