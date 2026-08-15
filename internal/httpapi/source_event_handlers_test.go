package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

func TestSourceEventIngressRequiresVerifiedIdentity(t *testing.T) {
	api := &API{deps: Dependencies{SourceCatalog: sourceaccess.NewCatalogService(nil, sourceaccess.EnvironmentSecretResolver{}, map[sourceaccess.AdapterKind]sourceaccess.Adapter{})}}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/source-bindings/binding-a/events", strings.NewReader(`{"event_id":"evt-1","payload":{"account_id":"A1"}}`))
	response := httptest.NewRecorder()

	api.ingestSourceBindingEvent(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestSourceEventIngressRejectsHumanIdentity(t *testing.T) {
	api := &API{deps: Dependencies{SourceCatalog: sourceaccess.NewCatalogService(nil, sourceaccess.EnvironmentSecretResolver{}, map[sourceaccess.AdapterKind]sourceaccess.Adapter{})}}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/source-bindings/binding-a/events", strings.NewReader(`{"event_id":"evt-1","payload":{"account_id":"A1"}}`))
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{TenantID: "tenant-a", PrincipalID: "person-a", Kind: "PERSON"}))
	response := httptest.NewRecorder()

	api.ingestSourceBindingEvent(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestSourceEventIngressDoesNotAcceptCallerTenantScope(t *testing.T) {
	api := &API{deps: Dependencies{SourceCatalog: sourceaccess.NewCatalogService(nil, sourceaccess.EnvironmentSecretResolver{}, map[sourceaccess.AdapterKind]sourceaccess.Adapter{})}}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/source-bindings/binding-a/events", strings.NewReader(`{"tenant_id":"other-tenant","event_id":"evt-1","payload":{"account_id":"A1"}}`))
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{TenantID: "tenant-a", PrincipalID: "service-a", Kind: "SERVICE"}))
	response := httptest.NewRecorder()

	api.ingestSourceBindingEvent(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}
