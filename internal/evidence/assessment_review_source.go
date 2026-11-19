package evidence

import (
	"context"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func (s *Service) ProgramSubmissionReview(ctx context.Context, programID, reviewID string) (SubmissionReview, error) {
	actor, err := identity.Require(ctx)
	if err != nil {
		return SubmissionReview{}, ErrRecipientMismatch
	}
	repo, ok := s.repo.(interface {
		ProgramSubmissionReview(context.Context, identity.Actor, string, string) (SubmissionReview, error)
	})
	if !ok {
		return SubmissionReview{}, ErrNotFound
	}
	return repo.ProgramSubmissionReview(ctx, actor, programID, reviewID)
}
