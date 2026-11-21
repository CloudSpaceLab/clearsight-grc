package sourceaccess

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRESTQuestionContractValidatesResultAndOptionalMessage(t *testing.T) {
	for _, tc := range []struct {
		name, resultType string
		valid, invalid   []string
	}{
		{"yes or no", "BOOLEAN", []string{`{"result":false}`, `{"result":true,"message":"Checked"}`}, []string{`{}`, `{"result":null}`, `{"result":0}`, `{"result":true,"message":null}`, `{"result":true,"message":4}`, `{"result":true,"other":1}`, `{"result":false,"result":true}`, `{"result":true,"message":"` + strings.Repeat("x", 1001) + `"}`}},
		{"numeric score", "NUMBER", []string{`{"result":0}`, `{"result":72.5,"message":"Checked"}`}, []string{`{"result":false}`, `{"result":"1"}`, `{"result":1e999}`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := tc.valid[0]
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
			defer opened.Close()
			session := opened.(*RESTJSONSession)
			view := restTestView(t, connection, RESTJSONViewDefinition{Path: "/check", Singleton: true, ResultType: tc.resultType, Pagination: RESTJSONPagination{Mode: RESTJSONPaginationNone}})
			view.StableKeys = nil
			schema, err := session.Inspect(context.Background(), view)
			if err != nil {
				t.Fatal(err)
			}
			view.NativeSchema, view.SchemaFingerprint = schema.Fields, schema.Receipt.SchemaFingerprint
			binding := Binding{ID: "binding", ViewID: view.ID, Version: "1", Purpose: "question", Operations: []Operation{OperationPage}, SelectedFields: []string{"result"}, Limits: DefaultResourceLimits()}
			for _, body = range tc.valid {
				page, err := session.ReadPage(context.Background(), view, binding, PageRequest{Limit: 1})
				if err != nil || len(page.Records) != 1 {
					t.Fatalf("valid response %s: %+v %v", body, page, err)
				}
			}
			for _, body = range tc.invalid {
				if _, err := session.ReadPage(context.Background(), view, binding, PageRequest{Limit: 1}); err == nil {
					t.Fatalf("invalid response accepted: %s", body)
				}
			}
		})
	}
}

func TestRESTQuestionContractRequiresSingletonAndSupportedType(t *testing.T) {
	connection := restTestConnection(t, "https://bank.example", RESTJSONAuthentication{Kind: RESTJSONAuthNone}, "")
	for _, definition := range []RESTJSONViewDefinition{
		{Path: "/check", ResultType: "BOOLEAN"},
		{Path: "/check", Singleton: true, ResultType: "TEXT"},
	} {
		if _, err := decodeRESTJSONView(restTestView(t, connection, definition)); err == nil {
			t.Fatalf("invalid contract accepted: %+v", definition)
		}
	}
}
