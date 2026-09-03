package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
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
	addExpected(http.MethodGet, "/api/v1/oversight", identity.PermissionOversightRead)
	addExpected(http.MethodPost, "/api/v1/governance/policies", identity.PermissionConfigWrite)
	addExpected(http.MethodPost, "/api/v1/authority/simulate", identity.PermissionConfigRead)
	addExpected(http.MethodGet, "/api/v1/system-activity", identity.PermissionPlatformOperationsRead)
	addExpected(http.MethodGet, "/api/v1/system-activity/{event_id}", identity.PermissionPlatformOperationsRead)
	addExpected(http.MethodGet, "/api/v1/operations/projections", identity.PermissionPlatformOperationsRead)
	addExpected(http.MethodPost, "/api/v1/operations/projections/reconcile", identity.PermissionPlatformOperationsWrite)
	addExpected(http.MethodGet, "/api/v1/operations/background-jobs", identity.PermissionPlatformJobsRead)
	addExpected(http.MethodGet, "/api/v1/compliance/automation-policies", identity.PermissionConfigRead)
	addExpected(http.MethodGet, "/api/v1/ai-governance/policies", identity.PermissionConfigRead)
	addExpected(http.MethodPost, "/api/v1/ai-governance/policies", identity.PermissionConfigWrite)
	addExpected(http.MethodPost, "/api/v1/ai-governance/policies/{id}/activate", identity.PermissionConfigWrite)
	addExpected(http.MethodGet, "/api/v1/ai-governance/workloads", identity.PermissionConfigRead)
	addExpected(http.MethodPost, "/api/v1/ai-governance/workloads", identity.PermissionConfigWrite)
	addExpected(http.MethodPost, "/api/v1/ai-governance/workloads/{id}/activate", identity.PermissionConfigWrite)
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

func TestThirdPartyActivationApprovalUsesBusinessAuthorityNotConfigurationPermission(t *testing.T) {
	for _, route := range (&API{}).routes() {
		if route.Method != http.MethodPost || route.Path != "/api/v1/third-party-activation-policies/{id}/approve" {
			continue
		}
		if route.Permission != "" {
			t.Fatalf("activation-policy approval requires unrelated administrative permission %q", route.Permission)
		}
		if route.Command == nil || route.Command.Name != "thirdparty.activation_policy.approve" {
			t.Fatalf("activation-policy approval lacks its material command: %#v", route.Command)
		}
		if route.Command.Policy.Responsibility != authority.ResponsibilityAuthorizer {
			t.Fatalf("activation-policy approval responsibility = %q, want %q", route.Command.Policy.Responsibility, authority.ResponsibilityAuthorizer)
		}
		return
	}
	t.Fatal("activation-policy approval route was not registered")
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

func TestMatterEditRoutesAreMaterialCommands(t *testing.T) {
	expected := map[string]string{
		"/api/v1/matters/{id}/details":                        "matter.details.update",
		"/api/v1/matters/{id}/context-changes":                "matter.context.change",
		"/api/v1/matters/{id}/assignment":                     "matter.assign",
		"/api/v1/matters/{id}/actions/{action_id}":            "matter.action.update",
		"/api/v1/matters/{id}/actions/{action_id}/assignment": "matter.action.assign",
	}
	seen := map[string]bool{}
	for _, route := range (&API{}).routes() {
		name, wanted := expected[route.Path]
		if !wanted || route.Method != http.MethodPost {
			continue
		}
		seen[route.Path] = true
		if route.Class != routeMaterialCommand || route.Command == nil || route.Command.Name != name {
			t.Fatalf("%s is not the governed command %s: %#v", route.Path, name, route)
		}
	}
	if len(seen) != len(expected) {
		t.Fatalf("missing Matter edit routes: got %#v want %#v", seen, expected)
	}
}

func TestProgramReviewAcceptanceIsAGovernedReviewerCommand(t *testing.T) {
	for _, route := range (&API{}).routes() {
		if route.Method != http.MethodPost || route.Path != "/api/v1/programs/{id}/reviews" {
			continue
		}
		if route.Class != routeMaterialCommand || route.Command == nil {
			t.Fatalf("Program review acceptance is not a material command: %#v", route)
		}
		if route.Command.Name != "program.review.accept" || route.Command.Policy.ObjectType != "PROGRAM" || route.Command.Policy.Responsibility != authority.ResponsibilityReviewer || route.Command.Policy.Materiality != 3 || route.Command.Policy.ActorField != noActorField {
			t.Fatalf("Program review acceptance has the wrong authority contract: %#v", route.Command)
		}
		return
	}
	t.Fatal("Program review acceptance route is missing")
}

func TestGovernanceMutationRoutesAreMaterialCommands(t *testing.T) {
	expected := map[string]string{
		"/api/v1/governance/policies":                 "governance.policy.create",
		"/api/v1/governance/policies/{id}/submit":     "governance.policy.submit",
		"/api/v1/governance/policies/{id}/approve":    "governance.policy.approve",
		"/api/v1/governance/policies/{id}/reject":     "governance.policy.reject",
		"/api/v1/governance/policies/{id}/retire":     "governance.policy.retire",
		"/api/v1/governance/delegations":              "governance.delegation.create",
		"/api/v1/governance/delegations/{id}/submit":  "governance.delegation.submit",
		"/api/v1/governance/delegations/{id}/approve": "governance.delegation.approve",
		"/api/v1/governance/delegations/{id}/revoke":  "governance.delegation.revoke",
	}
	seen := map[string]bool{}
	for _, route := range (&API{}).routes() {
		name, wanted := expected[route.Path]
		if !wanted || route.Method != http.MethodPost {
			continue
		}
		seen[route.Path] = true
		if route.Class != routeMaterialCommand || route.Command == nil || route.Command.Name != name {
			t.Fatalf("%s is not the governed command %s: %#v", route.Path, name, route)
		}
	}
	if len(seen) != len(expected) {
		t.Fatalf("missing governance material routes: got %#v want %#v", seen, expected)
	}
}
