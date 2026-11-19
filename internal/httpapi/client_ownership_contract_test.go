package httpapi

import (
	"encoding/json"
	"os"
	"testing"
)

// An explicit boundary, not a wildcard exemption from legacy client coverage.
// Adding an operation to V2 requires updating this API gate and the paired V2 gate.
func TestV2ProgramClientOwnership(t *testing.T) {
	expected := map[string]string{
		"saveProgramFrameworkAdoptionDraft": "/api/v1/programs/{id}/standards/draft",
		"selectFrameworkQuestionnaire":      "/api/v1/programs/{id}/standards/{adoption_id}/questionnaire",
		"proposeRequirementReview":          "/api/v1/programs/{id}/requirements/{requirement_id}/review",
		"approveRequirementReview":          "/api/v1/programs/{id}/requirement-reviews/{review_id}/approve",
		"rejectRequirementReview":           "/api/v1/programs/{id}/requirement-reviews/{review_id}/reject",
		"proposeAssessmentSchedule":         "/api/v1/programs/{id}/assessment-schedules",
		"approveAssessmentSchedule":         "/api/v1/programs/{id}/assessment-schedules/{schedule_id}/approve",
		"stopAssessmentSchedule":            "/api/v1/programs/{id}/assessment-schedules/{schedule_id}/stop",
		"createAdoptedAssessment":           "/api/v1/programs/{id}/assessments",
		"reviewProgramAssessment":           "/api/v1/programs/{id}/assessment-reviews/{request_id}",
	}
	raw, err := os.ReadFile("../../api/runtime.openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var contract struct {
		Paths map[string]map[string]struct {
			ID         string `json:"operationId"`
			Client     string `json:"x-clearsight-client"`
			Class      string `json:"x-clearsight-route-class"`
			Permission string `json:"x-clearsight-permission"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(raw, &contract); err != nil {
		t.Fatal(err)
	}
	for path, methods := range contract.Paths {
		for method, op := range methods {
			if op.Client == "" {
				continue
			}
			want, ok := expected[op.ID]
			if !ok || op.Client != "clearsight2-grc" || path != want || method != "post" || op.Class != "MATERIAL_COMMAND" {
				t.Fatalf("unreviewed client ownership or weakened command contract: %s %s (%s)", method, path, op.ID)
			}
			delete(expected, op.ID)
		}
	}
	if len(expected) != 0 {
		t.Fatalf("missing V2-owned Program operations: %v", expected)
	}
}
