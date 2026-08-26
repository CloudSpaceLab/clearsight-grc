package thirdparty

import (
	"os"
	"strings"
	"testing"
)

func TestVendorBrandMigrationAddsScopedDurableStateWithoutParallelFoundations(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile("../../migrations/000047_vendor_brand_assets.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	schema := string(content)
	for _, required := range []string{
		"ALTER TABLE third_parties",
		"ADD COLUMN website_domain",
		"CREATE TABLE third_party_vendor_brand_assets",
		"CREATE TABLE third_party_vendor_brand_jobs",
		"UNIQUE (tenant_id,vendor_id)",
		"third_party_vendor_brand_jobs_claim_idx",
		"third_party_vendor_brand_assets_current_idx",
		"state IN ('READY','LEASED','COMPLETED','FAILED','CANCELLED')",
		"lease_token uuid",
		"lease_expires_at timestamptz",
		"artifact_key text",
		"source_digest text",
		"media_type text",
		"pixel_width integer",
		"pixel_height integer",
		"byte_size bigint",
		"'VENDOR'",
		"'VendorIdentityCreated'",
		"'VendorIdentityUpdated'",
	} {
		if !strings.Contains(schema, required) {
			t.Fatalf("vendor brand migration missing %q", required)
		}
	}
	for _, prohibited := range []string{"CREATE TABLE third_party_vendors", "CREATE TABLE outbox_events", "CREATE TABLE third_party_events", "remote_content", "image_bytes"} {
		if strings.Contains(schema, prohibited) {
			t.Fatalf("vendor brand migration duplicates or stores unsafe state with %q", prohibited)
		}
	}
}

func TestVendorBrandRollbackOnlyRemovesMigrationOwnedState(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile("../../migrations/000047_vendor_brand_assets.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	schema := string(content)
	for _, required := range []string{
		"DROP TABLE third_party_vendor_brand_jobs",
		"DROP TABLE third_party_vendor_brand_assets",
		"DROP COLUMN website_domain",
	} {
		if !strings.Contains(schema, required) {
			t.Fatalf("vendor brand rollback missing %q", required)
		}
	}
	for _, prohibited := range []string{"DROP TABLE third_parties", "DROP TABLE third_party_events", "DROP TABLE outbox_events", "DELETE FROM", "TRUNCATE"} {
		if strings.Contains(schema, prohibited) {
			t.Fatalf("vendor brand rollback changes pre-000047 state with %q", prohibited)
		}
	}
	if strings.Contains(schema, "ALTER TABLE third_party_events") || strings.Contains(schema, "CREATE OR REPLACE FUNCTION third_party_event_aggregate_guard") {
		t.Fatal("vendor brand rollback must preserve compatible event vocabulary and aggregate validation for retained vendor identity history")
	}
}
