//go:build postgres && postgresintegration

package thirdparty

import (
	"context"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/jackc/pgx/v5"
)

func TestPostgresVendorWorkUsesCanonicalOTPRouteAndSubmitsAfterAutosave(t *testing.T) {
	pool := assessmentPostgresPool(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	const (
		programID = "33333333-3333-7333-8333-333333333375"
		linkID    = "33333333-3333-7333-8333-333333333376"
		workID    = "33333333-3333-7333-8333-333333333377"
		captureID = "33333333-3333-7333-8333-333333333378"
	)
	relationship := seedAssessmentRelationship(t, pool, "Canonical vendor-work capture")
	if _, err := pool.Exec(ctx, `
		INSERT INTO programs(id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,jurisdiction,effective_from)
		VALUES($1::uuid,$2::uuid,$3::uuid,'VENDOR-OTP','Vendor OTP capture','COMPLIANCE','ACTIVE','Third-party risk','NG',$4);
		INSERT INTO third_party_relationship_program_links(id,tenant_id,legal_entity_id,relationship_id,program_id,purpose_code,purpose_label,state,created_by_principal_id,version,created_at,updated_at)
		VALUES($5::uuid,$2::uuid,$3::uuid,$6::uuid,$1::uuid,'EVIDENCE_REFRESH','Evidence refresh','ACTIVE',$7::uuid,1,$4,$4)`,
		pgx.QueryExecModeSimpleProtocol, programID, thirdPartyTenantID, thirdPartyEntityA, now, linkID, relationship.Relationship.ID, thirdPartyPrincipal); err != nil {
		t.Fatal(err)
	}

	repository := NewPostgresRepository(pool)
	work, err := repository.CreateVendorWork(ctx, VendorWorkRequest{
		ID: workID, TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA,
		RelationshipID: relationship.Relationship.ID, RelationshipLinkID: linkID,
		TargetType: LinkTargetProgram, TargetID: programID, RequestKind: VendorWorkGeneral,
		Purpose: "Confirm current vendor controls.", Instructions: "Confirm the current control information.",
		OwnerPrincipalID: thirdPartyPrincipal, FormTemplateID: assessmentTemplateID, FormTemplateVersion: 3,
		Presentation: formcontract.PresentationWizard, State: VendorWorkPreparing, DeliveryState: VendorWorkDeliveryNotSent,
		DueAt: now.Add(24 * time.Hour), Version: 1, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}

	evidenceRepository := evidence.NewPostgresRepository(pool)
	var recipientKey, accessKey [32]byte
	for index := range recipientKey {
		recipientKey[index], accessKey[index] = byte(index+1), byte(index+33)
	}
	keyring, err := evidence.NewRecipientKeyring("vendor-work-v1", map[string][32]byte{"vendor-work-v1": recipientKey})
	if err != nil {
		t.Fatal(err)
	}
	distributionStore := evidence.NewPostgresDistributionStore(evidenceRepository, keyring)
	distributions := evidence.NewDistributionService(distributionStore)
	otp := &vendorWorkOTPDelivery{}
	access, err := evidence.NewDistributionAccessService(distributionStore, keyring, otp, accessKey, 20*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	dispatcher := evidence.NewWorkflowDistributionDispatcher(distributions, access)
	origin := evidence.RequestOrigin{Type: VendorWorkOrigin, ID: work.ID, Version: 1}
	dispatched, err := dispatcher.Dispatch(evidence.WithRequestOriginAuthority(ctx, VendorWorkOrigin), evidence.WorkflowDistributionDispatchInput{
		Request: evidence.CreateRequestInput{
			TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA,
			SubjectType: "VENDOR_RELATIONSHIP", SubjectID: relationship.Relationship.ID,
			Title: "Vendor due diligence", Purpose: work.Purpose, WhyYou: work.Instructions,
			Sensitivity: "INTERNAL", AudienceType: "VENDOR",
			Recipient:        evidence.RecipientInput{Type: evidence.RecipientExternalAudience, Audience: "review@vendor.example"},
			EstimatedMinutes: 5, Deadline: work.DueAt, Origin: origin,
			Presentation: evidenceAssessmentPresentation(), Sections: []formcontract.Section{{ID: "company", Title: "Company details"}},
			Fields:         []evidence.Field{{ID: "confirmed", SectionID: "company", Label: "Confirm the supplied details", Type: string(formcontract.TypeYesNo), Required: true}},
			FormTemplateID: assessmentTemplateID, FormTemplateVersion: 3, CreatedBy: thirdPartyPrincipal,
		},
		AccessPolicy: evidence.AccessDirectEmailOTP, RouteExpiresAt: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	work, err = repository.AttachVendorWorkCapture(ctx, Scope{TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA}, work.ID, work.Version, VendorWorkCaptureLink{
		ID: captureID, TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA,
		WorkRequestID: work.ID, RequestID: dispatched.Request.ID, Sequence: 1, Purpose: "INITIAL", OriginVersion: 1, CreatedAt: now,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	replacement, err := dispatcher.Resume(ctx, work.TenantID, work.LegalEntityID, dispatched.Request.ID, thirdPartyPrincipal, now.Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	work, err = repository.ReserveVendorWorkInvitation(ctx, Scope{TenantID: work.TenantID, LegalEntityID: work.LegalEntityID}, work.ID, work.Version, replacement.Route.RouteID, now)
	if err != nil {
		t.Fatal(err)
	}
	work, err = repository.MarkVendorWorkSent(ctx, Scope{TenantID: work.TenantID, LegalEntityID: work.LegalEntityID}, work.ID, work.Version, replacement.Route.RouteID, VendorWorkDeliveryLinkAvailable, "Copy the secure link or retry email delivery.", now)
	if err != nil {
		t.Fatal(err)
	}
	if work.CurrentInvitationID != replacement.Route.RouteID {
		t.Fatalf("vendor-work route proof = %q, want %q", work.CurrentInvitationID, replacement.Route.RouteID)
	}

	start, err := access.StartDistributionAccess(ctx, replacement.Route.Selector)
	if err != nil || start.Policy != evidence.AccessDirectEmailOTP || len(start.Recipients) != 1 {
		t.Fatalf("start canonical vendor-work route = (%#v, %v)", start, err)
	}
	if _, err := access.StartDistributionAccess(ctx, replacement.Route.Selector); err != nil {
		t.Fatalf("unexpired canonical vendor-work route could not be reopened: %v", err)
	}
	otpReceipt, err := access.SendOTP(ctx, replacement.Route.Selector, start.Recipients[0].SelectorID)
	if err != nil || len(otp.values) != 1 {
		t.Fatalf("send canonical vendor-work OTP = (%#v, %v)", otpReceipt, err)
	}
	verified, err := access.VerifyOTP(ctx, replacement.Route.Selector, otpReceipt.ChallengeID, otp.values[0].Code)
	if err != nil {
		t.Fatal(err)
	}
	workspace, err := access.GetResponseWorkspace(ctx, verified.SessionToken)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := access.SaveResponseWorkspace(ctx, verified.SessionToken, evidence.SaveWorkspaceInput{
		ExpectedVersion: workspace.Workspace.Version, PresentationMode: workspace.PresentationMode,
		Edits: []evidence.FieldEdit{{FieldID: "confirmed", Value: formcontract.TextAnswer("Yes"), BaseSequence: workspace.FieldSequences["confirmed"]}},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := access.SubmitResponseWorkspace(ctx, verified.SessionToken, evidence.SubmitWorkspaceInput{ExpectedVersion: saved.Workspace.Version})
	if err != nil {
		t.Fatalf("submit canonical vendor-work response after autosave: %v", err)
	}
	if result.Submission.SubmissionID == "" {
		t.Fatalf("submission result = %#v", result)
	}
	var legacyInvitations int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM capture_invitations WHERE request_id=$1::uuid`, dispatched.Request.ID).Scan(&legacyInvitations); err != nil {
		t.Fatal(err)
	}
	if legacyInvitations != 0 {
		t.Fatalf("vendor-work canonical request created %d legacy invitations", legacyInvitations)
	}
}

func TestPostgresAttachVendorWorkCapturePersistsIntegerSequenceAndBigintOriginVersion(t *testing.T) {
	pool := assessmentPostgresPool(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	const (
		programID = "33333333-3333-7333-8333-333333333371"
		linkID    = "33333333-3333-7333-8333-333333333372"
		workID    = "33333333-3333-7333-8333-333333333373"
		captureID = "33333333-3333-7333-8333-333333333374"
	)
	relationship := seedAssessmentRelationship(t, pool, "Address verification attachment")
	if _, err := pool.Exec(ctx, `
		INSERT INTO programs(id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,jurisdiction,effective_from)
		VALUES($1::uuid,$2::uuid,$3::uuid,'VENDOR-CAPTURE','Vendor capture','COMPLIANCE','ACTIVE','Third-party risk','NG',$4);
		INSERT INTO third_party_relationship_program_links(id,tenant_id,legal_entity_id,relationship_id,program_id,purpose_code,purpose_label,state,created_by_principal_id,version,created_at,updated_at)
		VALUES($5::uuid,$2::uuid,$3::uuid,$6::uuid,$1::uuid,'EVIDENCE_REFRESH','Evidence refresh','ACTIVE',$7::uuid,1,$4,$4)`,
		pgx.QueryExecModeSimpleProtocol, programID, thirdPartyTenantID, thirdPartyEntityA, now, linkID, relationship.Relationship.ID, thirdPartyPrincipal); err != nil {
		t.Fatal(err)
	}

	repository := NewPostgresRepository(pool)
	created, err := repository.CreateVendorWork(ctx, VendorWorkRequest{
		ID: workID, TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA,
		RelationshipID: relationship.Relationship.ID, RelationshipLinkID: linkID,
		TargetType: LinkTargetProgram, TargetID: programID, RequestKind: VendorWorkGeneral,
		Purpose: "Refresh vendor evidence.", Instructions: "Provide the requested current evidence.",
		OwnerPrincipalID: thirdPartyPrincipal, FormTemplateID: assessmentTemplateID, FormTemplateVersion: 3,
		Presentation: formcontract.PresentationWizard, State: VendorWorkPreparing, DeliveryState: VendorWorkDeliveryNotSent,
		DueAt: now.Add(24 * time.Hour), Version: 1, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}

	origin := evidence.RequestOrigin{Type: VendorWorkOrigin, ID: created.ID, Version: 1}
	evidenceContext := evidence.WithRequestOriginAuthority(ctx, origin.Type)
	evidenceService := evidence.NewService(evidence.NewPostgresRepository(pool), evidence.NewMemoryObjectStore())
	request, err := evidenceService.CreateRequest(evidenceContext, evidence.CreateRequestInput{
		TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA,
		SubjectType: "VENDOR_RELATIONSHIP", SubjectID: relationship.Relationship.ID,
		Title: "Confirm the vendor address", Purpose: "Record the address verification result.",
		WhyYou:      "You are responsible for confirming the address and providing evidence.",
		Sensitivity: "CONFIDENTIAL", AudienceType: "VENDOR",
		Recipient:        evidence.RecipientInput{Type: evidence.RecipientExternalAudience, Audience: "review@vendor.example"},
		EstimatedMinutes: 10, Deadline: now.Add(24 * time.Hour), Origin: origin,
		Presentation:   formcontract.Presentation{DefaultMode: formcontract.PresentationWizard},
		Sections:       []formcontract.Section{{ID: "address", Title: "Address verification"}},
		Fields:         []evidence.Field{{ID: "confirmed", SectionID: "address", Label: "Was the address verified?", Type: string(formcontract.TypeYesNo), Required: true}},
		FormTemplateID: assessmentTemplateID, FormTemplateVersion: 3, CreatedBy: thirdPartyPrincipal,
	})
	if err != nil {
		t.Fatal(err)
	}

	updated, err := repository.AttachVendorWorkCapture(ctx, Scope{TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA}, created.ID, created.Version, VendorWorkCaptureLink{
		ID: captureID, TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA,
		WorkRequestID: created.ID, RequestID: request.ID, Sequence: 1, Purpose: "INITIAL", OriginVersion: 1, CreatedAt: now,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if updated.CurrentRequestID != request.ID || updated.CurrentCaptureSequence != 1 || updated.Version != 2 {
		t.Fatalf("updated work = %#v", updated)
	}
	captures, err := repository.ListVendorWorkCaptures(ctx, Scope{TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA}, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(captures) != 1 || captures[0].Sequence != 1 || captures[0].OriginVersion != 1 || captures[0].RequestID != request.ID {
		t.Fatalf("captures = %#v", captures)
	}
}
