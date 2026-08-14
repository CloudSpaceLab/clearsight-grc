package sourceaccess

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

const (
	hardMaxCatalogCodeBytes = 128
	hardMaxCatalogNameBytes = 512
)

func normalizeConnectionRevision(value ConnectionRevision) (ConnectionRevision, error) {
	value.Code = strings.TrimSpace(value.Code)
	value.Name = strings.TrimSpace(value.Name)
	value.AdapterKind = AdapterKind(strings.TrimSpace(string(value.AdapterKind)))
	value.AdapterVersion = strings.TrimSpace(value.AdapterVersion)
	value.SecretRef = strings.TrimSpace(value.SecretRef)
	value.OwnerPrincipalID = strings.TrimSpace(value.OwnerPrincipalID)
	value.DeclaredCapabilities = append([]Capability(nil), value.DeclaredCapabilities...)
	value.VerifiedCapabilities = append([]Capability(nil), value.VerifiedCapabilities...)
	value.Definition = defaultJSONObject(value.Definition)
	normalizeLifecycle(&value.RevisionLifecycle)
	if err := validateCatalogIdentity(value.RevisionID, value.ConnectionID, value.TenantID, value.SourceID, value.Code, value.Name, value.RevisionLifecycle); err != nil {
		return ConnectionRevision{}, err
	}
	if strings.TrimSpace(string(value.AdapterKind)) == "" || value.AdapterVersion == "" || len(value.AdapterVersion) > hardMaxAdapterVersionLen || containsControl(value.AdapterVersion) {
		return ConnectionRevision{}, catalogInvalid("adapter kind and adapter version are required")
	}
	if value.SecretRef != "" && (len(value.SecretRef) > HardMaxIdentifierBytes || containsControl(value.SecretRef)) {
		return ConnectionRevision{}, catalogInvalid("secret reference is invalid")
	}
	if value.OwnerPrincipalID != "" {
		if err := validateOpaqueID(value.OwnerPrincipalID, "owner principal id"); err != nil {
			return ConnectionRevision{}, catalogInvalid("owner principal id is invalid")
		}
	}
	definition, err := normalizeJSONObject(value.Definition, HardMaxDefinitionBytes, "connection definition")
	if err != nil {
		return ConnectionRevision{}, err
	}
	value.Definition = definition
	declared, err := normalizeCapabilities(value.DeclaredCapabilities)
	if err != nil {
		return ConnectionRevision{}, err
	}
	verified, err := normalizeCapabilities(value.VerifiedCapabilities)
	if err != nil {
		return ConnectionRevision{}, err
	}
	for _, capability := range verified {
		if !containsCapability(declared, capability) {
			return ConnectionRevision{}, catalogInvalid("verified capabilities must be declared")
		}
	}
	if value.AdapterKind == AdapterReference {
		if value.AdapterVersion != ReferenceAdapterVersion || value.SecretRef != "" || len(declared) != 0 || len(verified) != 0 {
			return ConnectionRevision{}, catalogInvalid("reference connections cannot carry credentials or execution capabilities")
		}
		var reference map[string]any
		if err := json.Unmarshal(value.Definition, &reference); err != nil {
			return ConnectionRevision{}, catalogInvalid("reference connection definition is invalid")
		}
		endpoint, ok := reference["endpoint"].(string)
		endpoint = strings.TrimSpace(endpoint)
		if !ok || endpoint == "" || containsControl(endpoint) {
			return ConnectionRevision{}, catalogInvalid("reference connections require a bounded endpoint without control characters")
		}
		reference["endpoint"] = endpoint
		encoded, err := json.Marshal(reference)
		if err != nil || len(encoded) > HardMaxDefinitionBytes {
			return ConnectionRevision{}, catalogInvalid("reference connection definition exceeds the size limit")
		}
		value.Definition = encoded
	}
	value.DeclaredCapabilities = declared
	value.VerifiedCapabilities = verified
	return value, nil
}

func normalizeViewRevision(value ViewRevision) (ViewRevision, error) {
	value.Code = strings.TrimSpace(value.Code)
	value.Name = strings.TrimSpace(value.Name)
	value.StableKeys = append([]string(nil), value.StableKeys...)
	value.NativeSchema = append([]NativeField(nil), value.NativeSchema...)
	value.Definition = cloneRawMessage(value.Definition)
	value.SchemaFingerprint = strings.TrimSpace(value.SchemaFingerprint)
	normalizeLifecycle(&value.RevisionLifecycle)
	if err := validateCatalogIdentity(value.RevisionID, value.ViewID, value.TenantID, value.SourceID, value.Code, value.Name, value.RevisionLifecycle); err != nil {
		return ViewRevision{}, err
	}
	if err := validateOpaqueID(value.ConnectionID, "connection id"); err != nil || value.ConnectionVersion < 1 {
		return ViewRevision{}, catalogInvalid("connection id and version are required")
	}
	if value.OutputKind != OutputRecords {
		return ViewRevision{}, catalogInvalid("view output kind is unsupported")
	}
	definition, err := normalizeJSONObject(value.Definition, HardMaxDefinitionBytes, "view definition")
	if err != nil {
		return ViewRevision{}, err
	}
	value.Definition = definition
	if len(value.StableKeys) > HardMaxStableKeyFields {
		return ViewRevision{}, catalogInvalid("too many stable keys")
	}
	if err := validateFields(value.StableKeys, HardMaxStableKeyFields); err != nil {
		return ViewRevision{}, errors.Join(ErrCatalogInvalid, err)
	}
	if len(value.NativeSchema) > HardMaxSchemaFields {
		return ViewRevision{}, catalogInvalid("source schema has too many fields")
	}
	seen := make(map[string]struct{}, len(value.NativeSchema))
	for index := range value.NativeSchema {
		field := &value.NativeSchema[index]
		field.NativeType = strings.TrimSpace(field.NativeType)
		if !ValidFieldName(field.Name) || field.NativeType == "" || len(field.NativeType) > HardMaxIdentifierBytes || containsControl(field.NativeType) {
			return ViewRevision{}, catalogInvalid("source schema contains an invalid field")
		}
		if _, exists := seen[field.Name]; exists {
			return ViewRevision{}, catalogInvalid("source schema contains duplicate fields")
		}
		seen[field.Name] = struct{}{}
	}
	if value.SchemaFingerprint != "" && !isLowerHex(value.SchemaFingerprint, 64) {
		return ViewRevision{}, catalogInvalid("schema fingerprint must be a lowercase SHA-256 value")
	}
	if value.IsCurrent && (value.SchemaFingerprint == "" || len(value.NativeSchema) == 0) {
		return ViewRevision{}, catalogInvalid("current views require an inspected schema")
	}
	return value, nil
}

func normalizeBindingRevision(value BindingRevision) (BindingRevision, error) {
	value.Code = strings.TrimSpace(value.Code)
	value.Name = strings.TrimSpace(value.Name)
	value.Purpose = strings.TrimSpace(value.Purpose)
	value.Operations = append([]Operation(nil), value.Operations...)
	value.SelectedFields = append([]string(nil), value.SelectedFields...)
	value.KeyFields = append([]string(nil), value.KeyFields...)
	value.Mapping = defaultJSONObject(value.Mapping)
	value.ParameterSchema = defaultJSONObject(value.ParameterSchema)
	value.OutputSchema = defaultJSONObject(value.OutputSchema)
	value.SensitivityHandling = defaultJSONObject(value.SensitivityHandling)
	if value.Completeness == "" {
		value.Completeness = CompletenessRequireFull
	}
	normalizeLifecycle(&value.RevisionLifecycle)
	if err := validateCatalogIdentity(value.RevisionID, value.BindingID, value.TenantID, value.SourceID, value.Code, value.Name, value.RevisionLifecycle); err != nil {
		return BindingRevision{}, err
	}
	if err := validateOpaqueID(value.ViewID, "view id"); err != nil || value.ViewVersion < 1 {
		return BindingRevision{}, catalogInvalid("view id and version are required")
	}
	if value.Purpose == "" || len(value.Purpose) > HardMaxIdentifierBytes || containsControl(value.Purpose) {
		return BindingRevision{}, catalogInvalid("binding purpose is invalid")
	}
	if len(value.Operations) == 0 || len(value.Operations) > 5 {
		return BindingRevision{}, catalogInvalid("binding operations are invalid")
	}
	seenOperations := make(map[Operation]struct{}, len(value.Operations))
	for _, operation := range value.Operations {
		if !validOperation(operation) {
			return BindingRevision{}, catalogInvalid("binding operation is unsupported")
		}
		if _, exists := seenOperations[operation]; exists {
			return BindingRevision{}, catalogInvalid("binding operations contain duplicates")
		}
		seenOperations[operation] = struct{}{}
	}
	if len(value.SelectedFields) == 0 || len(value.SelectedFields) > HardMaxSelectedFields {
		return BindingRevision{}, catalogInvalid("selected fields are invalid")
	}
	if err := validateFields(value.SelectedFields, HardMaxSelectedFields); err != nil {
		return BindingRevision{}, errors.Join(ErrCatalogInvalid, err)
	}
	if len(value.KeyFields) > HardMaxStableKeyFields {
		return BindingRevision{}, catalogInvalid("too many key fields")
	}
	if err := validateFields(value.KeyFields, HardMaxStableKeyFields); err != nil {
		return BindingRevision{}, errors.Join(ErrCatalogInvalid, err)
	}
	limits, err := value.Limits.Normalized()
	if err != nil {
		return BindingRevision{}, errors.Join(ErrCatalogInvalid, err)
	}
	value.Limits = limits
	for label, document := range map[string]json.RawMessage{
		"binding mapping":      value.Mapping,
		"parameter schema":     value.ParameterSchema,
		"output schema":        value.OutputSchema,
		"sensitivity handling": value.SensitivityHandling,
	} {
		normalized, normalizeErr := normalizeJSONObject(document, HardMaxDefinitionBytes, label)
		if normalizeErr != nil {
			return BindingRevision{}, normalizeErr
		}
		switch label {
		case "binding mapping":
			value.Mapping = normalized
		case "parameter schema":
			value.ParameterSchema = normalized
		case "output schema":
			value.OutputSchema = normalized
		case "sensitivity handling":
			value.SensitivityHandling = normalized
		}
	}
	if value.RequiredFreshnessMinutes < 0 || value.RequiredFreshnessMinutes > 525600 {
		return BindingRevision{}, catalogInvalid("required freshness is outside the supported range")
	}
	if value.Completeness != CompletenessAllowPartial && value.Completeness != CompletenessRequireFull {
		return BindingRevision{}, catalogInvalid("completeness requirement is unsupported")
	}
	return value, nil
}

func validateBindingAgainstView(binding BindingRevision, view ViewRevision) (BindingRevision, error) {
	binding, err := normalizeBindingRevision(binding)
	if err != nil {
		return BindingRevision{}, err
	}
	view, err = normalizeViewRevision(view)
	if err != nil {
		return BindingRevision{}, err
	}
	if binding.TenantID != view.TenantID || binding.SourceID != view.SourceID || binding.ViewID != view.ViewID || binding.ViewVersion != view.Version {
		return BindingRevision{}, catalogInvalid("binding does not match its view revision")
	}
	contract := Binding{
		ID:             binding.BindingID,
		ViewID:         binding.ViewID,
		Version:        "catalog",
		Purpose:        binding.Purpose,
		Operations:     binding.Operations,
		SelectedFields: binding.SelectedFields,
		KeyFields:      binding.KeyFields,
		Limits:         binding.Limits,
	}
	viewContract := View{
		ID:           view.ViewID,
		ConnectionID: view.ConnectionID,
		Version:      "catalog",
		OutputKind:   view.OutputKind,
		Definition:   view.Definition,
		StableKeys:   view.StableKeys,
	}
	if err := contract.Validate(viewContract); err != nil {
		return BindingRevision{}, errors.Join(ErrCatalogInvalid, err)
	}
	available := make(map[string]struct{}, len(view.NativeSchema))
	for _, field := range view.NativeSchema {
		available[field.Name] = struct{}{}
	}
	for _, field := range binding.SelectedFields {
		if _, exists := available[field]; !exists {
			return BindingRevision{}, catalogInvalid("binding selects a field outside the view schema")
		}
	}
	return binding, nil
}

func validateViewAgainstConnection(view ViewRevision, connection ConnectionRevision) (ViewRevision, error) {
	view, err := normalizeViewRevision(view)
	if err != nil {
		return ViewRevision{}, err
	}
	connection, err = normalizeConnectionRevision(connection)
	if err != nil {
		return ViewRevision{}, err
	}
	if view.TenantID != connection.TenantID || view.SourceID != connection.SourceID || view.ConnectionID != connection.ConnectionID || view.ConnectionVersion != connection.Version {
		return ViewRevision{}, catalogInvalid("view does not match its connection revision")
	}
	if connection.AdapterKind == AdapterReference {
		return ViewRevision{}, catalogInvalid("reference connections cannot own executable views")
	}
	return view, nil
}

func validateCatalogIdentity(revisionID, resourceID, tenantID, sourceID, code, name string, lifecycle RevisionLifecycle) error {
	for label, value := range map[string]string{
		"revision id": revisionID,
		"resource id": resourceID,
		"tenant id":   tenantID,
		"source id":   sourceID,
	} {
		if err := validateOpaqueID(value, label); err != nil {
			return catalogInvalid(label + " is invalid")
		}
	}
	if code == "" || len(code) > hardMaxCatalogCodeBytes || containsControl(code) {
		return catalogInvalid("catalog code is invalid")
	}
	if name == "" || len(name) > hardMaxCatalogNameBytes || containsControl(name) {
		return catalogInvalid("catalog name is invalid")
	}
	return validateRevisionLifecycle(lifecycle)
}

func validateRevisionLifecycle(value RevisionLifecycle) error {
	if value.Version < 1 || value.CreatedAt.IsZero() || value.UpdatedAt.IsZero() || value.UpdatedAt.Before(value.CreatedAt) {
		return catalogInvalid("revision version and timestamps are invalid")
	}
	if value.CreatedBy != "" {
		if err := validateOpaqueID(value.CreatedBy, "creator principal id"); err != nil {
			return catalogInvalid("creator principal id is invalid")
		}
	}
	switch value.Status {
	case RevisionActive, RevisionPaused:
		if !value.IsCurrent || value.EffectiveFrom == nil || value.EffectiveUntil != nil {
			return catalogInvalid("active and paused revisions must be current and open-ended")
		}
	case RevisionRetired:
		if value.IsCurrent || value.EffectiveFrom == nil || value.EffectiveUntil == nil || value.EffectiveUntil.Before(*value.EffectiveFrom) {
			return catalogInvalid("retired revisions require a closed effective period")
		}
	case RevisionDraft, RevisionPendingApproval, RevisionRejected:
		if value.IsCurrent || value.EffectiveFrom != nil || value.EffectiveUntil != nil {
			return catalogInvalid("unapproved revisions cannot be current or effective")
		}
	default:
		return catalogInvalid("revision status is unsupported")
	}
	return nil
}

func normalizeLifecycle(value *RevisionLifecycle) {
	value.CreatedBy = strings.TrimSpace(value.CreatedBy)
	value.CreatedAt = value.CreatedAt.UTC()
	value.UpdatedAt = value.UpdatedAt.UTC()
	if value.EffectiveFrom != nil {
		normalized := value.EffectiveFrom.UTC()
		value.EffectiveFrom = &normalized
	}
	if value.EffectiveUntil != nil {
		normalized := value.EffectiveUntil.UTC()
		value.EffectiveUntil = &normalized
	}
}

func normalizeCapabilities(values []Capability) ([]Capability, error) {
	seen := make(map[Capability]struct{}, len(values))
	result := make([]Capability, 0, len(values))
	for _, value := range values {
		if !validCapability(value) {
			return nil, catalogInvalid("capability is unsupported")
		}
		if _, exists := seen[value]; exists {
			return nil, catalogInvalid("capabilities contain duplicates")
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result, nil
}

func validCapability(value Capability) bool {
	switch value {
	case CapabilityInspect, CapabilityPage, CapabilityLookup, CapabilityAggregate, CapabilityChanges:
		return true
	default:
		return false
	}
}

func containsCapability(values []Capability, target Capability) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func normalizeJSONObject(value json.RawMessage, maximum int, label string) (json.RawMessage, error) {
	if len(value) == 0 || len(value) > maximum {
		return nil, catalogInvalid(label + " is missing or too large")
	}
	decoder := json.NewDecoder(bytes.NewReader(value))
	decoder.UseNumber()
	var decoded map[string]any
	if err := decoder.Decode(&decoded); err != nil || decoded == nil {
		return nil, catalogInvalid(label + " must be a JSON object")
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, catalogInvalid(label + " contains trailing data")
	}
	encoded, err := json.Marshal(decoded)
	if err != nil || len(encoded) > maximum {
		return nil, catalogInvalid(label + " cannot be encoded within the size limit")
	}
	return encoded, nil
}

func defaultJSONObject(value json.RawMessage) json.RawMessage {
	if len(bytes.TrimSpace(value)) == 0 {
		return json.RawMessage(`{}`)
	}
	return cloneRawMessage(value)
}

func cloneRawMessage(value json.RawMessage) json.RawMessage {
	return append(json.RawMessage(nil), value...)
}

func isLowerHex(value string, expected int) bool {
	if len(value) != expected {
		return false
	}
	for _, character := range value {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

func catalogInvalid(message string) error {
	return fmt.Errorf("%w: %s", ErrCatalogInvalid, message)
}

func catalogListLimit(value int) int {
	if value <= 0 {
		return 100
	}
	if value > HardMaxCatalogListRows {
		return HardMaxCatalogListRows
	}
	return value
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
