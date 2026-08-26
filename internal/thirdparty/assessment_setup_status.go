package thirdparty

import (
	"context"
	"strings"
)

func (s *AssessmentService) GetAssessmentSetupStatus(ctx context.Context, actor Actor, assessmentID string) (AssessmentSetupStatus, error) {
	assessmentID = strings.TrimSpace(assessmentID)
	if !validActor(actor) || !validAssessmentIdentifier(assessmentID) {
		return AssessmentSetupStatus{}, ErrInvalid
	}
	repository, ok := s.repo.(assessmentSetupStatusRepository)
	if !ok {
		return AssessmentSetupStatus{}, ErrAssessmentSetupStatusUnavailable
	}
	job, err := repository.GetAssessmentSetupJob(ctx, scopeFrom(actor), assessmentID)
	if err != nil {
		return AssessmentSetupStatus{}, err
	}
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
	return status, nil
}
