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
		"third_party_work_requests_target_idx",
	} {
		if !strings.Contains(schema, required) {
			t.Fatalf("schema missing %q", required)
		}
	}
	if strings.Contains(schema, "CREATE TABLE third_party_work_jobs") {
		t.Fatal("vendor work migration creates a retry ledger without a worker that can claim it")
	}
}

func TestVendorWorkInvitationReservationMigrationKeepsPreIssueAssociationAndAudit(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000046_vendor_work_invitation_reservations.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000046_vendor_work_invitation_reservations.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"CREATE TABLE third_party_work_invitation_reservations",
		"invitation_id uuid PRIMARY KEY",
		"state text NOT NULL CHECK (state IN ('PENDING','FINALIZED','SUPERSEDED'))",
		"third_party_work_invitation_reservations_pending_idx",
		"VendorWorkInvitationReserved",
		"VendorWorkInvitationReady",
	} {
		if !strings.Contains(string(up), required) {
			t.Fatalf("invitation reservation migration missing %q", required)
		}
	}
	if strings.Contains(string(up), "token_hash") || strings.Contains(string(down), "DELETE FROM") || strings.Contains(string(down), "DROP TABLE") {
		t.Fatal("invitation reservation migrations must not persist tokens or rewrite work history")
	}
}
