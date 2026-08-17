package sourceaccess

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRESTJSONAdapterRejectsWhitespaceCredentialMaterial(t *testing.T) {
	connection := restTestConnection(t, "https://bank.example", RESTJSONAuthentication{Kind: RESTJSONAuthBearer}, "secret://rest")
	if _, err := NewRESTJSONAdapter(DefaultRESTJSONOptions()).Open(context.Background(), connection, staticRESTSecretResolver(" token-with-space ")); !errors.Is(err, ErrCredentials) {
		t.Fatalf("credential material with surrounding whitespace was accepted: %v", err)
	}
}

func TestRESTJSONNoContentIsAnEmptyCompleteRead(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	connection := restTestConnection(t, server.URL, RESTJSONAuthentication{Kind: RESTJSONAuthNone}, "")
	sessionValue, err := NewRESTJSONAdapter(RESTJSONOptions{Client: server.Client()}).Open(context.Background(), connection, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer sessionValue.Close()
	view := restTestView(t, connection, RESTJSONViewDefinition{
		Path: "/empty", RecordsPointer: "/items", Pagination: RESTJSONPagination{Mode: RESTJSONPaginationNone},
	})
	binding := Binding{
		ID: "binding-empty", ViewID: view.ID, Version: "1", Purpose: "empty-read",
		Operations: []Operation{OperationPage}, SelectedFields: []string{"id"}, KeyFields: []string{"id"}, Limits: DefaultResourceLimits(),
	}
	page, err := sessionValue.(PageReader).ReadPage(context.Background(), view, binding, PageRequest{Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Records) != 0 || page.NextCursor != nil || page.Receipt.Completeness != CompletenessComplete {
		t.Fatalf("204 response was not represented as an empty complete read: %#v", page)
	}
}
