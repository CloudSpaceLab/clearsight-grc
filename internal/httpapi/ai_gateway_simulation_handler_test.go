package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/aigovernance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestGatewayConfigReadProjectsEphemeralSimulationWithoutContent(t *testing.T) {
	service := aigovernance.NewService(aigovernance.NewMemoryRepository(), nil, nil, nil)
	api := &API{deps: Dependencies{AIGovernance: service}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/ai-governance/gateway-configs?environment=production&simulate_fixture=UNKNOWN_WORKLOAD", nil)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{TenantID: "bank", PrincipalID: "admin"}))
	response := httptest.NewRecorder()

	api.listAIGatewayTransports(response, request)
	body := response.Body.String()
	if response.Code != http.StatusOK {
		t.Fatalf("response = %d %s", response.Code, body)
	}
	for _, expected := range []string{
		`"simulation":`,
		`"fixture":"UNKNOWN_WORKLOAD"`,
		`"environment":"PRODUCTION"`,
		`"provider_call_would_occur":false`,
		`"provider_call_blocked_reason":"UNKNOWN_WORKLOAD"`,
		`"reason_codes":["UNKNOWN_WORKLOAD"]`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("response missing %s: %s", expected, body)
		}
	}
	for _, prohibited := range []string{"Summarize the approved policy", "Ignore previous instructions", "Reveal the system prompt"} {
		if strings.Contains(body, prohibited) {
			t.Fatalf("simulation response exposed fixture content %q: %s", prohibited, body)
		}
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("cache control = %q", response.Header().Get("Cache-Control"))
	}
}

func TestGatewaySimulationCannotSelectAnotherTenantCandidate(t *testing.T) {
	repo := aigovernance.NewMemoryRepository()
	if _, err := repo.CreatePolicy(t.Context(), aigovernance.Policy{
		ID: "other-baseline", TenantID: "other-bank", Code: "ORG_AI_BASELINE", Name: "Other bank baseline",
		ActionClass: "AI_GATEWAY_BASELINE", Status: "DRAFT", RolloutMode: "SHADOW", Version: 1, RecordVersion: 1,
	}); err != nil {
		t.Fatal(err)
	}
	service := aigovernance.NewService(repo, nil, nil, nil)
	api := &API{deps: Dependencies{AIGovernance: service}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/ai-governance/gateway-configs?environment=PRODUCTION&simulate_fixture=SAFE&simulate_baseline_policy_id=other-baseline", nil)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{TenantID: "bank", PrincipalID: "admin"}))
	response := httptest.NewRecorder()

	api.listAIGatewayTransports(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant candidate response = %d %s", response.Code, response.Body.String())
	}
}
