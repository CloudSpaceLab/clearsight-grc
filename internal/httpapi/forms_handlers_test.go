package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
)

func TestFormsRoutesAreRegisteredAndClassified(t *testing.T) {
	want := map[string]routeClass{
		"GET /api/v1/forms/templates":                             routeAuthenticatedRead,
		"POST /api/v1/forms/templates":                            routeMaterialCommand,
		"GET /api/v1/forms/templates/{id}/revisions/{version}":    routeAuthenticatedRead,
		"POST /api/v1/forms/templates/{id}/revisions":             routeMaterialCommand,
		"POST /api/v1/forms/templates/{id}/transition":            routeMaterialCommand,
		"GET /api/v1/forms/starter-templates":                     routeAuthenticatedRead,
		"POST /api/v1/forms/starter-templates/{code}/instantiate": routeMaterialCommand,
		"GET /api/v1/forms/saved-views":                           routeAuthenticatedRead,
		"POST /api/v1/forms/saved-views":                          routeAuthenticatedWrite,
		"DELETE /api/v1/forms/saved-views/{id}":                   routeAuthenticatedWrite,
	}
	for _, route := range (&API{}).routes() {
		key := route.Method + " " + route.Path
		class, exists := want[key]
		if !exists {
			continue
		}
		if route.Class != class {
			t.Fatalf("%s class = %s, want %s", key, route.Class, class)
		}
		if class == routeMaterialCommand && (route.Command == nil || route.Command.Policy.ActorField != noActorField) {
			t.Fatalf("%s lacks an actor-free material command policy: %#v", key, route)
		}
		delete(want, key)
	}
	if len(want) != 0 {
		t.Fatalf("missing Forms routes: %#v", want)
	}
}

func formsTestHandler(t *testing.T) http.Handler {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	guard, err := commandauth.New(nil, commandauth.ModeOff, logger)
	if err != nil {
		t.Fatal(err)
	}
	service := monitoring.NewService(monitoring.NewMemoryRepository(), nil)
	service.ConfigureCommandGuard(guard)
	return New(Dependencies{
		Logger: logger, Identity: identity.NewDevelopmentAuthenticator("bank-a", "maker-a", "entity-a"),
		CommandGuard: guard, Monitoring: service,
	})
}

func TestFormsCreateListAndExactRevisionUseSignedScope(t *testing.T) {
	handler := formsTestHandler(t)
	body := []byte(`{"code":"VENDOR","name":"Vendor review","purpose":"Collect current vendor evidence.","presentation":{"default_mode":"AUTOMATIC"},"sections":[{"id":"identity","title":"Identity"}],"fields":[{"id":"name","section_id":"identity","label":"Registered name","type":"short_text","required":true}]}`)
	createdResponse := httptest.NewRecorder()
	handler.ServeHTTP(createdResponse, httptest.NewRequest(http.MethodPost, "/api/v1/forms/templates", bytes.NewReader(body)))
	if createdResponse.Code != http.StatusCreated {
		t.Fatalf("create returned %d: %s", createdResponse.Code, createdResponse.Body.String())
	}
	var created monitoring.FormTemplate
	if err := json.Unmarshal(createdResponse.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.TenantID != "bank-a" || created.LegalEntityID != "entity-a" || created.CreatedBy != "maker-a" {
		t.Fatalf("created scope = %#v", created)
	}

	listResponse := httptest.NewRecorder()
	handler.ServeHTTP(listResponse, httptest.NewRequest(http.MethodGet, "/api/v1/forms/templates?search=vendor&limit=25", nil))
	if listResponse.Code != http.StatusOK || !bytes.Contains(listResponse.Body.Bytes(), []byte(created.ID)) {
		t.Fatalf("list returned %d: %s", listResponse.Code, listResponse.Body.String())
	}

	exactResponse := httptest.NewRecorder()
	handler.ServeHTTP(exactResponse, httptest.NewRequest(http.MethodGet, "/api/v1/forms/templates/"+created.ID+"/revisions/1", nil))
	if exactResponse.Code != http.StatusOK || !bytes.Contains(exactResponse.Body.Bytes(), []byte(`"version":1`)) {
		t.Fatalf("exact revision returned %d: %s", exactResponse.Code, exactResponse.Body.String())
	}
}

func TestFormsRejectUnknownScopeAndQueryOverrides(t *testing.T) {
	handler := formsTestHandler(t)
	for _, request := range []*http.Request{
		httptest.NewRequest(http.MethodGet, "/api/v1/forms/templates?tenant_id=bank-b", nil),
		httptest.NewRequest(http.MethodPost, "/api/v1/forms/templates", bytes.NewBufferString(`{"tenant_id":"bank-b","code":"VENDOR","name":"Vendor review","purpose":"Collect evidence.","fields":[]}`)),
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code < 400 {
			t.Fatalf("scope override returned %d: %s", response.Code, response.Body.String())
		}
	}
}
