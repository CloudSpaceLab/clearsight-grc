package thirdparty

import (
	"os"
	"strings"
	"testing"
)

func TestVendorWorkSchemaKeepsTypedTargetCaptureHistoryAndRecovery(t *testing.T) {
	raw, err := os.ReadFile("../../migrations/000042_third_party_work_requests.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	schema := string(raw)
	for _, required := range []string{
		"CREATE TABLE third_party_work_requests", "program_link_id", "matter_link_id",
		"REFERENCES third_party_relationship_program_links", "REFERENCES third_party_relationship_matter_links",
		"CREATE TABLE third_party_work_capture_links", "origin_type='THIRD_PARTY_WORK'",
		"CREATE TABLE third_party_work_reactions", "CREATE TABLE third_party_work_events",
		"CREATE TABLE third_party_work_jobs", "third_party_work_requests_target_idx",
	} {
		if !strings.Contains(schema, required) {
			t.Fatalf("schema missing %q", required)
		}
	}
}
