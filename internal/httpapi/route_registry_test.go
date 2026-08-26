package httpapi

import (
	"encoding/json"
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
		http.MethodGet + " /health/live":           true,
		http.MethodGet + " /health/ready":          true,
		http.MethodGet + " /api/v1/session/status": true,
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

func TestAdministrativePermissionsLiveInRouteRegistry(t *testing.T) {
	routes := (&API{}).routes()
	expected := map[string]string{}
	addExpected := func(method, path, permission string) {
		expected[method+" "+path] = permission
	}
	addExpected(http.MethodGet, "/api/v1/governance/policies", identity.PermissionConfigRead)
	addExpected(http.MethodPost, "/api/v1/governance/policies", identity.PermissionConfigWrite)
	addExpected(http.MethodPost, "/api/v1/authority/simulate", identity.PermissionConfigRead)
	addExpected(http.MethodGet, "/api/v1/operations/projections", identity.PermissionPlatformOperationsRead)
	addExpected(http.MethodPost, "/api/v1/operations/projections/reconcile", identity.PermissionPlatformOperationsWrite)
	addExpected(http.MethodGet, "/api/v1/operations/background-jobs", identity.PermissionPlatformJobsRead)
	addExpected(http.MethodGet, "/api/v1/compliance/automation-policies", identity.PermissionConfigRead)
	addExpected(http.MethodGet, "/api/v1/access/overview", identity.PermissionIdentityRead)
	addExpected(http.MethodPost, "/api/v1/access/scim-sources", identity.PermissionIdentityConfigure)
	addExpected(http.MethodPost, "/api/v1/access/scim-sources/{id}/rotate-token", identity.PermissionIdentityConfigure)
	addExpected(http.MethodPost, "/api/v1/access/scim-sources/{id}/revoke", identity.PermissionIdentityConfigure)
	addExpected(http.MethodPost, "/api/v1/access/group-role-bindings", identity.PermissionIdentityConfigure)
	addExpected(http.MethodPost, "/api/v1/access/group-role-bindings/{id}/retire", identity.PermissionIdentityConfigure)
	addExpected(http.MethodPost, "/api/v1/access/escalation-guard-revisions", identity.PermissionIdentityConfigure)
	addExpected(http.MethodPost, "/api/v1/access/escalation-guard-revisions/{policy_id}/{version}/approve", identity.PermissionIdentityConfigure)
	addExpected(http.MethodPost, "/api/v1/access/escalations/preview", identity.PermissionIdentityRead)

	seen := map[string]string{}
	for _, route := range routes {
		key := route.Method + " " + route.Path
		if permission, ok := expected[key]; ok {
			seen[key] = route.Permission
			if route.Permission != permission {
				t.Fatalf("%s permission = %q, want %q", key, route.Permission, permission)
			}
		}
	}
	if len(seen) != len(expected) {
		t.Fatalf("missing administrative routes from registry: got %#v", seen)
	}
}

func TestVendorReadRoutesRequireVendorReadPermission(t *testing.T) {
	expected := map[string]bool{
		http.MethodGet + " /api/v1/vendors":                                 true,
		http.MethodGet + " /api/v1/vendors/{id}":                            true,
		http.MethodGet + " /api/v1/vendors/{id}/links":                      true,
		http.MethodGet + " /api/v1/vendor-links":                            true,
		http.MethodGet + " /api/v1/vendor-work":                             true,
		http.MethodGet + " /api/v1/vendor-work/{request_id}":                true,
		http.MethodGet + " /api/v1/vendors/{id}/work/{request_id}/response": true,
		http.MethodGet + " /api/v1/vendors/{id}/work/{request_id}/requests/{capture_request_id}/documents/{artifact_id}/open": true,
		http.MethodGet + " /api/v1/vendors/{id}/assessments/current":                                                          true,
		http.MethodGet + " /api/v1/vendor-assessments/{id}":                                                                   true,
		http.MethodGet + " /api/v1/vendor-assessments/{id}/requests/{request_id}/documents/{artifact_id}/open":                true,
	}
	for _, route := range (&API{}).routes() {
		key := route.Method + " " + route.Path
		if !expected[key] {
			continue
		}
		if route.Permission != identity.PermissionVendorRead {
			t.Fatalf("%s permission = %q, want %q", key, route.Permission, identity.PermissionVendorRead)
		}
		delete(expected, key)
	}
	if len(expected) != 0 {
		t.Fatalf("vendor read routes missing from registry: %#v", expected)
	}
}

func TestVendorReadRouteRequiresVerifiedPermission(t *testing.T) {
	withoutPermission := New(Dependencies{
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		Identity: identity.NewDevelopmentAuthenticator("bank-demo", "user", "bank-ng", "EVIDENCE_RESPONDENT"),
	})
	response := httptest.NewRecorder()
	withoutPermission.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/vendors", nil))
	if response.Code != http.StatusForbidden {
		t.Fatalf("vendor list without permission returned %d: %s", response.Code, response.Body.String())
	}

	withPermission := New(Dependencies{
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		Identity: identity.NewDevelopmentAuthenticator("bank-demo", "owner", "bank-ng", "BUSINESS_OWNER"),
	})
	response = httptest.NewRecorder()
	withPermission.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/vendors", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("vendor list with permission should reach the handler, got %d: %s", response.Code, response.Body.String())
	}
}

func TestActorContextReportsVendorReadCapability(t *testing.T) {
	now := time.Now().UTC()
	actor := identity.Actor{
		TenantID: "bank", PrincipalID: "owner", LegalEntityID: "entity", Kind: "PERSON", RoleCodes: []string{"BUSINESS_OWNER"},
		PermissionCodes: []string{identity.PermissionVendorRead}, AuthenticationMethod: "test", AssuranceLevel: "test", SessionID: "session", IssuedAt: now, ExpiresAt: now.Add(time.Hour),
	}
	handler := New(Dependencies{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: staticIdentityAuthenticator{actor: actor}})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/context", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("context response = %d: %s", response.Code, response.Body.String())
	}
	var body struct {
		Capabilities map[string]bool `json:"capabilities"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if !body.Capabilities["vendor_read"] {
		t.Fatalf("vendor_read capability = %#v", body.Capabilities)
	}
}

func TestAdministrativeRouteRequiresEffectivePermission(t *testing.T) {
	handler := New(Dependencies{
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		Identity: identity.NewDevelopmentAuthenticator("bank-demo", "user", "bank-ng"),
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/operations/background-jobs", nil))
	if response.Code != http.StatusForbidden {
		t.Fatalf("background jobs without permission returned %d: %s", response.Code, response.Body.String())
	}

	admin := New(Dependencies{
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		Identity: identity.NewDevelopmentAuthenticator("bank-demo", "admin", "bank-ng", "SYSTEM_ADMIN"),
	})
	response = httptest.NewRecorder()
	admin.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/operations/background-jobs", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("permitted background-jobs route should reach handler, got %d: %s", response.Code, response.Body.String())
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
