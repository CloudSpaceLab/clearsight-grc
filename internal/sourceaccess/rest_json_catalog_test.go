package sourceaccess

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCatalogServiceReusesInspectedRESTBindingForLookup(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer catalog-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.URL.Path != "/accounts" || r.URL.Query().Get("active") != "true" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if values := r.URL.Query()["id"]; len(values) > 0 {
			_, _ = w.Write([]byte(`{"items":[{"id":"1","name":"Chidi"},{"id":"3","name":"Ada"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"items":[{"id":"1","name":"Chidi"}]}`))
	}))
	defer server.Close()

	const (
		tenantID = "rest-bank"
		sourceID = "rest-source"
		actorID  = "rest-admin"
	)
	repository := NewMemoryCatalogRepository([]SourceScope{{TenantID: tenantID, SourceID: sourceID}})
	adapter := NewRESTJSONAdapter(RESTJSONOptions{Client: server.Client()})
	service := NewCatalogService(repository, staticRESTSecretResolver("catalog-token"), map[AdapterKind]Adapter{AdapterRESTJSON: adapter})
	service.now = func() time.Time { return time.Date(2026, 8, 15, 8, 15, 0, 0, time.UTC) }
	ids := []string{
		"61111111-1111-7111-8111-111111111111",
		"62222222-2222-7222-8222-222222222222",
		"63333333-3333-7333-8333-333333333333",
		"64444444-4444-7444-8444-444444444444",
		"65555555-5555-7555-8555-555555555555",
		"66666666-6666-7666-8666-666666666666",
		"67777777-7777-7777-8777-777777777777",
	}
	service.newID = func() (string, error) {
		value := ids[0]
		ids = ids[1:]
		return value, nil
	}
	actor := CatalogActor{TenantID: tenantID, PrincipalID: actorID}

	connectionDefinition, _ := json.Marshal(RESTJSONConnectionDefinition{
		BaseURL:        server.URL,
		Authentication: RESTJSONAuthentication{Kind: RESTJSONAuthBearer},
	})
	connection, err := service.CreateConnectionDraft(context.Background(), actor, sourceID, CreateConnectionDraftInput{
		Code: "CUSTOMER_API", Name: "Customer API", AdapterKind: AdapterRESTJSON, AdapterVersion: RESTJSONAdapterVersion,
		SecretRef: "secret://customer-api", Definition: connectionDefinition,
		DeclaredCapabilities: []Capability{CapabilityInspect, CapabilityPage, CapabilityLookup},
	})
	if err != nil {
		t.Fatal(err)
	}

	viewDefinition, _ := json.Marshal(RESTJSONViewDefinition{
		Path: "/accounts", RecordsPointer: "/items", FixedQuery: map[string]string{"active": "true"},
		Pagination: RESTJSONPagination{Mode: RESTJSONPaginationNone},
		Lookup:     &RESTJSONLookup{QueryParam: "id"},
	})
	view, err := service.CreateViewDraft(context.Background(), actor, connection.ConnectionID, CreateViewDraftInput{
		ConnectionVersion: connection.Version, Code: "ACTIVE_ACCOUNTS", Name: "Active accounts", Definition: viewDefinition,
	})
	if err != nil {
		t.Fatal(err)
	}
	inspected, err := service.InspectViewDraft(context.Background(), actor, view.ViewID, view.Version, InspectViewDraftInput{StableKeys: []string{"id"}})
	if err != nil {
		t.Fatal(err)
	}
	if inspected.View.SchemaFingerprint == "" || len(inspected.View.NativeSchema) != 2 {
		t.Fatalf("REST view inspection did not persist bounded native schema: %#v", inspected.View)
	}

	binding, err := service.CreateBindingDraft(context.Background(), actor, inspected.View.ViewID, CreateBindingDraftInput{
		ViewVersion: inspected.View.Version, Code: "CUSTOMER_IDENTITY_LOOKUP", Name: "Customer identity lookup", Purpose: "customer-validation",
		Operations: []Operation{OperationPage, OperationLookup}, SelectedFields: []string{"id", "name"}, KeyFields: []string{"id"},
		Limits: ResourceLimits{PageRows: 10, ResponseBytes: 64 << 10, LookupValues: 10, Timeout: 2 * time.Second},
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := service.LookupBinding(context.Background(), tenantID, binding.BindingID, binding.Version, LookupRequest{
		Values: []Scalar{StringValue("1"), StringValue("3")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Records) != 2 || result.Receipt.BindingID != binding.BindingID || result.Receipt.SchemaFingerprint != inspected.View.SchemaFingerprint {
		t.Fatalf("persisted REST Binding was not reused through the catalog: %#v", result)
	}
	if _, ok := DefaultCatalogAdapters()[AdapterRESTJSON]; !ok {
		t.Fatal("REST/JSON adapter is absent from default catalog registration")
	}
}
