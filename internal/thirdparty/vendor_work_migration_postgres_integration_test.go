//go:build postgres && postgresintegration

package thirdparty

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestVendorWorkCanonicalRouteMigrationPreservesHistoricalRowsAndEnforcesNewProofs(t *testing.T) {
	pool := assessmentPostgresPool(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	down := readVendorWorkMigration(t, "../../migrations/000074_vendor_work_canonical_access_routes.down.sql")
	up := readVendorWorkMigration(t, "../../migrations/000074_vendor_work_canonical_access_routes.up.sql")
	migrationApplied := true
	t.Cleanup(func() {
		if !migrationApplied {
			if _, err := pool.Exec(context.Background(), up); err != nil {
				t.Errorf("restore migration 74 after failed test: %v", err)
			}
		}
	})
	if _, err := pool.Exec(ctx, down); err != nil {
		t.Fatalf("prepare pre-74 schema: %v", err)
	}
	migrationApplied = false

	relationship := seedAssessmentRelationship(t, pool, "Historical address verification")
	repository := NewPostgresRepository(pool)
	legacyLinkID := "33333333-3333-7333-8333-333333333381"
	legacyWorkID := "33333333-3333-7333-8333-333333333382"
	legacyCaptureID := "33333333-3333-7333-8333-333333333383"
	seedVendorWorkProgramLink(t, pool, relationship.Relationship.ID, "33333333-3333-7333-8333-333333333380", legacyLinkID, "MIGRATION-HISTORY", now)
	legacyWork := createMigrationVendorWork(t, repository, relationship.Relationship.ID, legacyLinkID, legacyWorkID, "33333333-3333-7333-8333-333333333380", now)
	legacyRequest := createMigrationEvidenceRequest(t, pool, relationship.Relationship.ID, legacyWork.ID, 1, "legacy@vendor.example", now)
	legacyWork, err := repository.AttachVendorWorkCapture(ctx, Scope{TenantID: legacyWork.TenantID, LegalEntityID: legacyWork.LegalEntityID}, legacyWork.ID, legacyWork.Version, VendorWorkCaptureLink{
		ID: legacyCaptureID, TenantID: legacyWork.TenantID, LegalEntityID: legacyWork.LegalEntityID,
		WorkRequestID: legacyWork.ID, RequestID: legacyRequest.ID, Sequence: 1, Purpose: "INITIAL", OriginVersion: 1, CreatedAt: now,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	evidenceService := evidence.NewService(evidence.NewPostgresRepository(pool), evidence.NewMemoryObjectStore())
	legacyInvitation, err := evidenceService.IssueInvitation(ctx, evidence.IssueInvitationInput{
		TenantID: legacyWork.TenantID, LegalEntityID: legacyWork.LegalEntityID, RequestID: legacyRequest.ID,
		Audience: "legacy@vendor.example", Purpose: "Historical address verification", TTLMinutes: 60, CreatedBy: thirdPartyPrincipal,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `ALTER TABLE third_party_work_requests DISABLE TRIGGER reject_retired_vendor_address_work_request_write`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `ALTER TABLE third_party_work_requests ENABLE TRIGGER reject_retired_vendor_address_work_request_write`)
	})
	if _, err := pool.Exec(ctx, `
		UPDATE third_party_work_requests
		SET request_kind='ADDRESS_VERIFICATION',current_invitation_id=$2::uuid,state='AWAITING_VENDOR',delivery_state='DELIVERED',version=version+1,updated_at=$3
		WHERE id=$1::uuid;
		UPDATE third_party_work_capture_links SET invitation_id=$2::uuid WHERE id=$4::uuid;
		ALTER TABLE third_party_work_requests ENABLE TRIGGER reject_retired_vendor_address_work_request_write`,
		pgx.QueryExecModeSimpleProtocol, legacyWork.ID, legacyInvitation.InvitationID, now.Add(time.Second), legacyCaptureID); err != nil {
		t.Fatal(err)
	}

	if _, err := pool.Exec(ctx, up); err != nil {
		t.Fatalf("migration 74 rewrote historical address work: %v", err)
	}
	migrationApplied = true
	assertVendorWorkInvitationAssociation(t, pool, legacyWork.ID, legacyCaptureID, legacyInvitation.InvitationID)

	invalidLinkID := "33333333-3333-7333-8333-333333333385"
	invalidWorkID := "33333333-3333-7333-8333-333333333386"
	seedVendorWorkProgramLink(t, pool, relationship.Relationship.ID, "33333333-3333-7333-8333-333333333384", invalidLinkID, "MIGRATION-INVALID", now)
	invalidWork := createMigrationVendorWork(t, repository, relationship.Relationship.ID, invalidLinkID, invalidWorkID, "33333333-3333-7333-8333-333333333384", now)
	if _, err := pool.Exec(ctx, `UPDATE third_party_work_requests SET current_request_id=$2::uuid,current_invitation_id=$3::uuid WHERE id=$1::uuid`, invalidWork.ID, legacyRequest.ID, "33333333-3333-7333-8333-333333333399"); err == nil {
		t.Fatal("new vendor-work write bypassed canonical route/request proof")
	}

	if _, err := pool.Exec(ctx, down); err != nil {
		t.Fatalf("legacy-only rollback failed: %v", err)
	}
	migrationApplied = false
	assertVendorWorkInvitationAssociation(t, pool, legacyWork.ID, legacyCaptureID, legacyInvitation.InvitationID)
	if _, err := pool.Exec(ctx, up); err != nil {
		t.Fatalf("reapply canonical migration: %v", err)
	}
	migrationApplied = true

	canonicalLinkID := "33333333-3333-7333-8333-333333333388"
	canonicalWorkID := "33333333-3333-7333-8333-333333333389"
	canonicalCaptureID := "33333333-3333-7333-8333-333333333390"
	canonicalProgramID := "33333333-3333-7333-8333-333333333387"
	seedVendorWorkProgramLink(t, pool, relationship.Relationship.ID, canonicalProgramID, canonicalLinkID, "MIGRATION-CANONICAL", now)
	canonicalWork := createMigrationVendorWork(t, repository, relationship.Relationship.ID, canonicalLinkID, canonicalWorkID, canonicalProgramID, now)
	dispatcher, access := migrationDistributionDispatcher(t, pool)
	dispatched, err := dispatcher.Dispatch(evidence.WithRequestOriginAuthority(ctx, VendorWorkOrigin), evidence.WorkflowDistributionDispatchInput{
		Request:      migrationEvidenceRequestInput(relationship.Relationship.ID, canonicalWork.ID, 1, "canonical@vendor.example", now),
		AccessPolicy: evidence.AccessDirectEmailOTP, RouteExpiresAt: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	canonicalWork, err = repository.AttachVendorWorkCapture(ctx, Scope{TenantID: canonicalWork.TenantID, LegalEntityID: canonicalWork.LegalEntityID}, canonicalWork.ID, canonicalWork.Version, VendorWorkCaptureLink{
		ID: canonicalCaptureID, TenantID: canonicalWork.TenantID, LegalEntityID: canonicalWork.LegalEntityID,
		WorkRequestID: canonicalWork.ID, RequestID: dispatched.Request.ID, Sequence: 1, Purpose: "INITIAL", OriginVersion: 1, CreatedAt: now,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	route, err := dispatcher.Resume(ctx, canonicalWork.TenantID, canonicalWork.LegalEntityID, dispatched.Request.ID, thirdPartyPrincipal, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	canonicalWork, err = repository.ReserveVendorWorkInvitation(ctx, Scope{TenantID: canonicalWork.TenantID, LegalEntityID: canonicalWork.LegalEntityID}, canonicalWork.ID, canonicalWork.Version, route.Route.RouteID, now)
	if err != nil {
		t.Fatal(err)
	}
	canonicalWork, err = repository.MarkVendorWorkSent(ctx, Scope{TenantID: canonicalWork.TenantID, LegalEntityID: canonicalWork.LegalEntityID}, canonicalWork.ID, canonicalWork.Version, route.Route.RouteID, VendorWorkDeliveryLinkAvailable, "Copy the secure link or retry email delivery.", now)
	if err != nil {
		t.Fatal(err)
	}
	_ = access

	connection, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, rollbackErr := connection.Exec(ctx, down)
	_, _ = connection.Exec(ctx, "ROLLBACK")
	connection.Release()
	if rollbackErr == nil {
		t.Fatal("canonical route rollback unexpectedly discarded active audit proof")
	}
	assertVendorWorkInvitationAssociation(t, pool, canonicalWork.ID, canonicalCaptureID, route.Route.RouteID)
	var accessRouteColumn bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='third_party_work_invitation_reservations' AND column_name='access_route_id')`).Scan(&accessRouteColumn); err != nil || !accessRouteColumn {
		t.Fatalf("failed rollback changed canonical schema: column=%v err=%v", accessRouteColumn, err)
	}
}

func readVendorWorkMigration(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func seedVendorWorkProgramLink(t *testing.T, pool *pgxpool.Pool, relationshipID, programID, linkID, code string, now time.Time) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO programs(id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,jurisdiction,effective_from)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4,$4,'COMPLIANCE','ACTIVE','Third-party risk','NG',$5);
		INSERT INTO third_party_relationship_program_links(id,tenant_id,legal_entity_id,relationship_id,program_id,purpose_code,purpose_label,state,created_by_principal_id,version,created_at,updated_at)
		VALUES($6::uuid,$2::uuid,$3::uuid,$7::uuid,$1::uuid,'EVIDENCE_REFRESH','Evidence refresh','ACTIVE',$8::uuid,1,$5,$5)`,
		pgx.QueryExecModeSimpleProtocol, programID, thirdPartyTenantID, thirdPartyEntityA, code, now, linkID, relationshipID, thirdPartyPrincipal); err != nil {
		t.Fatal(err)
	}
}

func createMigrationVendorWork(t *testing.T, repository *PostgresRepository, relationshipID, linkID, workID, programID string, now time.Time) VendorWorkRequest {
	t.Helper()
	work, err := repository.CreateVendorWork(context.Background(), VendorWorkRequest{
		ID: workID, TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA,
		RelationshipID: relationshipID, RelationshipLinkID: linkID, TargetType: LinkTargetProgram, TargetID: programID,
		RequestKind: VendorWorkGeneral, Purpose: "Confirm current vendor controls.", Instructions: "Confirm the current control information.",
		OwnerPrincipalID: thirdPartyPrincipal, FormTemplateID: assessmentTemplateID, FormTemplateVersion: 3,
		Presentation: formcontract.PresentationWizard, State: VendorWorkPreparing, DeliveryState: VendorWorkDeliveryNotSent,
		DueAt: now.Add(24 * time.Hour), Version: 1, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	return work
}

func createMigrationEvidenceRequest(t *testing.T, pool *pgxpool.Pool, relationshipID, workID string, sequence int64, audience string, now time.Time) evidence.Request {
	t.Helper()
	service := evidence.NewService(evidence.NewPostgresRepository(pool), evidence.NewMemoryObjectStore())
	request, err := service.CreateRequest(evidence.WithRequestOriginAuthority(context.Background(), VendorWorkOrigin), migrationEvidenceRequestInput(relationshipID, workID, sequence, audience, now))
	if err != nil {
		t.Fatal(err)
	}
	return request
}

func migrationEvidenceRequestInput(relationshipID, workID string, sequence int64, audience string, now time.Time) evidence.CreateRequestInput {
	return evidence.CreateRequestInput{
		TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA, SubjectType: "VENDOR_RELATIONSHIP", SubjectID: relationshipID,
		Title: "Vendor due diligence", Purpose: "Confirm current vendor controls.", WhyYou: "Confirm the current control information.",
		Sensitivity: "INTERNAL", AudienceType: "VENDOR", Recipient: evidence.RecipientInput{Type: evidence.RecipientExternalAudience, Audience: audience},
		EstimatedMinutes: 5, Deadline: now.Add(24 * time.Hour), Origin: evidence.RequestOrigin{Type: VendorWorkOrigin, ID: workID, Version: sequence},
		Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationWizard}, Sections: []formcontract.Section{{ID: "company", Title: "Company details"}},
		Fields:         []evidence.Field{{ID: "confirmed", SectionID: "company", Label: "Confirm the supplied details", Type: string(formcontract.TypeYesNo), Required: true}},
		FormTemplateID: assessmentTemplateID, FormTemplateVersion: 3, CreatedBy: thirdPartyPrincipal,
	}
}

func migrationDistributionDispatcher(t *testing.T, pool *pgxpool.Pool) (*evidence.WorkflowDistributionDispatcher, *evidence.DistributionAccessService) {
	t.Helper()
	repository := evidence.NewPostgresRepository(pool)
	var recipientKey, accessKey [32]byte
	for index := range recipientKey {
		recipientKey[index], accessKey[index] = byte(index+1), byte(index+33)
	}
	keyring, err := evidence.NewRecipientKeyring("migration-v1", map[string][32]byte{"migration-v1": recipientKey})
	if err != nil {
		t.Fatal(err)
	}
	store := evidence.NewPostgresDistributionStore(repository, keyring)
	access, err := evidence.NewDistributionAccessService(store, keyring, &vendorWorkOTPDelivery{}, accessKey, 20*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	return evidence.NewWorkflowDistributionDispatcher(evidence.NewDistributionService(store), access), access
}

func assertVendorWorkInvitationAssociation(t *testing.T, pool *pgxpool.Pool, workID, captureID, invitationID string) {
	t.Helper()
	var workInvitation, captureInvitation string
	if err := pool.QueryRow(context.Background(), `SELECT current_invitation_id::text FROM third_party_work_requests WHERE id=$1::uuid`, workID).Scan(&workInvitation); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT invitation_id::text FROM third_party_work_capture_links WHERE id=$1::uuid`, captureID).Scan(&captureInvitation); err != nil {
		t.Fatal(err)
	}
	if workInvitation != invitationID || captureInvitation != invitationID {
		t.Fatalf("invitation history changed: work=%q capture=%q want=%q", workInvitation, captureInvitation, invitationID)
	}
}
