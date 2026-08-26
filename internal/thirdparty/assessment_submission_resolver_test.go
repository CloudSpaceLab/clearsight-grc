package thirdparty

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

func TestAssessmentSubmissionConsumerWithMemoryRepositoriesTransitionsOnce(t *testing.T) {
	assessments := submissionResolverMemoryFixture()
	requests := evidence.NewMemoryRepository(nil, []evidence.Request{{
		ID: "request-1", TenantID: "bank-a",
		Origin: evidence.RequestOrigin{Type: AssessmentRequestOrigin, ID: "assessment-1", Version: 1},
	}})
	inbox := workflowruntime.NewMemoryRepository()
	consumer := &AssessmentConsumer{
		Inbox: inbox, Requests: requests, Resolver: assessments,
		Reactions: NewAssessmentService(assessments, nil),
	}
	event := assessmentSubmissionEvent()

	if err := consumer.Publish(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if err := consumer.Publish(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	updated, err := assessments.GetAssessment(context.Background(), Scope{TenantID: "bank-a", LegalEntityID: "entity-a"}, "assessment-1")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != AssessmentSubmitted || updated.SubmissionID != "submission-1" || updated.Version != 5 {
		t.Fatalf("assessment = %#v", updated)
	}
	processed, err := inbox.InboxProcessed(context.Background(), "bank-a", assessmentSubmissionConsumerName, event.ID)
	if err != nil || !processed {
		t.Fatalf("inbox processed=%v err=%v", processed, err)
	}
}

func TestAssessmentSubmissionConsumerWithMemoryRepositoriesFailsClosedOnWrongScope(t *testing.T) {
	assessments := submissionResolverMemoryFixture()
	requests := evidence.NewMemoryRepository(nil, []evidence.Request{{
		ID: "request-1", TenantID: "bank-a",
		Origin: evidence.RequestOrigin{Type: AssessmentRequestOrigin, ID: "assessment-1", Version: 1},
	}})
	inbox := workflowruntime.NewMemoryRepository()
	consumer := &AssessmentConsumer{
		Inbox: inbox, Requests: requests, Resolver: assessments,
		Reactions: NewAssessmentService(assessments, nil),
	}
	event := assessmentSubmissionEvent()
	event.TenantID = "bank-b"

	if err := consumer.Publish(context.Background(), event); err == nil {
		t.Fatal("wrong-scope event was accepted")
	}
	unchanged, err := assessments.GetAssessment(context.Background(), Scope{TenantID: "bank-a", LegalEntityID: "entity-a"}, "assessment-1")
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Status != AssessmentCollecting || unchanged.Version != 4 {
		t.Fatalf("assessment changed = %#v", unchanged)
	}
	processed, err := inbox.InboxProcessed(context.Background(), "bank-b", assessmentSubmissionConsumerName, event.ID)
	if err != nil || processed {
		t.Fatalf("wrong-scope inbox processed=%v err=%v", processed, err)
	}
}

func TestMemoryAssessmentRepositoryResolvesExactCurrentAssessmentRequest(t *testing.T) {
	repository := submissionResolverMemoryFixture()

	target, err := repository.ResolveAssessmentRequest(context.Background(), "bank-a", evidence.RequestOrigin{
		Type: AssessmentRequestOrigin, ID: "assessment-1", Version: 1,
	}, "request-1")
	if err != nil {
		t.Fatal(err)
	}
	if target != (AssessmentSubmissionTarget{
		Scope:        Scope{TenantID: "bank-a", LegalEntityID: "entity-a"},
		AssessmentID: "assessment-1", AssessmentVersion: 4, RequestID: "request-1",
	}) {
		t.Fatalf("target = %#v", target)
	}
}

func TestMemoryAssessmentRepositoryRejectsMismatchedAssessmentRequestScope(t *testing.T) {
	repository := submissionResolverMemoryFixture()
	cases := []struct {
		name      string
		tenant    string
		origin    evidence.RequestOrigin
		requestID string
	}{
		{name: "wrong tenant", tenant: "bank-b", origin: evidence.RequestOrigin{Type: AssessmentRequestOrigin, ID: "assessment-1", Version: 1}, requestID: "request-1"},
		{name: "wrong origin type", tenant: "bank-a", origin: evidence.RequestOrigin{Type: "MONITORING_CHECK", ID: "assessment-1", Version: 1}, requestID: "request-1"},
		{name: "wrong assessment", tenant: "bank-a", origin: evidence.RequestOrigin{Type: AssessmentRequestOrigin, ID: "assessment-2", Version: 1}, requestID: "request-1"},
		{name: "wrong sequence", tenant: "bank-a", origin: evidence.RequestOrigin{Type: AssessmentRequestOrigin, ID: "assessment-1", Version: 2}, requestID: "request-1"},
		{name: "wrong request", tenant: "bank-a", origin: evidence.RequestOrigin{Type: AssessmentRequestOrigin, ID: "assessment-1", Version: 1}, requestID: "request-2"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := repository.ResolveAssessmentRequest(context.Background(), testCase.tenant, testCase.origin, testCase.requestID)
			if !errors.Is(err, ErrNotFound) {
				t.Fatalf("error = %v, want ErrNotFound", err)
			}
		})
	}
}

func TestMemoryAssessmentRepositoryRejectsNonCurrentOrNonCollectingRequest(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		mutate func(*MemoryAssessmentRepository)
	}{
		{name: "request no longer current", mutate: func(repository *MemoryAssessmentRepository) {
			assessment := repository.assessments["assessment-1"]
			assessment.CurrentRequestID = "request-2"
			repository.assessments[assessment.ID] = assessment
		}},
		{name: "assessment no longer collecting", mutate: func(repository *MemoryAssessmentRepository) {
			assessment := repository.assessments["assessment-1"]
			assessment.Status = AssessmentSubmitted
			repository.assessments[assessment.ID] = assessment
		}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			repository := submissionResolverMemoryFixture()
			testCase.mutate(repository)
			_, err := repository.ResolveAssessmentRequest(context.Background(), "bank-a", evidence.RequestOrigin{
				Type: AssessmentRequestOrigin, ID: "assessment-1", Version: 1,
			}, "request-1")
			if !errors.Is(err, ErrNotFound) {
				t.Fatalf("error = %v, want ErrNotFound", err)
			}
		})
	}
}

func submissionResolverMemoryFixture() *MemoryAssessmentRepository {
	repository := NewMemoryAssessmentRepository()
	now := time.Date(2026, 8, 26, 18, 0, 0, 0, time.UTC)
	repository.assessments["assessment-1"] = Assessment{
		ID: "assessment-1", TenantID: "bank-a", LegalEntityID: "entity-a", RelationshipID: "relationship-1",
		Status: AssessmentCollecting, CurrentRequestID: "request-1", Version: 4, UpdatedAt: now,
	}
	repository.requestLinks["assessment-1"] = []AssessmentRequestLink{{
		TenantID: "bank-a", LegalEntityID: "entity-a", AssessmentID: "assessment-1", RequestID: "request-1",
		Purpose: AssessmentRequestInitial, Sequence: 1, OriginType: AssessmentRequestOrigin, OriginID: "assessment-1", OriginSequence: 1,
	}}
	return repository
}
