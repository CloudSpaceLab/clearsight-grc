package continuity

import (
	"context"
	"fmt"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

// RecordDecisionLifecycle is the production lifecycle-aware decision command.
// AuthorityPrincipalID is the verified command actor at the HTTP boundary; it
// is normalized into the stage-specific actor field before persistence.
func (s *Service) RecordDecisionLifecycle(ctx context.Context, input AddDecisionInput) (MatterAggregate, error) {
	aggregate, err := s.matterForMutation(ctx, input.TenantID, input.MatterID, input.ExpectedVersion)
	if err != nil {
		return MatterAggregate{}, err
	}
	decisionType := strings.TrimSpace(input.Type)
	if decisionType == "" || strings.TrimSpace(input.Rationale) == "" {
		return MatterAggregate{}, fmt.Errorf("type and rationale are required")
	}
	if err := ValidateDecisionLifecycle(aggregate.Decisions, decisionType, input.Status); err != nil {
		return MatterAggregate{}, err
	}
	actorID := strings.TrimSpace(input.AuthorityPrincipalID)
	if actorID == "" {
		return MatterAggregate{}, fmt.Errorf("verified lifecycle actor is required")
	}
	switch input.Status {
	case DecisionProposed, DecisionInReview, DecisionChallenged, DecisionReturned:
	case DecisionApproved, DecisionConditionallyApproved, DecisionRejected, DecisionExpired, DecisionSuperseded:
		if strings.TrimSpace(input.SelectedOption) == "" && input.Status != DecisionExpired && input.Status != DecisionSuperseded {
			return MatterAggregate{}, fmt.Errorf("selected_option is required for an authority decision")
		}
	default:
		return MatterAggregate{}, fmt.Errorf("unsupported decision status")
	}
	options, err := normalizedJSON(input.Options, `[]`)
	if err != nil {
		return MatterAggregate{}, err
	}
	conditions, err := normalizedJSON(input.Conditions, `[]`)
	if err != nil {
		return MatterAggregate{}, err
	}
	valueID, err := id.NewUUIDv7()
	if err != nil {
		return MatterAggregate{}, err
	}
	now := s.now().UTC()
	value := Decision{
		ID: valueID, TenantID: input.TenantID, MatterID: input.MatterID,
		Type: decisionType, Status: input.Status, Options: options,
		SelectedOption: strings.TrimSpace(input.SelectedOption),
		Rationale:      strings.TrimSpace(input.Rationale), Conditions: conditions,
		ExpiresAt: input.ExpiresAt, CreatedAt: now, UpdatedAt: now, Version: 1,
	}
	setDecisionActor(&value, input.Status, actorID)
	if input.Status == DecisionApproved || input.Status == DecisionConditionallyApproved || input.Status == DecisionRejected || input.Status == DecisionExpired || input.Status == DecisionSuperseded {
		value.DecidedAt = &now
	}
	if err = s.applyMatterValue(ctx, input.TenantID, input.MatterID, input.ExpectedVersion, EventDecisionAdded, value, actorID); err != nil {
		return MatterAggregate{}, err
	}
	return s.GetMatter(ctx, input.TenantID, input.MatterID)
}
