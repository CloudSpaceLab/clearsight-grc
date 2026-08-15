package documentimport

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

func TestTabularArtifactSourceAccessReusesImportedCSV(t *testing.T) {
	ctx := context.Background()
	service := NewService(NewMemoryRepository(), evidence.NewMemoryObjectStore())
	service.Configure(1 << 20)
	document, err := service.Import(ctx, ImportInput{TenantID: "tenant-a", FileName: "accounts.csv", MediaType: "text/csv", CreatedBy: "actor-a"}, strings.NewReader("id,name\n1,Ada\n2,Grace\n"))
	if err != nil || document.Tabular == nil {
		t.Fatalf("import failed: %#v err=%v", document, err)
	}
	connection := sourceaccess.Connection{TenantID: "tenant-a", ID: "connection-a", SourceID: "source-a", Version: "1", AdapterKind: sourceaccess.AdapterTabularArtifact, AdapterVersion: sourceaccess.TabularArtifactAdapterVersion}
	session, err := service.SourceAccessAdapter().Open(ctx, connection, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	view := sourceaccess.View{ID: "view-a", ConnectionID: connection.ID, Version: "1", OutputKind: sourceaccess.OutputRecords, Definition: []byte(`{"document_id":"` + document.ID + `","resource":"records"}`)}
	inspected, err := session.(sourceaccess.SchemaReader).Inspect(ctx, view)
	if err != nil {
		t.Fatal(err)
	}
	if len(inspected.Fields) != 2 || inspected.Receipt.ArtifactID != document.ID || inspected.Receipt.ArtifactSHA256 != document.SHA256 || inspected.Receipt.ParserVersion != TabularParserVersion {
		t.Fatalf("unexpected inspection receipt: %#v", inspected)
	}
	view.NativeSchema = inspected.Fields
	view.SchemaFingerprint = inspected.Receipt.SchemaFingerprint
	view.StableKeys = []string{"id"}
	binding := sourceaccess.Binding{
		ID: "binding-a", ViewID: view.ID, Version: "1", Purpose: "account-identity",
		Operations:     []sourceaccess.Operation{sourceaccess.OperationPage, sourceaccess.OperationLookup},
		SelectedFields: []string{"id", "name"}, KeyFields: []string{"id"},
		Limits: sourceaccess.ResourceLimits{PageRows: 10, ResponseBytes: 64 << 10, LookupValues: 10, Timeout: time.Second},
	}
	page, err := session.(sourceaccess.PageReader).ReadPage(ctx, view, binding, sourceaccess.PageRequest{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Records) != 1 || page.Records[0]["id"].Text != "1" || page.NextCursor == nil || page.NextCursor.Kind != sourceaccess.ScalarNumber || page.Receipt.Completeness != sourceaccess.CompletenessPartial {
		t.Fatalf("unexpected first page: %#v", page)
	}
	second, err := session.(sourceaccess.PageReader).ReadPage(ctx, view, binding, sourceaccess.PageRequest{Limit: 2, After: page.NextCursor})
	if err != nil || len(second.Records) != 1 || second.Records[0]["id"].Text != "2" || second.NextCursor != nil || second.Receipt.Completeness != sourceaccess.CompletenessComplete {
		t.Fatalf("unexpected second page: %#v err=%v", second, err)
	}
	lookup, err := session.(sourceaccess.LookupReader).Lookup(ctx, view, binding, sourceaccess.LookupRequest{Values: []sourceaccess.Scalar{sourceaccess.StringValue("2")}})
	if err != nil || len(lookup.Records) != 1 || lookup.Records[0]["name"].Text != "Grace" || lookup.Receipt.ArtifactID != document.ID {
		t.Fatalf("unexpected lookup: %#v err=%v", lookup, err)
	}
}

func TestTabularArtifactSourceAccessPreservesPartialRowTruthAndTenantScope(t *testing.T) {
	ctx := context.Background()
	service := NewService(NewMemoryRepository(), evidence.NewMemoryObjectStore())
	document, err := service.Import(ctx, ImportInput{TenantID: "tenant-a", FileName: "users.ndjson", MediaType: "application/x-ndjson", CreatedBy: "actor-a"}, strings.NewReader("{\"id\":\"1\",\"name\":\"Ada\"}\nnot-json\n{\"id\":\"2\",\"name\":\"Grace\"}\n"))
	if err != nil || document.Tabular == nil || document.Tabular.RowsRejected != 1 {
		t.Fatalf("NDJSON import failed: %#v err=%v", document, err)
	}
	connection := sourceaccess.Connection{TenantID: "tenant-a", ID: "connection-a", SourceID: "source-a", Version: "1", AdapterKind: sourceaccess.AdapterTabularArtifact, AdapterVersion: sourceaccess.TabularArtifactAdapterVersion}
	session, err := service.SourceAccessAdapter().Open(ctx, connection, nil)
	if err != nil {
		t.Fatal(err)
	}
	view := sourceaccess.View{ID: "view-a", ConnectionID: connection.ID, Version: "1", OutputKind: sourceaccess.OutputRecords, Definition: []byte(`{"document_id":"` + document.ID + `"}`)}
	inspected, err := session.(sourceaccess.SchemaReader).Inspect(ctx, view)
	if err != nil {
		t.Fatal(err)
	}
	view.NativeSchema, view.SchemaFingerprint, view.StableKeys = inspected.Fields, inspected.Receipt.SchemaFingerprint, []string{"id"}
	binding := sourceaccess.Binding{ID: "binding-a", ViewID: view.ID, Version: "1", Purpose: "user-page", Operations: []sourceaccess.Operation{sourceaccess.OperationPage}, SelectedFields: []string{"id", "name"}, KeyFields: []string{"id"}, Limits: sourceaccess.DefaultResourceLimits()}
	page, err := session.(sourceaccess.PageReader).ReadPage(ctx, view, binding, sourceaccess.PageRequest{Limit: 10})
	if err != nil || len(page.Records) != 2 || page.Receipt.Completeness != sourceaccess.CompletenessPartial {
		t.Fatalf("row rejection was not retained as partial truth: %#v err=%v", page, err)
	}
	foreign := sourceaccess.Connection{TenantID: "tenant-b", ID: "connection-b", SourceID: "source-a", Version: "1", AdapterKind: sourceaccess.AdapterTabularArtifact, AdapterVersion: sourceaccess.TabularArtifactAdapterVersion}
	foreignSession, err := service.SourceAccessAdapter().Open(ctx, foreign, nil)
	if err != nil {
		t.Fatal(err)
	}
	foreignView := view
	foreignView.ConnectionID = foreign.ID
	if _, err := foreignSession.(sourceaccess.SchemaReader).Inspect(ctx, foreignView); err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-tenant artifact resolution leaked import identity: %v", err)
	}
}

func TestTabularArtifactSourceAccessRejectsChangedArtifactAndParserVersion(t *testing.T) {
	ctx := context.Background()
	store := evidence.NewMemoryObjectStore()
	repository := NewMemoryRepository()
	service := NewService(repository, store)
	document, err := service.Import(ctx, ImportInput{TenantID: "tenant-a", FileName: "accounts.csv", MediaType: "text/csv", CreatedBy: "actor-a"}, strings.NewReader("id,name\n1,Ada\n"))
	if err != nil {
		t.Fatal(err)
	}
	connection := sourceaccess.Connection{TenantID: "tenant-a", ID: "connection-a", SourceID: "source-a", Version: "1", AdapterKind: sourceaccess.AdapterTabularArtifact, AdapterVersion: sourceaccess.TabularArtifactAdapterVersion}
	session, err := service.SourceAccessAdapter().Open(ctx, connection, nil)
	if err != nil {
		t.Fatal(err)
	}
	view := sourceaccess.View{ID: "view-a", ConnectionID: connection.ID, Version: "1", OutputKind: sourceaccess.OutputRecords, Definition: []byte(`{"document_id":"` + document.ID + `"}`)}
	inspected, err := session.(sourceaccess.SchemaReader).Inspect(ctx, view)
	if err != nil {
		t.Fatal(err)
	}
	view.NativeSchema, view.SchemaFingerprint, view.StableKeys = inspected.Fields, inspected.Receipt.SchemaFingerprint, []string{"id"}
	binding := sourceaccess.Binding{ID: "binding-a", ViewID: view.ID, Version: "1", Purpose: "account-page", Operations: []sourceaccess.Operation{sourceaccess.OperationPage}, SelectedFields: []string{"id", "name"}, KeyFields: []string{"id"}, Limits: sourceaccess.DefaultResourceLimits()}
	if _, err := store.Put(ctx, document.StorageKey, strings.NewReader("id,name\n1,Mallory\n"), 1<<20); err != nil {
		t.Fatal(err)
	}
	if _, err := session.(sourceaccess.SchemaReader).Inspect(ctx, view); !errors.Is(err, sourceaccess.ErrExecution) {
		t.Fatalf("changed artifact remained inspectable: %v", err)
	}
	if _, err := session.(sourceaccess.PageReader).ReadPage(ctx, view, binding, sourceaccess.PageRequest{}); !errors.Is(err, sourceaccess.ErrExecution) {
		t.Fatalf("changed artifact was accepted: %v", err)
	}

	store2 := evidence.NewMemoryObjectStore()
	repository2 := NewMemoryRepository()
	service2 := NewService(repository2, store2)
	document2, err := service2.Import(ctx, ImportInput{TenantID: "tenant-a", FileName: "accounts.csv", MediaType: "text/csv", CreatedBy: "actor-a"}, strings.NewReader("id,name\n1,Ada\n"))
	if err != nil {
		t.Fatal(err)
	}
	repository2.mu.Lock()
	stale := repository2.items[document2.ID]
	metadata := *stale.Tabular
	metadata.ParserVersion = "TABULAR_OLD"
	stale.Tabular = &metadata
	repository2.items[document2.ID] = stale
	repository2.mu.Unlock()
	session2, err := service2.SourceAccessAdapter().Open(ctx, connection, nil)
	if err != nil {
		t.Fatal(err)
	}
	staleView := sourceaccess.View{ID: "view-b", ConnectionID: connection.ID, Version: "1", OutputKind: sourceaccess.OutputRecords, Definition: []byte(`{"document_id":"` + document2.ID + `"}`)}
	if _, err := session2.(sourceaccess.SchemaReader).Inspect(ctx, staleView); !errors.Is(err, sourceaccess.ErrExecution) {
		t.Fatalf("old parser version remained executable: %v", err)
	}
}
