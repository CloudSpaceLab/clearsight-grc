package sourceaccess

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRESTSingletonReadsChangingResultWithoutRecordKey(t *testing.T) {
	body := `{"result":true,"message":"Enabled"}`
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()
	connection := restTestConnection(t, server.URL, RESTJSONAuthentication{Kind: RESTJSONAuthNone}, "")
	opened, err := NewRESTJSONAdapter(RESTJSONOptions{Client: server.Client()}).Open(context.Background(), connection, nil)
	if err != nil {
		t.Fatal(err)
	}
	session := opened.(*RESTJSONSession)
	defer session.Close()
	view := restTestView(t, connection, RESTJSONViewDefinition{Path: "/check", Singleton: true, Pagination: RESTJSONPagination{Mode: RESTJSONPaginationNone}})
	view.StableKeys = nil
	inspected, err := session.Inspect(context.Background(), view)
	if err != nil {
		t.Fatal(err)
	}
	view.NativeSchema, view.SchemaFingerprint = inspected.Fields, inspected.Receipt.SchemaFingerprint
	binding := Binding{ID: "binding", ViewID: view.ID, Version: "1", Purpose: "check", Operations: []Operation{OperationPage}, SelectedFields: []string{"result"}, Limits: DefaultResourceLimits()}
	for _, response := range []string{`{"result":true,"message":"Enabled"}`, `{"result":false,"message":"Disabled"}`} {
		body = response
		page, err := session.ReadPage(context.Background(), view, binding, PageRequest{Limit: 1})
		if err != nil || len(page.Records) != 1 || page.NextCursor != nil || page.Receipt.Completeness != CompletenessComplete {
			t.Fatalf("page=%+v err=%v", page, err)
		}
	}
	body = `[{"result":true,"message":"Enabled"}]`
	if _, err = session.ReadPage(context.Background(), view, binding, PageRequest{Limit: 1}); err == nil {
		t.Fatal("array accepted as singleton")
	}
	if _, err = session.Inspect(context.Background(), view); err == nil {
		t.Fatal("array accepted during inspection")
	}
	body = `{"result":72,"message":"Score"}`
	if _, err = session.ReadPage(context.Background(), view, binding, PageRequest{Limit: 1}); err == nil {
		t.Fatal("changed result type accepted")
	}
	if _, err = session.ReadPage(context.Background(), view, binding, PageRequest{After: ptrScalar(StringValue("cursor"))}); err == nil {
		t.Fatal("cursor accepted")
	}
}

func TestRESTSingletonRejectsPaginationLookupAndKeylessOrdinaryViews(t *testing.T) {
	connection := restTestConnection(t, "https://bank.example", RESTJSONAuthentication{Kind: RESTJSONAuthNone}, "")
	for _, definition := range []RESTJSONViewDefinition{
		{Path: "/check", Singleton: true, Pagination: RESTJSONPagination{Mode: RESTJSONPaginationETag}},
		{Path: "/check", Singleton: true, Pagination: RESTJSONPagination{Mode: RESTJSONPaginationNone}, Lookup: &RESTJSONLookup{QueryParam: "id"}},
		{Path: "/check", Singleton: true, RecordsPointer: "/result", Pagination: RESTJSONPagination{Mode: RESTJSONPaginationNone}},
	} {
		view := restTestView(t, connection, definition)
		if _, err := decodeRESTJSONView(view); !errors.Is(err, ErrDefinitionInvalid) {
			t.Fatalf("invalid singleton accepted: %+v, %v", definition, err)
		}
	}
	view := restTestView(t, connection, RESTJSONViewDefinition{Path: "/check", Pagination: RESTJSONPagination{Mode: RESTJSONPaginationNone}})
	binding := Binding{ID: "binding", ViewID: view.ID, Version: "1", Purpose: "check", Operations: []Operation{OperationPage}, SelectedFields: []string{"result"}, Limits: DefaultResourceLimits()}
	if err := binding.Validate(view); err == nil {
		t.Fatal("ordinary page accepted without a key")
	}
	singleton := restTestView(t, connection, RESTJSONViewDefinition{Path: "/check", Singleton: true, Pagination: RESTJSONPagination{Mode: RESTJSONPaginationNone}})
	binding.Operations = []Operation{OperationPage, OperationLookup}
	binding.SelectedFields = []string{"id", "result"}
	binding.KeyFields = []string{"id"}
	if err := binding.Validate(singleton); err == nil {
		t.Fatal("singleton binding accepted lookup or a record key")
	}
}

func TestRESTSingletonReadsZeroScore(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":0}`))
	}))
	defer server.Close()
	connection := restTestConnection(t, server.URL, RESTJSONAuthentication{Kind: RESTJSONAuthNone}, "")
	opened, err := NewRESTJSONAdapter(RESTJSONOptions{Client: server.Client()}).Open(context.Background(), connection, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	session := opened.(*RESTJSONSession)
	view := restTestView(t, connection, RESTJSONViewDefinition{Path: "/check", Singleton: true, Pagination: RESTJSONPagination{Mode: RESTJSONPaginationNone}})
	view.StableKeys = nil
	schema, err := session.Inspect(context.Background(), view)
	if err != nil {
		t.Fatal(err)
	}
	view.NativeSchema, view.SchemaFingerprint = schema.Fields, schema.Receipt.SchemaFingerprint
	binding := Binding{ID: "binding", ViewID: view.ID, Version: "1", Purpose: "check", Operations: []Operation{OperationPage}, SelectedFields: []string{"result"}, Limits: DefaultResourceLimits()}
	page, err := session.ReadPage(context.Background(), view, binding, PageRequest{Limit: 1})
	if err != nil || len(page.Records) != 1 || page.Records[0]["result"].Text != "0" {
		t.Fatalf("zero result lost: %+v, %v", page, err)
	}
}

func TestRESTSingletonCatalogDraftPreviewKeepsTenantScope(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":false,"message":"Not enabled"}`))
	}))
	defer server.Close()
	repo := NewMemoryCatalogRepository([]SourceScope{{TenantID: "bank", SourceID: "source"}})
	service := NewCatalogService(repo, staticRESTSecretResolver(""), map[AdapterKind]Adapter{AdapterRESTJSON: NewRESTJSONAdapter(RESTJSONOptions{Client: server.Client()})})
	actor := CatalogActor{TenantID: "bank", PrincipalID: "admin"}
	connectionDefinition, _ := json.Marshal(RESTJSONConnectionDefinition{BaseURL: server.URL, Authentication: RESTJSONAuthentication{Kind: RESTJSONAuthNone}})
	connection, err := service.CreateConnectionDraft(context.Background(), actor, "source", CreateConnectionDraftInput{Code: "API", Name: "API", AdapterKind: AdapterRESTJSON, AdapterVersion: RESTJSONAdapterVersion, Definition: connectionDefinition, DeclaredCapabilities: []Capability{CapabilityInspect, CapabilityPage}})
	if err != nil {
		t.Fatal(err)
	}
	viewDefinition, _ := json.Marshal(RESTJSONViewDefinition{Path: "/check", Singleton: true, Pagination: RESTJSONPagination{Mode: RESTJSONPaginationNone}})
	view, err := service.CreateViewDraft(context.Background(), actor, connection.ConnectionID, CreateViewDraftInput{ConnectionVersion: connection.Version, Code: "CHECK", Name: "Check", Definition: viewDefinition})
	if err != nil {
		t.Fatal(err)
	}
	inspected, err := service.InspectViewDraft(context.Background(), actor, view.ViewID, view.Version, InspectViewDraftInput{})
	if err != nil {
		t.Fatal(err)
	}
	binding, err := service.CreateBindingDraft(context.Background(), actor, view.ViewID, CreateBindingDraftInput{ViewVersion: inspected.View.Version, Code: "RESULT", Name: "Result", Purpose: "question", Operations: []Operation{OperationPage}, SelectedFields: []string{"result"}, Limits: DefaultResourceLimits()})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.PreviewBinding(context.Background(), "bank", binding.BindingID, binding.Version, PageRequest{Limit: 1})
	if err != nil || len(result.Records) != 1 || result.Records[0]["result"].Text != "false" {
		t.Fatalf("preview=%+v err=%v", result, err)
	}
	if _, err := service.PreviewBinding(context.Background(), "other-bank", binding.BindingID, binding.Version, PageRequest{Limit: 1}); err == nil {
		t.Fatal("cross-tenant preview accepted")
	}
}

func ptrScalar(value Scalar) *Scalar { return &value }
