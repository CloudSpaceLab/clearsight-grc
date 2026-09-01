package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestListCompletedFormResponsesUsesVerifiedScopeAndRejectsInvalidFilters(t *testing.T) {
	store := evidence.NewMemoryDistributionStore(evidence.NewMemoryRepository(nil, nil), nil, nil)
	api := &API{deps: Dependencies{FormDistributions: evidence.NewDistributionService(store)}}
	actor := identity.Actor{
		TenantID: "tenant-a", LegalEntityID: "entity-a", PrincipalID: "principal-a", Kind: "PERSON",
		AuthenticationMethod: "TEST", AssuranceLevel: "HIGH", SessionID: "session-a",
		IssuedAt: time.Now().Add(-time.Minute), ExpiresAt: time.Now().Add(time.Hour),
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/forms/responses?legal_entity_id=entity-b&sort=CONCERN_DESC&limit=25", nil)
	request = request.WithContext(identity.WithActor(context.Background(), actor))
	response := httptest.NewRecorder()
	api.listCompletedFormResponses(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"items":[]`) {
		t.Fatalf("verified-scope list status=%d body=%s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/forms/responses?raw_min=101", nil)
	request = request.WithContext(identity.WithActor(context.Background(), actor))
	response = httptest.NewRecorder()
	api.listCompletedFormResponses(response, request)
	if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), "response_filters_invalid") {
		t.Fatalf("invalid filter status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestCompletedResponseRoutesAreAuthenticatedReads(t *testing.T) {
	api := &API{}
	want := map[string]bool{
		"/api/v1/forms/responses":               false,
		"/api/v1/forms/responses/{revision_id}": false,
	}
	for _, route := range api.formDistributionRoutes() {
		if _, ok := want[route.Path]; !ok {
			continue
		}
		want[route.Path] = route.Class == routeAuthenticatedRead && route.Method == http.MethodGet
	}
	for path, valid := range want {
		if !valid {
			t.Fatalf("completed response route %s is missing or not an authenticated read", path)
		}
	}
}
