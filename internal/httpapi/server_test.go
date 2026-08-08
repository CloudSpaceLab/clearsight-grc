package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/onboarding"
	"github.com/CloudSpaceLab/clearsight-grc/internal/today"
	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
)

const demoExternalEvidenceRequestID = "019fd444-4444-7444-8444-444444444444"

func testHandler() http.Handler {
	version, rules := authority.DemoPolicySet()
	auto := autonomy.NewService(autonomy.NewMemoryRepository())
	autonomy.SeedDemo(context.Background(), auto)
	requests := evidence.DemoRequests()
	if len(requests) > 0 {
		requests[0].Recipient = evidence.Recipient{Type: evidence.RecipientInternalPrincipal, PrincipalID: "role-cro"}
		requests[0].CreatedBy = "role-cro"
		external := requests[0]
		external.ID = demoExternalEvidenceRequestID
		external.Title = "Confirm external resilience response"
		external.Purpose = "Collect one bounded response from the designated external contact."
		external.WhyYou = "You are the designated external respondent."
		external.Sensitivity = "CONFIDENTIAL"
		external.AudienceType = "EXTERNAL"
		external.Recipient = evidence.Recipient{Type: evidence.RecipientExternalAudience, AudienceHint: "m***@example.com"}
		external.CreatedBy = "role-cro"
		requests = append(requests, external)
	}
	evidenceService := evidence.NewService(evidence.NewMemoryRepository(evidence.DemoSources(), requests), evidence.NewMemoryObjectStore())
	return New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), AllowedOrigin: "http://localhost:5173", Mode: "test-memory",
		Identity: identity.NewDevelopmentAuthenticator("bank-demo", "role-cro", "bank-ng"), Authority: authority.NewResolver(version, rules),
		Evidence: evidenceService,
		Today:    today.NewService(today.DemoItems()), Workflow: workflow.NewService(workflow.NewMemoryRepository(workflow.DemoTasks())),
		Onboarding: onboarding.NewService(onboarding.NewMemoryRepository()), Autonomy: auto, MaxArtifactBytes: 1 << 20,
	})
}

func TestTodayEndpoint(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/today", nil)
	response := httptest.NewRecorder()
	testHandler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var body struct {
		Items []today.AttentionItem `json:"items"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Items) != 3 {
		t.Fatalf("expected 3 attention items, got %d", len(body.Items))
	}
}

func TestAuthorityResolutionEndpoint(t *testing.T) {
	payload := []byte(`{"tenant_id":"bank-demo","legal_entity_id":"bank-ng","object_type":"MATTER","object_id":"matter-1","responsibility":"AUTHORIZER","materiality":5}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/authority/resolve", bytes.NewReader(payload))
	response := httptest.NewRecorder()
	testHandler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
}

func TestReadinessEndpointDoesNotFabricateBaseline(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/compliance/readiness?tenant_id=bank-demo", nil)
	response := httptest.NewRecorder()
	testHandler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var body autonomy.Readiness
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.BaselineKnown || body.Dimensions.Current != 0 {
		t.Fatalf("unexpected fabricated baseline: %#v", body)
	}
}

func TestWorkflowListUsesVerifiedTenant(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/workflow/tasks", nil)
	response := httptest.NewRecorder()
	testHandler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
}

func TestWorkflowTaskMutationRouteIsNotExposed(t *testing.T) {
	payload := []byte(`{"tenant_id":"bank-demo","status":"IN_PROGRESS","expected_version":1}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/workflow/tasks/task_review_cbn/transition", bytes.NewReader(payload))
	response := httptest.NewRecorder()
	testHandler().ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected projected tasks to be read-only at the HTTP boundary, got %d: %s", response.Code, response.Body.String())
	}
}

func TestDuplicateSignalReturnsNoSyntheticDrift(t *testing.T) {
	payload := []byte(`{"tenant_id":"bank-demo","type":"EVIDENCE_EXPIRED","subject_type":"CLAIM","subject_id":"claim-1","dedupe_key":"claim-1-expired","source":"scheduler"}`)
	handler := testHandler()
	for attempt := 0; attempt < 2; attempt++ {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/compliance/signals", bytes.NewReader(payload))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusAccepted {
			t.Fatalf("expected 202, got %d: %s", response.Code, response.Body.String())
		}
		if attempt == 1 {
			var body struct {
				Inserted bool            `json:"inserted"`
				Drift    *autonomy.Drift `json:"drift"`
			}
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body.Inserted || body.Drift != nil {
				t.Fatalf("expected duplicate without drift, got inserted=%v drift=%#v", body.Inserted, body.Drift)
			}
		}
	}
}
