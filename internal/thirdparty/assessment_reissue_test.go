package thirdparty

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

type collectingRequestFixture struct {
	assessmentService *AssessmentService
	repository        *MemoryAssessmentRepository
	evidenceService   *evidence.Service
	requestService    *AssessmentRequestService
	assessment        Assessment
	request           evidence.Request
	invitationToken   string
	audience          string
}

type captureReissueRepository struct {
	AssessmentRepository
	prepared PrepareRequestReissueRecord
	record   FinalizeRequestReissueRecord
	err      error
}

func (r *captureReissueRepository) PrepareRequestReissue(ctx context.Context, record PrepareRequestReissueRecord) (AssessmentRequestLink, Assessment, error) {
	r.prepared = record
	return r.AssessmentRepository.PrepareRequestReissue(ctx, record)
}

func (r *captureReissueRepository) FinalizeRequestReissue(ctx context.Context, record FinalizeRequestReissueRecord) (AssessmentRequestLink, Assessment, error) {
	r.record = record
	if r.err != nil {
		failure := r.err
		r.err = nil
		return AssessmentRequestLink{}, Assessment{}, failure
	}
	return r.AssessmentRepository.FinalizeRequestReissue(ctx, record)
}

type blockingIssueEvidence struct {
	assessmentRequestEvidence
	started chan struct{}
	release chan struct{}
	once    sync.Once
	mu      sync.Mutex
	issued  int
}

type wrongTenantRequestEvidence struct{ assessmentRequestEvidence }

type failingReissueEvidence struct {
	assessmentRequestEvidence
	service            *evidence.Service
	revoked            bool
	revokedBeforeIssue bool
	issueErr           error
}

func (e *failingReissueEvidence) RevokeRequestCapabilities(ctx context.Context, tenant, requestID string) error {
	e.revoked = true
	return e.service.RevokeRequestCapabilities(ctx, tenant, requestID)
}

func (e *failingReissueEvidence) IssueInvitation(context.Context, evidence.IssueInvitationInput) (evidence.IssuedInvitation, error) {
	e.revokedBeforeIssue = e.revoked
	return evidence.IssuedInvitation{}, e.issueErr
}

func (e wrongTenantRequestEvidence) GetRequestByOrigin(ctx context.Context, tenant string, origin evidence.RequestOrigin) (evidence.Request, error) {
	request, err := e.assessmentRequestEvidence.GetRequestByOrigin(ctx, tenant, origin)
	request.TenantID = "other-bank"
	return request, err
}

func (e *blockingIssueEvidence) IssueInvitation(ctx context.Context, input evidence.IssueInvitationInput) (evidence.IssuedInvitation, error) {
	e.mu.Lock()
	e.issued++
	e.mu.Unlock()
	e.once.Do(func() { close(e.started) })
	select {
	case <-e.release:
	case <-ctx.Done():
		return evidence.IssuedInvitation{}, ctx.Err()
	}
	return e.assessmentRequestEvidence.IssueInvitation(ctx, input)
}

func (e *blockingIssueEvidence) issueCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.issued
}

func newCollectingRequestFixture(t *testing.T) collectingRequestFixture {
	t.Helper()
	assessmentService, repository, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	assessment := mustReadyAssessment(t, assessmentService, mustStartAssessment(t, assessmentService, relationship))
	evidenceService := evidence.NewService(evidence.NewMemoryRepository(nil, nil), evidence.NewMemoryObjectStore())
	requestService, err := NewAssessmentRequestService(
		assessmentService, repository, evidenceService, assessmentFormReaderStub{form: activeAssessmentForm()}, nil,
		"https://capture.example.test/respond", "production",
	)
	if err != nil {
		t.Fatal(err)
	}
	audience := "security@vendor.example"
	outcome, err := requestService.SendRequest(assessmentContext(), assessmentActor(), assessment.ID, SendAssessmentRequestInput{
		ExpectedVersion: assessment.Version, Audience: audience,
		Deadline: time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC), InvitationTTLMinutes: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.State != SendRequestLinkCreatedEmailNotSent {
		t.Fatalf("initial request did not produce the reload recovery case: %#v", outcome)
	}
	return collectingRequestFixture{
		assessmentService: assessmentService, repository: repository, evidenceService: evidenceService,
		requestService: requestService, assessment: outcome.Assessment, request: outcome.Request,
		invitationToken: captureToken(t, outcome.CaptureURL), audience: audience,
	}
}

func TestReissueAssessmentRequestAfterReloadReplacesInvitationAndRedeemedSession(t *testing.T) {
	fixture := newCollectingRequestFixture(t)
	priorSession, err := fixture.evidenceService.RedeemInvitation(context.Background(), fixture.invitationToken, fixture.audience)
	if err != nil {
		t.Fatal(err)
	}

	outcome, err := fixture.requestService.ReissueRequest(assessmentContext(), Actor{TenantID: "forged", LegalEntityID: "forged", PrincipalID: "forged"}, fixture.assessment.ID, ReissueAssessmentRequestInput{
		ExpectedVersion: fixture.assessment.Version, Audience: fixture.audience, InvitationTTLMinutes: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.State != SendRequestLinkCreatedEmailNotSent || outcome.Assessment.Status != AssessmentCollecting || outcome.Assessment.Version != fixture.assessment.Version+2 {
		t.Fatalf("unexpected reissue outcome: %#v", outcome)
	}
	if outcome.Request.ID != fixture.request.ID || outcome.Invitation == nil || outcome.Invitation.Token != "" || outcome.CaptureURL == "" {
		t.Fatalf("replacement did not reuse the current request safely: %#v", outcome)
	}
	if _, _, err := fixture.evidenceService.SessionRequest(context.Background(), priorSession.SessionToken); !errors.Is(err, evidence.ErrSessionInvalid) {
		t.Fatalf("prior redeemed session remained usable: %v", err)
	}
	replacementToken := captureToken(t, outcome.CaptureURL)
	if replacementToken == fixture.invitationToken {
		t.Fatal("replacement reused the prior one-time token")
	}
	if _, err := fixture.evidenceService.RedeemInvitation(context.Background(), replacementToken, fixture.audience); err != nil {
		t.Fatalf("replacement invitation was not redeemable: %v", err)
	}
	link, err := fixture.repository.GetCurrentAssessmentRequestLink(context.Background(), scopeFromVerified(), fixture.assessment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if link.RequestID != fixture.request.ID || link.InvitationID != outcome.Invitation.InvitationID {
		t.Fatalf("current request link was not updated: %#v", link)
	}
	stored, err := json.Marshal([]any{outcome.Assessment, link})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(stored), fixture.audience) || strings.Contains(string(stored), replacementToken) {
		t.Fatalf("protected value leaked into assessment persistence: %s", stored)
	}
	audit, err := json.Marshal([]any{fixture.repository.assessmentEvents, fixture.repository.assessmentOutbox})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(audit), "AssessmentRequestReissuePrepared") || !strings.Contains(string(audit), "AssessmentRequestReissued") || strings.Contains(string(audit), fixture.audience) || strings.Contains(string(audit), replacementToken) {
		t.Fatalf("in-memory audit/outbox was incomplete or unsafe: %s", audit)
	}
}

func TestReissueAssessmentRequestIssuanceFailureLeavesPriorCapabilityRevokedAndRetryable(t *testing.T) {
	fixture := newCollectingRequestFixture(t)
	priorSession, err := fixture.evidenceService.RedeemInvitation(context.Background(), fixture.invitationToken, fixture.audience)
	if err != nil {
		t.Fatal(err)
	}
	failing := &failingReissueEvidence{
		assessmentRequestEvidence: fixture.evidenceService,
		service:                   fixture.evidenceService,
		issueErr:                  errors.New("invitation issuer unavailable"),
	}
	requestService, err := NewAssessmentRequestService(
		fixture.assessmentService, fixture.repository, failing, assessmentFormReaderStub{form: activeAssessmentForm()}, nil,
		"https://capture.example.test/respond", "production",
	)
	if err != nil {
		t.Fatal(err)
	}

	outcome, err := requestService.ReissueRequest(assessmentContext(), assessmentActor(), fixture.assessment.ID, ReissueAssessmentRequestInput{
		ExpectedVersion: fixture.assessment.Version, Audience: fixture.audience, InvitationTTLMinutes: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !failing.revokedBeforeIssue {
		t.Fatal("replacement issuance started before prior request capabilities were revoked")
	}
	if outcome.State != SendRequestReadyInvitationNotIssued || outcome.Assessment.Version != fixture.assessment.Version+1 || outcome.Invitation != nil || outcome.CaptureURL != "" || outcome.Recovery == "" {
		t.Fatalf("issuance failure did not return a truthful retry state: %#v", outcome)
	}
	if _, err := fixture.evidenceService.RedeemInvitation(context.Background(), fixture.invitationToken, fixture.audience); !errors.Is(err, evidence.ErrInvitationInvalid) {
		t.Fatalf("prior invitation remained usable after replacement failure: %v", err)
	}
	if _, _, err := fixture.evidenceService.SessionRequest(context.Background(), priorSession.SessionToken); !errors.Is(err, evidence.ErrSessionInvalid) {
		t.Fatalf("prior session remained usable after replacement failure: %v", err)
	}
}

func TestRequestCapabilityRevocationEndsEveryInvitationAndSessionIdempotently(t *testing.T) {
	fixture := newCollectingRequestFixture(t)
	priorSession, err := fixture.evidenceService.RedeemInvitation(context.Background(), fixture.invitationToken, fixture.audience)
	if err != nil {
		t.Fatal(err)
	}

	if err := fixture.evidenceService.RevokeRequestCapabilities(context.Background(), "bank", fixture.request.ID); err != nil {
		t.Fatal(err)
	}
	if err := fixture.evidenceService.RevokeRequestCapabilities(context.Background(), "bank", fixture.request.ID); err != nil {
		t.Fatalf("repeated revocation was not idempotent: %v", err)
	}
	if _, err := fixture.evidenceService.RedeemInvitation(context.Background(), fixture.invitationToken, fixture.audience); !errors.Is(err, evidence.ErrInvitationInvalid) {
		t.Fatalf("request invitation remained usable: %v", err)
	}
	if _, _, err := fixture.evidenceService.SessionRequest(context.Background(), priorSession.SessionToken); !errors.Is(err, evidence.ErrSessionInvalid) {
		t.Fatalf("request session remained usable: %v", err)
	}
}

func TestReissueAssessmentRequestWithoutCaptureAddressRevokesPriorCapabilityBeforeRetryState(t *testing.T) {
	fixture := newCollectingRequestFixture(t)
	priorSession, err := fixture.evidenceService.RedeemInvitation(context.Background(), fixture.invitationToken, fixture.audience)
	if err != nil {
		t.Fatal(err)
	}
	requestService, err := NewAssessmentRequestService(
		fixture.assessmentService, fixture.repository, fixture.evidenceService, assessmentFormReaderStub{form: activeAssessmentForm()}, nil,
		"", "development",
	)
	if err != nil {
		t.Fatal(err)
	}

	outcome, err := requestService.ReissueRequest(assessmentContext(), assessmentActor(), fixture.assessment.ID, ReissueAssessmentRequestInput{
		ExpectedVersion: fixture.assessment.Version, Audience: fixture.audience, InvitationTTLMinutes: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.State != SendRequestReadyInvitationNotIssued || outcome.Assessment.Version != fixture.assessment.Version+1 || outcome.Recovery == "" {
		t.Fatalf("missing capture address did not leave a prepared retry state: %#v", outcome)
	}
	if _, err := fixture.evidenceService.RedeemInvitation(context.Background(), fixture.invitationToken, fixture.audience); !errors.Is(err, evidence.ErrInvitationInvalid) {
		t.Fatalf("prior invitation remained usable without a configured capture address: %v", err)
	}
	if _, _, err := fixture.evidenceService.SessionRequest(context.Background(), priorSession.SessionToken); !errors.Is(err, evidence.ErrSessionInvalid) {
		t.Fatalf("prior session remained usable without a configured capture address: %v", err)
	}
}

func TestReissueAssessmentRequestRejectsStaleVersionBeforeReplacement(t *testing.T) {
	fixture := newCollectingRequestFixture(t)
	linkBefore, err := fixture.repository.GetCurrentAssessmentRequestLink(context.Background(), scopeFromVerified(), fixture.assessment.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = fixture.requestService.ReissueRequest(assessmentContext(), assessmentActor(), fixture.assessment.ID, ReissueAssessmentRequestInput{
		ExpectedVersion: fixture.assessment.Version - 1, Audience: fixture.audience, InvitationTTLMinutes: 60,
	})
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale reissue error = %v", err)
	}
	linkAfter, err := fixture.repository.GetCurrentAssessmentRequestLink(context.Background(), scopeFromVerified(), fixture.assessment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if linkAfter.InvitationID != linkBefore.InvitationID {
		t.Fatalf("stale command replaced invitation: before=%#v after=%#v", linkBefore, linkAfter)
	}
}

func TestReissueAssessmentRequestDeliversWithoutReturningLinkAndAuditsVerifiedOwner(t *testing.T) {
	fixture := newCollectingRequestFixture(t)
	repository := &captureReissueRepository{AssessmentRepository: fixture.repository}
	assessmentService := NewAssessmentService(repository, newAssessmentGuard())
	assessmentService.now = fixture.assessmentService.now
	adapter := &invitationDeliveryStub{}
	requestService, err := NewAssessmentRequestService(assessmentService, repository, fixture.evidenceService, assessmentFormReaderStub{form: activeAssessmentForm()}, evidence.NewInvitationDeliveryService(adapter), "https://capture.example.test/respond", "production")
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := requestService.ReissueRequest(assessmentContext(), Actor{TenantID: "forged", LegalEntityID: "forged", PrincipalID: "forged"}, fixture.assessment.ID, ReissueAssessmentRequestInput{
		ExpectedVersion: fixture.assessment.Version, Audience: fixture.audience, InvitationTTLMinutes: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.State != SendRequestDelivered || outcome.CaptureURL != "" || len(adapter.requests) != 1 {
		t.Fatalf("delivered replacement exposed a fallback link: outcome=%#v requests=%#v", outcome, adapter.requests)
	}
	if repository.prepared.ActorPrincipalID != "verified-owner" || repository.record.ActorPrincipalID != "verified-owner" || repository.record.RequestID != fixture.request.ID {
		t.Fatalf("replacement audit did not use the verified owner and current request: %#v", repository.record)
	}
	storedRecord, err := json.Marshal(repository.record)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(storedRecord), fixture.audience) || strings.Contains(string(storedRecord), "capture_invite") {
		t.Fatalf("replacement audit record contained protected delivery data: %s", storedRecord)
	}
	if adapter.requests[0].RecipientAddress != fixture.audience || !strings.Contains(adapter.requests[0].InvitationLink, "capture_invite=") {
		t.Fatalf("protected delivery boundary received the wrong replacement values: %#v", adapter.requests[0])
	}
}

func TestReissueAssessmentRequestRecoversAfterAuditFinalizationFailure(t *testing.T) {
	fixture := newCollectingRequestFixture(t)
	repository := &captureReissueRepository{AssessmentRepository: fixture.repository, err: errors.New("audit unavailable")}
	assessmentService := NewAssessmentService(repository, newAssessmentGuard())
	assessmentService.now = fixture.assessmentService.now
	requestService, err := NewAssessmentRequestService(assessmentService, repository, fixture.evidenceService, assessmentFormReaderStub{form: activeAssessmentForm()}, nil, "https://capture.example.test/respond", "production")
	if err != nil {
		t.Fatal(err)
	}
	input := ReissueAssessmentRequestInput{ExpectedVersion: fixture.assessment.Version, Audience: fixture.audience, InvitationTTLMinutes: 60}
	if _, err := requestService.ReissueRequest(assessmentContext(), assessmentActor(), fixture.assessment.ID, input); err == nil {
		t.Fatal("expected replacement audit failure")
	}
	current, loadErr := fixture.repository.GetAssessment(context.Background(), scopeFromVerified(), fixture.assessment.ID)
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	input.ExpectedVersion = current.Version
	recovered, err := requestService.ReissueRequest(assessmentContext(), assessmentActor(), fixture.assessment.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.State != SendRequestLinkCreatedEmailNotSent || recovered.Assessment.Version != fixture.assessment.Version+3 {
		t.Fatalf("replacement did not recover after finalization failure: %#v", recovered)
	}
	if _, err := fixture.evidenceService.RedeemInvitation(context.Background(), captureToken(t, recovered.CaptureURL), fixture.audience); err != nil {
		t.Fatalf("recovered replacement invitation was not redeemable: %v", err)
	}
}

func TestReissueAssessmentRequestSerializesBeforeInvitationIssuance(t *testing.T) {
	fixture := newCollectingRequestFixture(t)
	blocking := &blockingIssueEvidence{assessmentRequestEvidence: fixture.evidenceService, started: make(chan struct{}), release: make(chan struct{})}
	requestService, err := NewAssessmentRequestService(fixture.assessmentService, fixture.repository, blocking, assessmentFormReaderStub{form: activeAssessmentForm()}, nil, "https://capture.example.test/respond", "production")
	if err != nil {
		t.Fatal(err)
	}
	input := ReissueAssessmentRequestInput{ExpectedVersion: fixture.assessment.Version, Audience: fixture.audience, InvitationTTLMinutes: 60}
	type result struct {
		outcome SendRequestOutcome
		err     error
	}
	firstResult := make(chan result, 1)
	go func() {
		outcome, callErr := requestService.ReissueRequest(assessmentContext(), assessmentActor(), fixture.assessment.ID, input)
		firstResult <- result{outcome: outcome, err: callErr}
	}()
	<-blocking.started
	if _, err := requestService.ReissueRequest(assessmentContext(), assessmentActor(), fixture.assessment.ID, input); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("interleaved stale replacement error = %v", err)
	}
	if blocking.issueCount() != 1 {
		t.Fatalf("interleaved command issued another invitation: count=%d", blocking.issueCount())
	}
	close(blocking.release)
	first := <-firstResult
	if first.err != nil {
		t.Fatal(first.err)
	}
	if first.outcome.Assessment.Version != fixture.assessment.Version+2 || first.outcome.State != SendRequestLinkCreatedEmailNotSent {
		t.Fatalf("reserved replacement did not finalize: %#v", first.outcome)
	}
	if _, err := fixture.evidenceService.RedeemInvitation(context.Background(), captureToken(t, first.outcome.CaptureURL), fixture.audience); err != nil {
		t.Fatalf("serialized replacement invitation was not usable: %v", err)
	}
}

func TestReissueAssessmentRequestRequiresCollectingCurrentRequestAndAudience(t *testing.T) {
	assessmentService, repository, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	ready := mustReadyAssessment(t, assessmentService, mustStartAssessment(t, assessmentService, relationship))
	evidenceService := evidence.NewService(evidence.NewMemoryRepository(nil, nil), evidence.NewMemoryObjectStore())
	requestService, err := NewAssessmentRequestService(assessmentService, repository, evidenceService, assessmentFormReaderStub{form: activeAssessmentForm()}, nil, "https://capture.example.test/respond", "production")
	if err != nil {
		t.Fatal(err)
	}
	_, err = requestService.ReissueRequest(assessmentContext(), assessmentActor(), ready.ID, ReissueAssessmentRequestInput{ExpectedVersion: ready.Version, Audience: "security@vendor.example", InvitationTTLMinutes: 60})
	if !errors.Is(err, ErrInvalidAssessmentTransition) {
		t.Fatalf("ready assessment reissue error = %v", err)
	}

	fixture := newCollectingRequestFixture(t)
	_, err = fixture.requestService.ReissueRequest(assessmentContext(), assessmentActor(), fixture.assessment.ID, ReissueAssessmentRequestInput{
		ExpectedVersion: fixture.assessment.Version, Audience: "other@vendor.example", InvitationTTLMinutes: 60,
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("wrong current recipient error = %v", err)
	}
}

func TestReissueAssessmentRequestRejectsRelationshipOutsideOnboardingBeforeReservation(t *testing.T) {
	fixture := newCollectingRequestFixture(t)
	fixture.repository.MemoryRepository.mu.Lock()
	relationship := fixture.repository.MemoryRepository.relationships[fixture.assessment.RelationshipID]
	relationship.Status = RelationshipExiting
	fixture.repository.MemoryRepository.relationships[relationship.ID] = relationship
	fixture.repository.MemoryRepository.mu.Unlock()

	_, err := fixture.requestService.ReissueRequest(assessmentContext(), assessmentActor(), fixture.assessment.ID, ReissueAssessmentRequestInput{
		ExpectedVersion: fixture.assessment.Version, Audience: fixture.audience, InvitationTTLMinutes: 60,
	})
	if !errors.Is(err, ErrInvalidAssessmentTransition) {
		t.Fatalf("exiting relationship reissue error = %v", err)
	}
	current, err := fixture.repository.GetAssessment(context.Background(), scopeFromVerified(), fixture.assessment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Version != fixture.assessment.Version {
		t.Fatalf("rejected relationship reserved a replacement: %#v", current)
	}
	if _, err := fixture.evidenceService.RedeemInvitation(context.Background(), fixture.invitationToken, fixture.audience); err != nil {
		t.Fatalf("rejected relationship changed the current invitation: %v", err)
	}
}

func TestReissueAssessmentRequestRejectsFaultyCrossTenantRequestReader(t *testing.T) {
	fixture := newCollectingRequestFixture(t)
	requestService, err := NewAssessmentRequestService(
		fixture.assessmentService, fixture.repository, wrongTenantRequestEvidence{assessmentRequestEvidence: fixture.evidenceService},
		assessmentFormReaderStub{form: activeAssessmentForm()}, nil, "https://capture.example.test/respond", "production",
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = requestService.ReissueRequest(assessmentContext(), assessmentActor(), fixture.assessment.ID, ReissueAssessmentRequestInput{
		ExpectedVersion: fixture.assessment.Version, Audience: fixture.audience, InvitationTTLMinutes: 60,
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("cross-tenant request reader error = %v", err)
	}
	current, err := fixture.repository.GetAssessment(context.Background(), scopeFromVerified(), fixture.assessment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Version != fixture.assessment.Version {
		t.Fatalf("cross-tenant request reserved a replacement: %#v", current)
	}
}

func TestReissueAssessmentRequestFailsClosedForScopeAndOwner(t *testing.T) {
	fixture := newCollectingRequestFixture(t)
	_, err := fixture.requestService.ReissueRequest(assessmentContextFor("bank", "other-entity", "verified-owner"), assessmentActor(), fixture.assessment.ID, ReissueAssessmentRequestInput{
		ExpectedVersion: fixture.assessment.Version, Audience: fixture.audience, InvitationTTLMinutes: 60,
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("wrong legal-entity scope error = %v", err)
	}

	guard := newAssessmentGuard()
	guard.err = commandauth.ErrNotAuthorized
	assessmentService := NewAssessmentService(fixture.repository, guard)
	assessmentService.now = fixture.assessmentService.now
	requestService, err := NewAssessmentRequestService(assessmentService, fixture.repository, fixture.evidenceService, assessmentFormReaderStub{form: activeAssessmentForm()}, nil, "https://capture.example.test/respond", "production")
	if err != nil {
		t.Fatal(err)
	}
	_, err = requestService.ReissueRequest(assessmentContext(), assessmentActor(), fixture.assessment.ID, ReissueAssessmentRequestInput{
		ExpectedVersion: fixture.assessment.Version, Audience: fixture.audience, InvitationTTLMinutes: 60,
	})
	if !errors.Is(err, commandauth.ErrNotAuthorized) {
		t.Fatalf("non-owner reissue error = %v", err)
	}
}

func captureToken(t *testing.T, rawURL string) string {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	token := parsed.Query().Get("capture_invite")
	if token == "" {
		t.Fatalf("capture URL did not contain a one-time token: %q", rawURL)
	}
	return token
}
