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

func TestMonitoringMigrationIncludesCollectionRenewal(t *testing.T) {
	content, err := os.ReadFile("../../migrations/000036_program_collection_renewal.up.sql")
	if err != nil {
		t.Fatalf("read collection renewal migration: %v", err)
	}
	schema := string(content)
	for _, required := range []string{
		"validity_months",
		"renewal_window_days",
		"reminder_count",
		"origin_type",
		"origin_id",
		"origin_sequence",
		"predecessor_request_id",
		"previous_responses",
		"CREATE TABLE monitoring_collection_cycles",
		"monitoring_collection_cycles_due_idx",
		"monitoring_collection_cycles_program_idx",
		"capture_requests_origin_idx",
		"jsonb_typeof(previous_responses)='object'",
	} {
		if !strings.Contains(schema, required) {
			t.Fatalf("collection renewal migration missing %q", required)
		}
	}
	if !strings.HasPrefix(strings.TrimSpace(schema), "BEGIN;") || !strings.HasSuffix(strings.TrimSpace(schema), "COMMIT;") {
		t.Fatal("collection renewal migration must use the repository outer transaction convention")
	}
}
