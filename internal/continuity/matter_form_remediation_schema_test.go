package continuity

import (
	"os"
	"strings"
	"testing"
)

func TestMatterFormRemediationSchemaBindsExactActiveExternalSubject(t *testing.T) {
	raw, err := os.ReadFile("../../migrations/000071_matter_form_remediation.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	schema := string(raw)
	for _, required := range []string{
		"subject_type text NOT NULL CHECK (subject_type = 'MATTER')",
		"subject_id uuid NOT NULL CHECK (subject_id = matter_id)",
		"audience_class text NOT NULL CHECK (audience_class IN ('EXTERNAL'))",
		"status text NOT NULL CHECK (status = 'ACTIVE')",
		"effective_from timestamptz NOT NULL",
		"FOREIGN KEY (tenant_id, subject_id) REFERENCES matters(tenant_id, id)",
		"WHERE status = 'ACTIVE'",
	} {
		if !strings.Contains(schema, required) {
			t.Fatalf("schema missing %q", required)
		}
	}
}

func TestMatterFormApplicationSchedulesExactOutcomeReconciliationInMaterialTransaction(t *testing.T) {
	raw, err := os.ReadFile("matter_form_remediation_postgres.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	for _, required := range []string{
		"EventMatterFormVerificationDue", "verification_contract_id", "vc.observation_period_minutes",
		"GREATEST($7,COALESCE(ma.implemented_at,vc.created_at)", "ON CONFLICT DO NOTHING",
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("outcome reconciliation schedule missing %q", required)
		}
	}
	commit := strings.LastIndex(source, "r.commitContinuityEvents")
	schedule := strings.Index(source, "EventMatterFormVerificationDue")
	if schedule < 0 || commit < 0 || schedule > commit {
		t.Fatal("outcome reconciliation was not queued before the material transaction commit")
	}
}
