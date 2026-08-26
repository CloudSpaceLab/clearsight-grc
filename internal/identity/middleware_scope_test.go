package identity

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMiddlewareRejectsConflictingQueryScope(t *testing.T) {
	authenticator := NewDevelopmentAuthenticator("bank-a", "person-a", "entity-a")
	handler := Middleware(authenticator, slog.New(slog.NewTextHandler(io.Discard, nil)))(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	for _, path := range []string{
		"/api/v1/programs?tenant_id=bank-b",
		"/api/v1/onboarding/state?principal_id=person-b",
		"/api/v1/programs?legal_entity_id=entity-b",
	} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("%s returned %d", path, recorder.Code)
		}
		if !strings.Contains(recorder.Body.String(), "scope_not_found") {
			t.Fatalf("%s did not return a non-enumerating scope error: %s", path, recorder.Body.String())
		}
	}
}

func TestMiddlewareAcceptsVerifiedOrImplicitScope(t *testing.T) {
	authenticator := NewDevelopmentAuthenticator("bank-a", "person-a", "entity-a")
	handler := Middleware(authenticator, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, ok := FromContext(r.Context())
		if !ok || actor.TenantID != "bank-a" {
			t.Fatal("verified actor was not added to the request context")
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	for _, path := range []string{
		"/api/v1/context",
		"/api/v1/programs?tenant_id=bank-a&principal_id=person-a&legal_entity_id=entity-a",
	} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("%s returned %d: %s", path, recorder.Code, recorder.Body.String())
		}
	}
}

func TestMiddlewareAllowsWildcardLegalEntityActorToBindOneExactEntity(t *testing.T) {
	authenticator := NewDevelopmentAuthenticator("bank-a", "admin-a", "*")
	handler := Middleware(authenticator, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/evidence/sources?tenant_id=bank-a&legal_entity_id=entity-a", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("exact entity binding returned %d: %s", response.Code, response.Body.String())
	}
}
