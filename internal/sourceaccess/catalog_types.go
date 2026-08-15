package sourceaccess

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"
)

const (
	AdapterReference        AdapterKind = "REFERENCE"
	ReferenceAdapterVersion             = "reference-v1"
	ReferenceConnectionCode             = "PRIMARY_REFERENCE"
	ReferenceConnectionName             = "Primary reference"
	HardMaxCatalogListRows              = 500
)

type RevisionStatus string

const (
	RevisionDraft           RevisionStatus = "DRAFT"
	RevisionPendingApproval RevisionStatus = "PENDING_APPROVAL"
	RevisionActive          RevisionStatus = "ACTIVE"
	RevisionPaused          RevisionStatus = "PAUSED"
	RevisionRejected        RevisionStatus = "REJECTED"
	RevisionRetired         RevisionStatus = "RETIRED"
)

type CompletenessRequirement string

const (
	CompletenessAllowPartial CompletenessRequirement = "ALLOW_PARTIAL"
	CompletenessRequireFull  CompletenessRequirement = "REQUIRE_COMPLETE"
)

var (
	ErrCatalogNotFound = errors.New("source catalog object not found")
	ErrCatalogConflict = errors.New("source catalog version conflict")
	ErrCatalogInvalid  = errors.New("source catalog object is invalid")
	ErrCatalogStorage  = errors.New("source catalog storage failed")
)

type SourceScope struct {
	TenantID string `json:"tenant_id"`
	SourceID string `json:"source_id"`
}

type RevisionLifecycle struct {
	Status         RevisionStatus `json:"status"`
	IsCurrent      bool           `json:"is_current"`
	EffectiveFrom  *time.Time     `json:"effective_from,omitempty"`
	EffectiveUntil *time.Time     `json:"effective_until,omitempty"`
	Version        int64          `json:"version"`
	CreatedBy      string         `json:"created_by,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type ConnectionRevision struct {
	RevisionID           string          `json:"revision_id"`
	ConnectionID         string          `json:"connection_id"`
	TenantID             string          `json:"tenant_id"`
	SourceID             string          `json:"source_id"`
	Code                 string          `json:"code"`
	Name                 string          `json:"name"`
	AdapterKind          AdapterKind     `json:"adapter_kind"`
	AdapterVersion       string          `json:"adapter_version"`
	SecretRef            string          `json:"secret_ref,omitempty"`
	Definition           json.RawMessage `json:"definition"`
	DeclaredCapabilities []Capability    `json:"declared_capabilities"`
	VerifiedCapabilities []Capability    `json:"verified_capabilities"`
	OwnerPrincipalID     string          `json:"owner_principal_id,omitempty"`
	RevisionLifecycle
}

type ViewRevision struct {
	RevisionID        string          `json:"revision_id"`
	ViewID            string          `json:"view_id"`
	TenantID          string          `json:"tenant_id"`
	SourceID          string          `json:"source_id"`
	ConnectionID      string          `json:"connection_id"`
	ConnectionVersion int64           `json:"connection_version"`
	Code              string          `json:"code"`
	Name              string          `json:"name"`
	Definition        json.RawMessage `json:"definition"`
	OutputKind        OutputKind      `json:"output_kind"`
	StableKeys        []string        `json:"stable_keys"`
	NativeSchema      []NativeField   `json:"native_schema"`
	SchemaFingerprint string          `json:"schema_fingerprint,omitempty"`
	RevisionLifecycle
}

type BindingRevision struct {
	RevisionID               string                  `json:"revision_id"`
	BindingID                string                  `json:"binding_id"`
	TenantID                 string                  `json:"tenant_id"`
	SourceID                 string                  `json:"source_id"`
	ViewID                   string                  `json:"view_id"`
	ViewVersion              int64                   `json:"view_version"`
	Code                     string                  `json:"code"`
	Name                     string                  `json:"name"`
	Purpose                  string                  `json:"purpose"`
	Operations               []Operation             `json:"operations"`
	SelectedFields           []string                `json:"selected_fields"`
	KeyFields                []string                `json:"key_fields,omitempty"`
	Limits                   ResourceLimits          `json:"limits"`
	Mapping                  json.RawMessage         `json:"mapping"`
	ParameterSchema          json.RawMessage         `json:"parameter_schema"`
	OutputSchema             json.RawMessage         `json:"output_schema"`
	RequiredFreshnessMinutes int                     `json:"required_freshness_minutes"`
	Completeness             CompletenessRequirement `json:"completeness"`
	SensitivityHandling      json.RawMessage         `json:"sensitivity_handling"`
	RevisionLifecycle
}

type CatalogRepository interface {
	CreateConnectionRevision(context.Context, ConnectionRevision) (ConnectionRevision, error)
	ConnectionRevision(context.Context, string, string, int64) (ConnectionRevision, error)
	CurrentConnection(context.Context, string, string) (ConnectionRevision, error)
	ListCurrentConnections(context.Context, string, string, int) ([]ConnectionRevision, error)
	ListConnectionRevisions(context.Context, string, string, int) ([]ConnectionRevision, error)

	CreateViewRevision(context.Context, ViewRevision) (ViewRevision, error)
	ViewRevision(context.Context, string, string, int64) (ViewRevision, error)
	CurrentView(context.Context, string, string) (ViewRevision, error)
	ListCurrentViews(context.Context, string, string, int) ([]ViewRevision, error)
	ListViewRevisions(context.Context, string, string, int) ([]ViewRevision, error)

	CreateBindingRevision(context.Context, BindingRevision) (BindingRevision, error)
	BindingRevision(context.Context, string, string, int64) (BindingRevision, error)
	CurrentBinding(context.Context, string, string) (BindingRevision, error)
	ListCurrentBindings(context.Context, string, string, int) ([]BindingRevision, error)
	ListBindingRevisions(context.Context, string, string, int) ([]BindingRevision, error)
}

func (value ConnectionRevision) Contract() (Connection, error) {
	value, err := normalizeConnectionRevision(value)
	if err != nil {
		return Connection{}, err
	}
	definition := cloneRawMessage(value.Definition)
	if string(definition) == "{}" {
		definition = nil
	}
	return Connection{
		ID:             value.ConnectionID,
		SourceID:       value.SourceID,
		Version:        strconv.FormatInt(value.Version, 10),
		AdapterKind:    value.AdapterKind,
		AdapterVersion: value.AdapterVersion,
		SecretRef:      value.SecretRef,
		Definition:     definition,
	}, nil
}

func (value ViewRevision) Contract(connection ConnectionRevision) (View, error) {
	value, err := validateViewAgainstConnection(value, connection)
	if err != nil {
		return View{}, err
	}
	contract := View{
		ID:                value.ViewID,
		ConnectionID:      value.ConnectionID,
		Version:           strconv.FormatInt(value.Version, 10),
		OutputKind:        value.OutputKind,
		Definition:        cloneRawMessage(value.Definition),
		StableKeys:        append([]string(nil), value.StableKeys...),
		NativeSchema:      append([]NativeField(nil), value.NativeSchema...),
		SchemaFingerprint: value.SchemaFingerprint,
	}
	connectionContract, err := connection.Contract()
	if err != nil {
		return View{}, err
	}
	if err := contract.Validate(connectionContract); err != nil {
		return View{}, errors.Join(ErrCatalogInvalid, err)
	}
	return contract, nil
}

func (value BindingRevision) Contract(view ViewRevision) (Binding, error) {
	value, err := validateBindingAgainstView(value, view)
	if err != nil {
		return Binding{}, err
	}
	contract := Binding{
		ID:             value.BindingID,
		ViewID:         value.ViewID,
		Version:        strconv.FormatInt(value.Version, 10),
		Purpose:        value.Purpose,
		Operations:     append([]Operation(nil), value.Operations...),
		SelectedFields: append([]string(nil), value.SelectedFields...),
		KeyFields:      append([]string(nil), value.KeyFields...),
		Limits:         value.Limits,
	}
	viewContract := View{
		ID:                view.ViewID,
		ConnectionID:      view.ConnectionID,
		Version:           strconv.FormatInt(view.Version, 10),
		OutputKind:        view.OutputKind,
		Definition:        cloneRawMessage(view.Definition),
		StableKeys:        append([]string(nil), view.StableKeys...),
		NativeSchema:      append([]NativeField(nil), view.NativeSchema...),
		SchemaFingerprint: view.SchemaFingerprint,
	}
	if err := contract.Validate(viewContract); err != nil {
		return Binding{}, errors.Join(ErrCatalogInvalid, err)
	}
	return contract, nil
}
