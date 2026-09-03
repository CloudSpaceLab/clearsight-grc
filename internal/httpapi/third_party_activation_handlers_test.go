package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

func TestThirdPartyActivationApprovalSeparatesBusinessAuthorityFromPlatformAdministration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	requestBody := `{"expected_version":2,"simulation_id":"simulation-1","rationale":"Approve after independent review of the complete simulation."}`

	t.Run("business authorizer does not need configuration write", func(t *testing.T) {
		guard, err := commandauth.New(commandAuthorityStub{principal: "business-checker"}, commandauth.ModeEnforce, logger)
		if err != nil {
			t.Fatal(err)
		}
		handler := New(Dependencies{
			Logger:       logger,
			Identity:     identity.NewDevelopmentAuthenticator("bank", "business-checker", "entity", "CRO"),
			CommandGuard: guard,
		})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/third-party-activation-policies/policy-1/approve", strings.NewReader(requestBody)))

		if response.Code == http.StatusForbidden && strings.Contains(response.Body.String(), "permission_required") {
			t.Fatalf("business authorizer was blocked by configuration permission: %s", response.Body.String())
		}
		if response.Header().Get("X-ClearSight-Command-Authorization") != "enforced" {
			t.Fatalf("business approval did not pass through enforced authority: %d %s", response.Code, response.Body.String())
		}
		if response.Code != http.StatusServiceUnavailable {
			t.Fatalf("business authorizer did not reach the policy handler: %d %s", response.Code, response.Body.String())
		}
	})

	t.Run("platform administrator is not the business checker", func(t *testing.T) {
		guard, err := commandauth.New(commandAuthorityStub{principal: "business-checker"}, commandauth.ModeEnforce, logger)
		if err != nil {
			t.Fatal(err)
		}
		handler := New(Dependencies{
			Logger:       logger,
			Identity:     identity.NewDevelopmentAuthenticator("bank", "platform-admin", "entity", "SYSTEM_ADMIN"),
			CommandGuard: guard,
		})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/third-party-activation-policies/policy-1/approve", strings.NewReader(requestBody)))

		if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), "approval_required") {
			t.Fatalf("platform administrator bypassed business authorization: %d %s", response.Code, response.Body.String())
		}
	})
}

func TestThirdPartyActivationCommandsAcceptServerBoundIdentity(t *testing.T) {
	tests := []struct {
		name string
		body string
		new  func() any
	}{
		{
			name: "propose policy",
			body: `{"tenant_id":"bank","legal_entity_id":"entity-a","allowed_conclusions":["SATISFACTORY"],"maximum_assessment_age_days":365,"address_verification_required":true,"conditional_conclusion_needs_terms":true,"effective_from":"2026-09-03T12:00:00Z","rationale":"Install the approved reference activation policy."}`,
			new:  func() any { return &activationPolicyProposalRequest{} },
		},
		{
			name: "submit or approve policy",
			body: `{"tenant_id":"bank","expected_version":2,"simulation_id":"simulation-1","rationale":"Approve the independently simulated policy revision."}`,
			new:  func() any { return &activationPolicyTransitionRequest{} },
		},
		{
			name: "prepare rollback",
			body: `{"tenant_id":"bank","effective_from":"2026-09-04T12:00:00Z","rationale":"Restore the previously approved policy after review."}`,
			new:  func() any { return &activationPolicyRollbackRequest{} },
		},
		{
			name: "activate relationship",
			body: `{"tenant_id":"bank","expected_version":3,"intended_effective_at":"2026-09-03T12:00:00Z","rationale":"Activate after every current policy gate passed."}`,
			new:  func() any { return &activateVendorRelationshipRequest{} },
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("POST", "/", strings.NewReader(test.body))
			response := httptest.NewRecorder()
			if err := httpx.DecodeJSON(response, request, test.new()); err != nil {
				t.Fatalf("server-bound identity must remain decodable: %v", err)
			}
		})
	}
}

func TestThirdPartyActivationSimulationFailuresIdentifyTheSafeRecoveryStage(t *testing.T) {
	tests := []struct {
		err  error
		code string
	}{
		{err: thirdparty.ErrActivationCandidateList, code: "activation_candidate_list_failed"},
		{err: thirdparty.ErrActivationCandidateFacts, code: "activation_candidate_evaluation_failed"},
		{err: thirdparty.ErrActivationSimulationStore, code: "activation_simulation_store_failed"},
	}
	for _, test := range tests {
		response := httptest.NewRecorder()
		writeThirdPartyActivationError(response, test.err)
		if response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), `"error":"`+test.code+`"`) {
			t.Fatalf("%s response = %d %s", test.code, response.Code, response.Body.String())
		}
	}
}
