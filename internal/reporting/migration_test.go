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
		"status IN ('DRAFT','PENDING_REVIEW','REVIEWED','ACTIVE','RETIRED')",
		"maker_id uuid NOT NULL",
		"checker_id uuid",
		"reviewer_id uuid",
		"report_definitions_maker_tenant_fk",
		"report_definitions_checker_tenant_fk",
		"report_definitions_reviewer_tenant_fk",
		"report_definition_revisions_maker_tenant_fk",
		"report_definition_revisions_reviewer_tenant_fk",
		"report_definition_revisions_approver_tenant_fk",
		"reviewed_by uuid",
		"reviewed_at timestamptz",
		"definition_checksum text NOT NULL",
		"source_boundary jsonb NOT NULL",
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
	for _, dataset := range []string{
		"'PROCESSING_ACTIVITIES'",
		"'PROCESSING_ACTIVITY_EXCEPTIONS'",
		"'PROGRAMS'",
		"'MATTER_EXCEPTIONS'",
	} {
		if count := strings.Count(sql, dataset); count < 3 {
			t.Errorf("migration must permit %s in definitions, revisions, and runs; found %d occurrences", dataset, count)
		}
	}
	for _, forbidden := range []string{
		"report_definitions_owner_fk",
		"definition_code text NOT NULL CHECK (code ~",
		"maker_id text",
		"checker_id text",
		"reviewer_id text",
		"reviewed_by text",
		"approved_by text",
	} {
		if strings.Contains(sql, forbidden) {
			t.Errorf("migration retains misleading or non-principal constraint %q", forbidden)
		}
	}
}

func TestMigrationRevisionTriggerAllowsOnlyForwardDecisionColumns(t *testing.T) {
	up, err := os.ReadFile(filepath.Join("..", "..", "migrations", "000093_report_builder.up.sql"))
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(up)
	start := strings.Index(sql, "CREATE FUNCTION report_definition_revisions_immutable()")
	end := strings.Index(sql[start:], "CREATE TRIGGER report_definition_revisions_immutable")
	if start < 0 || end < 0 {
		t.Fatal("revision immutability function or trigger is missing")
	}
	function := sql[start : start+end]
	for _, required := range []string{
		"TG_OP = 'DELETE'",
		"NEW.definition_id",
		"NEW.tenant_id",
		"NEW.legal_entity_id",
		"NEW.version",
		"NEW.dataset",
		"NEW.scope_kind",
		"NEW.scope_ref",
		"NEW.format",
		"NEW.filter",
		"NEW.checksum",
		"NEW.maker_id",
		"NEW.created_at",
		"step integer",
		"NEW.decision",
		"OLD.decision IN ('APPROVED','REJECTED','RETIRED')",
	} {
		if !strings.Contains(function, required) {
			t.Errorf("revision trigger is missing %q", required)
		}
	}
	if strings.Contains(function, "RAISE EXCEPTION 'report definition revisions are immutable';") {
		t.Error("revision trigger still rejects every decision update")
	}
}
