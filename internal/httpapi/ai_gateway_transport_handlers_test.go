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
		status: aigateway.TransportApplyStatus{TenantID: "bank", Environment: "PRODUCTION", DesiredRevision: 4, AppliedRevision: 3, Degraded: true, ErrorCode: "TRANSPORT_APPLY_FAILED"},
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
	if response.Code != http.StatusOK || !strings.Contains(body, `"desired_revision":4`) || !strings.Contains(body, `"applied_revision":3`) || !strings.Contains(body, `"available":true`) {
		t.Fatalf("response = %d %s", response.Code, body)
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
