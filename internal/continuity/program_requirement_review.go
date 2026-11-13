package continuity

import (
	"strings"
	"time"
)

// RequirementReview is the proposed independent decision for one draft
// Program requirement. It is not an approved requirement or a compliance result.
type RequirementReview struct {
	ID                string     `json:"id"`
	TenantID          string     `json:"tenant_id"`
	ProgramID         string     `json:"program_id"`
	RequirementID     string     `json:"requirement_id"`
	ProposedBy        string     `json:"proposed_by"`
	ProposalRationale string     `json:"proposal_rationale"`
	ProposedAt        time.Time  `json:"proposed_at"`
	ReviewedBy        string     `json:"reviewed_by,omitempty"`
	ReviewRationale   string     `json:"review_rationale,omitempty"`
	ReviewedAt        *time.Time `json:"reviewed_at,omitempty"`
	Status            string     `json:"status"`
	Version           int64      `json:"version"`
}

// prepareRequirementReviewDecision is a pure transition. The command layer
// must authenticate the actor, re-evaluate authority and persist the decision,
// Requirement status, event and outbox in the same transaction.
func prepareRequirementReviewDecision(program Program, requirement Requirement, review RequirementReview, checker, rationale string, approved bool, now time.Time) (RequirementReview, error) {
	checker, rationale = strings.TrimSpace(checker), strings.TrimSpace(rationale)
	if program.Status == ProgramRetired || checker == "" || checker != program.AuthorityPrincipalID || checker == review.ProposedBy ||
		review.Status != "PENDING" || review.Version < 1 || review.ID == "" || review.ProposedBy == "" || review.ProposedAt.IsZero() ||
		review.TenantID != program.TenantID || review.ProgramID != program.ID ||
		requirement.ID != review.RequirementID || requirement.ProgramID != program.ID ||
		requirement.TenantID != program.TenantID || requirement.Status != RequirementDraft ||
		rationale == "" || len(rationale) > 2000 || now.Before(review.ProposedAt) {
		return RequirementReview{}, ErrInvalidState
	}
	value := review
	value.ReviewedBy, value.ReviewRationale = checker, rationale
	at := now.UTC()
	value.ReviewedAt = &at
	value.Status = "REJECTED"
	if approved {
		value.Status = "APPROVED"
	}
	value.Version++
	return value, nil
}
