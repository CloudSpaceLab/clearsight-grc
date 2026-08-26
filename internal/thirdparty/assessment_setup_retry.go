package thirdparty

import (
	"context"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
)

func (s *AssessmentService) RetryAssessmentSetup(ctx context.Context, _ Actor, assessmentID string, input RetryAssessmentSetupInput) (AssessmentSetupRetryOutcome, error) {
	assessmentID = strings.TrimSpace(assessmentID)
	if !validAssessmentIdentifier(assessmentID) || input.ExpectedVersion < 1 {
		return AssessmentSetupRetryOutcome{}, ErrInvalid
	}
	actor, err := s.authorize(ctx, assessmentID, assessmentObjectType, AssessmentSetupRetryCommand, authority.ResponsibilityOwner)
	if err != nil {
		return AssessmentSetupRetryOutcome{}, err
	}
	job, assessment, err := s.repo.RequeueAssessmentSetup(ctx, RequeueAssessmentSetupRecord{
		Scope: scopeFrom(actor), AssessmentID: assessmentID, ExpectedVersion: input.ExpectedVersion,
		ActorPrincipalID: actor.PrincipalID, QueuedAt: s.now().UTC(),
	})
	if err != nil {
		return AssessmentSetupRetryOutcome{}, err
	}
	return AssessmentSetupRetryOutcome{Assessment: assessment, Setup: assessmentSetupStatus(job)}, nil
}

func assessmentSetupStatus(job AssessmentSetupJob) AssessmentSetupStatus {
	status := AssessmentSetupStatus{
		AssessmentID: job.AssessmentID, State: job.State, Attempts: job.Attempts,
		LeaseUntil: cloneAssessmentTime(job.LeaseExpiresAt), FailureCode: job.LastFailureCode, UpdatedAt: job.UpdatedAt,
	}
	switch job.State {
	case AssessmentJobReady:
		value := job.AvailableAt
		status.NextAttemptAt = &value
	case AssessmentJobFailed:
		value := job.UpdatedAt
		status.TerminalAt = &value
	}
	return status
}
