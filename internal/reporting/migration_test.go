package reporting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrationDeclaresReportDefinitionSchema(t *testing.T) {
	up, err := os.ReadFile(filepath.Join("..", "..", "migrations", "000093_report_builder.up.sql"))
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(up)
	for _, required := range []string{
		"CREATE TABLE report_definitions",
		"CREATE TABLE report_definition_revisions",
		"CREATE TABLE report_runs",
		"report_definitions_owner_fk",
		"report_runs_definition_fk",
		"ALTER TABLE ropa_processing_activities ADD COLUMN matter_id",
		"ropa_matter_idx",
		"report_definition_revisions_immutable",
		"report_runs_generation_guard",
	} {
		if !strings.Contains(sql, required) {
			t.Errorf("migration is missing %q", required)
		}
	}
}
