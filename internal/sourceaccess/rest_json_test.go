package sourceaccess

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRESTJSONAdapterInspectPageLookupAndCursor(t *testing.T) {
	var requests int
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Header.Get("Authorization") != "Bearer test-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.URL.Query().Get("active") != "true" {
			http.Error(w, "missing fixed query", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/bank/accounts":
			if cursor := r.URL.Query().Get("cursor"); cursor == "c2" {
				_, _ = w.Write([]byte(`{"data":{"items":[{"id":"3","name":"Ada","status":"ACTIVE"}],"next":null}}`))
				return
			}
			_, _ = w.Write([]byte(`{"data":{"items":[{"id":"1","name":"Chidi","status":"ACTIVE"},{"id":"2","name":"Bola","status":"ACTIVE"}],"next":"c2"}}`))
		case "/bank/accounts/lookup":
			values := r.URL.Query()["id"]
			if !reflect.DeepEqual(values, []string{"1", "3"}) {
				http.Error(w, "wrong lookup values", http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(`{"data":{"items":[{"id":"1","name":"Chidi","status":"ACTIVE"},{"id":"3","name":"Ada","status":"ACTIVE"}]}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	connection := restTestConnection(t, server.URL+"/bank", RESTJSONAuthentication{Kind: RESTJSONAuthBearer}, "secret://api")
	adapter := NewRESTJSONAdapter(RESTJSONOptions{Client: server.Client()})
	sessionValue, err := adapter.Open(context.Background(), connection, staticRESTSecretResolver("test-token"))
	if err != nil {
		t.Fatal(err)
	}
	defer sessionValue.Close()
	session := sessionValue.(*RESTJSONSession)
	view := restTestView(t, connection, RESTJSONViewDefinition{
		Path: "/accounts", RecordsPointer: "/data/items", FixedQuery: map[string]string{"active": "true"},
		Pagination: RESTJSONPagination{Mode: RESTJSONPaginationCursor, CursorQueryParam: "cursor", NextCursorPointer: "/data/next", PageSizeQueryParam: "limit"},
		Lookup:     &RESTJSONLookup{Path: "/accounts/lookup", QueryParam: "id"},
	})

	inspected, err := session.Inspect(context.Background(), view)
	if err != nil {
		t.Fatal(err)
	}
	if len(inspected.Fields) != 3 || inspected.Receipt.SchemaFingerprint == "" || inspected.Receipt.Operation != OperationInspect {
		t.Fatalf("unexpected inspection: %#v", inspected)
	}
	view.NativeSchema = inspected.Fields
	view.SchemaFingerprint = inspected.Receipt.SchemaFingerprint
	binding := Binding{
		ID: "binding-1", ViewID: view.ID, Version: "1", Purpose: "test",
		Operations: []Operation{OperationPage, OperationLookup}, SelectedFields: []string{"id", "name", "status"}, KeyFields: []string{"id"},
		Limits: ResourceLimits{PageRows: 10, ResponseBytes: 64 << 10, LookupValues: 10, Timeout: 2 * time.Second},
	}

	page, err := session.ReadPage(context.Background(), view, binding, PageRequest{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Records) != 2 || page.NextCursor == nil || page.NextCursor.Text != "c2" || page.Receipt.Position == nil || page.Receipt.Position.Kind != CheckpointCursor || page.Receipt.Position.Value != "c2" || page.Receipt.Completeness != CompletenessPartial {
		t.Fatalf("unexpected first page: %#v", page)
	}
	if page.Receipt.RetryIdentity == "" || strings.Contains(page.Receipt.RetryIdentity, "test-token") {
		t.Fatalf("unsafe retry identity: %#v", page.Receipt)
	}
	second, err := session.ReadPage(context.Background(), view, binding, PageRequest{After: page.NextCursor, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Records) != 1 || second.NextCursor != nil || second.Receipt.Completeness != CompletenessComplete {
		t.Fatalf("unexpected second page: %#v", second)
	}

	lookup, err := session.Lookup(context.Background(), view, binding, LookupRequest{Values: []Scalar{StringValue("1"), StringValue("3")}})
	if err != nil {
		t.Fatal(err)
	}
	if len(lookup.Records) != 2 || lookup.Receipt.Operation != OperationLookup || lookup.Receipt.RetryIdentity == "" {
		t.Fatalf("unexpected lookup: %#v", lookup)
	}
	if requests < 4 {
		t.Fatalf("expected inspect, two pages and lookup; requests=%d", requests)
	}
}

func TestRESTJSONAdapterETagPaginationAnd304(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-None-Match") == `"v1"` {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("ETag", `"v1"`)
		_, _ = w.Write([]byte(`{"items":[{"id":"1","name":"Current"}]}`))
	}))
	defer server.Close()
	connection := restTestConnection(t, server.URL, RESTJSONAuthentication{Kind: RESTJSONAuthNone}, "")
	sessionValue, err := NewRESTJSONAdapter(RESTJSONOptions{Client: server.Client()}).Open(context.Background(), connection, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer sessionValue.Close()
	session := sessionValue.(*RESTJSONSession)
	view := restTestView(t, connection, RESTJSONViewDefinition{Path: "/etag", RecordsPointer: "/items", Pagination: RESTJSONPagination{Mode: RESTJSONPaginationETag}})
	inspection, err := session.Inspect(context.Background(), view)
	if err != nil {
		t.Fatal(err)
	}
	view.NativeSchema, view.SchemaFingerprint = inspection.Fields, inspection.Receipt.SchemaFingerprint
	binding := Binding{ID: "binding-etag", ViewID: view.ID, Version: "1", Purpose: "etag", Operations: []Operation{OperationPage}, SelectedFields: []string{"id", "name"}, KeyFields: []string{"id"}, Limits: DefaultResourceLimits()}
	first, err := session.ReadPage(context.Background(), view, binding, PageRequest{Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if first.NextCursor == nil || first.NextCursor.Text != `"v1"` || first.Receipt.Position == nil || first.Receipt.Position.Kind != CheckpointETag {
		t.Fatalf("ETag was not returned as the next source position: %#v", first)
	}
	unchanged, err := session.ReadPage(context.Background(), view, binding, PageRequest{After: first.NextCursor, Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(unchanged.Records) != 0 || unchanged.NextCursor != nil || unchanged.Receipt.Completeness != CompletenessComplete {
		t.Fatalf("304 was not represented as an unchanged complete read: %#v", unchanged)
	}
}

func TestRESTJSONAdapterRejectsSchemaDrift(t *testing.T) {
	drift := false
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if drift {
			_, _ = w.Write([]byte(`{"items":[{"id":"1","status":7}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"items":[{"id":"1","status":"ACTIVE"}]}`))
	}))
	defer server.Close()
	connection := restTestConnection(t, server.URL, RESTJSONAuthentication{Kind: RESTJSONAuthNone}, "")
	sessionValue, err := NewRESTJSONAdapter(RESTJSONOptions{Client: server.Client()}).Open(context.Background(), connection, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer sessionValue.Close()
	session := sessionValue.(*RESTJSONSession)
	view := restTestView(t, connection, RESTJSONViewDefinition{Path: "/accounts", RecordsPointer: "/items", Pagination: RESTJSONPagination{Mode: RESTJSONPaginationNone}})
	inspection, err := session.Inspect(context.Background(), view)
	if err != nil {
		t.Fatal(err)
	}
	view.NativeSchema, view.SchemaFingerprint = inspection.Fields, inspection.Receipt.SchemaFingerprint
	binding := Binding{ID: "binding-drift", ViewID: view.ID, Version: "1", Purpose: "drift", Operations: []Operation{OperationPage}, SelectedFields: []string{"id", "status"}, KeyFields: []string{"id"}, Limits: DefaultResourceLimits()}
	drift = true
	if _, err := session.ReadPage(context.Background(), view, binding, PageRequest{Limit: 5}); !errors.Is(err, ErrSchemaDrift) {
		t.Fatalf("schema drift was not blocked: %v", err)
	}
}

func TestRESTJSONAdapterRejectsUnsafeTemplatesRedirectsAndOversize(t *testing.T) {
	connection := restTestConnection(t, "https://bank.example/api", RESTJSONAuthentication{Kind: RESTJSONAuthHeader, HeaderName: "X-API-Key"}, "secret://key")
	for name, definition := range map[string]RESTJSONViewDefinition{
		"query in path": {Path: "/accounts?admin=true", Pagination: RESTJSONPagination{Mode: RESTJSONPaginationNone}},
		"traversal":     {Path: "/accounts/../admin", Pagination: RESTJSONPagination{Mode: RESTJSONPaginationNone}},
		"cursor collision": {Path: "/accounts", FixedQuery: map[string]string{"cursor": "fixed"}, Pagination: RESTJSONPagination{Mode: RESTJSONPaginationCursor, CursorQueryParam: "cursor", NextCursorPointer: "/next"}},
	} {
		t.Run(name, func(t *testing.T) {
			encoded, _ := json.Marshal(definition)
			view := View{ID: "view", ConnectionID: connection.ID, Version: "1", OutputKind: OutputRecords, Definition: encoded, StableKeys: []string{"id"}}
			if _, err := decodeRESTJSONView(view); err == nil {
				t.Fatal("unsafe REST view definition was accepted")
			}
		})
	}

	redirectTarget := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer redirectTarget.Close()
	redirectServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, redirectTarget.URL, http.StatusFound)
	}))
	defer redirectServer.Close()
	redirectConnection := restTestConnection(t, redirectServer.URL, RESTJSONAuthentication{Kind: RESTJSONAuthNone}, "")
	redirectSessionValue, err := NewRESTJSONAdapter(RESTJSONOptions{Client: redirectServer.Client()}).Open(context.Background(), redirectConnection, nil)
	if err != nil {
		t.Fatal(err)
	}
	redirectView := restTestView(t, redirectConnection, RESTJSONViewDefinition{Path: "/redirect", Pagination: RESTJSONPagination{Mode: RESTJSONPaginationNone}})
	if _, err := redirectSessionValue.(SchemaReader).Inspect(context.Background(), redirectView); err == nil {
		t.Fatal("REST redirect was followed or accepted")
	}

	oversizeServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"1","payload":"` + strings.Repeat("x", 4096) + `"}]`))
	}))
	defer oversizeServer.Close()
	overConnection := restTestConnection(t, oversizeServer.URL, RESTJSONAuthentication{Kind: RESTJSONAuthNone}, "")
	overSessionValue, err := NewRESTJSONAdapter(RESTJSONOptions{Client: oversizeServer.Client()}).Open(context.Background(), overConnection, nil)
	if err != nil {
		t.Fatal(err)
	}
	overView := restTestView(t, overConnection, RESTJSONViewDefinition{Path: "/large", Pagination: RESTJSONPagination{Mode: RESTJSONPaginationNone}})
	overView.NativeSchema = []NativeField{{Name: "id", NativeType: "json:string", Nullable: true}, {Name: "payload", NativeType: "json:string", Nullable: true}}
	overView.SchemaFingerprint, _ = nativeSchemaFingerprint(overView.NativeSchema)
	binding := Binding{ID: "binding-large", ViewID: overView.ID, Version: "1", Purpose: "limit", Operations: []Operation{OperationPage}, SelectedFields: []string{"id", "payload"}, KeyFields: []string{"id"}, Limits: ResourceLimits{PageRows: 10, ResponseBytes: 128, LookupValues: 10, Timeout: time.Second}}
	if _, err := overSessionValue.(PageReader).ReadPage(context.Background(), overView, binding, PageRequest{Limit: 5}); !errors.Is(err, ErrLimitExceeded) {
		t.Fatalf("oversized response was not rejected: %v", err)
	}
}

func TestRESTJSONConnectionRequiresHTTPSAndSafeAuthHeader(t *testing.T) {
	for name, definition := range map[string]RESTJSONConnectionDefinition{
		"http": {BaseURL: "http://bank.example", Authentication: RESTJSONAuthentication{Kind: RESTJSONAuthNone}},
		"credentials": {BaseURL: "https://user:pass@bank.example", Authentication: RESTJSONAuthentication{Kind: RESTJSONAuthNone}},
		"reserved header": {BaseURL: "https://bank.example", Authentication: RESTJSONAuthentication{Kind: RESTJSONAuthHeader, HeaderName: "Host"}},
	} {
		t.Run(name, func(t *testing.T) {
			raw, _ := json.Marshal(definition)
			secret := ""
			if definition.Authentication.Kind == RESTJSONAuthHeader {
				secret = "secret://key"
			}
			if _, err := normalizeRESTJSONConnectionDefinition(raw, secret); err == nil {
				t.Fatal("unsafe REST connection definition was accepted")
			}
		})
	}
}

func restTestConnection(t *testing.T, baseURL string, authentication RESTJSONAuthentication, secretRef string) Connection {
	t.Helper()
	definition, err := json.Marshal(RESTJSONConnectionDefinition{BaseURL: baseURL, Authentication: authentication})
	if err != nil {
		t.Fatal(err)
	}
	return Connection{ID: "connection-1", SourceID: "source-1", Version: "1", AdapterKind: AdapterRESTJSON, AdapterVersion: RESTJSONAdapterVersion, SecretRef: secretRef, Definition: definition}
}

func restTestView(t *testing.T, connection Connection, definition RESTJSONViewDefinition) View {
	t.Helper()
	encoded, err := json.Marshal(definition)
	if err != nil {
		t.Fatal(err)
	}
	return View{ID: "view-1", ConnectionID: connection.ID, Version: "1", OutputKind: OutputRecords, Definition: encoded, StableKeys: []string{"id"}}
}

type staticRESTSecretResolver string

func (r staticRESTSecretResolver) Resolve(context.Context, string) (string, error) {
	return string(r), nil
}

func TestRESTJSONRetryIdentityDoesNotExposeLookupValues(t *testing.T) {
	view := View{ID: "view", ConnectionID: "connection", Version: "1"}
	binding := Binding{ID: "binding", Version: "2"}
	identity := restRetryIdentity(view, binding, OperationLookup, nil, []Scalar{StringValue("customer-secret-123")})
	if identity == "" || strings.Contains(identity, "customer-secret-123") {
		t.Fatalf("retry identity exposes source lookup material: %q", identity)
	}
	if _, err := url.Parse(identity); err != nil {
		t.Fatal(err)
	}
	_ = fmt.Sprint(identity)
}
