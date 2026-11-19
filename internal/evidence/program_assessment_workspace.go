package evidence

import (
	"context"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

// ProgramAssessmentWorkspace is an actor-scoped read model over authoritative
// requests, immutable submissions and append-only reviews, not a compliance score.
type ProgramAssessmentWorkspace struct {
	RequirementReviews []RequirementReviewCoverage `json:"requirement_reviews,omitempty"`
	Items              []Request                   `json:"items"`
	Limited            bool                        `json:"limited"`
	AssessmentCount    int                         `json:"assessment_count"`
	ResponsesToReview  int                         `json:"responses_to_review"`
	DocumentsToReview  int                         `json:"documents_to_review"`
	Activities         []AssessmentActivity        `json:"activities"`
	ActivitiesLimited  bool                        `json:"activities_limited"`
}
type RequirementReviewCoverage struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Area    string `json:"area"`
	Outcome string `json:"outcome"`
}
type AssessmentActivity struct {
	ActorName string `json:"actor_name,omitempty"`
	ID        string `json:"id"`
	RequestID string `json:"request_id"`
	Title     string `json:"title"`
	Event     string `json:"event"`
	Actor     string `json:"actor"`
	At        string `json:"at"`
}

func (s *Service) ProgramAssessmentWorkspace(ctx context.Context, program string) (ProgramAssessmentWorkspace, error) {
	actor, err := identity.Require(ctx)
	if err != nil {
		return ProgramAssessmentWorkspace{}, ErrRecipientMismatch
	}
	repo, ok := s.repo.(interface {
		ProgramAssessmentWorkspace(context.Context, identity.Actor, string) (ProgramAssessmentWorkspace, error)
	})
	if !ok {
		return ProgramAssessmentWorkspace{}, ErrNotFound
	}
	value, err := repo.ProgramAssessmentWorkspace(ctx, actor, program)
	if err != nil {
		return value, err
	}
	for i, request := range value.Items {
		request, err = hydrateRequestRecipient(ctx, s.repo, request)
		if err != nil {
			return ProgramAssessmentWorkspace{}, err
		}
		request = effectiveRequest(request, s.now().UTC())
		if request.CreatedBy != actor.PrincipalID && RequestAssignedTo(request, actor.PrincipalID) {
			request = RespondentRequest(request)
		}
		value.Items[i] = request
	}
	return value, nil
}
