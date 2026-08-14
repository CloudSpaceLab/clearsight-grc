package sourceaccess

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCatalogServiceCreatesInspectedServerOwnedDraftHierarchy(t *testing.T) {
	repository := NewMemoryCatalogRepository([]SourceScope{{TenantID: catalogTenantID, SourceID: catalogSourceID}})
	session := &catalogFakeSession{capabilities: NewCapabilitySet(CapabilityInspect, CapabilityPage)}
	adapter := &catalogFakeAdapter{session: session}
	service := NewCatalogService(repository, catalogFakeSecrets{}, map[AdapterKind]Adapter{AdapterPostgres: adapter})
	service.now = func() time.Time { return time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC) }
	ids := []string{
		"51111111-1111-7111-8111-111111111111",
		"52222222-2222-7222-8222-222222222222",
		"53333333-3333-7333-8333-333333333333",
		"54444444-4444-7444-8444-444444444444",
		"55555555-5555-7555-8555-555555555555",
		"56666666-6666-7666-8666-666666666666",
		"57777777-7777-7777-8777-777777777777",
	}
	service.newID = func() (string, error) {
		value := ids[0]
		ids = ids[1:]
		return value, nil
	}
	actor := CatalogActor{TenantID: catalogTenantID, PrincipalID: catalogActorID}

	connection, err := service.CreateConnectionDraft(context.Background(), actor, catalogSourceID, CreateConnectionDraftInput{
		Code: "RISK_DATABASE", Name: "Risk database", AdapterKind: AdapterPostgres,
		AdapterVersion: PostgresAdapterVersion, SecretRef: "vault://bank-a/source-readers/risk",
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
	connections, err := service.Connections(context.Background(), actor.TenantID, catalogSourceID, 20)
	if err != nil || len(connections) != 1 || connections[0].ConnectionID != connection.ConnectionID {
		t.Fatalf("created draft disappeared from configuration list: values=%#v err=%v", connections, err)
	}

	if _, err := service.CreateViewDraft(context.Background(), actor, connection.ConnectionID, CreateViewDraftInput{
		ConnectionVersion: connection.Version, Code: "INVALID", Name: "Invalid",
		Definition: []byte(`{"query":"SELECT account_id FROM active_accounts"}`), StableKeys: []string{"account_id"},
	}); !errors.Is(err, ErrCatalogInvalid) {
		t.Fatalf("unobserved stable keys should fail, got %v", err)
	}

	view, err := service.CreateViewDraft(context.Background(), actor, connection.ConnectionID, CreateViewDraftInput{
		ConnectionVersion: connection.Version, Code: "ACTIVE_ACCOUNTS", Name: "Active accounts",
		Definition: []byte(`{"query":"SELECT account_id,status FROM active_accounts"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if view.ConnectionID != connection.ConnectionID || view.ConnectionVersion != connection.Version || view.SourceID != connection.SourceID || view.Status != RevisionDraft || view.IsCurrent || len(view.NativeSchema) != 0 || len(view.StableKeys) != 0 {
		t.Fatalf("view parent scope or pre-inspection lifecycle changed: %#v", view)
	}

	inspected, err := service.InspectViewDraft(context.Background(), actor, view.ViewID, view.Version, InspectViewDraftInput{StableKeys: []string{"account_id"}})
	if err != nil {
		t.Fatal(err)
	}
	if inspected.View.ViewID != view.ViewID || inspected.View.Version != 2 || inspected.View.Status != RevisionDraft || inspected.View.IsCurrent || inspected.View.SchemaFingerprint == "" || len(inspected.View.NativeSchema) != 1 || len(inspected.View.StableKeys) != 1 {
		t.Fatalf("inspection did not create an immutable schema-bearing revision: %#v", inspected.View)
	}
	views, err := service.Views(context.Background(), actor.TenantID, connection.ConnectionID, 20)
	if err != nil || len(views) != 2 || views[0].Version != 2 || views[1].Version != 1 {
		t.Fatalf("View revision history is incomplete or unordered: values=%#v err=%v", views, err)
	}

	binding, err := service.CreateBindingDraft(context.Background(), actor, inspected.View.ViewID, CreateBindingDraftInput{
		ViewVersion: inspected.View.Version, Code: "ACCOUNT_PAGE", Name: "Account page", Purpose: "account-review",
		Operations: []Operation{OperationPage}, SelectedFields: []string{"account_id"}, KeyFields: []string{"account_id"},
		Limits: ResourceLimits{PageRows: 25, ResponseBytes: 64 << 10, LookupValues: 10, Timeout: 2 * time.Second},
	})
	if err != nil {
		t.Fatal(err)
	}
	if binding.ViewID != view.ViewID || binding.ViewVersion != inspected.View.Version || binding.Status != RevisionDraft || binding.IsCurrent {
		t.Fatalf("binding parent scope or lifecycle changed: %#v", binding)
	}
	bindings, err := service.Bindings(context.Background(), actor.TenantID, view.ViewID, 20)
	if err != nil || len(bindings) != 1 || bindings[0].BindingID != binding.BindingID {
		t.Fatalf("created Binding draft disappeared from configuration list: values=%#v err=%v", bindings, err)
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
	ids := []string{"58888888-8888-7888-8888-888888888888"}
	service.newID = func() (string, error) { value := ids[0]; ids = ids[1:]; return value, nil }
	service.now = func() time.Time { return now.Add(time.Minute) }

	inspected, err := service.InspectViewDraft(ctx, CatalogActor{TenantID: catalogTenantID, PrincipalID: catalogActorID}, view.ViewID, view.Version, InspectViewDraftInput{StableKeys: []string{"account_id"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(inspected.Schema.Fields) != 1 || inspected.Schema.Fields[0].Name != "account_id" || adapter.opens != 1 || session.closes != 1 {
		t.Fatalf("inspect result/session lifecycle changed: result=%#v opens=%d closes=%d", inspected, adapter.opens, session.closes)
	}

	page, err := service.PreviewBinding(ctx, catalogTenantID, binding.BindingID, binding.Version, PageRequest{Limit: 500})
	if err != nil {
		t.Fatal(err)
	}
	if session.pageLimit != binding.Limits.PageRows || len(page.Records) != binding.Limits.PageRows || adapter.opens != 2 || session.closes != 2 {
		t.Fatalf("preview was not bounded by the activated binding: limit=%d records=%d opens=%d closes=%d", session.pageLimit, len(page.Records), adapter.opens, session.closes)
	}

	retired := binding
	retired.RevisionID = "59999999-9999-7999-8999-999999999999"
	retired.Version = 2
	retired.Status = RevisionRetired
	retired.IsCurrent = false
	retired.EffectiveUntil = timePointer(now.Add(time.Hour))
	retired.UpdatedAt = now.Add(time.Hour)
	if _, err := repository.CreateBindingRevision(ctx, retired); err != nil {
		t.Fatal(err)
	}
	if _, err := service.PreviewBinding(ctx, catalogTenantID, binding.BindingID, retired.Version, PageRequest{}); !errors.Is(err, ErrCatalogInvalid) {
		t.Fatalf("retired Binding remained previewable, got %v", err)
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

func TestCatalogServiceRejectsOwnerSpoofingAndKeepsSecretReferencesOpaque(t *testing.T) {
	ctx := context.Background()
	repository := NewMemoryCatalogRepository([]SourceScope{{TenantID: catalogTenantID, SourceID: catalogSourceID}})
	adapter := &catalogFakeAdapter{session: &catalogFakeSession{}}
	service := NewCatalogService(repository, catalogFakeSecrets{}, map[AdapterKind]Adapter{AdapterPostgres: adapter})
	actor := CatalogActor{TenantID: catalogTenantID, PrincipalID: catalogActorID}
	if _, err := service.CreateConnectionDraft(ctx, actor, catalogSourceID, CreateConnectionDraftInput{
		Code: "SPOOFED_OWNER", Name: "Spoofed owner", AdapterKind: AdapterPostgres, AdapterVersion: PostgresAdapterVersion,
		SecretRef: "vault://bank-a/risk-reader", OwnerPrincipalID: "60000000-0000-7000-8000-000000000000",
	}); !errors.Is(err, ErrCatalogInvalid) {
		t.Fatalf("unverified owner override should fail, got %v", err)
	}
	created, err := service.CreateConnectionDraft(ctx, actor, catalogSourceID, CreateConnectionDraftInput{
		Code: "OPAQUE_SECRET", Name: "Opaque secret", AdapterKind: AdapterPostgres, AdapterVersion: PostgresAdapterVersion,
		SecretRef: "vault://bank-a/risk-reader",
	})
	if err != nil || created.SecretRef != "vault://bank-a/risk-reader" {
		t.Fatalf("opaque secret reference was not retained: value=%#v err=%v", created, err)
	}

	resolver := EnvironmentSecretResolver{}
	if _, err := resolver.Resolve(ctx, created.SecretRef); !errors.Is(err, ErrCredentials) {
		t.Fatalf("default environment resolver should not reinterpret opaque references, got %v", err)
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

func (s *catalogFakeSession) Connection() Connection      { return s.connection }
func (s *catalogFakeSession) Capabilities() CapabilitySet { return s.capabilities }
func (s *catalogFakeSession) Close() error                { s.closes++; return nil }
func (s *catalogFakeSession) Inspect(_ context.Context, view View) (SchemaResult, error) {
	fingerprint := strings.Repeat("a", 64)
	return SchemaResult{
		Fields:  []NativeField{{Name: "account_id", NativeType: "uuid"}},
		Receipt: OperationReceipt{SourceID: s.connection.SourceID, ConnectionID: s.connection.ID, ViewID: view.ID, Operation: OperationInspect, SchemaFingerprint: fingerprint, Completeness: CompletenessComplete},
	}, nil
}
func (s *catalogFakeSession) ReadPage(_ context.Context, view View, binding Binding, request PageRequest) (RecordPage, error) {
	s.pageLimit = request.Limit
	records := make([]Record, request.Limit)
	for index := range records {
		records[index] = Record{"account_id": StringValue("account")}
	}
	return RecordPage{Records: records, Receipt: OperationReceipt{SourceID: s.connection.SourceID, ConnectionID: s.connection.ID, ViewID: view.ID, BindingID: binding.ID, Operation: OperationPage, Count: int64(len(records)), Bytes: int64(len(records) * 16), Completeness: CompletenessPartial}}, nil
}
