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
