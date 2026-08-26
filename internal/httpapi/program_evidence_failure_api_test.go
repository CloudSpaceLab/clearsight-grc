package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestProgramEvidenceFailureAPIRecordsResultAndReturnsLinkedIssueInWorkList(t *testing.T) {
	service := continuity.NewService(continuity.NewMemoryRepository())
	handler := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Mode: "test-memory",
		Identity: identity.NewDevelopmentAuthenticator("bank", "reviewer-owner", "bank-ng"), Continuity: service,
		Authority: &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
			authority.ResponsibilityOwner: {
				Principal:           authority.Principal{ID: "reviewer-owner", DisplayName: "Program owner"},
				CandidatePrincipals: []authority.Principal{{ID: "reviewer-owner", DisplayName: "Program owner"}},
			},
			authority.ResponsibilityReviewer: {
				Principal:           authority.Principal{ID: "reviewer-owner", DisplayName: "Evidence reviewer"},
				CandidatePrincipals: []authority.Principal{{ID: "reviewer-owner", DisplayName: "Evidence reviewer"}},
			},
			authority.ResponsibilityAuthorizer: {
				Principal:           authority.Principal{ID: "approver", DisplayName: "Approval authority"},
				CandidatePrincipals: []authority.Principal{{ID: "approver", DisplayName: "Approval authority"}},
			},
		}},
	})
	post := func(path, body string, target any) *httptest.ResponseRecorder {
		t.Helper()
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, strings.NewReader(body)))
		if response.Code < 200 || response.Code >= 300 {
			t.Fatalf("POST %s: %d %s", path, response.Code, response.Body.String())
		}
		if target != nil {
			if err := json.NewDecoder(response.Body).Decode(target); err != nil {
				t.Fatal(err)
			}
		}
		return response
	}

	var program continuity.ProgramAggregate
	post("/api/v1/programs", `{"tenant_id":"bank","legal_entity_id":"bank-ng","code":"EVIDENCE","name":"Evidence oversight","type":"COMPLIANCE","owning_function":"Compliance","owner_candidate_id":"reviewer-owner","approval_authority_candidate_id":"approver","scope":{},"effective_from":"2026-08-26T00:00:00Z"}`, &program)
	post("/api/v1/programs/"+program.Program.ID+"/requirements", `{"tenant_id":"bank","expected_version":1,"code":"REQ","title":"Retain evidence","statement":"Evidence must be retained.","status":"APPROVED","effective_from":"2026-08-26T00:00:00Z"}`, &program)
	post("/api/v1/programs/"+program.Program.ID+"/evidence-contracts", `{"tenant_id":"bank","expected_version":2,"requirement_id":"`+program.Requirements[0].ID+`","code":"CHECK","name":"Retention evidence","claim":"Required evidence is retained.","acceptable_source_ids":[],"population_scope":{},"freshness_minutes":60,"minimum_coverage":1,"contradiction_policy":"REVIEW","failure_action":"MATTER","status":"ACTIVE"}`, &program)
	post("/api/v1/programs/"+program.Program.ID+"/evidence-assessments", `{"tenant_id":"bank","expected_version":3,"contract_id":"`+program.EvidenceContracts[0].ID+`","conclusion":"UNSUPPORTED","coverage":0.5,"basis":{"missing":1},"assessed_by":"forged-reviewer","assessed_at":"2026-08-26T10:00:00Z"}`, &program)
	if len(program.EvidenceAssessments) != 1 || program.EvidenceAssessments[0].AssessedBy != "reviewer-owner" {
		t.Fatalf("API assessment identity/result = %#v", program.EvidenceAssessments)
	}

	work := httptest.NewRecorder()
	handler.ServeHTTP(work, httptest.NewRequest(http.MethodGet, "/api/v1/matters?tenant_id=bank&status=OPEN", nil))
	if work.Code != http.StatusOK || !strings.Contains(work.Body.String(), "Resolve failed evidence check: Retention evidence") || !strings.Contains(work.Body.String(), `"legal_entity_id":"bank-ng"`) {
		t.Fatalf("linked failure issue was not returned in work list: %d %s", work.Code, work.Body.String())
	}
}
