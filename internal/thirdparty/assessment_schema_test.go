package thirdparty

import (
	"os"
	"strings"
	"testing"
)

func TestAssessmentMigrationOwnsScopedAtomicWorkflowState(t *testing.T) {
	content, err := os.ReadFile("../../migrations/000037_third_party_due_diligence.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	schema := string(content)
	for _, required := range []string{
		"CREATE TABLE third_party_assessments",
		"UNIQUE (tenant_id,legal_entity_id,stable_episode_key)",
		"FOREIGN KEY (tenant_id,form_template_id,form_template_version)",
		"CREATE TABLE third_party_assessment_matter_links",
		"CREATE TABLE third_party_assessment_request_links",
		"CREATE TABLE third_party_assessment_reactions",
		"CREATE TABLE third_party_documents",
		"CREATE TABLE third_party_assessment_jobs",
		"third_party_assessment_jobs_claim_idx",
		"'THIRD_PARTY_ASSESSMENT'",
	} {
		if !strings.Contains(schema, required) {
			t.Fatalf("assessment migration missing %q", required)
		}
	}
	for _, prohibited := range []string{"recipient_email", "recipient_address", "invitation_token", "vendor_answers", "document_contents"} {
		if strings.Contains(strings.ToLower(schema), prohibited) {
			t.Fatalf("assessment migration contains protected payload column %q", prohibited)
		}
	}
}
