package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestRouteRegistryHasExplicitAccessClasses(t *testing.T) {
	api := &API{}
	routes := api.routes()
	if err := validateRoutes(routes); err != nil {
		t.Fatal(err)
	}
	public := map[string]bool{}
	for _, route := range routes {
		key := route.Method + " " + route.Path
		if route.Class == routePublic {
			public[key] = true
		}
		if route.Class == routeMaterialCommand && route.Command == nil {
			t.Fatalf("material route lacks command policy: %s", key)
		}
		if route.Method != http.MethodGet && route.Method != http.MethodOptions && route.Class == routeAuthenticatedRead {
			t.Fatalf("mutating route classified as read: %s", key)
		}
	}
	expected := map[string]bool{
		http.MethodGet + " /health/live":  true,
		http.MethodGet + " /health/ready": true,
	}
	if len(public) != len(expected) {
		t.Fatalf("unexpected public route set: %#v", public)
	}
	for key := range expected {
		if !public[key] {
			t.Fatalf("expected public route %s, got %#v", key, public)
		}
	}
}

func TestProtectedRoutesRequireVerifiedIdentity(t *testing.T) {
	authenticator, err := identity.NewSignedAuthenticator(strings.Repeat("s", 32), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	handler := New(Dependencies{
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		Identity: authenticator,
	})

	health := httptest.NewRecorder()
	handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/health/live", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("public health route returned %d: %s", health.Code, health.Body.String())
	}

	protected := httptest.NewRecorder()
	handler.ServeHTTP(protected, httptest.NewRequest(http.MethodGet, "/api/v1/context", nil))
	if protected.Code != http.StatusUnauthorized {
		t.Fatalf("protected route without identity returned %d: %s", protected.Code, protected.Body.String())
	}
}

func TestCORSAllowsBearerCapabilityHeader(t *testing.T) {
	authenticator, err := identity.NewSignedAuthenticator(strings.Repeat("s", 32), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	handler := New(Dependencies{
		Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		AllowedOrigin: "https://app.example",
		Identity:      authenticator,
	})
	request := httptest.NewRequest(http.MethodOptions, "/api/v1/evidence/session", nil)
	request.Header.Set("Origin", "https://app.example")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("expected preflight 204, got %d", response.Code)
	}
	allowed := response.Header().Get("Access-Control-Allow-Headers")
	if !strings.Contains(allowed, "Authorization") {
		t.Fatalf("bearer capability header missing from CORS allowlist: %q", allowed)
	}
}
