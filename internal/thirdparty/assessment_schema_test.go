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
	downContent, err := os.ReadFile("../../migrations/000037_third_party_due_diligence.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	downSchema := string(downContent)
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
		"'AssessmentRequestPrepared'",
		"'AssessmentRequestReissued'",
		"'AssessmentRequestReissuePrepared'",
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
	for _, existingConstraint := range []string{"capture_submissions_id_tenant_request_key", "capture_invitations_id_tenant_request_key"} {
		if strings.Contains(schema, existingConstraint) {
			t.Fatalf("assessment migration attempts to recreate existing constraint %q", existingConstraint)
		}
		if strings.Contains(downSchema, existingConstraint) {
			t.Fatalf("assessment rollback attempts to remove existing constraint %q", existingConstraint)
		}
	}
	if !strings.Contains(schema, "invitation_id uuid,") || strings.Contains(schema, "invitation_id uuid NOT NULL") {
		t.Fatal("assessment request links must support the prepared state before invitation issuance")
	}
}
