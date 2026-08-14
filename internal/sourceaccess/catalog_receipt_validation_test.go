package sourceaccess

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCatalogServiceRejectsMismatchedInspectReceipt(t *testing.T) {
	ctx := context.Background()
	repository, connection, view, _ := receiptCatalogFixture(t)
	session := &receiptValidationSession{
		connection:   mustConnection(t, connection),
		capabilities: NewCapabilitySet(CapabilityInspect),
		inspectResult: SchemaResult{
			Fields: []NativeField{{Name: "account_id", NativeType: "uuid"}},
			Receipt: OperationReceipt{
				SourceID: connection.SourceID, ConnectionID: connection.ConnectionID, ConnectionVersion: "1",
				AdapterKind: AdapterPostgres, AdapterVersion: PostgresAdapterVersion,
				ViewID: "wrong-view", ViewVersion: "1", Operation: OperationInspect,
				Count: 1, Completeness: CompletenessComplete,
			},
		},
	}
	service := NewCatalogService(repository, catalogFakeSecrets{}, map[AdapterKind]Adapter{AdapterPostgres: receiptValidationAdapter{session: session}})
	service.newID = func() (string, error) { return "7c111111-1111-7111-8111-111111111111", nil }
	service.now = func() time.Time { return time.Date(2026, 8, 14, 13, 0, 0, 0, time.UTC) }

	_, err := service.InspectViewDraft(ctx, CatalogActor{TenantID: catalogTenantID, PrincipalID: catalogActorID}, view.ViewID, view.Version, InspectViewDraftInput{StableKeys: []string{"account_id"}})
	if !errors.Is(err, ErrExecution) {
		t.Fatalf("mismatched inspect receipt should fail, got %v", err)
	}
	history, err := repository.ListViewRevisions(ctx, catalogTenantID, connection.ConnectionID, 20)
	if err != nil || len(history) != 1 {
		t.Fatalf("invalid inspection persisted a View revision: history=%#v err=%v", history, err)
	}
}

func TestCatalogServiceRejectsMismatchedPreviewReceipt(t *testing.T) {
	ctx := context.Background()
	repository, connection, view, binding := receiptCatalogFixture(t)
	session := &receiptValidationSession{
		connection:   mustConnection(t, connection),
		capabilities: NewCapabilitySet(CapabilityPage),
		pageResult: RecordPage{
			Records: []Record{{"account_id": StringValue("account-1")}},
			Receipt: OperationReceipt{
				SourceID: connection.SourceID, ConnectionID: connection.ConnectionID, ConnectionVersion: "1",
				AdapterKind: AdapterPostgres, AdapterVersion: PostgresAdapterVersion,
				ViewID: view.ViewID, ViewVersion: "1", BindingID: "wrong-binding", BindingVersion: "1",
				Operation: OperationPage, Count: 1, Bytes: 16, Completeness: CompletenessComplete,
			},
		},
	}
	service := NewCatalogService(repository, catalogFakeSecrets{}, map[AdapterKind]Adapter{AdapterPostgres: receiptValidationAdapter{session: session}})

	_, err := service.PreviewBinding(ctx, catalogTenantID, binding.BindingID, binding.Version, PageRequest{Limit: 1})
	if !errors.Is(err, ErrExecution) {
		t.Fatalf("mismatched preview receipt should fail, got %v", err)
	}
}

func TestCatalogServiceRejectsReceiptCountMismatch(t *testing.T) {
	ctx := context.Background()
	repository, connection, view, binding := receiptCatalogFixture(t)
	session := &receiptValidationSession{
		connection:   mustConnection(t, connection),
		capabilities: NewCapabilitySet(CapabilityPage),
		pageResult: RecordPage{
			Records: []Record{{"account_id": StringValue("account-1")}},
			Receipt: OperationReceipt{
				SourceID: connection.SourceID, ConnectionID: connection.ConnectionID, ConnectionVersion: "1",
				AdapterKind: AdapterPostgres, AdapterVersion: PostgresAdapterVersion,
				ViewID: view.ViewID, ViewVersion: "1", BindingID: binding.BindingID, BindingVersion: "1",
				Operation: OperationPage, Count: 2, Bytes: 16, Completeness: CompletenessComplete,
			},
		},
	}
	service := NewCatalogService(repository, catalogFakeSecrets{}, map[AdapterKind]Adapter{AdapterPostgres: receiptValidationAdapter{session: session}})

	_, err := service.PreviewBinding(ctx, catalogTenantID, binding.BindingID, binding.Version, PageRequest{Limit: 1})
	if !errors.Is(err, ErrExecution) {
		t.Fatalf("receipt count mismatch should fail, got %v", err)
	}
}

func receiptCatalogFixture(t *testing.T) (*MemoryCatalogRepository, ConnectionRevision, ViewRevision, BindingRevision) {
	t.Helper()
	repository := NewMemoryCatalogRepository([]SourceScope{{TenantID: catalogTenantID, SourceID: catalogSourceID}})
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	connection := catalogConnectionRevision(now)
	view := catalogViewRevision(now)
	binding := catalogBindingRevision(now)
	if _, err := repository.CreateConnectionRevision(context.Background(), connection); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.CreateViewRevision(context.Background(), view); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.CreateBindingRevision(context.Background(), binding); err != nil {
		t.Fatal(err)
	}
	return repository, connection, view, binding
}

type receiptValidationAdapter struct{ session *receiptValidationSession }

func (a receiptValidationAdapter) Open(_ context.Context, connection Connection, _ SecretResolver) (Session, error) {
	a.session.connection = connection
	return a.session, nil
}

type receiptValidationSession struct {
	connection    Connection
	capabilities  CapabilitySet
	inspectResult SchemaResult
	pageResult    RecordPage
}

func (s *receiptValidationSession) Connection() Connection     { return s.connection }
func (s *receiptValidationSession) Capabilities() CapabilitySet { return s.capabilities }
func (s *receiptValidationSession) Close() error                { return nil }
func (s *receiptValidationSession) Inspect(context.Context, View) (SchemaResult, error) {
	return s.inspectResult, nil
}
func (s *receiptValidationSession) ReadPage(context.Context, View, Binding, PageRequest) (RecordPage, error) {
	return s.pageResult, nil
}
