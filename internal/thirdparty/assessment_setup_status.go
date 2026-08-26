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
	return assessmentSetupStatus(job), nil
}
