package sourceaccess

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

func TestCatalogServiceCreatesServerOwnedDrafts(t *testing.T) {
	repository := NewMemoryCatalogRepository([]SourceScope{{TenantID: catalogTenantID, SourceID: catalogSourceID}})
	service := NewCatalogService(repository, nil, nil)
	service.now = func() time.Time { return time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC) }
	ids := []string{
		"51111111-1111-7111-8111-111111111111",
		"52222222-2222-7222-8222-222222222222",
		"53333333-3333-7333-8333-333333333333",
		"54444444-4444-7444-8444-444444444444",
	}
	service.newID = func() (string, error) {
		value := ids[0]
		ids = ids[1:]
		return value, nil
	}
	actor := CatalogActor{TenantID: catalogTenantID, PrincipalID: catalogActorID}

	connection, err := service.CreateConnectionDraft(context.Background(), actor, catalogSourceID, CreateConnectionDraftInput{
		Code: "RISK_DATABASE", Name: "Risk database", AdapterKind: AdapterPostgres,
		AdapterVersion: PostgresAdapterVersion, SecretRef: "env://RISK_READER_DSN",
		DeclaredCapabilities: []Capability{CapabilityInspect, CapabilityPage},
	})
	if err != nil {
		t.Fatal(err)
	}
	if connection.TenantID != actor.TenantID || connection.SourceID != catalogSourceID || connection.CreatedBy != actor.PrincipalID || connection.OwnerPrincipalID != actor.PrincipalID {
		t.Fatalf("server scope was not applied: %#v", connection)
	}
	if connection.Status != RevisionDraft || connection.IsCurrent || connection.Version != 1 || connection.EffectiveFrom != nil || len(connection.VerifiedCapabilities) != 0 {
		t.Fatalf("connection was not created as an unverified draft: %#v", connection)
	}

	view, err := service.CreateViewDraft(context.Background(), actor, connection.ConnectionID, CreateViewDraftInput{
		ConnectionVersion: connection.Version, Code: "ACTIVE_ACCOUNTS", Name: "Active accounts",
		Definition: []byte(`{"query":"SELECT account_id,status FROM active_accounts"}`), StableKeys: []string{"account_id"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if view.ConnectionID != connection.ConnectionID || view.ConnectionVersion != connection.Version || view.SourceID != connection.SourceID || view.Status != RevisionDraft || view.IsCurrent {
		t.Fatalf("view parent scope or lifecycle changed: %#v", view)
	}
	if _, err := service.CreateBindingDraft(context.Background(), actor, view.ViewID, CreateBindingDraftInput{
		ViewVersion: view.Version, Code: "ACCOUNT_LOOKUP", Name: "Account lookup", Purpose: "validation",
		Operations: []Operation{OperationPage}, SelectedFields: []string{"account_id"}, KeyFields: []string{"account_id"},
	}); !errors.Is(err, ErrCatalogInvalid) {
		t.Fatalf("binding over an uninspected draft view should fail, got %v", err)
	}
}

func TestCatalogServiceInspectPreviewAndWhereUsedAreBounded(t *testing.T) {
	ctx := context.Background()
	repository := NewMemoryCatalogRepository([]SourceScope{{TenantID: catalogTenantID, SourceID: catalogSourceID}})
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	connection := catalogConnectionRevision(now)
	view := catalogViewRevision(now)
	binding := catalogBindingRevision(now)
	if _, err := repository.CreateConnectionRevision(ctx, connection); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.CreateViewRevision(ctx, view); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.CreateBindingRevision(ctx, binding); err != nil {
		t.Fatal(err)
	}

	session := &catalogFakeSession{connection: mustConnection(t, connection), capabilities: NewCapabilitySet(CapabilityInspect, CapabilityPage)}
	adapter := &catalogFakeAdapter{session: session}
	service := NewCatalogService(repository, catalogFakeSecrets{}, map[AdapterKind]Adapter{AdapterPostgres: adapter})

	schema, err := service.InspectView(ctx, catalogTenantID, view.ViewID, view.Version)
	if err != nil {
		t.Fatal(err)
	}
	if len(schema.Fields) != 1 || schema.Fields[0].Name != "account_id" || adapter.opens != 1 || session.closes != 1 {
		t.Fatalf("inspect result/session lifecycle changed: schema=%#v opens=%d closes=%d", schema, adapter.opens, session.closes)
	}

	page, err := service.PreviewBinding(ctx, catalogTenantID, binding.BindingID, binding.Version, PageRequest{Limit: 500})
	if err != nil {
		t.Fatal(err)
	}
	if session.pageLimit != binding.Limits.PageRows || len(page.Records) != binding.Limits.PageRows || adapter.opens != 2 || session.closes != 2 {
		t.Fatalf("preview was not bounded by the activated binding: limit=%d records=%d opens=%d closes=%d", session.pageLimit, len(page.Records), adapter.opens, session.closes)
	}

	connectionUsage, err := service.WhereUsed(ctx, catalogTenantID, UsageConnection, connection.ConnectionID, 20)
	if err != nil {
		t.Fatal(err)
	}
	viewUsage, err := service.WhereUsed(ctx, catalogTenantID, UsageView, view.ViewID, 20)
	if err != nil {
		t.Fatal(err)
	}
	bindingUsage, err := service.WhereUsed(ctx, catalogTenantID, UsageBinding, binding.BindingID, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(connectionUsage.Children) != 1 || connectionUsage.Children[0].ID != view.ViewID || len(viewUsage.Children) != 1 || viewUsage.Children[0].ID != binding.BindingID {
		t.Fatalf("catalog where-used hierarchy changed: connection=%#v view=%#v", connectionUsage, viewUsage)
	}
	if !bindingUsage.Complete || len(bindingUsage.Consumers) != 0 || len(bindingUsage.ConsumerDomains) != 5 {
		t.Fatalf("binding usage truth changed: %#v", bindingUsage)
	}
}

func TestCatalogServiceRejectsMissingAdapterAndUnsafeSecretReference(t *testing.T) {
	ctx := context.Background()
	repository := NewMemoryCatalogRepository([]SourceScope{{TenantID: catalogTenantID, SourceID: catalogSourceID}})
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	connection := catalogConnectionRevision(now)
	view := catalogViewRevision(now)
	if _, err := repository.CreateConnectionRevision(ctx, connection); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.CreateViewRevision(ctx, view); err != nil {
		t.Fatal(err)
	}
	service := NewCatalogService(repository, nil, nil)
	if _, err := service.InspectView(ctx, catalogTenantID, view.ViewID, view.Version); !errors.Is(err, ErrCapabilityUnavailable) {
		t.Fatalf("missing adapter boundary should be explicit, got %v", err)
	}

	resolver := EnvironmentSecretResolver{}
	if _, err := resolver.Resolve(ctx, "plain-secret-name"); !errors.Is(err, ErrCredentials) {
		t.Fatalf("plain secret reference should fail, got %v", err)
	}
	t.Setenv("CATALOG_TEST_DSN", "postgres://reader:secret@example.invalid/risk")
	value, err := resolver.Resolve(ctx, "env://CATALOG_TEST_DSN")
	if err != nil || value == "" {
		t.Fatalf("environment secret reference did not resolve: value=%q err=%v", value, err)
	}
	_ = os.Unsetenv("CATALOG_TEST_DSN")
}

func mustConnection(t *testing.T, revision ConnectionRevision) Connection {
	t.Helper()
	value, err := revision.Contract()
	if err != nil {
		t.Fatal(err)
	}
	return value
}

type catalogFakeSecrets struct{}

func (catalogFakeSecrets) Resolve(context.Context, string) (string, error) {
	return "secret", nil
}

type catalogFakeAdapter struct {
	session *catalogFakeSession
	opens   int
}

func (a *catalogFakeAdapter) Open(_ context.Context, connection Connection, _ SecretResolver) (Session, error) {
	a.opens++
	a.session.connection = connection
	return a.session, nil
}

type catalogFakeSession struct {
	connection   Connection
	capabilities CapabilitySet
	closes       int
	pageLimit    int
}

func (s *catalogFakeSession) Connection() Connection       { return s.connection }
func (s *catalogFakeSession) Capabilities() CapabilitySet   { return s.capabilities }
func (s *catalogFakeSession) Close() error                  { s.closes++; return nil }
func (s *catalogFakeSession) Inspect(_ context.Context, view View) (SchemaResult, error) {
	return SchemaResult{Fields: []NativeField{{Name: "account_id", NativeType: "uuid"}}, Receipt: OperationReceipt{SourceID: s.connection.SourceID, ConnectionID: s.connection.ID, ViewID: view.ID, Operation: OperationInspect, Completeness: CompletenessComplete}}, nil
}
func (s *catalogFakeSession) ReadPage(_ context.Context, view View, binding Binding, request PageRequest) (RecordPage, error) {
	s.pageLimit = request.Limit
	records := make([]Record, request.Limit)
	for index := range records {
		records[index] = Record{"account_id": StringValue("account")}
	}
	return RecordPage{Records: records, Receipt: OperationReceipt{SourceID: s.connection.SourceID, ConnectionID: s.connection.ID, ViewID: view.ID, BindingID: binding.ID, Operation: OperationPage, Count: int64(len(records)), Bytes: int64(len(records) * 16), Completeness: CompletenessPartial}}, nil
}
