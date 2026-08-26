package thirdparty

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

func TestRetryAssessmentSetupRequeuesSameTerminalJobWithVerifiedOwnerAudit(t *testing.T) {
	service, repository, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	assessment, err := service.StartAssessment(assessmentContext(), Actor{TenantID: "forged", LegalEntityID: "forged", PrincipalID: "forged"}, relationship.Relationship.ID, validStartAssessmentInput(relationship.Relationship.Version))
	if err != nil {
		t.Fatal(err)
	}
	terminal := terminalizeAssessmentSetupJob(t, repository, assessment, AssessmentSetupFailureMatter)
	retryAt := assessment.CreatedAt.Add(5 * time.Minute)
	service.now = func() time.Time { return retryAt }
	eventsBefore, outboxBefore := len(repository.assessmentEvents), len(repository.assessmentOutbox)

	outcome, err := service.RetryAssessmentSetup(assessmentContext(), Actor{TenantID: "forged", LegalEntityID: "forged", PrincipalID: "forged"}, assessment.ID, RetryAssessmentSetupInput{ExpectedVersion: assessment.Version})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Assessment.ID != assessment.ID || outcome.Assessment.Status != AssessmentSetupPending || outcome.Assessment.Version != assessment.Version+1 {
		t.Fatalf("retried assessment = %#v", outcome.Assessment)
	}
	if outcome.Setup.AssessmentID != assessment.ID || outcome.Setup.State != AssessmentJobReady || outcome.Setup.Attempts != 0 || outcome.Setup.FailureCode != "" || outcome.Setup.NextAttemptAt == nil || !outcome.Setup.NextAttemptAt.Equal(retryAt) {
		t.Fatalf("retried setup = %#v", outcome.Setup)
	}
	jobs, err := repository.ListAssessmentSetupJobs(context.Background(), scopeFromVerified(), assessment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 || jobs[0].ID != terminal.ID || jobs[0].DedupeKey != terminal.DedupeKey {
		t.Fatalf("setup job was duplicated or replaced: %#v", jobs)
	}
	if len(repository.assessmentEvents) != eventsBefore+1 || len(repository.assessmentOutbox) != outboxBefore+1 {
		t.Fatalf("retry audit counts = events %d outbox %d", len(repository.assessmentEvents), len(repository.assessmentOutbox))
	}
	event := repository.assessmentEvents[len(repository.assessmentEvents)-1]
	if event.Type != "AssessmentSetupRetryQueued" || event.ActorPrincipalID != "verified-owner" || event.AssessmentVersion != outcome.Assessment.Version || event.Payload["setup_job_id"] != terminal.ID || event.Payload["previous_failure_code"] != AssessmentSetupFailureMatter {
		t.Fatalf("unsafe or incomplete retry event = %#v", event)
	}
	outbox := repository.assessmentOutbox[len(repository.assessmentOutbox)-1]
	if outbox.ActorPrincipalID != "" || outbox.Payload["setup_job_id"] != terminal.ID || outbox.Payload["previous_failure_code"] != AssessmentSetupFailureMatter {
		t.Fatalf("unsafe or incomplete retry outbox = %#v", outbox)
	}
}

func TestRetryAssessmentSetupReplayAndConcurrencyDoNotDuplicateMutation(t *testing.T) {
	service, repository, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	assessment, err := service.StartAssessment(assessmentContext(), assessmentActor(), relationship.Relationship.ID, validStartAssessmentInput(relationship.Relationship.Version))
	if err != nil {
		t.Fatal(err)
	}
	terminalizeAssessmentSetupJob(t, repository, assessment, AssessmentSetupFailureAttemptsExhausted)
	service.now = func() time.Time { return assessment.CreatedAt.Add(5 * time.Minute) }
	input := RetryAssessmentSetupInput{ExpectedVersion: assessment.Version}

	const callers = 8
	results := make(chan AssessmentSetupRetryOutcome, callers)
	errorsSeen := make(chan error, callers)
	var wait sync.WaitGroup
	for index := 0; index < callers; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			value, callErr := service.RetryAssessmentSetup(assessmentContext(), assessmentActor(), assessment.ID, input)
			results <- value
			errorsSeen <- callErr
		}()
	}
	wait.Wait()
	close(results)
	close(errorsSeen)
	for callErr := range errorsSeen {
		if callErr != nil {
			t.Fatalf("concurrent retry error = %v", callErr)
		}
	}
	for value := range results {
		if value.Assessment.Version != assessment.Version+1 || value.Setup.State != AssessmentJobReady {
			t.Fatalf("concurrent retry outcome = %#v", value)
		}
	}
	queued := 0
	for _, event := range repository.assessmentEvents {
		if event.AssessmentID == assessment.ID && event.Type == "AssessmentSetupRetryQueued" {
			queued++
		}
	}
	if queued != 1 {
		t.Fatalf("retry events = %d", queued)
	}
	jobs, err := repository.ListAssessmentSetupJobs(context.Background(), scopeFromVerified(), assessment.ID)
	if err != nil || len(jobs) != 1 {
		t.Fatalf("setup jobs = (%#v, %v)", jobs, err)
	}
}

func TestRetryAssessmentSetupRejectsIneligibleStateScopeVersionAndAuthority(t *testing.T) {
	service, repository, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	assessment, err := service.StartAssessment(assessmentContext(), assessmentActor(), relationship.Relationship.ID, validStartAssessmentInput(relationship.Relationship.Version))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.RetryAssessmentSetup(assessmentContext(), assessmentActor(), assessment.ID, RetryAssessmentSetupInput{ExpectedVersion: assessment.Version}); !errors.Is(err, ErrInvalidAssessmentTransition) {
		t.Fatalf("non-terminal retry error = %v", err)
	}
	terminalizeAssessmentSetupJob(t, repository, assessment, AssessmentSetupFailureMatter)
	if _, err := service.RetryAssessmentSetup(assessmentContext(), assessmentActor(), assessment.ID, RetryAssessmentSetupInput{ExpectedVersion: assessment.Version + 9}); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale retry error = %v", err)
	}
	wrongScope := assessmentContextFor("bank", "other-entity", "verified-owner")
	if _, err := service.RetryAssessmentSetup(wrongScope, assessmentActor(), assessment.ID, RetryAssessmentSetupInput{ExpectedVersion: assessment.Version}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-entity retry error = %v", err)
	}
	deniedGuard := newAssessmentGuard()
	deniedGuard.err = commandauth.ErrNotAuthorized
	denied := NewAssessmentService(repository, deniedGuard)
	denied.now = service.now
	if _, err := denied.RetryAssessmentSetup(assessmentContext(), assessmentActor(), assessment.ID, RetryAssessmentSetupInput{ExpectedVersion: assessment.Version}); !errors.Is(err, commandauth.ErrNotAuthorized) {
		t.Fatalf("denied retry error = %v", err)
	}
}

func TestRetryAssessmentSetupReusesCanonicalMatterOnWorkerRecovery(t *testing.T) {
	service, repository, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	assessment, err := service.StartAssessment(assessmentContext(), assessmentActor(), relationship.Relationship.ID, validStartAssessmentInput(relationship.Relationship.Version))
	if err != nil {
		t.Fatal(err)
	}
	terminalizeAssessmentSetupJob(t, repository, assessment, AssessmentSetupFailureCompletion)
	retryAt := assessment.CreatedAt.Add(5 * time.Minute)
	service.now = func() time.Time { return retryAt }
	if _, err := service.RetryAssessmentSetup(assessmentContext(), assessmentActor(), assessment.ID, RetryAssessmentSetupInput{ExpectedVersion: assessment.Version}); err != nil {
		t.Fatal(err)
	}
	triggerKey := "thirdparty-assessment:" + assessment.ID
	matterService := &recordingAssessmentMatterService{matters: map[string]continuity.MatterAggregate{
		triggerKey: {Matter: continuity.Matter{ID: "matter-1", TenantID: assessment.TenantID, Type: continuity.MatterVendorReview, TriggerKey: triggerKey, Version: 1}},
	}}
	provisioner := NewAssessmentProvisioner(repository, matterService, "worker-recovery")
	if completed, err := provisioner.Maintain(context.Background(), retryAt, 1); err != nil || completed != 1 {
		t.Fatalf("recovered setup = (%d, %v)", completed, err)
	}
	current, err := repository.GetAssessment(context.Background(), scopeFromVerified(), assessment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Status != AssessmentReadyToSend || current.ReviewMatterID != "matter-1" || matterService.created != 0 || matterService.lookupCalls != 1 {
		t.Fatalf("canonical Matter was not reused: assessment=%#v service=%#v", current, matterService)
	}
	jobs, err := repository.ListAssessmentSetupJobs(context.Background(), scopeFromVerified(), assessment.ID)
	if err != nil || len(jobs) != 1 || jobs[0].State != AssessmentJobCompleted {
		t.Fatalf("recovered setup jobs = (%#v, %v)", jobs, err)
	}
}

func TestRetryAssessmentSetupUsesOwnerAuthorityRoute(t *testing.T) {
	guard := newAssessmentGuard()
	service, repository, relationship := newAssessmentServiceFixture(t, guard)
	assessment, err := service.StartAssessment(assessmentContext(), assessmentActor(), relationship.Relationship.ID, validStartAssessmentInput(relationship.Relationship.Version))
	if err != nil {
		t.Fatal(err)
	}
	terminalizeAssessmentSetupJob(t, repository, assessment, AssessmentSetupFailureMatter)
	guard.requests = nil
	if _, err := service.RetryAssessmentSetup(assessmentContext(), assessmentActor(), assessment.ID, RetryAssessmentSetupInput{ExpectedVersion: assessment.Version}); err != nil {
		t.Fatal(err)
	}
	if len(guard.requests) != 1 || guard.requests[0].DecisionType != AssessmentSetupRetryCommand || guard.requests[0].Responsibility != authority.ResponsibilityOwner || guard.requests[0].ObjectID != assessment.ID {
		t.Fatalf("retry authority request = %#v", guard.requests)
	}
}

func terminalizeAssessmentSetupJob(t *testing.T, repository *MemoryAssessmentRepository, assessment Assessment, failureCode string) AssessmentSetupJob {
	t.Helper()
	claimed, err := repository.ClaimAssessmentSetupJobs(context.Background(), "worker-terminal", assessment.CreatedAt, time.Minute, 1, 1)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim setup job = (%#v, %v)", claimed, err)
	}
	terminal, err := repository.FailAssessmentSetupJob(context.Background(), claimed[0], 1, failureCode, assessment.CreatedAt.Add(time.Second), assessment.CreatedAt.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if terminal.State != AssessmentJobFailed {
		t.Fatalf("terminal setup job = %#v", terminal)
	}
	return terminal
}
