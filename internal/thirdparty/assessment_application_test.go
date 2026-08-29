package thirdparty

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestApplyAssessmentResponseUpdatesAcceptedIdentityAndStoresIdempotentReceipt(t *testing.T) {
	service, application, repository, actor, assessment := assessmentApplicationFixture(t)
	input := ApplyAssessmentResponseInput{ExpectedAssessmentVersion: assessment.Version, ExpectedSubmissionRevision: 1, Decisions: []FieldApplicationDecision{{FieldID: "legal_name", Decision: FieldApplicationAccept, Rationale: "Registration evidence supports the corrected name."}}}
	ctx := assessmentContextFor(actor.TenantID, actor.LegalEntityID, actor.PrincipalID)

	first, err := service.ApplyResponse(ctx, Actor{}, assessment.ID, "submission-1", input)
	if err != nil {
		t.Fatal(err)
	}
	if first.Receipt.PriorVendorVersion != 1 || first.Receipt.ResultVendorVersion != 2 || len(first.Receipt.AcceptedFieldIDs) != 1 || first.Receipt.AcceptedFieldIDs[0] != "legal_name" {
		t.Fatalf("unexpected application receipt: %#v", first.Receipt)
	}
	updated, err := repository.GetVendor(context.Background(), Scope{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID}, "vendor-1")
	if err != nil || updated.LegalName != "Vendor Holdings Limited" || updated.TradingName != "Vendor" || updated.Version != 2 {
		t.Fatalf("updated vendor = %#v err=%v", updated, err)
	}

	second, err := service.ApplyResponse(ctx, Actor{}, assessment.ID, "submission-1", input)
	if err != nil || second.Receipt.ID != first.Receipt.ID || second.Receipt.ResultVendorVersion != 2 || len(repository.applicationReceipts) != 1 {
		t.Fatalf("idempotent replay = %#v err=%v receipts=%d", second, err, len(repository.applicationReceipts))
	}
	_ = application
}

func TestApplyAssessmentResponseRejectsStaleHeldVendorVersionWithoutMutation(t *testing.T) {
	service, _, repository, actor, assessment := assessmentApplicationFixture(t)
	repository.mu.Lock()
	stale := repository.vendors["vendor-1"]
	stale.Version = 2
	repository.vendors["vendor-1"] = stale
	repository.mu.Unlock()
	ctx := assessmentContextFor(actor.TenantID, actor.LegalEntityID, actor.PrincipalID)
	_, err := service.ApplyResponse(ctx, Actor{}, assessment.ID, "submission-1", ApplyAssessmentResponseInput{ExpectedAssessmentVersion: assessment.Version, ExpectedSubmissionRevision: 1, Decisions: []FieldApplicationDecision{{FieldID: "legal_name", Decision: FieldApplicationAccept, Rationale: "Reviewed."}}})
	if !errors.Is(err, ErrVersionConflict) || len(repository.applicationReceipts) != 0 {
		t.Fatalf("stale application error=%v receipts=%d", err, len(repository.applicationReceipts))
	}
}

func TestApplyAssessmentResponseUsesVerifiedReviewerAndRejectsBodyActor(t *testing.T) {
	service, _, _, actor, assessment := assessmentApplicationFixture(t)
	service.assessments.guard = &assessmentGuardStub{decisionActor: verifiedActorFor(actor), err: commandauth.ErrNotAuthorized}
	ctx := assessmentContextFor(actor.TenantID, actor.LegalEntityID, actor.PrincipalID)
	_, err := service.ApplyResponse(ctx, Actor{TenantID: "forged", LegalEntityID: "forged", PrincipalID: "forged"}, assessment.ID, "submission-1", ApplyAssessmentResponseInput{ExpectedAssessmentVersion: assessment.Version, ExpectedSubmissionRevision: 1, Decisions: []FieldApplicationDecision{{FieldID: "legal_name", Decision: FieldApplicationAccept, Rationale: "Reviewed."}}})
	if !errors.Is(err, commandauth.ErrNotAuthorized) {
		t.Fatalf("unauthorized application error = %v", err)
	}
}

func assessmentApplicationFixture(t *testing.T) (*AssessmentApplicationService, *AssessmentReviewService, *MemoryAssessmentRepository, Actor, Assessment) {
	t.Helper()
	review, actor, assessment, reader := assessmentReviewFixture(t)
	repository := review.links.(*MemoryAssessmentRepository)
	assessment.Status = AssessmentUnderReview
	assessment.CompletedAt = nil
	assessment.Conclusion = ""
	assessment.ConclusionRationale = ""
	assessment.Version = 5
	repository.assessments[assessment.ID] = assessment
	vendor := repository.vendors["vendor-1"]
	vendor.LegalName = "Vendor Limited"
	vendor.TradingName = "Vendor"
	vendor.Version = 1
	repository.vendors[vendor.ID] = vendor
	request := reader.requests["request-1"]
	request.Fields = []evidence.Field{{ID: "legal_name", SectionID: "identity", Label: "Registered legal name", Type: string(formcontract.TypeShortText), Required: true, CollectionIntent: formcontract.IntentConfirmOrCorrect, RecordTarget: &formcontract.RecordTarget{Key: "VENDOR.IDENTITY.LEGAL_NAME", RequiredSubjectType: "VENDOR_RELATIONSHIP"}, RecordBaseline: &evidence.RecordBaseline{TargetKey: "VENDOR.IDENTITY.LEGAL_NAME", SubjectType: "VENDOR_RELATIONSHIP", SubjectID: assessment.RelationshipID, RecordID: vendor.ID, RecordVersion: 1, DisplayValue: vendor.LegalName}}}
	request.Sections = []formcontract.Section{{ID: "identity", Title: "Identity"}}
	reader.requests["request-1"] = request
	submission := reader.submissions["submission-1"]
	submission.Answers = map[string]formcontract.AnswerValue{"legal_name": formcontract.TextAnswer("Vendor Holdings Limited")}
	submission.AnswerProvenance = map[string]evidence.AnswerProvenance{"legal_name": {Origin: evidence.AnswerRespondentCorrected}}
	reader.submissions["submission-1"] = submission
	guard := &assessmentGuardStub{decisionActor: verifiedActorFor(actor)}
	review.assessments.guard = guard
	review.assessments.now = func() time.Time { return time.Date(2026, 8, 26, 12, 30, 0, 0, time.UTC) }
	return NewAssessmentApplicationService(review.assessments, review, repository), review, repository, actor, assessment
}

func verifiedActorFor(actor Actor) identity.Actor {
	value := verifiedIdentity()
	value.TenantID, value.LegalEntityID, value.PrincipalID = actor.TenantID, actor.LegalEntityID, actor.PrincipalID
	return value
}
