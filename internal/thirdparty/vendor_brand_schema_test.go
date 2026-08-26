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
		"CREATE FUNCTION third_party_website_domain_valid(value text)",
		"char_length(label) BETWEEN 1 AND 63",
		"cardinality(string_to_array(value,'.')) BETWEEN 1 AND 4",
		"label ~ '^(?:[0-9]+|0x[0-9a-f]+)$'",
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
		"INSERT INTO third_party_events",
		"'VENDOR',p.id,p.version,NULL::uuid,'VendorIdentityCreated'",
		"ON CONFLICT (tenant_id,aggregate_type,aggregate_id,aggregate_version) DO NOTHING",
	} {
		if !strings.Contains(schema, required) {
			t.Fatalf("vendor brand migration missing %q", required)
		}
	}
	if strings.Contains(schema, "INSERT INTO outbox_events") {
		t.Fatal("vendor identity baseline backfill must not redeliver historical state through the outbox")
	}
	if got := strings.Count(schema, "third_party_website_domain_valid("); got != 4 {
		t.Fatalf("hostname validator definition/use count = %d, want one definition and three checks", got)
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
		"DROP FUNCTION third_party_website_domain_valid(text)",
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

func TestVendorIdentityPostgresEventsCarryReconstructableSafeState(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile("vendor_brand_postgres.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	for _, required := range []string{"'legal_name'", "'trading_name'", "'registration_ref'", "'jurisdiction'", "'website_domain'", "'status'"} {
		if got := strings.Count(source, required); got != 2 {
			t.Fatalf("vendor identity event/outbox payload field %s count = %d, want one in each payload", required, got)
		}
	}
	for _, prohibited := range []string{"'source_id'", "'external_ref'"} {
		if strings.Contains(source, prohibited) {
			t.Fatalf("vendor identity event/outbox payload includes unnecessary field %s", prohibited)
		}
	}
}
