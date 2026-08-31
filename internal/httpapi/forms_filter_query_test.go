package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestFormsAdvancedFilterReturnsAuthoritativeTotalAndStatusFacets(t *testing.T) {
	handler := formsTestHandler(t)
	body := []byte(`{"code":"VENDOR","name":"Vendor review","purpose":"Collect current vendor evidence.","presentation":{"default_mode":"AUTOMATIC"},"sections":[{"id":"identity","title":"Identity"}],"fields":[{"id":"name","section_id":"identity","label":"Registered name","type":"short_text","required":true}]}`)
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, httptest.NewRequest(http.MethodPost, "/api/v1/forms/templates", bytes.NewReader(body)))
	if created.Code != http.StatusCreated {
		t.Fatalf("create returned %d: %s", created.Code, created.Body.String())
	}

	expression := `{"kind":"condition","field":"status","operator":"is","value":"draft"}`
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/forms/templates?filter="+url.QueryEscape(expression)+"&facets=status", nil)
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("advanced list returned %d: %s", response.Code, response.Body.String())
	}
	for _, expected := range [][]byte{[]byte(`"total":1`), []byte(`"facets":{"status":{"DRAFT":1}}`)} {
		if !bytes.Contains(response.Body.Bytes(), expected) {
			t.Fatalf("advanced list missing %s: %s", expected, response.Body.String())
		}
	}
}

func TestFormsAdvancedFilterRejectsUnsupportedFieldsAndFacets(t *testing.T) {
	handler := formsTestHandler(t)
	for _, target := range []string{
		"/api/v1/forms/templates?filter=" + url.QueryEscape(`{"kind":"condition","field":"reviewer","operator":"is","value":"person-a"}`),
		"/api/v1/forms/templates?facets=owner",
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, target, nil))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("%s returned %d: %s", target, response.Code, response.Body.String())
		}
	}
}
