package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/aigateway"
	"github.com/CloudSpaceLab/clearsight-grc/internal/aigovernance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type testAIGatewayOperationsReader struct {
	status aigateway.TransportApplyStatus
	err    error
	seen   func(string, string)
}

func (reader testAIGatewayOperationsReader) TransportStatus(_ context.Context, tenantID, environment string) (aigateway.TransportApplyStatus, error) {
	if reader.seen != nil {
		reader.seen(tenantID, environment)
	}
	return reader.status, reader.err
}

func TestListAIGatewayTransportsProjectsVerifiedRuntimeScope(t *testing.T) {
	service := aigovernance.NewService(aigovernance.NewMemoryRepository(), nil, nil, nil)
	reader := testAIGatewayOperationsReader{
		status: aigateway.TransportApplyStatus{
			TenantID: "bank", Environment: "PRODUCTION", DesiredRevision: 4, AppliedRevision: 3,
			EmergencySupported: true, EmergencyRevision: 2, OutboundFrozen: true,
			Degraded: true, ErrorCode: "TRANSPORT_APPLY_FAILED",
		},
		seen: func(tenantID, environment string) {
			if tenantID != "bank" || environment != "PRODUCTION" {
				t.Fatalf("operations scope = %s/%s", tenantID, environment)
			}
		},
	}
	api := &API{deps: Dependencies{AIGovernance: service, AIGatewayOperations: reader}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/ai-governance/gateway-configs?environment=production", nil)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{TenantID: "bank", PrincipalID: "admin"}))
	response := httptest.NewRecorder()
	api.listAIGatewayTransports(response, request)
	body := response.Body.String()
	for _, expected := range []string{`"desired_revision":4`, `"applied_revision":3`, `"available":true`, `"emergency_supported":true`, `"emergency_revision":2`, `"outbound_frozen":true`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("response missing %s: %d %s", expected, response.Code, body)
		}
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("cache control = %q", response.Header().Get("Cache-Control"))
	}
}

func TestListAIGatewayTransportsReportsOperationsBridgeAvailabilityTruthfully(t *testing.T) {
	service := aigovernance.NewService(aigovernance.NewMemoryRepository(), nil, nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/ai-governance/gateway-configs?environment=PRODUCTION", nil)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{TenantID: "bank", PrincipalID: "admin"}))

	response := httptest.NewRecorder()
	(&API{deps: Dependencies{AIGovernance: service}}).listAIGatewayTransports(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"configured":false`) || !strings.Contains(response.Body.String(), `"available":false`) {
		t.Fatalf("unconfigured response = %d %s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	api := &API{deps: Dependencies{AIGovernance: service, AIGatewayOperations: testAIGatewayOperationsReader{err: errors.New("offline")}}}
	api.listAIGatewayTransports(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"configured":true`) || !strings.Contains(response.Body.String(), `"error_code":"GATEWAY_STATUS_UNAVAILABLE"`) {
		t.Fatalf("unavailable response = %d %s", response.Code, response.Body.String())
	}
}

func TestListAIGatewayTransportsProjectsOnlyPublicProxyAndWorkloadIngress(t *testing.T) {
	service := aigovernance.NewService(aigovernance.NewMemoryRepository(), nil, nil, nil)
	api := &API{deps: Dependencies{
		AIGovernance:           service,
		AIGatewayPublicBaseURL: "https://ai.bank.example/proxy",
	}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/ai-governance/gateway-configs?environment=PRODUCTION", nil)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{TenantID: "bank", PrincipalID: "admin"}))
	response := httptest.NewRecorder()
	api.listAIGatewayTransports(response, request)
	body := response.Body.String()
	for _, expected := range []string{
		`"proxy":{"configured":true`,
		`"base_url":"https://ai.bank.example/proxy"`,
		`"path":"/v1/models"`,
		`"path":"/v1/chat/completions"`,
		`"path":"/v1/responses"`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("response missing %s: %s", expected, body)
		}
	}
	for _, prohibited := range []string{"/metrics", "/health/config", "/health/live", "/health/ready"} {
		if strings.Contains(body, prohibited) {
			t.Fatalf("response exposed non-workload route %q: %s", prohibited, body)
		}
	}
}

func TestAIGatewayEmergencyCommandUsesVerifiedActorAndTenant(t *testing.T) {
	service := aigovernance.NewService(aigovernance.NewMemoryRepository(), nil, nil, nil)
	api := &API{deps: Dependencies{AIGovernance: service}}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/ai-governance/gateway-configs/emergency", strings.NewReader(`{
		"tenant_id":"spoofed-bank",
		"actor_id":"spoofed-user",
		"environment":"production",
		"frozen":true,
		"reason":"Potential provider credential compromise",
		"expected_version":0
	}`))
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{TenantID: "bank", PrincipalID: "admin"}))
	response := httptest.NewRecorder()
	api.setAIGatewayEmergencyControl(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
	value, err := service.GatewayEmergencyControl(context.Background(), "bank", "PRODUCTION")
	if err != nil {
		t.Fatal(err)
	}
	if !value.Frozen || value.TenantID != "bank" || value.ActorID != "admin" || value.RecordVersion != 1 {
		t.Fatalf("stored control = %#v", value)
	}
	if _, err := service.GatewayEmergencyControl(context.Background(), "spoofed-bank", "PRODUCTION"); err != nil {
		t.Fatal(err)
	}
}
