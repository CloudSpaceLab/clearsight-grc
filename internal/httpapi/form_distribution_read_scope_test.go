package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestDistributionReadsFailClosedBeforeRepositoryAccess(t *testing.T) {
	// A service without a store cannot satisfy a repository read. The temporary
	// denial must happen first, for every registered distribution read route.
	api := &API{deps: Dependencies{FormDistributions: evidence.NewDistributionService(nil)}}
	want := map[string]bool{"/api/v1/forms/distributions": false, "/api/v1/forms/distributions/{id}": false, "/api/v1/forms/distributions/{id}/responses": false}
	for _, route := range api.formDistributionRoutes() {
		if route.Method != http.MethodGet || (route.Path != "/api/v1/forms/distributions" && !strings.HasPrefix(route.Path, "/api/v1/forms/distributions/")) {
			continue
		}
		if _, expected := want[route.Path]; expected {
			want[route.Path] = true
		}
		r := httptest.NewRequest(http.MethodGet, "/api/v1/forms/distributions/test", nil)
		r.SetPathValue("id", "test")
		r = r.WithContext(identity.WithActor(r.Context(), identity.Actor{TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "person", Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour)}))
		w := httptest.NewRecorder()
		route.Handler(w, r)
		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("%s status %d: %s", route.Path, w.Code, w.Body.String())
		}
	}
	for path, found := range want {
		if !found {
			t.Errorf("missing route %s", path)
		}
	}
}

func TestDistributionReadRoutesDenyAuthenticatedRequests(t *testing.T) {
	handler := New(Dependencies{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Mode: "test-memory", Identity: identity.NewDevelopmentAuthenticator("tenant", "person", "entity", "GRC_ADMIN"), FormDistributions: evidence.NewDistributionService(nil)})
	for _, path := range []string{"/api/v1/forms/distributions", "/api/v1/forms/distributions/example", "/api/v1/forms/distributions/example/responses"} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, path, nil)
		handler.ServeHTTP(w, r)
		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("%s status %d: %s", path, w.Code, w.Body.String())
		}
	}
}
