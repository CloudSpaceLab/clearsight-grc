package organization

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOrganizationScopeMigrationPreservesGovernedHierarchyAndGuardedRollback(t *testing.T) {
	up := readOrganizationMigration(t, "000102_organization_scope_foundation.up.sql")
	down := readOrganizationMigration(t, "000102_organization_scope_foundation.down.sql")

	for _, required := range []string{
		"CREATE TABLE organization_scopes",
		"CREATE FUNCTION organization_scope_for_department_path",
		"CREATE TRIGGER org_positions_bind_organization_scope",
		"CREATE TRIGGER directory_group_role_bindings_bind_organization_scope",
		"SELECT reconcile_organization_scopes();",
		"AND origin='LEGACY_DEPARTMENT_PATH'",
	} {
		if !strings.Contains(up, required) {
			t.Fatalf("up migration lacks %q", required)
		}
	}

	if !strings.Contains(down, "DO $$") || !strings.Contains(down, "$$;") {
		t.Fatal("down migration must use a valid guarded DO block")
	}
	guard := strings.Index(down, "Retain managed organization scope history before downgrade")
	firstDDL := strings.Index(down, "DROP TRIGGER")
	if guard < 0 || firstDDL < 0 || guard > firstDDL {
		t.Fatal("down migration must refuse to erase managed scope history before changing schema")
	}
	if strings.Contains(strings.ToUpper(down), "DELETE FROM ORGANIZATION_SCOPES") {
		t.Fatal("down migration must not delete managed organization history")
	}
}

func readOrganizationMigration(t *testing.T, name string) string {
	t.Helper()
	payload, err := os.ReadFile(filepath.Join("..", "..", "migrations", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(payload)
}
