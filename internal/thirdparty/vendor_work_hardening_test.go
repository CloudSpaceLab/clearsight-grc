package thirdparty

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
)

func TestRetryVendorWorkCompletesIncompletePreparation(t *testing.T) {
	fixture := newVendorWorkFixture(t)
	fixture.service.evidence = &vendorWorkEvidenceFailure{vendorWorkEvidence: fixture.evidence, createFailures: 1}
	prepared, err := fixture.service.Prepare(context.Background(), fixture.actor, PrepareVendorWorkInput{
		RelationshipID: "relationship-1", RelationshipLinkID: fixture.link.ID,
		Purpose: "Confirm service information.", Instructions: "Review the request.",
		FormTemplateID: "form-1", FormTemplateVersion: 3, VendorAudience: fixture.audience, DueAt: fixture.now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if prepared.CurrentRequestID != "" || prepared.DeliveryState != VendorWorkDeliveryRetryRequired {
		t.Fatalf("incomplete preparation = %#v", prepared)
	}

	outcome, err := fixture.service.Retry(context.Background(), fixture.actor, prepared.ID, RetryVendorWorkInput{
		ExpectedVersion: prepared.Version, VendorAudience: fixture.audience, InvitationTTLMinutes: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Work.CurrentRequestID == "" || outcome.Work.CurrentInvitationID == "" || outcome.Work.State != VendorWorkAwaitingVendor {
		t.Fatalf("recovered preparation = %#v", outcome)
	}
}

type ambiguousVendorWorkCreateRepository struct{ *MemoryVendorWorkRepository }

func (r *ambiguousVendorWorkCreateRepository) CreateVendorWork(ctx context.Context, value VendorWorkRequest) (VendorWorkRequest, error) {
	if _, err := r.MemoryVendorWorkRepository.CreateVendorWork(ctx, value); err != nil {
		return VendorWorkRequest{}, err
	}
	return VendorWorkRequest{}, errors.New("commit result unavailable")
}

func TestPrepareVendorWorkReconcilesCommittedWorkCreation(t *testing.T) {
	fixture := newVendorWorkFixture(t)
	fixture.service.repo = &ambiguousVendorWorkCreateRepository{MemoryVendorWorkRepository: fixture.repository}
	prepared, err := fixture.service.Prepare(context.Background(), fixture.actor, PrepareVendorWorkInput{
		RelationshipID: "relationship-1", RelationshipLinkID: fixture.link.ID, Purpose: "Confirm service information.", Instructions: "Review the request.",
		FormTemplateID: "form-1", FormTemplateVersion: 3, VendorAudience: fixture.audience, DueAt: fixture.now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if prepared.ID == "" || prepared.CurrentRequestID == "" || prepared.Recovery != "" {
		t.Fatalf("reconciled preparation = %#v", prepared)
	}
}

type ambiguousVendorWorkCaptureRepository struct{ *MemoryVendorWorkRepository }

func (r *ambiguousVendorWorkCaptureRepository) AttachVendorWorkCapture(ctx context.Context, scope Scope, id string, expected int64, link VendorWorkCaptureLink, now time.Time) (VendorWorkRequest, error) {
	if _, err := r.MemoryVendorWorkRepository.AttachVendorWorkCapture(ctx, scope, id, expected, link, now); err != nil {
		return VendorWorkRequest{}, err
	}
	return VendorWorkRequest{}, errors.New("commit result unavailable")
}

func TestPrepareVendorWorkReconcilesCommittedCaptureAttachment(t *testing.T) {
	fixture := newVendorWorkFixture(t)
	fixture.service.repo = &ambiguousVendorWorkCaptureRepository{MemoryVendorWorkRepository: fixture.repository}
	prepared, err := fixture.service.Prepare(context.Background(), fixture.actor, PrepareVendorWorkInput{
		RelationshipID: "relationship-1", RelationshipLinkID: fixture.link.ID, Purpose: "Confirm service information.", Instructions: "Review the request.",
		FormTemplateID: "form-1", FormTemplateVersion: 3, VendorAudience: fixture.audience, DueAt: fixture.now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if prepared.CurrentRequestID == "" || prepared.Recovery != "" {
		t.Fatalf("reconciled capture attachment = %#v", prepared)
	}
}

type ambiguousVendorWorkDeliveryRepository struct{ *MemoryVendorWorkRepository }

func (r *ambiguousVendorWorkDeliveryRepository) MarkVendorWorkSent(ctx context.Context, scope Scope, id string, expected int64, invitationID string, delivery VendorWorkDeliveryState, recovery string, now time.Time) (VendorWorkRequest, error) {
	if _, err := r.MemoryVendorWorkRepository.MarkVendorWorkSent(ctx, scope, id, expected, invitationID, delivery, recovery, now); err != nil {
		return VendorWorkRequest{}, err
	}
	return VendorWorkRequest{}, errors.New("commit result unavailable")
}

func TestSendVendorWorkReconcilesCommittedInvitationBeforeRevocation(t *testing.T) {
	fixture := newVendorWorkFixture(t)
	fixture.service.repo = &ambiguousVendorWorkDeliveryRepository{MemoryVendorWorkRepository: fixture.repository}
	prepared, err := fixture.service.Prepare(context.Background(), fixture.actor, PrepareVendorWorkInput{
		RelationshipID: "relationship-1", RelationshipLinkID: fixture.link.ID, Purpose: "Confirm service information.", Instructions: "Review the request.",
		FormTemplateID: "form-1", FormTemplateVersion: 3, VendorAudience: fixture.audience, DueAt: fixture.now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := fixture.service.Send(context.Background(), fixture.actor, prepared.ID, SendVendorWorkInput{ExpectedVersion: prepared.Version, VendorAudience: fixture.audience, InvitationTTLMinutes: 60})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Work.CurrentInvitationID == "" || outcome.CaptureURL == "" {
		t.Fatalf("reconciled send outcome = %#v", outcome)
	}
	if _, err := fixture.evidence.RedeemInvitation(context.Background(), vendorWorkCaptureToken(t, outcome.CaptureURL), fixture.audience); err != nil {
		t.Fatalf("committed invitation was revoked: %v", err)
	}
}

type failedVendorWorkDeliveryRepository struct {
	*MemoryVendorWorkRepository
	markAttempted bool
}

func (r *failedVendorWorkDeliveryRepository) MarkVendorWorkSent(context.Context, Scope, string, int64, string, VendorWorkDeliveryState, string, time.Time) (VendorWorkRequest, error) {
	r.markAttempted = true
	return VendorWorkRequest{}, errors.New("delivery state unavailable")
}

func (r *failedVendorWorkDeliveryRepository) GetVendorWork(ctx context.Context, scope Scope, id string) (VendorWorkRequest, error) {
	if r.markAttempted {
		return VendorWorkRequest{}, errors.New("delivery state read unavailable")
	}
	return r.MemoryVendorWorkRepository.GetVendorWork(ctx, scope, id)
}

type revocationTrackingVendorWorkEvidence struct {
	vendorWorkEvidence
	revokedRequests int
	revokeFailures  int
	issued          []evidence.IssuedInvitation
}

func (s *revocationTrackingVendorWorkEvidence) RevokeRequestCapabilities(ctx context.Context, tenantID, requestID string) error {
	s.revokedRequests++
	if s.revokeFailures > 0 {
		s.revokeFailures--
		return errors.New("capability revocation unavailable")
	}
	return s.vendorWorkEvidence.RevokeRequestCapabilities(ctx, tenantID, requestID)
}

func (s *revocationTrackingVendorWorkEvidence) IssueInvitation(ctx context.Context, input evidence.IssueInvitationInput) (evidence.IssuedInvitation, error) {
	issued, err := s.vendorWorkEvidence.IssueInvitation(ctx, input)
	if err == nil {
		s.issued = append(s.issued, issued)
	}
	return issued, err
}

func TestSendVendorWorkReturnsRecoverableStateWhenInvitationPersistenceIsAmbiguous(t *testing.T) {
	fixture := newVendorWorkFixture(t)
	tracking := &revocationTrackingVendorWorkEvidence{vendorWorkEvidence: fixture.evidence}
	fixture.service.evidence = tracking
	fixture.service.repo = &failedVendorWorkDeliveryRepository{MemoryVendorWorkRepository: fixture.repository}
	prepared, err := fixture.service.Prepare(context.Background(), fixture.actor, PrepareVendorWorkInput{
		RelationshipID: "relationship-1", RelationshipLinkID: fixture.link.ID, Purpose: "Confirm service information.", Instructions: "Review the request.",
		FormTemplateID: "form-1", FormTemplateVersion: 3, VendorAudience: fixture.audience, DueAt: fixture.now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := fixture.service.Send(context.Background(), fixture.actor, prepared.ID, SendVendorWorkInput{ExpectedVersion: prepared.Version, VendorAudience: fixture.audience, InvitationTTLMinutes: 60})
	if err != nil {
		t.Fatalf("recoverable send = %v", err)
	}
	if outcome.State != VendorWorkDeliveryRetryRequired || outcome.Work.DeliveryState != VendorWorkDeliveryRetryRequired || outcome.CaptureURL != "" || tracking.revokedRequests != 1 {
		t.Fatalf("recoverable invitation outcome = %#v, revocations=%d", outcome, tracking.revokedRequests)
	}
}

type failedVendorWorkFinalizationRepository struct {
	*MemoryVendorWorkRepository
	finalizeFailures  int
	readFailures      int
	finalizeAttempted bool
}

func (r *failedVendorWorkFinalizationRepository) MarkVendorWorkSent(ctx context.Context, scope Scope, id string, expected int64, invitationID string, delivery VendorWorkDeliveryState, recovery string, now time.Time) (VendorWorkRequest, error) {
	if invitationID != "" && r.finalizeFailures > 0 {
		r.finalizeFailures--
		r.finalizeAttempted = true
		return VendorWorkRequest{}, errors.New("invitation finalization unavailable")
	}
	return r.MemoryVendorWorkRepository.MarkVendorWorkSent(ctx, scope, id, expected, invitationID, delivery, recovery, now)
}

func (r *failedVendorWorkFinalizationRepository) GetVendorWork(ctx context.Context, scope Scope, id string) (VendorWorkRequest, error) {
	if r.finalizeAttempted && r.readFailures > 0 {
		r.readFailures--
		return VendorWorkRequest{}, errors.New("vendor work read unavailable")
	}
	return r.MemoryVendorWorkRepository.GetVendorWork(ctx, scope, id)
}

func TestSendVendorWorkDurablyReservesInvitationBeforeIssueWhenFinalizationAndRevocationFail(t *testing.T) {
	fixture := newVendorWorkFixture(t)
	repository := &failedVendorWorkFinalizationRepository{MemoryVendorWorkRepository: fixture.repository, finalizeFailures: 1, readFailures: 1}
	tracking := &revocationTrackingVendorWorkEvidence{vendorWorkEvidence: fixture.evidence, revokeFailures: 1}
	fixture.service.repo = repository
	fixture.service.evidence = tracking
	prepared, err := fixture.service.Prepare(context.Background(), fixture.actor, PrepareVendorWorkInput{
		RelationshipID: "relationship-1", RelationshipLinkID: fixture.link.ID, Purpose: "Confirm service information.", Instructions: "Review the request.",
		FormTemplateID: "form-1", FormTemplateVersion: 3, VendorAudience: fixture.audience, DueAt: fixture.now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	failed, err := fixture.service.Send(context.Background(), fixture.actor, prepared.ID, SendVendorWorkInput{ExpectedVersion: prepared.Version, VendorAudience: fixture.audience, InvitationTTLMinutes: 60})
	if err != nil {
		t.Fatalf("recoverable send = %v", err)
	}
	if len(tracking.issued) != 1 || failed.Work.PendingInvitationID == "" || failed.Work.PendingInvitationID != tracking.issued[0].InvitationID || failed.Work.PendingInvitationRequestID != prepared.CurrentRequestID {
		t.Fatalf("reserved invitation outcome=%#v issued=%#v", failed, tracking.issued)
	}
	if failed.State != VendorWorkDeliveryRetryRequired || failed.CaptureURL != "" || tracking.revokedRequests != 1 {
		t.Fatalf("failed finalization outcome=%#v revocations=%d", failed, tracking.revokedRequests)
	}

	recovered, err := fixture.service.Retry(context.Background(), fixture.actor, prepared.ID, RetryVendorWorkInput{ExpectedVersion: failed.Work.Version, VendorAudience: fixture.audience, InvitationTTLMinutes: 60})
	if err != nil {
		t.Fatal(err)
	}
	if len(tracking.issued) != 2 || recovered.Work.PendingInvitationID != "" || recovered.Work.CurrentInvitationID != tracking.issued[1].InvitationID || recovered.Work.CurrentInvitationID == tracking.issued[0].InvitationID {
		t.Fatalf("recovered invitation outcome=%#v issued=%#v", recovered, tracking.issued)
	}
	if _, err := fixture.evidence.RedeemInvitation(context.Background(), tracking.issued[0].Token, fixture.audience); !errors.Is(err, evidence.ErrInvitationInvalid) {
		t.Fatalf("reserved predecessor remained usable: %v", err)
	}
}

type failingVendorWorkChangesRepository struct {
	*MemoryVendorWorkRepository
	failure error
}

type ambiguousVendorWorkChangesRepository struct{ *MemoryVendorWorkRepository }

func (r *ambiguousVendorWorkChangesRepository) RecordVendorWorkChanges(ctx context.Context, scope Scope, id string, expected int64, link VendorWorkCaptureLink, actor, message string, dueAt, now time.Time) (VendorWorkRequest, error) {
	if _, err := r.MemoryVendorWorkRepository.RecordVendorWorkChanges(ctx, scope, id, expected, link, actor, message, dueAt, now); err != nil {
		return VendorWorkRequest{}, err
	}
	return VendorWorkRequest{}, errors.New("commit result unavailable")
}

func TestRequestVendorWorkChangesReconcilesCommittedReplacement(t *testing.T) {
	fixture := newVendorWorkFixture(t)
	_, reviewing := vendorWorkUnderReview(t, fixture)
	fixture.service.repo = &ambiguousVendorWorkChangesRepository{MemoryVendorWorkRepository: fixture.repository}
	revisedDueAt := fixture.now.Add(5 * 24 * time.Hour)
	outcome, err := fixture.service.RequestChanges(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "reviewer-1"}, reviewing.ID, RequestVendorWorkChangesInput{
		ExpectedVersion: reviewing.Version, Message: "Confirm the updated service owner.", FieldIDs: []string{"service_current"},
		VendorAudience: fixture.audience, DueAt: revisedDueAt, InvitationTTLMinutes: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Work.State != VendorWorkChangesRequested || outcome.Work.CurrentCaptureSequence != reviewing.CurrentCaptureSequence+1 || outcome.Work.CurrentInvitationID == "" || !outcome.Work.DueAt.Equal(revisedDueAt) {
		t.Fatalf("reconciled clarification = %#v", outcome)
	}
}

func (r *failingVendorWorkChangesRepository) TransitionVendorWork(ctx context.Context, scope Scope, id string, expected int64, target VendorWorkState, actor, detail string, now time.Time) (VendorWorkRequest, error) {
	if target == VendorWorkChangesRequested {
		return VendorWorkRequest{}, r.failure
	}
	return r.MemoryVendorWorkRepository.TransitionVendorWork(ctx, scope, id, expected, target, actor, detail, now)
}

func (r *failingVendorWorkChangesRepository) RecordVendorWorkChanges(context.Context, Scope, string, int64, VendorWorkCaptureLink, string, string, time.Time, time.Time) (VendorWorkRequest, error) {
	return VendorWorkRequest{}, r.failure
}

func TestRequestVendorWorkChangesDoesNotPartiallyReplaceCurrentResponse(t *testing.T) {
	fixture := newVendorWorkFixture(t)
	prepared, reviewing := vendorWorkUnderReview(t, fixture)
	failed := &failingVendorWorkChangesRepository{MemoryVendorWorkRepository: fixture.repository, failure: errors.New("store unavailable")}
	fixture.service.repo = failed
	originalDueAt := reviewing.DueAt

	_, err := fixture.service.RequestChanges(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "reviewer-1"}, reviewing.ID, RequestVendorWorkChangesInput{
		ExpectedVersion: reviewing.Version, Message: "Confirm the updated service owner.", FieldIDs: []string{"service_current"},
		VendorAudience: fixture.audience, DueAt: fixture.now.Add(5 * 24 * time.Hour), InvitationTTLMinutes: 60,
	})
	if err == nil {
		t.Fatal("request changes unexpectedly succeeded")
	}
	stored, readErr := fixture.repository.GetVendorWork(context.Background(), scopeFrom(fixture.actor), reviewing.ID)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if stored.State != VendorWorkUnderReview || stored.CurrentRequestID != prepared.CurrentRequestID || stored.SubmissionID == "" || !stored.DueAt.Equal(originalDueAt) || stored.Version != reviewing.Version {
		t.Fatalf("partial clarification state = %#v", stored)
	}
}

func TestRequestVendorWorkChangesStoresCurrentDeadlineAtomically(t *testing.T) {
	fixture := newVendorWorkFixture(t)
	_, reviewing := vendorWorkUnderReview(t, fixture)
	revisedDueAt := fixture.now.Add(5 * 24 * time.Hour)
	outcome, err := fixture.service.RequestChanges(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "reviewer-1"}, reviewing.ID, RequestVendorWorkChangesInput{
		ExpectedVersion: reviewing.Version, Message: "Confirm the updated service owner.", FieldIDs: []string{"service_current"},
		VendorAudience: fixture.audience, DueAt: revisedDueAt, InvitationTTLMinutes: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !outcome.Work.DueAt.Equal(revisedDueAt) {
		t.Fatalf("work deadline = %s, want %s", outcome.Work.DueAt, revisedDueAt)
	}
}

func vendorWorkUnderReview(t *testing.T, fixture vendorWorkFixture) (VendorWorkRequest, VendorWorkRequest) {
	t.Helper()
	prepared, err := fixture.service.Prepare(context.Background(), fixture.actor, PrepareVendorWorkInput{
		RelationshipID: "relationship-1", RelationshipLinkID: fixture.link.ID, Purpose: "Confirm service information.", Instructions: "Review the request.",
		FormTemplateID: "form-1", FormTemplateVersion: 3, VendorAudience: fixture.audience, DueAt: fixture.now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	sent, err := fixture.service.Send(context.Background(), fixture.actor, prepared.ID, SendVendorWorkInput{ExpectedVersion: prepared.Version, VendorAudience: fixture.audience, InvitationTTLMinutes: 60})
	if err != nil {
		t.Fatal(err)
	}
	session, err := fixture.evidence.RedeemInvitation(context.Background(), vendorWorkCaptureToken(t, sent.CaptureURL), fixture.audience)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := fixture.evidence.SubmitSession(context.Background(), session.SessionToken, map[string]formcontract.AnswerValue{"service_current": formcontract.TextAnswer("Yes")}, 1)
	if err != nil {
		t.Fatal(err)
	}
	received, err := fixture.service.RecordSubmission(context.Background(), VendorWorkSubmissionInput{TenantID: fixture.actor.TenantID, WorkRequestID: prepared.ID, RequestID: prepared.CurrentRequestID, SubmissionID: receipt.SubmissionID, CausationID: "response-event"})
	if err != nil {
		t.Fatal(err)
	}
	reviewing, err := fixture.service.StartReview(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "reviewer-1"}, received.ID, StartVendorWorkReviewInput{ExpectedVersion: received.Version})
	if err != nil {
		t.Fatal(err)
	}
	return prepared, reviewing
}

func TestVendorWorkResponseReturnsBoundDocumentBeforeScanCompletes(t *testing.T) {
	fixture := newVendorWorkFixture(t)
	forms := fixture.service.forms.(*monitoring.MemoryRepository)
	_, err := forms.CreateFormRevision(context.Background(), monitoring.FormTemplate{
		ID: "form-document", TenantID: "bank", Name: "Executed document", Purpose: "Collect the executed document.",
		Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationClassic}, Sections: []formcontract.Section{{ID: "document", Title: "Document"}},
		Fields:    []monitoring.TemplateField{{ID: "executed_document", SectionID: "document", Label: "Executed document", Type: formcontract.TypeVendorDocument, Required: true, AcceptedFormats: []string{"application/pdf"}}},
		Lifecycle: monitoring.Lifecycle{Status: monitoring.LifecycleActive, IsCurrent: true, Version: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := fixture.service.Prepare(context.Background(), fixture.actor, PrepareVendorWorkInput{
		RelationshipID: "relationship-1", RelationshipLinkID: fixture.link.ID, Purpose: "Collect the executed document.", Instructions: "Upload the signed document.",
		FormTemplateID: "form-document", FormTemplateVersion: 1, VendorAudience: fixture.audience, DueAt: fixture.now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	sent, err := fixture.service.Send(context.Background(), fixture.actor, prepared.ID, SendVendorWorkInput{ExpectedVersion: prepared.Version, VendorAudience: fixture.audience, InvitationTTLMinutes: 60})
	if err != nil {
		t.Fatal(err)
	}
	session, err := fixture.evidence.RedeemInvitation(context.Background(), vendorWorkCaptureToken(t, sent.CaptureURL), fixture.audience)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := fixture.evidence.StoreArtifact(context.Background(), evidence.ArtifactInput{TenantID: fixture.actor.TenantID, RequestID: prepared.CurrentRequestID, SessionToken: session.SessionToken, FileName: "agreement.pdf", MediaType: "application/pdf"}, strings.NewReader("%PDF-1.7\n1 0 obj\n<< /Type /Catalog >>\nendobj\n%%EOF"))
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := fixture.evidence.SubmitSession(context.Background(), session.SessionToken, map[string]formcontract.AnswerValue{"executed_document": {Document: &formcontract.DocumentAnswer{ArtifactID: artifact.ID, DocumentType: "Executed agreement"}}}, 1)
	if err != nil {
		t.Fatal(err)
	}
	received, err := fixture.service.RecordSubmission(context.Background(), VendorWorkSubmissionInput{TenantID: fixture.actor.TenantID, WorkRequestID: prepared.ID, RequestID: prepared.CurrentRequestID, SubmissionID: receipt.SubmissionID, CausationID: "document-event"})
	if err != nil {
		t.Fatal(err)
	}
	view, err := fixture.service.Response(context.Background(), fixture.actor, received.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Documents) != 1 || view.Documents[0].ArtifactID != artifact.ID || view.Documents[0].ArtifactStatus != evidence.ArtifactStoredUnscanned {
		t.Fatalf("document response = %#v", view.Documents)
	}
}

func TestAcceptVendorWorkBlocksUnavailableCurrentResponseDocument(t *testing.T) {
	for _, status := range []evidence.ArtifactStatus{evidence.ArtifactStoredUnscanned, evidence.ArtifactQuarantined, evidence.ArtifactDeleted} {
		t.Run(string(status), func(t *testing.T) {
			fixture := newVendorWorkFixture(t)
			forms := fixture.service.forms.(*monitoring.MemoryRepository)
			_, err := forms.CreateFormRevision(context.Background(), monitoring.FormTemplate{
				ID: "form-document", TenantID: "bank", Name: "Executed document", Purpose: "Collect the executed document.",
				Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationClassic}, Sections: []formcontract.Section{{ID: "document", Title: "Document"}},
				Fields:    []monitoring.TemplateField{{ID: "executed_document", SectionID: "document", Label: "Executed document", Type: formcontract.TypeVendorDocument, Required: true, AcceptedFormats: []string{"application/pdf"}}},
				Lifecycle: monitoring.Lifecycle{Status: monitoring.LifecycleActive, IsCurrent: true, Version: 1},
			})
			if err != nil {
				t.Fatal(err)
			}
			prepared, err := fixture.service.Prepare(context.Background(), fixture.actor, PrepareVendorWorkInput{
				RelationshipID: "relationship-1", RelationshipLinkID: fixture.link.ID, Purpose: "Collect the executed document.", Instructions: "Upload the signed document.",
				FormTemplateID: "form-document", FormTemplateVersion: 1, VendorAudience: fixture.audience, DueAt: fixture.now.Add(24 * time.Hour),
			})
			if err != nil {
				t.Fatal(err)
			}
			sent, err := fixture.service.Send(context.Background(), fixture.actor, prepared.ID, SendVendorWorkInput{ExpectedVersion: prepared.Version, VendorAudience: fixture.audience, InvitationTTLMinutes: 60})
			if err != nil {
				t.Fatal(err)
			}
			session, err := fixture.evidence.RedeemInvitation(context.Background(), vendorWorkCaptureToken(t, sent.CaptureURL), fixture.audience)
			if err != nil {
				t.Fatal(err)
			}
			artifact, err := fixture.evidence.StoreArtifact(context.Background(), evidence.ArtifactInput{TenantID: fixture.actor.TenantID, RequestID: prepared.CurrentRequestID, SessionToken: session.SessionToken, FileName: "agreement.pdf", MediaType: "application/pdf"}, strings.NewReader("%PDF-1.7\n1 0 obj\n<< /Type /Catalog >>\nendobj\n%%EOF"))
			if err != nil {
				t.Fatal(err)
			}
			receipt, err := fixture.evidence.SubmitSession(context.Background(), session.SessionToken, map[string]formcontract.AnswerValue{"executed_document": {Document: &formcontract.DocumentAnswer{ArtifactID: artifact.ID, DocumentType: "Executed agreement"}}}, 1)
			if err != nil {
				t.Fatal(err)
			}
			received, err := fixture.service.RecordSubmission(context.Background(), VendorWorkSubmissionInput{TenantID: fixture.actor.TenantID, WorkRequestID: prepared.ID, RequestID: prepared.CurrentRequestID, SubmissionID: receipt.SubmissionID, CausationID: "document-event"})
			if err != nil {
				t.Fatal(err)
			}
			reviewing, err := fixture.service.StartReview(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "reviewer-1"}, received.ID, StartVendorWorkReviewInput{ExpectedVersion: received.Version})
			if err != nil {
				t.Fatal(err)
			}
			fixture.service.evidence = vendorWorkArtifactStatusStub{vendorWorkEvidence: fixture.evidence, artifactID: artifact.ID, status: status}
			_, err = fixture.service.Accept(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "reviewer-1"}, reviewing.ID, AcceptVendorWorkInput{ExpectedVersion: reviewing.Version, Rationale: "The executed agreement addresses the request."})
			if !errors.Is(err, ErrVendorWorkAcceptanceBlocked) {
				t.Fatalf("accept with %s document = %v", status, err)
			}
			stored, readErr := fixture.repository.GetVendorWork(context.Background(), scopeFrom(fixture.actor), reviewing.ID)
			if readErr != nil || stored.State != VendorWorkUnderReview || stored.Version != reviewing.Version {
				t.Fatalf("blocked acceptance changed work: value=%#v err=%v", stored, readErr)
			}
		})
	}
}

type vendorWorkArtifactStatusStub struct {
	vendorWorkEvidence
	artifactID string
	status     evidence.ArtifactStatus
}

func (s vendorWorkArtifactStatusStub) GetArtifact(ctx context.Context, tenantID, requestID, artifactID string) (evidence.Artifact, error) {
	artifact, err := s.vendorWorkEvidence.GetArtifact(ctx, tenantID, requestID, artifactID)
	if err == nil && artifact.ID == s.artifactID {
		artifact.Status = s.status
	}
	return artifact, err
}

func TestMemoryVendorWorkListUsesStableCursor(t *testing.T) {
	repo := NewMemoryVendorWorkRepository()
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	for index, work := range []VendorWorkRequest{
		{ID: "00000000-0000-0000-0000-000000000003", TenantID: "bank", LegalEntityID: "entity-a", RelationshipID: "relationship-1", RelationshipLinkID: "link-3"},
		{ID: "00000000-0000-0000-0000-000000000002", TenantID: "bank", LegalEntityID: "entity-a", RelationshipID: "relationship-1", RelationshipLinkID: "link-2"},
		{ID: "00000000-0000-0000-0000-000000000001", TenantID: "bank", LegalEntityID: "entity-a", RelationshipID: "relationship-1", RelationshipLinkID: "link-1"},
	} {
		work.State, work.Version, work.UpdatedAt = VendorWorkAccepted, 1, now.Add(-time.Duration(index)*time.Minute)
		if _, err := repo.CreateVendorWork(context.Background(), work); err != nil {
			t.Fatal(err)
		}
	}
	first, err := repo.ListVendorWork(context.Background(), Scope{TenantID: "bank", LegalEntityID: "entity-a"}, VendorWorkListInput{RelationshipID: "relationship-1", Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 2 || first.NextCursor == "" {
		t.Fatalf("first page = %#v", first)
	}
	second, err := repo.ListVendorWork(context.Background(), Scope{TenantID: "bank", LegalEntityID: "entity-a"}, VendorWorkListInput{RelationshipID: "relationship-1", Cursor: first.NextCursor, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Items) != 1 || second.Items[0].ID == first.Items[0].ID || second.NextCursor != "" {
		t.Fatalf("second page = %#v", second)
	}
}

func TestVendorWorkListFillsPageAfterRestrictedTargetsAreRemoved(t *testing.T) {
	fixture := newVendorWorkFixture(t)
	base := fixture.now.UTC()
	for _, value := range []VendorWorkRequest{
		{ID: "work-hidden", TenantID: "bank", LegalEntityID: "entity-a", RelationshipID: "relationship-1", RelationshipLinkID: "link-hidden", TargetType: LinkTargetProgram, TargetID: "program-hidden", Purpose: "Hidden", OwnerPrincipalID: fixture.actor.PrincipalID, State: VendorWorkAwaitingVendor, DeliveryState: VendorWorkDeliveryDelivered, DueAt: base.Add(24 * time.Hour), Version: 1, CreatedAt: base, UpdatedAt: base.Add(time.Minute)},
		{ID: "work-visible", TenantID: "bank", LegalEntityID: "entity-a", RelationshipID: "relationship-1", RelationshipLinkID: fixture.link.ID, TargetType: LinkTargetProgram, TargetID: "program-1", Purpose: "Visible", OwnerPrincipalID: fixture.actor.PrincipalID, State: VendorWorkAwaitingVendor, DeliveryState: VendorWorkDeliveryDelivered, DueAt: base.Add(24 * time.Hour), Version: 1, CreatedAt: base, UpdatedAt: base},
	} {
		if _, err := fixture.repository.CreateVendorWork(context.Background(), value); err != nil {
			t.Fatal(err)
		}
	}
	fixture.service.ConfigureTargetReader(vendorWorkTargetReader{programs: map[string]continuity.ProgramAggregate{
		"program-1": {Program: continuity.Program{ID: "program-1", TenantID: "bank", LegalEntityID: "entity-a"}},
	}})

	page, err := fixture.service.List(context.Background(), fixture.actor, VendorWorkListInput{RelationshipID: "relationship-1", Limit: 1})
	if err != nil || len(page.Items) != 1 || page.Items[0].ID != "work-visible" || page.NextCursor != "" {
		t.Fatalf("visible work page = %#v, %v", page, err)
	}
}

type vendorWorkReadAuthority struct{ ownerID, reviewerID string }

func (a vendorWorkReadAuthority) Resolve(_ context.Context, input authority.ResolveInput) (authority.Resolution, error) {
	id := a.reviewerID
	if input.Responsibility == authority.ResponsibilityOwner {
		if input.DecisionType != "thirdparty.work.send" {
			return authority.Resolution{}, nil
		}
		id = a.ownerID
	} else if input.DecisionType != "thirdparty.work.review" {
		return authority.Resolution{}, nil
	}
	return authority.Resolution{Principal: authority.Principal{ID: id}}, nil
}
func (vendorWorkReadAuthority) Simulate(context.Context, authority.ResolveInput) (authority.Simulation, error) {
	return authority.Simulation{}, nil
}
func (vendorWorkReadAuthority) Integrity(context.Context, string) ([]authority.IntegrityFinding, error) {
	return nil, nil
}
func (vendorWorkReadAuthority) Policies(context.Context, string) ([]authority.PolicySummary, error) {
	return nil, nil
}

func TestVendorWorkReadAllowsCurrentOwnerAfterAssignmentChanges(t *testing.T) {
	fixture := newVendorWorkFixture(t)
	prepared, err := fixture.service.Prepare(context.Background(), fixture.actor, PrepareVendorWorkInput{
		RelationshipID: "relationship-1", RelationshipLinkID: fixture.link.ID, Purpose: "Confirm service information.", Instructions: "Review the request.",
		FormTemplateID: "form-1", FormTemplateVersion: 3, VendorAudience: fixture.audience, DueAt: fixture.now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	fixture.service.ConfigureReadAuthority(vendorWorkReadAuthority{ownerID: "new-owner", reviewerID: "reviewer-1"})
	if _, err := fixture.service.Get(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "new-owner"}, prepared.ID); err != nil {
		t.Fatalf("current owner read = %v", err)
	}
}

type recordingVendorWorkGuard struct{ requests []commandauth.Request }

func (g *recordingVendorWorkGuard) Authorize(ctx context.Context, request commandauth.Request) (commandauth.Decision, error) {
	g.requests = append(g.requests, request)
	actor, _ := identity.FromContext(ctx)
	return commandauth.Decision{Allowed: true, Actor: actor}, nil
}

func TestAcceptVendorWorkUsesAcceptAuthorityDecision(t *testing.T) {
	fixture := newVendorWorkFixture(t)
	_, reviewing := vendorWorkUnderReview(t, fixture)
	guard := &recordingVendorWorkGuard{}
	fixture.service.ConfigureAuthority(guard)
	actor := Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "reviewer-1"}
	ctx := identity.WithActor(context.Background(), identity.Actor{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PrincipalID: actor.PrincipalID, Kind: "PERSON", IssuedAt: fixture.now.Add(-time.Minute), ExpiresAt: fixture.now.Add(time.Hour)})
	if _, err := fixture.service.Accept(ctx, actor, reviewing.ID, AcceptVendorWorkInput{ExpectedVersion: reviewing.Version, Rationale: "The submitted response meets the request."}); err != nil {
		t.Fatal(err)
	}
	if len(guard.requests) != 1 || guard.requests[0].DecisionType != "thirdparty.work.accept" || guard.requests[0].Responsibility != authority.ResponsibilityReviewer {
		t.Fatalf("accept authority request = %#v", guard.requests)
	}
}

func TestPrepareVendorWorkRequiresRelationshipAndTargetOwnerAuthority(t *testing.T) {
	fixture := newVendorWorkFixture(t)
	guard := &recordingVendorWorkGuard{}
	fixture.service.ConfigureAuthority(guard)
	actor := fixture.actor
	ctx := identity.WithActor(context.Background(), identity.Actor{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PrincipalID: actor.PrincipalID, Kind: "PERSON", IssuedAt: fixture.now.Add(-time.Minute), ExpiresAt: fixture.now.Add(time.Hour)})

	if _, err := fixture.service.Prepare(ctx, actor, PrepareVendorWorkInput{
		RelationshipID: "relationship-1", RelationshipLinkID: fixture.link.ID, Purpose: "Confirm service information.", Instructions: "Review the request.",
		FormTemplateID: "form-1", FormTemplateVersion: 3, VendorAudience: fixture.audience, DueAt: fixture.now.Add(24 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if len(guard.requests) != 2 || guard.requests[0].ObjectType != "VENDOR_RELATIONSHIP" || guard.requests[1].ObjectType != "PROGRAM" || guard.requests[1].ObjectID != "program-1" || guard.requests[1].Responsibility != authority.ResponsibilityOwner {
		t.Fatalf("authority requests = %#v", guard.requests)
	}
}

type vendorWorkTargetReader struct {
	programs map[string]continuity.ProgramAggregate
	matters  map[string]continuity.MatterAggregate
}

func (r vendorWorkTargetReader) GetProgram(_ context.Context, _, id string) (continuity.ProgramAggregate, error) {
	value, ok := r.programs[id]
	if !ok {
		return continuity.ProgramAggregate{}, continuity.ErrNotFound
	}
	return value, nil
}
func (r vendorWorkTargetReader) GetMatter(_ context.Context, _, id string) (continuity.MatterAggregate, error) {
	value, ok := r.matters[id]
	if !ok {
		return continuity.MatterAggregate{}, continuity.ErrNotFound
	}
	return value, nil
}

func TestVendorWorkReadAndPrepareHideRestrictedTarget(t *testing.T) {
	fixture := newVendorWorkFixture(t)
	prepared, err := fixture.service.Prepare(context.Background(), fixture.actor, PrepareVendorWorkInput{
		RelationshipID: "relationship-1", RelationshipLinkID: fixture.link.ID, Purpose: "Confirm service information.", Instructions: "Review the request.",
		FormTemplateID: "form-1", FormTemplateVersion: 3, VendorAudience: fixture.audience, DueAt: fixture.now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	fixture.service.ConfigureTargetReader(vendorWorkTargetReader{programs: map[string]continuity.ProgramAggregate{}})
	if _, err := fixture.service.Get(context.Background(), fixture.actor, prepared.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("restricted work read = %v", err)
	}

	second := newVendorWorkFixture(t)
	second.service.ConfigureTargetReader(vendorWorkTargetReader{programs: map[string]continuity.ProgramAggregate{}})
	if _, err := second.service.Prepare(context.Background(), second.actor, PrepareVendorWorkInput{
		RelationshipID: "relationship-1", RelationshipLinkID: second.link.ID, Purpose: "Confirm service information.", Instructions: "Review the request.",
		FormTemplateID: "form-1", FormTemplateVersion: 3, VendorAudience: second.audience, DueAt: second.now.Add(24 * time.Hour),
	}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("restricted work prepare = %v", err)
	}
}
