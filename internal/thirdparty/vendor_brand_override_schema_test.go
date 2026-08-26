package thirdparty

import (
	"os"
	"strings"
	"testing"
)

func TestVendorBrandOverrideMigrationOwnsRecoveryAndEventVocabulary(t *testing.T) {
	body, err := os.ReadFile("../../migrations/000048_vendor_brand_overrides.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(body)
	for _, required := range []string{"third_party_vendor_brand_upload_reservations", "third_party_vendor_brand_command_receipts", "VendorBrandApproved", "VendorBrandRemoved", "expected_brand_version", "result_brand_version", "lease_token", "third_party_vendor_brand_upload_cleanup_idx", "third_party_vendor_brand_upload_artifact_idx"} {
		if !strings.Contains(sql, required) {
			t.Errorf("migration missing %q", required)
		}
	}
}
func TestVendorBrandOverrideRollbackPreservesAppendOnlyHistory(t *testing.T) {
	body, err := os.ReadFile("../../migrations/000048_vendor_brand_overrides.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(body)
	for _, prohibited := range []string{"DELETE FROM third_party_events", "DELETE FROM outbox_events", "DROP TYPE", "DROP CONSTRAINT third_party_events_event_type_check"} {
		if strings.Contains(sql, prohibited) {
			t.Errorf("rollback changes append-only history with %q", prohibited)
		}
	}
}
