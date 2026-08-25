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

func TestSharedFormCaptureMigrationPersistsPresentationSectionsAndExpandedFieldBound(t *testing.T) {
	content, err := os.ReadFile("../../migrations/000036_shared_form_capture_contract.up.sql")
	if err != nil {
		t.Fatalf("read shared form capture migration: %v", err)
	}
	schema := string(content)
	for _, required := range []string{
		"ADD COLUMN presentation jsonb",
		"ADD COLUMN sections jsonb",
		"jsonb_array_length(fields) BETWEEN 1 AND 200",
		"ADD COLUMN origin_type text",
		"CREATE TABLE capture_response_drafts",
		"UNIQUE(tenant_id,request_id,session_id)",
		"FOREIGN KEY (session_id,tenant_id,request_id)",
		"octet_length(answers::text) <= 1048576",
	} {
		if !strings.Contains(schema, required) {
			t.Fatalf("shared form capture migration missing %q", required)
		}
	}
}
