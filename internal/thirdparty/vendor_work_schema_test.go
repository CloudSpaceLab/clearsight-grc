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

func TestVendorWorkRequestKindMigrationIsBoundedAndReversible(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000059_vendor_work_request_kind.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000059_vendor_work_request_kind.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"request_kind text NOT NULL DEFAULT 'GENERAL'", "'CERTIFICATION_REFRESH'"} {
		if !strings.Contains(string(up), required) {
			t.Fatalf("request-kind migration missing %q", required)
		}
	}
	if !strings.Contains(string(down), "DROP COLUMN IF EXISTS request_kind") || strings.Contains(string(down), "DELETE FROM") {
		t.Fatal("request-kind rollback must remove only the added column")
	}
}

func TestCurrentVendorWorkRequestKindsExcludeInternalAddressVerification(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000072_remove_vendor_work_address_compatibility.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	schema := string(up)
	for _, required := range []string{
		"CREATE TRIGGER reject_retired_vendor_address_work_request_write",
		"BEFORE INSERT OR UPDATE ON third_party_work_requests",
		"IF NEW.request_kind = 'ADDRESS_VERIFICATION'",
	} {
		if !strings.Contains(schema, required) {
			t.Fatalf("retired address-verification write guard missing %q", required)
		}
	}
	if strings.Contains(schema, "UPDATE third_party_work_requests") || strings.Contains(schema, "DELETE FROM third_party_work_requests") {
		t.Fatal("retiring the runtime path must not rewrite or delete historical work records")
	}
	if !strings.Contains(schema, "NEW.state = 'CANCELLED'") || !strings.Contains(schema, "OLD.request_kind = 'ADDRESS_VERIFICATION'") {
		t.Fatal("historical address work must allow only its governed terminal cancellation")
	}
}

func TestVendorWorkCanonicalAccessRouteMigrationRemovesLegacyInvitationProof(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000074_vendor_work_canonical_access_routes.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000074_vendor_work_canonical_access_routes.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	schema := string(up)
	for _, required := range []string{
		"BEGIN;", "COMMIT;",
		"third_party_work_requests_access_route_fk",
		"third_party_work_capture_links_access_route_fk",
		"access_route_id uuid",
		"validate_vendor_work_access_route_proof",
		"REFERENCES third_party_work_invitation_reservations(access_route_id,tenant_id,request_id)",
		"UPDATE third_party_work_requests SET current_invitation_id=NULL",
		"UPDATE third_party_work_capture_links SET invitation_id=NULL",
	} {
		if !strings.Contains(schema, required) {
			t.Fatalf("canonical vendor-work route migration missing %q", required)
		}
	}
	if strings.Contains(string(down), "UPDATE third_party_work_requests SET current_invitation_id=NULL") || strings.Contains(string(down), "UPDATE third_party_work_capture_links SET invitation_id=NULL") {
		t.Fatal("canonical route rollback must not erase vendor-work audit associations")
	}
	if !strings.Contains(string(down), "cannot roll back canonical vendor-work access routes") {
		t.Fatal("canonical route rollback must fail closed while route associations exist")
	}
}

func TestVendorWorkRuntimeCannotCallLegacyInvitationIssuer(t *testing.T) {
	raw, err := os.ReadFile("vendor_work.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	for _, retired := range []string{"IssueInvitation(", "RedeemInvitation(", "capture_invitations"} {
		if strings.Contains(source, retired) {
			t.Fatalf("vendor-work runtime retained legacy invitation path %q", retired)
		}
	}
}
