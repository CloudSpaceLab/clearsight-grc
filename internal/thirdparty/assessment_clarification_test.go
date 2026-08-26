package thirdparty

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type clarificationPrepareBarrier struct {
	AssessmentRepository
	entered chan struct{}
	release chan struct{}
}

func (r *clarificationPrepareBarrier) PrepareAssessmentRequest(ctx context.Context, record PrepareAssessmentRequestRecord) (AssessmentRequestLink, Assessment, error) {
	r.entered <- struct{}{}
	<-r.release
	return r.AssessmentRepository.PrepareAssessmentRequest(ctx, record)
}

type fixedAssessmentGuard struct{}

func (fixedAssessmentGuard) Authorize(ctx context.Context, _ commandauth.Request) (commandauth.Decision, error) {
	actor, err := identity.Require(ctx)
	return commandauth.Decision{Allowed: err == nil, Enforced: true, Actor: actor}, err
}

func TestRequestAssessmentClarificationCreatesNextRequestAndReturnsToCollection(t *testing.T) {
	assessmentService, repo, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	assessment := assessmentUnderReviewFixture(t, assessmentService, repo, relationship)
	evidenceStub := &assessmentEvidenceStub{repo: repo, assessmentID: assessment.ID}
	deliveryStub := &invitationDeliveryStub{}
	service, err := NewAssessmentRequestService(assessmentService, repo, evidenceStub, assessmentFormReaderStub{form: activeAssessmentForm()}, evidence.NewInvitationDeliveryService(deliveryStub), "https://capture.example.test/respond", "production")
	if err != nil {
		t.Fatal(err)
	}

	outcome, err := service.RequestClarification(assessmentContext(), Actor{TenantID: "forged", LegalEntityID: "forged", PrincipalID: "forged"}, assessment.ID, RequestAssessmentClarificationInput{
		ExpectedVersion: assessment.Version, RequestFields: []string{"contact_email"}, Message: "Provide the current security contact.",
		Audience: "security@vendor.example", Deadline: assessment.ReviewDueAt.Add(-time.Hour), InvitationTTLMinutes: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Assessment.Status != AssessmentCollecting || outcome.Assessment.Version != assessment.Version+2 || outcome.State != SendRequestDelivered || outcome.CaptureURL != "" {
		t.Fatalf("clarification outcome = %#v", outcome)
	}
	links, err := repo.ListAssessmentRequestLinks(context.Background(), scopeFromVerified(), assessment.ID)
	if err != nil || len(links) != 2 || links[0].RequestID == outcome.Assessment.CurrentRequestID || links[1].RequestID != outcome.Assessment.CurrentRequestID || links[1].Sequence != 2 || links[1].Purpose != AssessmentRequestClarification || links[1].InvitationID == "" {
		t.Fatalf("clarification history = (%#v, %v)", links, err)
	}
	if len(evidenceStub.created) != 1 || evidenceStub.created[0].Origin.Version != 2 || len(evidenceStub.created[0].Fields) != 1 || evidenceStub.created[0].Fields[0].ID != "contact_email" || evidenceStub.created[0].Purpose != "Provide the current security contact." {
		t.Fatalf("clarification request = %#v", evidenceStub.created)
	}
	if len(deliveryStub.requests) != 1 || deliveryStub.requests[0].RecipientAddress != "security@vendor.example" || !strings.Contains(deliveryStub.requests[0].InvitationLink, "capture_invite=one-time-token") {
		t.Fatalf("protected delivery = %#v", deliveryStub.requests)
	}
	if len(repo.assessmentEvents) < 2 || repo.assessmentEvents[len(repo.assessmentEvents)-2].Type != "AssessmentRequestPrepared" || repo.assessmentEvents[len(repo.assessmentEvents)-1].Type != "AssessmentRequestIssued" || repo.assessmentEvents[len(repo.assessmentEvents)-1].ActorPrincipalID != "verified-owner" {
		t.Fatalf("clarification audit = %#v", repo.assessmentEvents)
	}
	stored, _ := json.Marshal([]any{outcome.Assessment, links, repo.assessmentEvents, repo.assessmentOutbox})
	for _, protected := range []string{"security@vendor.example", "one-time-token"} {
		if strings.Contains(string(stored), protected) {
			t.Fatalf("protected value persisted: %s", stored)
		}
	}
}

func TestRequestAssessmentClarificationRecoversPreparedRequestWithoutDuplication(t *testing.T) {
	assessmentService, repo, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	assessment := assessmentUnderReviewFixture(t, assessmentService, repo, relationship)
	evidenceStub := &assessmentEvidenceStub{repo: repo, assessmentID: assessment.ID, issueErr: errors.New("provider unavailable")}
	service, _ := NewAssessmentRequestService(assessmentService, repo, evidenceStub, assessmentFormReaderStub{form: activeAssessmentForm()}, nil, "https://capture.example.test/respond", "production")
	input := RequestAssessmentClarificationInput{ExpectedVersion: assessment.Version, RequestFields: []string{"contact_email"}, Message: "Provide the current security contact.", Audience: "security@vendor.example", Deadline: assessment.ReviewDueAt.Add(-time.Hour), InvitationTTLMinutes: 60}
	first, err := service.RequestClarification(assessmentContext(), assessmentActor(), assessment.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	if first.State != SendRequestReadyInvitationNotIssued || first.Assessment.Status != AssessmentUnderReview {
		t.Fatalf("prepared outcome = %#v", first)
	}
	evidenceStub.issueErr = nil
	input.ExpectedVersion = first.Assessment.Version
	second, err := service.RequestClarification(assessmentContext(), assessmentActor(), assessment.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	if len(evidenceStub.created) != 1 || second.Assessment.Status != AssessmentCollecting {
		t.Fatalf("recovery duplicated request: %#v %#v", evidenceStub.created, second)
	}
}

func TestRequestAssessmentClarificationRequiresCurrentReviewerStateVersionAndFields(t *testing.T) {
	guard := newAssessmentGuard()
	assessmentService, repo, relationship := newAssessmentServiceFixture(t, guard)
	assessment := assessmentUnderReviewFixture(t, assessmentService, repo, relationship)
	evidenceStub := &assessmentEvidenceStub{repo: repo, assessmentID: assessment.ID}
	service, _ := NewAssessmentRequestService(assessmentService, repo, evidenceStub, assessmentFormReaderStub{form: activeAssessmentForm()}, nil, "https://capture.example.test/respond", "production")
	base := RequestAssessmentClarificationInput{ExpectedVersion: assessment.Version, RequestFields: []string{"contact_email"}, Message: "Provide the current security contact.", Audience: "security@vendor.example", Deadline: assessment.ReviewDueAt.Add(-time.Hour), InvitationTTLMinutes: 60}
	stale := base
	stale.ExpectedVersion--
	if _, err := service.RequestClarification(assessmentContext(), assessmentActor(), assessment.ID, stale); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale error = %v", err)
	}
	badField := base
	badField.RequestFields = []string{"unknown_field"}
	if _, err := service.RequestClarification(assessmentContext(), assessmentActor(), assessment.ID, badField); !errors.Is(err, ErrInvalid) {
		t.Fatalf("field error = %v", err)
	}
	repo.assessmentMu.Lock()
	current := repo.assessments[assessment.ID]
	current.Status = AssessmentCompleted
	repo.assessments[assessment.ID] = current
	repo.assessmentMu.Unlock()
	if _, err := service.RequestClarification(assessmentContext(), assessmentActor(), assessment.ID, base); !errors.Is(err, ErrInvalidAssessmentTransition) {
		t.Fatalf("state error = %v", err)
	}
	if len(evidenceStub.created) != 0 || len(evidenceStub.issued) != 0 {
		t.Fatal("ineligible clarification created external access")
	}
	guard.requests = nil
	repo.assessmentMu.Lock()
	current.Status = AssessmentUnderReview
	repo.assessments[assessment.ID] = current
	repo.assessmentMu.Unlock()
	_, _ = service.RequestClarification(assessmentContext(), assessmentActor(), assessment.ID, base)
	if len(guard.requests) < 1 || guard.requests[0].DecisionType != AssessmentClarificationCommand || guard.requests[0].Responsibility != authority.ResponsibilityReviewer {
		t.Fatalf("authority route = %#v", guard.requests)
	}
}

func TestConcurrentClarificationPreparationAllowsOneInvitationPath(t *testing.T) {
	baseService, repo, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	assessment := assessmentUnderReviewFixture(t, baseService, repo, relationship)
	barrier := &clarificationPrepareBarrier{AssessmentRepository: repo, entered: make(chan struct{}, 2), release: make(chan struct{})}
	assessmentService := NewAssessmentService(barrier, fixedAssessmentGuard{})
	assessmentService.now = baseService.now
	evidenceService := evidence.NewService(evidence.NewMemoryRepository(nil, nil), evidence.NewMemoryObjectStore())
	form := activeAssessmentForm()
	requested := map[string]struct{}{"contact_email": {}}
	fields, sections, err := clarificationForm(form, requested)
	if err != nil {
		t.Fatal(err)
	}
	deadline := assessment.ReviewDueAt.Add(-time.Hour)
	_, err = evidenceService.CreateRequest(context.Background(), clarificationEvidenceRequestInput(assessmentActor(), assessment, relationship, form, fields, sections, evidence.RequestOrigin{Type: AssessmentRequestOrigin, ID: assessment.ID, Version: 2}, "security@vendor.example", "Provide the current security contact.", deadline))
	if err != nil {
		t.Fatal(err)
	}
	service, _ := NewAssessmentRequestService(assessmentService, barrier, evidenceService, assessmentFormReaderStub{form: form}, nil, "https://capture.example.test/respond", "production")
	input := RequestAssessmentClarificationInput{ExpectedVersion: assessment.Version, RequestFields: []string{"contact_email"}, Message: "Provide the current security contact.", Audience: "security@vendor.example", Deadline: deadline, InvitationTTLMinutes: 60}
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			_, callErr := service.RequestClarification(assessmentContext(), assessmentActor(), assessment.ID, input)
			errs <- callErr
		}()
	}
	<-barrier.entered
	<-barrier.entered
	close(barrier.release)
	first, second := <-errs, <-errs
	conflicts := 0
	successes := 0
	for _, callErr := range []error{first, second} {
		if callErr == nil {
			successes++
		} else if errors.Is(callErr, ErrVersionConflict) {
			conflicts++
		} else {
			t.Fatalf("concurrent clarification error = %v", callErr)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("concurrent results successes=%d conflicts=%d", successes, conflicts)
	}
	links, _ := repo.ListAssessmentRequestLinks(context.Background(), scopeFromVerified(), assessment.ID)
	if len(links) != 2 || links[1].InvitationID == "" {
		t.Fatalf("concurrent clarification links = %#v", links)
	}
}

func assessmentUnderReviewFixture(t *testing.T, service *AssessmentService, repo *MemoryAssessmentRepository, relationship Aggregate) Assessment {
	t.Helper()
	assessment := mustStartAssessment(t, service, relationship)
	repo.assessmentMu.Lock()
	assessment.Status, assessment.Version = AssessmentUnderReview, 5
	assessment.CurrentRequestID, assessment.SubmissionID, assessment.ReviewMatterID = "request-1", "submission-1", "matter-review"
	assessment.ReviewerPrincipalID = "verified-owner"
	repo.assessments[assessment.ID] = assessment
	repo.requestLinks[assessment.ID] = []AssessmentRequestLink{{TenantID: assessment.TenantID, LegalEntityID: assessment.LegalEntityID, AssessmentID: assessment.ID, RequestID: "request-1", Purpose: AssessmentRequestInitial, Sequence: 1, OriginType: AssessmentRequestOrigin, OriginID: assessment.ID, OriginSequence: 1, InvitationID: "invitation-old", CreatedAt: assessment.CreatedAt}}
	repo.assessmentMu.Unlock()
	return assessment
}
