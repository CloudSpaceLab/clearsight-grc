package continuity

import (
	"os"
	"strings"
	"testing"
)

func TestRelationshipRetirementMigrationKeepsHistoryAndCurrentIndexes(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000043_relationship_retirement.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	content := string(up)
	for _, required := range []string{
		"retired_at timestamptz",
		"retired_by uuid",
		"requirement_control_links_retired_by_tenant_fk",
		"matter_links_retired_by_tenant_fk",
		"REFERENCES principals(id,tenant_id)",
		"retirement_reason text",
		"requirement_control_links_current_unique_idx",
		"matter_links_current_unique_idx",
		"WHERE retired_at IS NULL",
		"WHERE program_id IS NOT NULL AND retired_at IS NULL",
	} {
		if !strings.Contains(content, required) {
			t.Fatalf("relationship retirement migration is missing %q", required)
		}
	}
}

func TestRelationshipRetirementDownFailsBeforeDDLWhenHistoryExists(t *testing.T) {
	down, err := os.ReadFile("../../migrations/000043_relationship_retirement.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	content := string(down)
	preflight := strings.Index(content, "cannot roll back relationship retirement while retired material links exist")
	firstDDL := strings.Index(content, "DROP INDEX")
	if preflight < 0 || firstDDL < 0 || preflight > firstDDL {
		t.Fatal("down migration must fail closed before changing the schema when retired links exist")
	}
	if strings.Contains(strings.ToUpper(content), "DELETE FROM MATTER_LINKS") || strings.Contains(strings.ToUpper(content), "DELETE FROM REQUIREMENT_CONTROL_LINKS") {
		t.Fatal("down migration must not delete retired material relationship history")
	}
}
