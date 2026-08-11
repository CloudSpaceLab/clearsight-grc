package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type panicAuthenticator struct{}

func (panicAuthenticator) Authenticate(*http.Request) (identity.Actor, bool, error) {
	panic("browser identity middleware must not run for SCIM")
}

func TestSCIMProtocolEdgeBypassesBrowserIdentityMiddleware(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := New(Dependencies{
		Logger:        logger,
		AllowedOrigin: "https://app.example.test",
		Identity:      panicAuthenticator{},
		SCIM: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/scim/v2/ServiceProviderConfig" {
				t.Fatalf("unexpected SCIM path: %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusNoContent)
		}),
	})

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/scim/v2/ServiceProviderConfig", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected isolated SCIM handler response, got %d", recorder.Code)
	}
}
