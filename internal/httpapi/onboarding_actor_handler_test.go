package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/onboarding"
)

func TestOnboardingGuideUsesVerifiedActorRoles(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/onboarding/guide?role=GRC_ADMIN&surface=today", nil)
	request.Header.Set("X-ClearSight-Demo-Roles", "PROGRAM_OWNER")
	response := httptest.NewRecorder()
	testHandler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var guide onboarding.Guide
	if err := json.NewDecoder(response.Body).Decode(&guide); err != nil {
		t.Fatal(err)
	}
	if guide.Code != "program-owner-first-run" {
		t.Fatalf("client role override was trusted: %#v", guide)
	}
}

func TestOnboardingGuideRejectsClientRoleOverrideForVendorGuide(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/onboarding/guide?role=BUSINESS_OWNER&surface=vendors", nil)
	request.Header.Set("X-ClearSight-Demo-Roles", "PROGRAM_OWNER")
	response := httptest.NewRecorder()
	testHandler().ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", response.Code, response.Body.String())
	}
}

func TestOnboardingGuideRejectsExplicitGuideOutsideVerifiedRoleAndSurface(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/onboarding/guide?surface=today&code=vendor-operations-first-run", nil)
	request.Header.Set("X-ClearSight-Demo-Roles", "AUDITOR")
	response := httptest.NewRecorder()
	testHandler().ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", response.Code, response.Body.String())
	}
}

func TestOnboardingVendorGuideRequiresVerifiedPermission(t *testing.T) {
	now := time.Now().UTC()
	actor := identity.Actor{
		TenantID: "bank-demo", PrincipalID: "business-owner", LegalEntityID: "bank-ng", Kind: "PERSON",
		RoleCodes: []string{"BUSINESS_OWNER"}, AuthenticationMethod: "test", AssuranceLevel: "test", SessionID: "session",
		IssuedAt: now, ExpiresAt: now.Add(time.Hour),
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/onboarding/guide?surface=vendors&code=vendor-operations-first-run", nil)
	response := httptest.NewRecorder()
	onboardingGuideHandler(actor).ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected missing capability to return 404, got %d: %s", response.Code, response.Body.String())
	}

	actor.PermissionCodes = []string{identity.PermissionVendorRead}
	response = httptest.NewRecorder()
	onboardingGuideHandler(actor).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected verified capability to return 200, got %d: %s", response.Code, response.Body.String())
	}
	var guide onboarding.Guide
	if err := json.NewDecoder(response.Body).Decode(&guide); err != nil {
		t.Fatal(err)
	}
	if guide.Code != "vendor-operations-first-run" {
		t.Fatalf("unexpected vendor guide: %#v", guide)
	}
}

func onboardingGuideHandler(actor identity.Actor) http.Handler {
	return New(Dependencies{
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		Identity:   staticIdentityAuthenticator{actor: actor},
		Onboarding: onboarding.NewService(onboarding.NewMemoryRepository()),
	})
}
