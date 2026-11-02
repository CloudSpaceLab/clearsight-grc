package organization

import (
	"strings"
	"testing"
)

func TestOrganizationScopeLineageMigrationIsAppendOnlyAndGuarded(t *testing.T) {
	up := readOrganizationMigration(t, "000130_organization_scope_lineage.up.sql")
	down := readOrganizationMigration(t, "000130_organization_scope_lineage.down.sql")

	for _, required := range []string{
		"CREATE TABLE organization_scope_lineage_events",
		"CREATE FUNCTION capture_organization_scope_lineage",
		"CREATE TRIGGER organization_scopes_capture_lineage",
		"CREATE TRIGGER organization_scope_lineage_immutable",
		"event_kind IN ('BACKFILL','CREATE','MOVE','RETIRE','STRUCTURE_CHANGE')",
		"department_path",
		"effective_at",
	} {
		if !strings.Contains(up, required) {
			t.Fatalf("up migration lacks %q", required)
		}
	}

	if !strings.Contains(down, "Organization scope lineage history exists; refusing to erase it") {
		t.Fatal("down migration must refuse to erase non-backfill lineage history")
	}
	guard := strings.Index(down, "Organization scope lineage history exists; refusing to erase it")
	firstDrop := strings.Index(down, "DROP TRIGGER")
	if guard < 0 || firstDrop < 0 || guard > firstDrop {
		t.Fatal("rollback guard must run before destructive DDL")
	}
}
