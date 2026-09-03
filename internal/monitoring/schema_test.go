package monitoring

import (
	"os"
	"strings"
	"testing"
)

func TestMonitoringMigrationOwnsVersionedFormsChecksAndResults(t *testing.T) {
	content, err := os.ReadFile("../../migrations/000034_monitoring_setup.up.sql")
	if err != nil {
		t.Fatalf("read monitoring migration: %v", err)
	}
	schema := string(content)
	for _, required := range []string{
		"CREATE TABLE monitoring_form_templates",
		"CREATE TABLE monitoring_checks",
		"CREATE TABLE monitoring_results",
		"form_template_version",
		"collection_period_start",
		"UNIQUE(tenant_id, id, version)",
		"UNIQUE(tenant_id, monitoring_check_id, input_reference_id, evaluator_version)",
	} {
		if !strings.Contains(schema, required) {
			t.Fatalf("monitoring migration missing %q", required)
		}
	}
	if !strings.HasPrefix(strings.TrimSpace(schema), "BEGIN;") || !strings.HasSuffix(strings.TrimSpace(schema), "COMMIT;") {
		t.Fatal("monitoring migration must use the repository outer transaction convention")
	}
}

func TestMonitoringEventMigrationOwnsImmutableJournalAndOutboxDedupe(t *testing.T) {
	content, err := os.ReadFile("../../migrations/000040_monitoring_event_outbox.up.sql")
	if err != nil {
		t.Fatalf("read monitoring event migration: %v", err)
	}
	schema := string(content)
	for _, required := range []string{
		"ADD COLUMN legal_entity_id uuid",
		"ADD COLUMN program_id uuid",
		"monitoring_form_templates_program_entity_fk",
		"monitoring_checks_form_program_fk",
		"CREATE TABLE monitoring_events",
		"UNIQUE(tenant_id,aggregate_type,aggregate_id,aggregate_version)",
		"CREATE UNIQUE INDEX monitoring_outbox_event_uq",
		"MONITORING_FORM",
		"MONITORING_CHECK",
		"MONITORING_RESULT",
	} {
		if !strings.Contains(schema, required) {
			t.Fatalf("monitoring event migration missing %q", required)
		}
	}
	if !strings.HasPrefix(strings.TrimSpace(schema), "BEGIN;") || !strings.HasSuffix(strings.TrimSpace(schema), "COMMIT;") {
		t.Fatal("monitoring event migration must be transactional")
	}
}

func TestLegalEntityFormsMigrationPromotesWithoutRewritingHistoricalRows(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000053_legal_entity_forms.up.sql")
	if err != nil {
		t.Fatalf("read legal-entity Forms migration: %v", err)
	}
	down, err := os.ReadFile("../../migrations/000053_legal_entity_forms.down.sql")
	if err != nil {
		t.Fatalf("read legal-entity Forms rollback: %v", err)
	}
	for _, required := range []string{
		"ADD COLUMN owner_principal_id", "ADD COLUMN scoring_mode", "form_saved_views",
		"monitoring_form_templates_entity_current_code_idx", "monitoring_form_templates_unscoped_current_code_idx",
		"program_id IS NULL OR legal_entity_id IS NOT NULL", "jsonb_array_elements", "FOREIGN KEY (legal_entity_id,tenant_id)",
	} {
		if !strings.Contains(string(up), required) {
			t.Fatalf("legal-entity Forms migration missing %q", required)
		}
	}
	for _, prohibited := range []string{"DELETE FROM monitoring_form_templates", "UPDATE monitoring_form_templates SET legal_entity_id"} {
		if strings.Contains(string(up), prohibited) || strings.Contains(string(down), prohibited) {
			t.Fatalf("migration rewrites or deletes historical forms with %q", prohibited)
		}
	}
	if !strings.Contains(string(down), "RAISE EXCEPTION") || !strings.Contains(string(down), "program_id IS NULL AND legal_entity_id IS NOT NULL") {
		t.Fatal("rollback must fail closed after legal-entity-only Forms adoption")
	}
}

func TestMonitoringMigrationIncludesCollectionRenewal(t *testing.T) {
	content, err := os.ReadFile("../../migrations/000077_program_collection_renewal.up.sql")
	if err != nil {
		t.Fatalf("read collection renewal migration: %v", err)
	}
	schema := string(content)
	for _, required := range []string{
		"validity_months",
		"renewal_window_days",
		"reminder_count",
		"origin_type",
		"origin_version",
		"predecessor_request_id",
		"previous_responses",
		"latest_respondent_label",
		"CREATE TABLE monitoring_collection_cycles",
		"monitoring_collection_cycles_due_idx",
		"monitoring_collection_cycles_program_idx",
		"jsonb_typeof(previous_responses)='object'",
	} {
		if !strings.Contains(schema, required) {
			t.Fatalf("collection renewal migration missing %q", required)
		}
	}
	if !strings.HasPrefix(strings.TrimSpace(schema), "BEGIN;") || !strings.HasSuffix(strings.TrimSpace(schema), "COMMIT;") {
		t.Fatal("collection renewal migration must use the repository outer transaction convention")
	}
	if strings.Contains(schema, "ADD COLUMN origin_type") || strings.Contains(schema, "CREATE UNIQUE INDEX capture_requests_origin_idx") {
		t.Fatal("collection renewal migration must reuse the shared capture origin contract")
	}
}
