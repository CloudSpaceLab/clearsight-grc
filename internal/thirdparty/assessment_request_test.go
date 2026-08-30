package thirdparty

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
)

type invitationDeliveryStub struct {
	requests []evidence.InvitationDeliveryRequest
}

func (s *invitationDeliveryStub) Deliver(_ context.Context, request evidence.InvitationDeliveryRequest) (evidence.InvitationDeliveryReceipt, error) {
	s.requests = append(s.requests, request)
	deliveredAt := time.Date(2026, 8, 26, 10, 1, 0, 0, time.UTC)
	return evidence.InvitationDeliveryReceipt{Status: evidence.InvitationDelivered, DeliveredAt: &deliveredAt}, nil
}

type assessmentEvidenceStub struct {
	requests       map[evidence.RequestOrigin]evidence.Request
	created        []evidence.CreateRequestInput
	issued         []evidence.IssueInvitationInput
	issueErr       error
	revokeErr      error
	revoked        []string
	reassigned     []evidence.ReassignRecipientInput
	preparedBefore bool
	repo           AssessmentRepository
	assessmentID   string
}

func (s *assessmentEvidenceStub) CreateRequest(_ context.Context, input evidence.CreateRequestInput) (evidence.Request, error) {
	s.created = append(s.created, input)
	digest := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(input.Recipient.Audience))))
	request := evidence.Request{
		ID: fmt.Sprintf("request-%d", input.Origin.Version), TenantID: input.TenantID, SubjectType: input.SubjectType, SubjectID: input.SubjectID,
		Title: input.Title, Purpose: input.Purpose, AudienceType: input.AudienceType, Recipient: evidence.Recipient{Type: evidence.RecipientExternalAudience, AudienceHint: "s***@vendor.example", AudienceHash: digest[:], State: evidence.RecipientStateAssigned, Revision: 1},
		Deadline: input.Deadline, KnownFacts: input.KnownFacts, Presentation: input.Presentation, Sections: input.Sections,
		Fields: input.Fields, FormTemplateID: input.FormTemplateID, FormTemplateVersion: input.FormTemplateVersion,
		Origin: input.Origin, CreatedBy: input.CreatedBy, Status: evidence.RequestReady, Version: 1,
	}
	if s.requests == nil {
		s.requests = map[evidence.RequestOrigin]evidence.Request{}
	}
	s.requests[input.Origin] = request
	return request, nil
}

func (s *assessmentEvidenceStub) GetRequestByOrigin(_ context.Context, _ string, origin evidence.RequestOrigin) (evidence.Request, error) {
	value, ok := s.requests[origin]
	if !ok {
		return evidence.Request{}, evidence.ErrNotFound
	}
	return value, nil
}

func (s *assessmentEvidenceStub) ReassignRecipient(_ context.Context, input evidence.ReassignRecipientInput) (evidence.Request, error) {
	s.reassigned = append(s.reassigned, input)
	for origin, value := range s.requests {
		if value.ID == input.RequestID {
			value.Version++
			value.Recipient.AudienceHint = "r***@vendor.example"
			digest := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(input.Recipient.Audience))))
			value.Recipient.AudienceHash = digest[:]
			s.requests[origin] = value
			return value, nil
		}
	}
	return evidence.Request{}, evidence.ErrNotFound
}

type finalizeFailAssessmentRepository struct {
	AssessmentRepository
	err      error
	prepared PrepareAssessmentRequestRecord
	issued   RecordRequestIssuedRecord
}

func (r *finalizeFailAssessmentRepository) PrepareAssessmentRequest(ctx context.Context, record PrepareAssessmentRequestRecord) (AssessmentRequestLink, Assessment, error) {
	r.prepared = record
	return r.AssessmentRepository.PrepareAssessmentRequest(ctx, record)
}

func (r *finalizeFailAssessmentRepository) RecordRequestIssued(_ context.Context, record RecordRequestIssuedRecord) (AssessmentRequestLink, Assessment, error) {
	r.issued = record
	return AssessmentRequestLink{}, Assessment{}, r.err
}

func (s *assessmentEvidenceStub) IssueInvitation(ctx context.Context, input evidence.IssueInvitationInput) (evidence.IssuedInvitation, error) {
	s.issued = append(s.issued, input)
	links, err := s.repo.ListAssessmentRequestLinks(ctx, scopeFromVerified(), s.assessmentID)
	s.preparedBefore = err == nil && len(links) == 1 && links[0].RequestID == input.RequestID && links[0].InvitationID == ""
	if s.issueErr != nil {
		return evidence.IssuedInvitation{}, s.issueErr
	}
	return evidence.IssuedInvitation{InvitationID: "invitation-1", Token: "one-time-token", AudienceHint: "s***@vendor.example", ExpiresAt: time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)}, nil
}

func (s *assessmentEvidenceStub) RevokeInvitation(_ context.Context, _ string, invitationID string) error {
	s.revoked = append(s.revoked, invitationID)
	return s.revokeErr
}

func (s *assessmentEvidenceStub) RevokeRequestCapabilities(context.Context, string, string) error {
	return s.revokeErr
}

type assessmentFormReaderStub struct{ form monitoring.FormTemplate }

func (s assessmentFormReaderStub) ReusableFormRevision(context.Context, string, string, string, int64) (monitoring.FormTemplate, error) {
	return s.form, nil
}

func TestSendAssessmentRequestPreparesExactOriginBeforeInvitation(t *testing.T) {
	assessmentService, repo, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	assessment := mustReadyAssessment(t, assessmentService, mustStartAssessment(t, assessmentService, relationship))
	evidenceStub := &assessmentEvidenceStub{repo: repo, assessmentID: assessment.ID}
	service, err := NewAssessmentRequestService(assessmentService, repo, evidenceStub, assessmentFormReaderStub{form: activeAssessmentForm()}, nil, "https://capture.example.test/respond", "production")
	if err != nil {
		t.Fatal(err)
	}

	outcome, err := service.SendRequest(assessmentContext(), Actor{TenantID: "forged", LegalEntityID: "forged", PrincipalID: "forged"}, assessment.ID, SendAssessmentRequestInput{
		ExpectedVersion: assessment.Version, Audience: "security@vendor.example", Deadline: time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC), InvitationTTLMinutes: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !evidenceStub.preparedBefore {
		t.Fatal("assessment request was not linked before invitation issuance")
	}
	if len(evidenceStub.created) != 1 {
		t.Fatalf("expected one evidence request, got %d", len(evidenceStub.created))
	}
	created := evidenceStub.created[0]
	if created.Origin != (evidence.RequestOrigin{Type: AssessmentRequestOrigin, ID: assessment.ID, Version: 1}) {
		t.Fatalf("unexpected request origin: %#v", created.Origin)
	}
	if created.SubjectType != "VENDOR_RELATIONSHIP" || created.SubjectID != relationship.Relationship.ID || created.FormTemplateID != assessment.FormTemplateID || created.FormTemplateVersion != assessment.FormTemplateVersion {
		t.Fatalf("request did not preserve assessment scope and form: %#v", created)
	}
	if created.KnownFacts["vendor_legal_name"] != relationship.Vendor.LegalName || created.KnownFacts["service_name"] != relationship.Relationship.ServiceName {
		t.Fatalf("known relationship facts were not included: %#v", created.KnownFacts)
	}
	if outcome.State != SendRequestLinkCreatedEmailNotSent || outcome.Assessment.Status != AssessmentCollecting || outcome.Invitation == nil || !strings.Contains(outcome.CaptureURL, "#form_access=one-time-token") {
		t.Fatalf("unexpected send outcome: %#v", outcome)
	}
}

func TestSendAssessmentRequestSupportsPeriodicReviewForActiveRelationship(t *testing.T) {
	assessmentService, repo, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	repo.mu.Lock()
	stored := repo.relationships[relationship.Relationship.ID]
	stored.Status = RelationshipActive
	repo.relationships[relationship.Relationship.ID] = stored
	repo.mu.Unlock()
	input := validStartAssessmentInput(relationship.Relationship.Version)
	input.ReviewKind = AssessmentReviewPeriodic
	input.SourceTrigger = "annual-review-2026"
	assessment, err := assessmentService.StartAssessment(assessmentContext(), assessmentActor(), relationship.Relationship.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	assessment = mustReadyAssessment(t, assessmentService, assessment)
	evidenceStub := &assessmentEvidenceStub{repo: repo, assessmentID: assessment.ID}
	service, err := NewAssessmentRequestService(assessmentService, repo, evidenceStub, assessmentFormReaderStub{form: activeAssessmentForm()}, nil, "https://capture.example.test/respond", "production")
	if err != nil {
		t.Fatal(err)
	}

	outcome, err := service.SendRequest(assessmentContext(), assessmentActor(), assessment.ID, SendAssessmentRequestInput{
		ExpectedVersion: assessment.Version, Audience: "security@vendor.example", Deadline: time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC), InvitationTTLMinutes: 60,
	})
	if err != nil || outcome.Assessment.Status != AssessmentCollecting || outcome.Assessment.ReviewKind != AssessmentReviewPeriodic {
		t.Fatalf("periodic request = (%#v, %v)", outcome, err)
	}
}

func TestSendAssessmentRequestSuppliesVendorRegistrationMessageContext(t *testing.T) {
	assessmentService, repo, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	assessment := mustReadyAssessment(t, assessmentService, mustStartAssessment(t, assessmentService, relationship))
	evidenceStub := &assessmentEvidenceStub{repo: repo, assessmentID: assessment.ID}
	deliveryStub := &invitationDeliveryStub{}
	service, err := NewAssessmentRequestService(assessmentService, repo, evidenceStub, assessmentFormReaderStub{form: activeAssessmentForm()}, evidence.NewInvitationDeliveryService(deliveryStub), "https://capture.example.test/respond", "production")
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC)
	if _, err := service.SendRequest(assessmentContext(), assessmentActor(), assessment.ID, SendAssessmentRequestInput{ExpectedVersion: assessment.Version, Audience: "security@vendor.example", Deadline: deadline, InvitationTTLMinutes: 60}); err != nil {
		t.Fatal(err)
	}
	if len(deliveryStub.requests) != 1 {
		t.Fatalf("delivery requests = %d", len(deliveryStub.requests))
	}
	message := deliveryStub.requests[0].Message
	if message.Kind != evidence.InvitationMessageVendorRegistration || message.RecipientRole != "Vendor contact" || !message.DueAt.Equal(deadline) || !message.ExpiresAt.Equal(time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)) {
		t.Fatalf("message context = %#v", message)
	}
	if message.TaskTitle == "" || !strings.Contains(message.TaskSummary, relationship.Vendor.LegalName) {
		t.Fatalf("message did not identify the vendor task: %#v", message)
	}
}

func TestSendAssessmentRequestFreezesFocusedHeldValueBaselines(t *testing.T) {
	assessmentService, repo, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	input := validStartAssessmentInput(relationship.Relationship.Version)
	input.ScopeKind = AssessmentScopeFocused
	input.SelectedFieldIDs = []string{"registered_address"}
	assessment, err := assessmentService.StartAssessment(assessmentContext(), assessmentActor(), relationship.Relationship.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	assessment = mustReadyAssessment(t, assessmentService, assessment)
	form := activeAssessmentForm()
	form.Fields = []monitoring.TemplateField{
		{ID: "legal_name", SectionID: "organisation", Label: "Legal name", Type: formcontract.TypeShortText, CollectionIntent: formcontract.IntentConfirmOrCorrect, RecordTarget: &formcontract.RecordTarget{Key: "VENDOR.IDENTITY.LEGAL_NAME", RequiredSubjectType: "VENDOR_RELATIONSHIP"}},
		{ID: "registered_address", SectionID: "organisation", Label: "Registered address", Type: formcontract.TypeLongText, CollectionIntent: formcontract.IntentConfirmOrCorrect, RecordTarget: &formcontract.RecordTarget{Key: "VENDOR.IDENTITY.REGISTERED_ADDRESS", RequiredSubjectType: "VENDOR_RELATIONSHIP"}},
	}
	evidenceStub := &assessmentEvidenceStub{repo: repo, assessmentID: assessment.ID}
	service, err := NewAssessmentRequestService(assessmentService, repo, evidenceStub, assessmentFormReaderStub{form: form}, nil, "https://capture.example.test/respond", "production")
	if err != nil {
		t.Fatal(err)
	}
	service.ConfigureRecordTargetResolver(NewRecordTargetResolver(nil))

	_, err = service.SendRequest(assessmentContext(), assessmentActor(), assessment.ID, SendAssessmentRequestInput{ExpectedVersion: assessment.Version, Audience: "security@vendor.example", Deadline: time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC), InvitationTTLMinutes: 60})
	if err != nil {
		t.Fatal(err)
	}
	if len(evidenceStub.created) != 1 || len(evidenceStub.created[0].Fields) != 1 {
		t.Fatalf("focused request fields = %#v", evidenceStub.created)
	}
	field := evidenceStub.created[0].Fields[0]
	if field.ID != "registered_address" || field.RecordBaseline == nil || field.RecordBaseline.RecordID != relationship.Vendor.ID || field.RecordBaseline.RecordVersion != relationship.Vendor.Version || field.RecordBaseline.DisplayValue != relationship.Vendor.RegisteredAddress {
		t.Fatalf("focused baseline = %#v", field)
	}
}

func TestSendAssessmentRequestRejectsRelationshipThatLeftOnboarding(t *testing.T) {
	assessmentService, repo, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	assessment := mustReadyAssessment(t, assessmentService, mustStartAssessment(t, assessmentService, relationship))
	repo.mu.Lock()
	stored := repo.relationships[relationship.Relationship.ID]
	stored.Status = RelationshipTerminated
	repo.relationships[relationship.Relationship.ID] = stored
	repo.mu.Unlock()
	evidenceStub := &assessmentEvidenceStub{repo: repo, assessmentID: assessment.ID}
	service, err := NewAssessmentRequestService(assessmentService, repo, evidenceStub, assessmentFormReaderStub{form: activeAssessmentForm()}, nil, "https://capture.example.test/respond", "production")
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.SendRequest(assessmentContext(), assessmentActor(), assessment.ID, SendAssessmentRequestInput{
		ExpectedVersion: assessment.Version, Audience: "security@vendor.example", Deadline: time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC), InvitationTTLMinutes: 60,
	})
	if !errors.Is(err, ErrInvalidAssessmentTransition) {
		t.Fatalf("terminated relationship send error = %v", err)
	}
	if len(evidenceStub.created) != 0 || len(evidenceStub.issued) != 0 {
		t.Fatalf("terminated relationship created external access: created=%d issued=%d", len(evidenceStub.created), len(evidenceStub.issued))
	}
}

func TestSendAssessmentRequestReturnsRecoverablePreparedStateWithoutRecipientLeak(t *testing.T) {
	assessmentService, repo, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	assessment := mustReadyAssessment(t, assessmentService, mustStartAssessment(t, assessmentService, relationship))
	evidenceStub := &assessmentEvidenceStub{repo: repo, assessmentID: assessment.ID, issueErr: errors.New("provider unavailable")}
	service, err := NewAssessmentRequestService(assessmentService, repo, evidenceStub, assessmentFormReaderStub{form: activeAssessmentForm()}, nil, "https://capture.example.test/respond", "production")
	if err != nil {
		t.Fatal(err)
	}

	outcome, err := service.SendRequest(assessmentContext(), assessmentActor(), assessment.ID, SendAssessmentRequestInput{
		ExpectedVersion: assessment.Version, Audience: "security@vendor.example", Deadline: time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC), InvitationTTLMinutes: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.State != SendRequestReadyInvitationNotIssued || outcome.Assessment.Status != AssessmentReadyToSend || outcome.Request.ID != "request-1" {
		t.Fatalf("unexpected partial outcome: %#v", outcome)
	}
	links, linkErr := repo.ListAssessmentRequestLinks(context.Background(), scopeFromVerified(), assessment.ID)
	if linkErr != nil {
		t.Fatal(linkErr)
	}
	setupJob, jobErr := repo.GetAssessmentSetupJob(context.Background(), scopeFromVerified(), assessment.ID)
	if jobErr != nil {
		t.Fatal(jobErr)
	}
	stored, err := json.Marshal([]any{outcome.Assessment, links, setupJob})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(stored), "security@vendor.example") {
		t.Fatalf("raw recipient leaked into third-party storage: %s", stored)
	}
}

func TestSendAssessmentRequestRetryReusesPreparedRequest(t *testing.T) {
	assessmentService, repo, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	assessment := mustReadyAssessment(t, assessmentService, mustStartAssessment(t, assessmentService, relationship))
	evidenceStub := &assessmentEvidenceStub{repo: repo, assessmentID: assessment.ID, issueErr: errors.New("invitation unavailable")}
	service, err := NewAssessmentRequestService(assessmentService, repo, evidenceStub, assessmentFormReaderStub{form: activeAssessmentForm()}, nil, "https://capture.example.test/respond", "production")
	if err != nil {
		t.Fatal(err)
	}
	input := SendAssessmentRequestInput{ExpectedVersion: assessment.Version, Audience: "security@vendor.example", Deadline: time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC), InvitationTTLMinutes: 60}
	first, err := service.SendRequest(assessmentContext(), assessmentActor(), assessment.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	evidenceStub.issueErr = nil
	input.ExpectedVersion = first.Assessment.Version
	second, err := service.SendRequest(assessmentContext(), assessmentActor(), assessment.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	if first.Request.ID != second.Request.ID || len(evidenceStub.created) != 1 {
		t.Fatalf("retry duplicated the origin request: first=%#v second=%#v created=%d", first.Request, second.Request, len(evidenceStub.created))
	}
	links, err := repo.ListAssessmentRequestLinks(context.Background(), scopeFromVerified(), assessment.ID)
	if err != nil || len(links) != 1 || links[0].InvitationID != "invitation-1" {
		t.Fatalf("retry did not finalize one request link: links=%#v err=%v", links, err)
	}
}

func TestSendAssessmentRequestRejectsStalePreparedRecipientChange(t *testing.T) {
	assessmentService, repo, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	assessment := mustReadyAssessment(t, assessmentService, mustStartAssessment(t, assessmentService, relationship))
	evidenceStub := &assessmentEvidenceStub{repo: repo, assessmentID: assessment.ID, issueErr: errors.New("invitation unavailable")}
	service, err := NewAssessmentRequestService(assessmentService, repo, evidenceStub, assessmentFormReaderStub{form: activeAssessmentForm()}, nil, "https://capture.example.test/respond", "production")
	if err != nil {
		t.Fatal(err)
	}
	input := SendAssessmentRequestInput{ExpectedVersion: assessment.Version, Audience: "security@vendor.example", Deadline: time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC), InvitationTTLMinutes: 60}
	if _, err := service.SendRequest(assessmentContext(), assessmentActor(), assessment.ID, input); err != nil {
		t.Fatal(err)
	}
	input.Audience = "replacement@vendor.example"
	if _, err := service.SendRequest(assessmentContext(), assessmentActor(), assessment.ID, input); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale recipient change error = %v", err)
	}
	if len(evidenceStub.reassigned) != 0 {
		t.Fatalf("stale command changed recipient: %#v", evidenceStub.reassigned)
	}
}

func TestSendAssessmentRequestRevokesInvitationWhenFinalizationFails(t *testing.T) {
	baseService, baseRepo, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	assessment := mustReadyAssessment(t, baseService, mustStartAssessment(t, baseService, relationship))
	repo := &finalizeFailAssessmentRepository{AssessmentRepository: baseRepo, err: errors.New("finalization unavailable")}
	assessmentService := NewAssessmentService(repo, newAssessmentGuard())
	assessmentService.now = baseService.now
	evidenceStub := &assessmentEvidenceStub{repo: repo, assessmentID: assessment.ID}
	service, err := NewAssessmentRequestService(assessmentService, repo, evidenceStub, assessmentFormReaderStub{form: activeAssessmentForm()}, nil, "https://capture.example.test/respond", "production")
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := service.SendRequest(assessmentContext(), assessmentActor(), assessment.ID, SendAssessmentRequestInput{
		ExpectedVersion: assessment.Version, Audience: "security@vendor.example", Deadline: time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC), InvitationTTLMinutes: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.State != SendRequestReadyInvitationNotIssued || len(evidenceStub.revoked) != 1 || evidenceStub.revoked[0] != "invitation-1" {
		t.Fatalf("failed finalization was not compensated: outcome=%#v revoked=%#v", outcome, evidenceStub.revoked)
	}
	if repo.prepared.ActorPrincipalID != "verified-owner" || repo.issued.ActorPrincipalID != "verified-owner" {
		t.Fatalf("repository audit actor was not verified: prepared=%#v issued=%#v", repo.prepared, repo.issued)
	}
}

func TestSendAssessmentRequestKeepsRecoveryTruthfulWhenCompensationFails(t *testing.T) {
	baseService, baseRepo, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	assessment := mustReadyAssessment(t, baseService, mustStartAssessment(t, baseService, relationship))
	repo := &finalizeFailAssessmentRepository{AssessmentRepository: baseRepo, err: errors.New("finalization unavailable")}
	assessmentService := NewAssessmentService(repo, newAssessmentGuard())
	assessmentService.now = baseService.now
	evidenceStub := &assessmentEvidenceStub{repo: repo, assessmentID: assessment.ID, revokeErr: errors.New("revocation unavailable")}
	service, err := NewAssessmentRequestService(assessmentService, repo, evidenceStub, assessmentFormReaderStub{form: activeAssessmentForm()}, nil, "https://capture.example.test/respond", "production")
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := service.SendRequest(assessmentContext(), assessmentActor(), assessment.ID, SendAssessmentRequestInput{
		ExpectedVersion: assessment.Version, Audience: "security@vendor.example", Deadline: time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC), InvitationTTLMinutes: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.State != SendRequestReadyInvitationNotIssued || !strings.Contains(outcome.Recovery, "prior secure access will be revoked") || outcome.CaptureURL != "" || outcome.Invitation != nil {
		t.Fatalf("unsafe recovery outcome: %#v", outcome)
	}
}

func TestSendAssessmentRequestDeliversThroughProtectedBoundary(t *testing.T) {
	assessmentService, repo, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	assessment := mustReadyAssessment(t, assessmentService, mustStartAssessment(t, assessmentService, relationship))
	evidenceStub := &assessmentEvidenceStub{repo: repo, assessmentID: assessment.ID}
	adapter := &invitationDeliveryStub{}
	service, err := NewAssessmentRequestService(assessmentService, repo, evidenceStub, assessmentFormReaderStub{form: activeAssessmentForm()}, evidence.NewInvitationDeliveryService(adapter), "https://capture.example.test/respond?source=bank", "production")
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := service.SendRequest(assessmentContext(), assessmentActor(), assessment.ID, SendAssessmentRequestInput{
		ExpectedVersion: assessment.Version, Audience: "security@vendor.example", Deadline: time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC), InvitationTTLMinutes: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.State != SendRequestDelivered || outcome.CaptureURL != "" || outcome.Invitation == nil || outcome.Invitation.Token != "" || len(adapter.requests) != 1 {
		t.Fatalf("unexpected delivered outcome: %#v requests=%d", outcome, len(adapter.requests))
	}
	if adapter.requests[0].RecipientAddress != "security@vendor.example" || adapter.requests[0].InvitationLink != "https://capture.example.test/respond?source=bank#form_access=one-time-token" {
		t.Fatalf("protected delivery received the wrong values: %#v", adapter.requests[0])
	}
}

func TestSendAssessmentRequestReplacesChangedRecipientOnPreparedRequest(t *testing.T) {
	assessmentService, repo, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	assessment := mustReadyAssessment(t, assessmentService, mustStartAssessment(t, assessmentService, relationship))
	origin := evidence.RequestOrigin{Type: AssessmentRequestOrigin, ID: assessment.ID, Version: 1}
	digest := sha256.Sum256([]byte("old@vendor.example"))
	evidenceStub := &assessmentEvidenceStub{repo: repo, assessmentID: assessment.ID, issueErr: errors.New("invitation unavailable"), requests: map[evidence.RequestOrigin]evidence.Request{
		origin: {ID: "request-1", TenantID: "bank", SubjectType: "VENDOR_RELATIONSHIP", SubjectID: assessment.RelationshipID, AudienceType: "VENDOR", Recipient: evidence.Recipient{Type: evidence.RecipientExternalAudience, AudienceHash: digest[:], AudienceHint: "o***@vendor.example", State: evidence.RecipientStateAssigned, Revision: 1}, Deadline: time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC), FormTemplateID: assessment.FormTemplateID, FormTemplateVersion: assessment.FormTemplateVersion, Origin: origin, CreatedBy: "verified-owner", Status: evidence.RequestReady, Version: 1},
	}}
	service, err := NewAssessmentRequestService(assessmentService, repo, evidenceStub, assessmentFormReaderStub{form: activeAssessmentForm()}, nil, "https://capture.example.test/respond", "production")
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := service.SendRequest(assessmentContext(), assessmentActor(), assessment.ID, SendAssessmentRequestInput{
		ExpectedVersion: assessment.Version, Audience: "replacement@vendor.example", Deadline: time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC), InvitationTTLMinutes: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Request.Version != 2 || outcome.Request.Recipient.AudienceHint != "r***@vendor.example" {
		t.Fatalf("recipient was not replaced before invitation: %#v", outcome.Request)
	}
}

func TestCapturePublicBaseURLRequiresSecureConfiguredOrigin(t *testing.T) {
	for _, test := range []struct {
		name, value, environment string
		wantErr                  bool
	}{
		{name: "absolute https", value: "https://capture.example.test/respond", environment: "production"},
		{name: "production http", value: "http://capture.example.test/respond", environment: "production", wantErr: true},
		{name: "development external http", value: "http://capture.example.test/respond", environment: "development", wantErr: true},
		{name: "development localhost", value: "http://localhost:5173/respond", environment: "development"},
		{name: "relative", value: "/respond", environment: "development", wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseCapturePublicBaseURL(test.value, test.environment)
			if (err != nil) != test.wantErr {
				t.Fatalf("unexpected error %v", err)
			}
		})
	}
}

func TestAssessmentRequestEstimateStaysWithinEvidenceLimit(t *testing.T) {
	if got := estimateAssessmentMinutes(200); got != 60 {
		t.Fatalf("200-field estimate = %d, want 60", got)
	}
}

func TestSendAssessmentRequestCapsInvitationAtRequestDeadline(t *testing.T) {
	assessmentService, repo, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	assessment := mustReadyAssessment(t, assessmentService, mustStartAssessment(t, assessmentService, relationship))
	evidenceService := newAssessmentEvidenceService()
	service, err := NewAssessmentRequestService(assessmentService, repo, evidenceService, assessmentFormReaderStub{form: activeAssessmentForm()}, nil, "https://capture.example.test/respond", "production")
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().UTC().Add(24 * time.Hour)
	outcome, err := service.SendRequest(assessmentContext(), assessmentActor(), assessment.ID, SendAssessmentRequestInput{
		ExpectedVersion: assessment.Version, Audience: "security@vendor.example", Deadline: deadline, InvitationTTLMinutes: 2 * 24 * 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Invitation == nil || !outcome.Invitation.ExpiresAt.Equal(deadline) {
		t.Fatalf("invitation expiry = %#v, want %s", outcome.Invitation, deadline)
	}
}

func TestSendAssessmentRequestRecipientChangeRevokesPriorInvitation(t *testing.T) {
	assessmentService, repo, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	assessment := mustReadyAssessment(t, assessmentService, mustStartAssessment(t, assessmentService, relationship))
	evidenceService := newAssessmentEvidenceService()
	form := activeAssessmentForm()
	origin := evidence.RequestOrigin{Type: AssessmentRequestOrigin, ID: assessment.ID, Version: 1}
	deadline := time.Now().UTC().Add(24 * time.Hour)
	request, err := evidenceService.CreateRequest(evidence.WithRequestOriginAuthority(context.Background(), AssessmentRequestOrigin), assessmentEvidenceRequestInput(assessmentActor(), assessment, relationship, form, origin, "old@vendor.example", deadline))
	if err != nil {
		t.Fatal(err)
	}
	oldInvitation, err := evidenceService.IssueInvitation(context.Background(), evidence.IssueInvitationInput{
		TenantID: "bank", LegalEntityID: "entity", RequestID: request.ID, Audience: "old@vendor.example", Purpose: "Complete the vendor due-diligence request.", TTLMinutes: 60, CreatedBy: "verified-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, prepared, err := repo.PrepareAssessmentRequest(context.Background(), PrepareAssessmentRequestRecord{
		Scope: scopeFromVerified(), AssessmentID: assessment.ID, ExpectedVersion: assessment.Version, ActorPrincipalID: "verified-owner", RequestID: request.ID,
		Purpose: AssessmentRequestInitial, OriginType: AssessmentRequestOrigin, OriginID: assessment.ID, OriginSequence: 1,
		PreparedAt: time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewAssessmentRequestService(assessmentService, repo, evidenceService, assessmentFormReaderStub{form: form}, nil, "https://capture.example.test/respond", "production")
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := service.SendRequest(assessmentContext(), assessmentActor(), assessment.ID, SendAssessmentRequestInput{
		ExpectedVersion: prepared.Version, Audience: "replacement@vendor.example", Deadline: deadline, InvitationTTLMinutes: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Assessment.Status != AssessmentCollecting || outcome.Assessment.Version != prepared.Version+1 {
		t.Fatalf("replacement did not finalize collection: %#v", outcome.Assessment)
	}
	if _, err := evidenceService.RedeemInvitation(context.Background(), oldInvitation.Token, "old@vendor.example"); !errors.Is(err, evidence.ErrInvitationInvalid) {
		t.Fatalf("superseded invitation remained usable: %v", err)
	}
}

func activeAssessmentForm() monitoring.FormTemplate {
	return monitoring.FormTemplate{
		ID: "form-1", TenantID: "bank", Name: "Vendor due diligence", Purpose: "Provide the information required for this vendor review.",
		Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationWizard, AllowModeSwitch: true},
		Sections:     []formcontract.Section{{ID: "organisation", Title: "Organisation"}},
		Fields:       []monitoring.TemplateField{{ID: "contact_email", SectionID: "organisation", Label: "Contact email", Type: formcontract.TypeEmail, Required: true}},
		Lifecycle:    monitoring.Lifecycle{Status: monitoring.LifecycleActive, IsCurrent: true, Version: 3},
	}
}
