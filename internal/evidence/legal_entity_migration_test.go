package evidence

import (
	"os"
	"strings"
	"testing"
)

func TestEvidenceRequestLegalEntityMigrationIsFailClosedAndReversible(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000037_evidence_request_legal_entity_scope.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	upSQL := string(up)
	for _, required := range []string{
		"ALTER TABLE capture_requests ADD COLUMN legal_entity_id uuid",
		"subject_type='PROGRAM'",
		"subject_type='MATTER'",
		"unresolved_count",
		"ALTER COLUMN legal_entity_id SET NOT NULL",
		"FOREIGN KEY (legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id)",
		"capture_requests_legal_entity_immutable",
	} {
		if !strings.Contains(upSQL, required) {
			t.Fatalf("up migration lacks %q", required)
		}
	}
	down, err := os.ReadFile("../../migrations/000037_evidence_request_legal_entity_scope.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	downSQL := string(down)
	for _, required := range []string{
		"DROP TRIGGER IF EXISTS capture_requests_legal_entity_immutable ON capture_requests",
		"DROP COLUMN IF EXISTS legal_entity_id",
	} {
		if !strings.Contains(downSQL, required) {
			t.Fatalf("down migration lacks %q", required)
		}
	}
}

func TestFormDistributionMigrationPinsScopeAndIsReversible(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000054_form_distributions.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	upSQL := string(up)
	for _, required := range []string{
		"CREATE TABLE capture_form_distributions",
		"FOREIGN KEY (tenant_id,form_template_id,form_template_version) REFERENCES monitoring_form_templates(tenant_id,id,version)",
		"CHECK (route_expires_at <= deadline)",
		"CREATE TABLE capture_distribution_recipients",
		"CHECK ((role='TO' AND request_id IS NOT NULL) OR (role='CC' AND request_id IS NULL))",
		"CREATE UNIQUE INDEX capture_distribution_recipients_request_uq",
		"CREATE TABLE capture_access_routes",
		"CREATE TABLE capture_otp_challenges",
		"CREATE TABLE capture_response_workspaces",
		"UNIQUE (distribution_id)",
		"CREATE TABLE capture_response_workspace_edits",
		"CREATE TABLE capture_response_revisions",
		"achieved_assurance text NOT NULL CHECK (achieved_assurance IN ('LINK_POSSESSION','EMAIL_VERIFIED'))",
		"ADD COLUMN distribution_id uuid",
		"capture_form_distributions_deadline_idx",
		"capture_form_distributions_updated_idx",
		"capture_distribution_events",
		"aggregate_type='FORM_DISTRIBUTION'",
	} {
		if !strings.Contains(upSQL, required) {
			t.Fatalf("distribution up migration lacks %q", required)
		}
	}

	down, err := os.ReadFile("../../migrations/000054_form_distributions.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	downSQL := string(down)
	for _, required := range []string{
		"IF EXISTS (SELECT 1 FROM capture_form_distributions)",
		"cannot roll back form distributions after distribution or delivery history exists",
		"DROP TABLE IF EXISTS capture_response_revisions",
		"DROP TABLE IF EXISTS capture_response_workspaces",
		"DROP TABLE IF EXISTS capture_distribution_recipients",
		"DROP COLUMN IF EXISTS distribution_id",
		"DROP TABLE IF EXISTS capture_form_distributions",
	} {
		if !strings.Contains(downSQL, required) {
			t.Fatalf("distribution down migration lacks %q", required)
		}
	}
}
