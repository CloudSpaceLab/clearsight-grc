package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/onboarding"
)

func TestOnboardingGuideUsesVerifiedActorRoles(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/onboarding/guide?role=GRC_ADMIN", nil)
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

func TestOnboardingGuideRejectsGuideOutsideVerifiedRole(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/onboarding/guide?code=configure-admin-first-run", nil)
	request.Header.Set("X-ClearSight-Demo-Roles", "AUDITOR")
	response := httptest.NewRecorder()
	testHandler().ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", response.Code, response.Body.String())
	}
}
