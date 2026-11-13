package continuity

import (
	"context"
	"strings"

	platformid "github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

const (
	EventRequirementReviewProposed = "REQUIREMENT_REVIEW_PROPOSED"
	EventRequirementReviewApproved = "REQUIREMENT_REVIEW_APPROVED"
	EventRequirementReviewRejected = "REQUIREMENT_REVIEW_REJECTED"
)

type requirementReviewReader interface {
	GetRequirementReview(context.Context, string, string, string) (RequirementReview, error)
}
type ProposeRequirementReviewInput struct {
	TenantID        string `json:"tenant_id"`
	ProgramID       string `json:"program_id"`
	RequirementID   string `json:"requirement_id"`
	ExpectedVersion int64  `json:"expected_version"`
	ActorID         string `json:"actor_id"`
	Rationale       string `json:"rationale"`
}
type DecideRequirementReviewInput struct {
	TenantID        string `json:"tenant_id"`
	ProgramID       string `json:"program_id"`
	ReviewID        string `json:"review_id"`
	ExpectedVersion int64  `json:"expected_version"`
	ActorID         string `json:"actor_id"`
	Rationale       string `json:"rationale"`
}
type requirementReviewDecisionEvent struct {
	Review      RequirementReview `json:"review"`
	Requirement Requirement       `json:"requirement"`
}

// ProposeRequirementReview sends an existing draft for independent review.
func (s *Service) ProposeRequirementReview(ctx context.Context, in ProposeRequirementReviewInput) (RequirementReview, ProgramAggregate, error) {
	aggregate, err := s.programForMutation(ctx, in.TenantID, in.ProgramID, in.ExpectedVersion)
	if err != nil {
		return RequirementReview{}, ProgramAggregate{}, err
	}
	maker, reason := strings.TrimSpace(in.ActorID), strings.TrimSpace(in.Rationale)
	if !programComplianceActorMatches(ctx, in.TenantID, maker) || maker != aggregate.Program.OwnerPrincipalID || aggregate.Program.Status == ProgramRetired || reason == "" || len(reason) > 2000 {
		return RequirementReview{}, ProgramAggregate{}, ErrInvalidState
	}
	if _, ok := s.repo.(requirementReviewReader); !ok {
		return RequirementReview{}, ProgramAggregate{}, ErrInvalidState
	}
	found := false
	for _, req := range aggregate.Requirements {
		if req.ID == in.RequirementID && req.Status == RequirementDraft {
			found = true
			break
		}
	}
	if !found {
		return RequirementReview{}, ProgramAggregate{}, ErrInvalidState
	}
	id, err := platformid.NewUUIDv7()
	if err != nil {
		return RequirementReview{}, ProgramAggregate{}, err
	}
	review := RequirementReview{ID: id, TenantID: aggregate.Program.TenantID, ProgramID: aggregate.Program.ID, RequirementID: in.RequirementID, ProposedBy: maker, ProposalRationale: reason, ProposedAt: s.now().UTC(), Status: "PENDING", Version: 1}
	result, err := s.applyProgramValueAndResult(ctx, aggregate, in.TenantID, in.ProgramID, in.ExpectedVersion, EventRequirementReviewProposed, review, maker, id)
	if err != nil {
		return RequirementReview{}, ProgramAggregate{}, err
	}
	return review, result, nil
}

// DecideRequirementReview records an independent decision for the exact draft.
// Identity and current operation authority must also be enforced by the caller.
func (s *Service) DecideRequirementReview(ctx context.Context, in DecideRequirementReviewInput, approve bool) (RequirementReview, ProgramAggregate, error) {
	aggregate, err := s.programForMutation(ctx, in.TenantID, in.ProgramID, in.ExpectedVersion)
	if err != nil {
		return RequirementReview{}, ProgramAggregate{}, err
	}
	checker := strings.TrimSpace(in.ActorID)
	if !programComplianceActorMatches(ctx, in.TenantID, checker) {
		return RequirementReview{}, ProgramAggregate{}, ErrInvalidState
	}
	reader, ok := s.repo.(requirementReviewReader)
	if !ok {
		return RequirementReview{}, ProgramAggregate{}, ErrInvalidState
	}
	current, err := reader.GetRequirementReview(ctx, in.TenantID, in.ProgramID, in.ReviewID)
	if err != nil {
		return RequirementReview{}, ProgramAggregate{}, err
	}
	var requirement Requirement
	for _, r := range aggregate.Requirements {
		if r.ID == current.RequirementID {
			requirement = r
			break
		}
	}
	now := s.now().UTC()
	decision, err := prepareRequirementReviewDecision(aggregate.Program, requirement, current, checker, in.Rationale, approve, now)
	if err != nil {
		return RequirementReview{}, ProgramAggregate{}, err
	}
	event := EventRequirementReviewRejected
	if approve {
		event = EventRequirementReviewApproved
		requirement.Status = RequirementApproved
		requirement.Version++
		if requirement.EffectiveFrom.Before(now) {
			requirement.EffectiveFrom = now
		}
	}
	result, err := s.applyProgramValueAndResult(ctx, aggregate, in.TenantID, in.ProgramID, in.ExpectedVersion, event, requirementReviewDecisionEvent{Review: decision, Requirement: requirement}, checker, current.ID)
	if err != nil {
		return RequirementReview{}, ProgramAggregate{}, err
	}
	return decision, result, nil
}

type requirementReviewListReader interface {
	ListRequirementReviews(context.Context, string, string, string, int) ([]RequirementReview, error)
}

func (s *Service) ListRequirementReviews(ctx context.Context, tenant, programID, after string, limit int) ([]RequirementReview, error) {
	if _, err := s.GetProgram(ctx, tenant, programID); err != nil {
		return nil, err
	}
	reader, ok := s.repo.(requirementReviewListReader)
	if !ok {
		return nil, ErrInvalidState
	}
	return reader.ListRequirementReviews(ctx, tenant, programID, after, limit)
}
