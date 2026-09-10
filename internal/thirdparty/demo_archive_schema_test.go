package thirdparty

import (
	"os"
	"strings"
	"testing"
)

func TestDemoArchiveSchemaPreservesScopeAndRestorationHistory(t *testing.T) {
	raw, err := os.ReadFile("../../migrations/000089_demo_record_archives.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"CREATE TABLE demo_record_archives", "record_type='VENDOR_RELATIONSHIP'", "record_type='MATTER'",
		"FOREIGN KEY (vendor_relationship_id,tenant_id,legal_entity_id)", "FOREIGN KEY (matter_id,tenant_id,legal_entity_id)", "WHERE restored_at IS NULL",
		"source_manifest", "archived_by", "restored_by", "restoration_reason",
		"BEFORE UPDATE OR DELETE", "to_jsonb(NEW)", "to_jsonb(OLD)",
	} {
		if !strings.Contains(string(raw), required) {
			t.Fatalf("archive schema missing %q", required)
		}
	}
}
