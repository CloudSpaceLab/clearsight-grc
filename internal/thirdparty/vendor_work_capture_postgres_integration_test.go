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
	mismatched, err := dispatcher.Dispatch(evidence.WithRequestOriginAuthority(ctx, VendorWorkOrigin), evidence.WorkflowDistributionDispatchInput{
		Request: evidence.CreateRequestInput{
			TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA,
			SubjectType: "VENDOR_RELATIONSHIP", SubjectID: relationship.Relationship.ID,
			Title: "Unrelated vendor clarification", Purpose: work.Purpose, WhyYou: work.Instructions,
			Sensitivity: "INTERNAL", AudienceType: "VENDOR",
			Recipient:        evidence.RecipientInput{Type: evidence.RecipientExternalAudience, Audience: "other@vendor.example"},
			EstimatedMinutes: 5, Deadline: work.DueAt,
			Origin:       evidence.RequestOrigin{Type: VendorWorkOrigin, ID: work.ID, Version: 2},
			Presentation: evidenceAssessmentPresentation(), Sections: []formcontract.Section{{ID: "company", Title: "Company details"}},
			Fields:         []evidence.Field{{ID: "confirmed", SectionID: "company", Label: "Confirm the supplied details", Type: string(formcontract.TypeYesNo), Required: true}},
			FormTemplateID: assessmentTemplateID, FormTemplateVersion: 3, CreatedBy: thirdPartyPrincipal,
		},
		AccessPolicy: evidence.AccessDirectEmailOTP, RouteExpiresAt: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.ReserveVendorWorkInvitation(ctx, Scope{TenantID: work.TenantID, LegalEntityID: work.LegalEntityID}, work.ID, work.Version, mismatched.Route.RouteID, now); err == nil {
		t.Fatal("route for a different request was accepted as vendor-work proof")
	}
	unchanged, err := repository.GetVendorWork(ctx, Scope{TenantID: work.TenantID, LegalEntityID: work.LegalEntityID}, work.ID)
	if err != nil || unchanged.Version != work.Version || unchanged.PendingInvitationID != "" {
		t.Fatalf("mismatched route changed vendor work = %#v err=%v", unchanged, err)
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
	// Add one submitted file fixture and its exact capture receipt. The file
	// inventory must follow the relationship capture, then the work target ACL.
	if _, err := pool.Exec(ctx, `
	 UPDATE capture_requests SET fields=fields||'[{"id":"policy","label":"Policy","type":"file"}]'::jsonb WHERE id=$1::uuid;
	 UPDATE capture_submissions SET answers=answers||jsonb_build_object('policy',jsonb_build_object('artifact_ids',jsonb_build_array(md5('work-file')::uuid::text))) WHERE id=$2::uuid;
	 INSERT INTO capture_artifacts(id,tenant_id,request_id,submission_id,file_name,media_type,size_bytes,sha256,storage_key,status,created_at)
	 VALUES(md5('work-file')::uuid,$3::uuid,$1::uuid,$2::uuid,'policy.pdf','application/pdf',4,repeat('a',64),'work-file','STORED_UNSCANNED',$4);
	`, pgx.QueryExecModeSimpleProtocol, dispatched.Request.ID, result.Submission.SubmissionID, thirdPartyTenantID, now); err != nil {
		t.Fatal(err)
	}
	files, err := distributionStore.ListDocuments(ctx, evidence.DocumentQuery{TenantID: thirdPartyTenantID, LegalEntityID: thirdPartyEntityA, PrincipalID: thirdPartyPrincipal, RelationshipID: relationship.Relationship.ID, Limit: 1})
	if err != nil || len(files.Items) != 1 || files.Items[0].WorkRequestID != work.ID {
		t.Fatalf("work document inventory = %#v, %v", files, err)
	}
	// A successor capture has its own distribution. Currency belongs to the
	// merged work response, not the per-distribution revision flag.
	if _, err := pool.Exec(ctx, `
 UPDATE capture_requests SET fields=fields||'[{"id":"photo","label":"Site","type":"photo"}]'::jsonb WHERE id=$1::uuid;
 UPDATE capture_submissions SET answers=answers||jsonb_build_object('photo',jsonb_build_object('artifact_ids',jsonb_build_array(md5('work-photo')::uuid::text))) WHERE id=$2::uuid;
 INSERT INTO capture_artifacts SELECT (jsonb_populate_record(NULL::capture_artifacts,to_jsonb(a)||jsonb_build_object('id',md5('work-photo')::uuid,'file_name','site.png','media_type','image/png','storage_key','work-photo'))).* FROM capture_artifacts a WHERE id=md5('work-file')::uuid;
 INSERT INTO third_party_work_capture_links SELECT (jsonb_populate_record(NULL::third_party_work_capture_links,to_jsonb(c)||jsonb_build_object('id',md5('work-successor-link')::uuid,'request_id',$3::uuid,'sequence',2,'origin_version',2,'submission_id',NULL,'invitation_id',NULL))).* FROM third_party_work_capture_links c WHERE id=$4::uuid;
 UPDATE third_party_work_requests SET current_request_id=$3::uuid,current_invitation_id=NULL WHERE id=$5::uuid;
 UPDATE capture_requests SET fields='[{"id":"policy","label":"Policy","type":"file"}]' WHERE id=$3::uuid;
 `, pgx.QueryExecModeSimpleProtocol, dispatched.Request.ID, result.Submission.SubmissionID, mismatched.Request.ID, captureID, work.ID); err != nil {
		t.Fatal(err)
	}
	query := evidence.DocumentQuery{TenantID: thirdPartyTenantID, LegalEntityID: thirdPartyEntityA, PrincipalID: thirdPartyPrincipal, RelationshipID: relationship.Relationship.ID, CurrentOnly: true, Limit: 100}
	assertCurrent := func(stage string, expected int, replacement bool) {
		t.Helper()
		page, err := distributionStore.ListDocuments(ctx, query)
		if err != nil || len(page.Items) != expected {
			t.Fatalf("%s current documents: %+v %v", stage, page, err)
		}
		for _, file := range page.Items {
			if replacement && file.FieldID == "policy" && file.RequestID != mismatched.Request.ID {
				t.Fatalf("%s retained replaced policy: %+v", stage, file)
			}
		}
	}
	assertCurrent("pending successor", 2, false)
	if _, err := pool.Exec(ctx, `
 INSERT INTO capture_submissions SELECT (jsonb_populate_record(NULL::capture_submissions,to_jsonb(s)||jsonb_build_object('id',md5('work-successor-submission')::uuid,'request_id',$2::uuid,'distribution_id',(SELECT distribution_id FROM capture_requests WHERE id=$2::uuid),'answers',jsonb_build_object('policy',jsonb_build_object('artifact_ids',jsonb_build_array(md5('work-replacement')::uuid::text))),'submitted_at',s.submitted_at+interval '1 minute'))).* FROM capture_submissions s WHERE id=$1::uuid;
 INSERT INTO capture_artifacts SELECT (jsonb_populate_record(NULL::capture_artifacts,to_jsonb(a)||jsonb_build_object('id',md5('work-replacement')::uuid,'request_id',$2::uuid,'submission_id',md5('work-successor-submission')::uuid,'storage_key','work-replacement'))).* FROM capture_artifacts a WHERE id=md5('work-file')::uuid;
 INSERT INTO capture_response_revisions SELECT (jsonb_populate_record(NULL::capture_response_revisions,to_jsonb(r)||jsonb_build_object('id',md5('work-successor-revision')::uuid,'distribution_id',(SELECT distribution_id FROM capture_requests WHERE id=$2::uuid),'workspace_id',(SELECT id FROM capture_response_workspaces WHERE distribution_id=(SELECT distribution_id FROM capture_requests WHERE id=$2::uuid)),'submission_id',md5('work-successor-submission')::uuid,'supersedes_revision_id',NULL))).* FROM capture_response_revisions r WHERE submission_id=$1::uuid;
 `, pgx.QueryExecModeSimpleProtocol, result.Submission.SubmissionID, mismatched.Request.ID); err != nil {
		t.Fatal(err)
	}
	assertCurrent("submitted replacement", 2, true)
	query.CurrentOnly = false
	all, err := distributionStore.ListDocuments(ctx, query)
	if err != nil || len(all.Items) != 3 {
		t.Fatalf("replacement history: %+v %v", all, err)
	}
	query.CurrentOnly = true
	// A work/form mismatch must deny otherwise-valid captures, not merely fail
	// the independent request/distribution equality check.
	if _, err := pool.Exec(ctx, `
 INSERT INTO monitoring_form_templates SELECT (jsonb_populate_record(NULL::monitoring_form_templates,to_jsonb(f)||jsonb_build_object('revision_id',md5('work-form-v4')::uuid,'version',4,'is_current',false,'status','DRAFT','effective_from',NULL,'effective_until',NULL))).* FROM monitoring_form_templates f WHERE tenant_id=$2::uuid AND id=$3::uuid AND version=3;
 INSERT INTO monitoring_form_templates SELECT (jsonb_populate_record(NULL::monitoring_form_templates,to_jsonb(f)||jsonb_build_object('revision_id',md5('wrong-work-form-revision')::uuid,'id',md5('wrong-work-form')::uuid,'code','WRONG-WORK-FORM','is_current',false,'status','DRAFT','effective_from',NULL,'effective_until',NULL))).* FROM monitoring_form_templates f WHERE tenant_id=$2::uuid AND id=$3::uuid AND version=3;
 UPDATE third_party_work_requests SET form_template_version=4 WHERE id=$1::uuid`, pgx.QueryExecModeSimpleProtocol, work.ID, thirdPartyTenantID, assessmentTemplateID); err != nil {
		t.Fatal(err)
	}
	assertCurrent("wrong work form version", 0, false)
	if _, err := pool.Exec(ctx, `UPDATE third_party_work_requests SET form_template_version=3,form_template_id=md5('wrong-work-form')::uuid WHERE id=$1::uuid`, work.ID); err != nil {
		t.Fatal(err)
	}
	assertCurrent("wrong work form ID", 0, false)
	if _, err := pool.Exec(ctx, `UPDATE third_party_work_requests SET form_template_version=3,form_template_id=$3::uuid WHERE id=$1::uuid; DELETE FROM capture_response_revisions WHERE submission_id IN ($2::uuid,md5('work-successor-submission')::uuid)`, pgx.QueryExecModeSimpleProtocol, work.ID, result.Submission.SubmissionID, assessmentTemplateID); err != nil {
		t.Fatal(err)
	}
	assertCurrent("legacy replacement", 2, true)
	if _, err := pool.Exec(ctx, `UPDATE capture_submissions SET answers='{}' WHERE id=md5('work-successor-submission')::uuid`); err != nil {
		t.Fatal(err)
	}
	assertCurrent("omitted replacement answer", 1, true)
	if _, err := pool.Exec(ctx, `DELETE FROM capture_artifacts WHERE id=md5('work-replacement')::uuid; DELETE FROM capture_submissions WHERE id=md5('work-successor-submission')::uuid`, pgx.QueryExecModeSimpleProtocol); err != nil {
		t.Fatal(err)
	}
	assertCurrent("legacy pending successor", 2, false)
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
