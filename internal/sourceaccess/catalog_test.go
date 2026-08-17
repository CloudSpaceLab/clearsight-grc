package sourceaccess

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

const (
	catalogTenantID     = "11111111-1111-7111-8111-111111111111"
	catalogSourceID     = "12222222-2222-7222-8222-222222222222"
	catalogConnectionID = "13333333-3333-7333-8333-333333333333"
	catalogViewID       = "14444444-4444-7444-8444-444444444444"
	catalogBindingID    = "15555555-5555-7555-8555-555555555555"
	catalogActorID      = "16666666-6666-7666-8666-666666666666"
)

func TestMemoryCatalogPersistsExactReusableRevisions(t *testing.T) {
	ctx := context.Background()
	repository := NewMemoryCatalogRepository([]SourceScope{{TenantID: catalogTenantID, SourceID: catalogSourceID}})
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)

	connection := catalogConnectionRevision(now)
	createdConnection, err := repository.CreateConnectionRevision(ctx, connection)
	if err != nil {
		t.Fatal(err)
	}
	if createdConnection.Version != 1 || createdConnection.AdapterKind != AdapterPostgres || !createdConnection.IsCurrent {
		t.Fatalf("unexpected connection revision: %#v", createdConnection)
	}

	view := catalogViewRevision(now)
	createdView, err := repository.CreateViewRevision(ctx, view)
	if err != nil {
		t.Fatal(err)
	}
	binding := catalogBindingRevision(now)
	createdBinding, err := repository.CreateBindingRevision(ctx, binding)
	if err != nil {
		t.Fatal(err)
	}

	connectionContract, err := createdConnection.Contract()
	if err != nil {
		t.Fatal(err)
	}
	viewContract, err := createdView.Contract(createdConnection)
	if err != nil {
		t.Fatal(err)
	}
	bindingContract, err := createdBinding.Contract(createdView)
	if err != nil {
		t.Fatal(err)
	}
	if connectionContract.ID != catalogConnectionID || viewContract.ConnectionID != catalogConnectionID || bindingContract.ViewID != catalogViewID || bindingContract.Limits.PageRows != 25 {
		t.Fatalf("compiled contracts lost catalog identity: connection=%#v view=%#v binding=%#v", connectionContract, viewContract, bindingContract)
	}

	connections, err := repository.ListCurrentConnections(ctx, catalogTenantID, catalogSourceID, 10)
	if err != nil || len(connections) != 1 {
		t.Fatalf("connections=%#v err=%v", connections, err)
	}
	views, err := repository.ListCurrentViews(ctx, catalogTenantID, catalogConnectionID, 10)
	if err != nil || len(views) != 1 {
		t.Fatalf("views=%#v err=%v", views, err)
	}
	bindings, err := repository.ListCurrentBindings(ctx, catalogTenantID, catalogViewID, 10)
	if err != nil || len(bindings) != 1 {
		t.Fatalf("bindings=%#v err=%v", bindings, err)
	}

	connections[0].Definition[0] = '['
	bindings[0].SelectedFields[0] = "changed"
	reloadedConnection, err := repository.CurrentConnection(ctx, catalogTenantID, catalogConnectionID)
	if err != nil {
		t.Fatal(err)
	}
	reloadedBinding, err := repository.CurrentBinding(ctx, catalogTenantID, catalogBindingID)
	if err != nil {
		t.Fatal(err)
	}
	if string(reloadedConnection.Definition) != `{}` || reloadedBinding.SelectedFields[0] != "account_id" {
		t.Fatal("catalog returned mutable internal state")
	}
}

func TestMemoryCatalogRejectsCrossScopeAndCurrentParentViolations(t *testing.T) {
	ctx := context.Background()
	repository := NewMemoryCatalogRepository([]SourceScope{{TenantID: catalogTenantID, SourceID: catalogSourceID}})
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	if _, err := repository.CreateConnectionRevision(ctx, catalogConnectionRevision(now)); err != nil {
		t.Fatal(err)
	}

	duplicateCode := catalogConnectionRevision(now)
	duplicateCode.RevisionID = "17777777-7777-7777-8777-777777777777"
	duplicateCode.ConnectionID = "18888888-8888-7888-8888-888888888888"
	if _, err := repository.CreateConnectionRevision(ctx, duplicateCode); !errors.Is(err, ErrCatalogConflict) {
		t.Fatalf("duplicate current connection code should fail, got %v", err)
	}

	missingSource := catalogConnectionRevision(now)
	missingSource.RevisionID = "19999999-9999-7999-8999-999999999999"
	missingSource.ConnectionID = "21111111-1111-7111-8111-111111111111"
	missingSource.SourceID = "22222222-2222-7222-8222-222222222222"
	if _, err := repository.CreateConnectionRevision(ctx, missingSource); !errors.Is(err, ErrCatalogNotFound) {
		t.Fatalf("missing source should fail, got %v", err)
	}

	historical := catalogConnectionRevision(now)
	historical.RevisionID = "23333333-3333-7333-8333-333333333333"
	historical.Version = 2
	historical.Status = RevisionRetired
	historical.IsCurrent = false
	historical.EffectiveUntil = timePointer(now.Add(time.Hour))
	if _, err := repository.CreateConnectionRevision(ctx, historical); err != nil {
		t.Fatal(err)
	}
	view := catalogViewRevision(now)
	view.RevisionID = "24444444-4444-7444-8444-444444444444"
	view.ViewID = "25555555-5555-7555-8555-555555555555"
	view.ConnectionVersion = 2
	if _, err := repository.CreateViewRevision(ctx, view); !errors.Is(err, ErrCatalogNotFound) {
		t.Fatalf("current view over historical connection should fail, got %v", err)
	}

	reference := catalogConnectionRevision(now)
	reference.RevisionID = "26666666-6666-7666-8666-666666666666"
	reference.ConnectionID = "27777777-7777-7777-8777-777777777777"
	reference.Code = ReferenceConnectionCode
	reference.Name = ReferenceConnectionName
	reference.AdapterKind = AdapterReference
	reference.AdapterVersion = ReferenceAdapterVersion
	reference.SecretRef = ""
	reference.Definition = json.RawMessage(`{"endpoint":"https://example.invalid/source"}`)
	reference.DeclaredCapabilities = nil
	reference.VerifiedCapabilities = nil
	if _, err := repository.CreateConnectionRevision(ctx, reference); err != nil {
		t.Fatal(err)
	}
	referenceView := catalogViewRevision(now)
	referenceView.RevisionID = "28888888-8888-7888-8888-888888888888"
	referenceView.ViewID = "29999999-9999-7999-8999-999999999999"
	referenceView.ConnectionID = reference.ConnectionID
	if _, err := repository.CreateViewRevision(ctx, referenceView); !errors.Is(err, ErrCatalogInvalid) {
		t.Fatalf("reference connection should not own a view, got %v", err)
	}
}

func TestCatalogBindingMustStayWithinTheViewSchema(t *testing.T) {
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	view := catalogViewRevision(now)
	binding := catalogBindingRevision(now)
	binding.SelectedFields = append(binding.SelectedFields, "unprojected_field")
	if _, err := validateBindingAgainstView(binding, view); !errors.Is(err, ErrCatalogInvalid) {
		t.Fatalf("unprojected binding field should fail, got %v", err)
	}
}

func TestCatalogNormalizesJSONCapabilitiesAndLimits(t *testing.T) {
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	connection := catalogConnectionRevision(now)
	connection.Definition = json.RawMessage(" { \"database\" : \"risk\" } ")
	connection.DeclaredCapabilities = []Capability{CapabilityLookup, CapabilityInspect}
	connection.VerifiedCapabilities = []Capability{CapabilityInspect}
	normalized, err := normalizeConnectionRevision(connection)
	if err != nil {
		t.Fatal(err)
	}
	if string(normalized.Definition) != `{"database":"risk"}` || normalized.DeclaredCapabilities[0] != CapabilityInspect {
		t.Fatalf("connection was not normalized: %#v", normalized)
	}
	binding := catalogBindingRevision(now)
	binding.Limits = ResourceLimits{}
	normalizedBinding, err := normalizeBindingRevision(binding)
	if err != nil {
		t.Fatal(err)
	}
	if normalizedBinding.Limits != DefaultResourceLimits() || normalizedBinding.Completeness != CompletenessRequireFull {
		t.Fatalf("binding defaults were not fixed at revision creation: %#v", normalizedBinding)
	}
}

func catalogConnectionRevision(now time.Time) ConnectionRevision {
	return ConnectionRevision{
		RevisionID:           "31111111-1111-7111-8111-111111111111",
		ConnectionID:         catalogConnectionID,
		TenantID:             catalogTenantID,
		SourceID:             catalogSourceID,
		Code:                 "RISK_DATABASE",
		Name:                 "Risk database",
		AdapterKind:          AdapterPostgres,
		AdapterVersion:       PostgresAdapterVersion,
		SecretRef:            "secret://risk/reader",
		Definition:           json.RawMessage(`{}`),
		DeclaredCapabilities: []Capability{CapabilityInspect, CapabilityPage, CapabilityLookup, CapabilityAggregate},
		VerifiedCapabilities: []Capability{CapabilityInspect, CapabilityLookup, CapabilityAggregate},
		OwnerPrincipalID:     catalogActorID,
		RevisionLifecycle:    activeCatalogLifecycle(now, 1),
	}
}

func catalogViewRevision(now time.Time) ViewRevision {
	return ViewRevision{
		RevisionID:        "32222222-2222-7222-8222-222222222222",
		ViewID:            catalogViewID,
		TenantID:          catalogTenantID,
		SourceID:          catalogSourceID,
		ConnectionID:      catalogConnectionID,
		ConnectionVersion: 1,
		Code:              "ACTIVE_ACCOUNTS",
		Name:              "Active accounts",
		Definition:        json.RawMessage(`{"query":"SELECT account_id,status,balance FROM active_accounts"}`),
		OutputKind:        OutputRecords,
		StableKeys:        []string{"account_id"},
		NativeSchema: []NativeField{
			{Name: "account_id", NativeType: "uuid", Nullable: false},
			{Name: "status", NativeType: "text", Nullable: false},
			{Name: "balance", NativeType: "numeric", Nullable: false},
		},
		SchemaFingerprint: strings.Repeat("a", 64),
		RevisionLifecycle: activeCatalogLifecycle(now, 1),
	}
}

func catalogBindingRevision(now time.Time) BindingRevision {
	return BindingRevision{
		RevisionID:               "33333333-3333-7333-8333-333333333333",
		BindingID:                catalogBindingID,
		TenantID:                 catalogTenantID,
		SourceID:                 catalogSourceID,
		ViewID:                   catalogViewID,
		ViewVersion:              1,
		Code:                     "ACCOUNT_STATUS_LOOKUP",
		Name:                     "Account status lookup",
		Purpose:                  "account-status-validation",
		Operations:               []Operation{OperationLookup, OperationPage, OperationAggregate},
		SelectedFields:           []string{"account_id", "status", "balance"},
		KeyFields:                []string{"account_id"},
		Limits:                   ResourceLimits{PageRows: 25, ResponseBytes: 64 << 10, LookupValues: 10, Timeout: 2 * time.Second},
		Mapping:                  json.RawMessage(`{"account_id":"account_id"}`),
		ParameterSchema:          json.RawMessage(`{"type":"object"}`),
		OutputSchema:             json.RawMessage(`{"type":"object"}`),
		RequiredFreshnessMinutes: 15,
		Completeness:             CompletenessRequireFull,
		SensitivityHandling:      json.RawMessage(`{"classification":"RESTRICTED"}`),
		RevisionLifecycle:        activeCatalogLifecycle(now, 1),
	}
}

func activeCatalogLifecycle(now time.Time, version int64) RevisionLifecycle {
	return RevisionLifecycle{
		Status:        RevisionActive,
		IsCurrent:     true,
		EffectiveFrom: timePointer(now),
		Version:       version,
		CreatedBy:     catalogActorID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func timePointer(value time.Time) *time.Time {
	return &value
}
