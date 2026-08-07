package httpapi

import (
	"context"
	"fmt"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

func (a *API) lifecycleCommandPolicy(ctx context.Context, tenant, name string, payload map[string]any, policy commandPolicy) (commandPolicy, error) {
	switch name {
	case "matter.transition":
		matterID := stringValue(payload["id"])
		if matterID == "" {
			matterID = stringValue(payload["matter_id"])
		}
		if matterID == "" {
			return policy, nil
		}
		aggregate, err := a.deps.Continuity.GetMatter(ctx, tenant, matterID)
		if err != nil {
			return policy, err
		}
		target := continuity.MatterStatus(strings.ToUpper(stringValue(payload["to"])))
		if target == continuity.MatterDecisionRequired || target == continuity.MatterClosed || target == continuity.MatterCancelled || aggregate.Matter.Status == continuity.MatterClosed {
			policy.Responsibility = authority.ResponsibilityAuthorizer
			policy.Materiality = max(policy.Materiality, 4)
		}
		return policy, nil

	case "matter.decision.record":
		matterID := stringValue(payload["matter_id"])
		if matterID == "" {
			return policy, nil
		}
		aggregate, err := a.deps.Continuity.GetMatter(ctx, tenant, matterID)
		if err != nil {
			return policy, err
		}
		decisionType := stringValue(payload["type"])
		target := continuity.DecisionStatus(strings.ToUpper(stringValue(payload["status"])))
		if err := continuity.ValidateDecisionLifecycle(aggregate.Decisions, decisionType, target); err != nil {
			return policy, err
		}
		policy.ActorField = "authority_principal_id"
		switch target {
		case continuity.DecisionProposed:
			policy.Responsibility = authority.ResponsibilityProposer
			policy.Materiality = max(policy.Materiality, 2)
		case continuity.DecisionInReview, continuity.DecisionReturned:
			policy.Responsibility = authority.ResponsibilityReviewer
			policy.Materiality = max(policy.Materiality, 3)
		case continuity.DecisionChallenged:
			policy.Responsibility = authority.ResponsibilityChallenger
			policy.Materiality = max(policy.Materiality, 3)
		case continuity.DecisionApproved, continuity.DecisionConditionallyApproved, continuity.DecisionRejected, continuity.DecisionExpired, continuity.DecisionSuperseded:
			policy.Responsibility = authority.ResponsibilityAuthorizer
			policy.Materiality = max(policy.Materiality, 4)
		default:
			return policy, fmt.Errorf("%w: unsupported decision lifecycle target %s", continuity.ErrInvalidState, target)
		}
		return policy, nil

	case "matter.response.transition":
		matterID := stringValue(payload["matter_id"])
		responseID := stringValue(payload["response_id"])
		if matterID == "" || responseID == "" {
			return policy, nil
		}
		aggregate, err := a.deps.Continuity.GetMatter(ctx, tenant, matterID)
		if err != nil {
			return policy, err
		}
		var current *continuity.ResponsePackage
		for index := range aggregate.ResponsePackages {
			if aggregate.ResponsePackages[index].ID == responseID {
				value := aggregate.ResponsePackages[index]
				current = &value
				break
			}
		}
		if current == nil {
			return policy, continuity.ErrNotFound
		}
		target := continuity.ResponseStatus(strings.ToUpper(stringValue(payload["to"])))
		responsibility, materiality, err := responseLifecyclePolicy(current.Status, target)
		if err != nil {
			return policy, err
		}
		policy.Responsibility = responsibility
		policy.Materiality = max(policy.Materiality, materiality)
		return policy, nil
	default:
		return policy, nil
	}
}

func responseLifecyclePolicy(from, target continuity.ResponseStatus) (authority.Responsibility, int, error) {
	allowed := map[continuity.ResponseStatus]map[continuity.ResponseStatus]bool{
		continuity.ResponseDraft: {continuity.ResponseInReview: true, continuity.ResponseWithdrawn: true},
		continuity.ResponseInReview: {continuity.ResponseApproved: true, continuity.ResponseRejected: true, continuity.ResponseDraft: true, continuity.ResponseWithdrawn: true},
		continuity.ResponseApproved: {continuity.ResponseTransmitted: true, continuity.ResponseWithdrawn: true},
		continuity.ResponseTransmitted: {continuity.ResponseAcknowledged: true},
		continuity.ResponseRejected: {continuity.ResponseDraft: true},
	}
	if !allowed[from][target] {
		return "", 0, fmt.Errorf("%w: response cannot move from %s to %s", continuity.ErrInvalidState, from, target)
	}
	switch target {
	case continuity.ResponseInReview, continuity.ResponseRejected:
		return authority.ResponsibilityReviewer, 3, nil
	case continuity.ResponseApproved:
		return authority.ResponsibilitySignatory, 4, nil
	case continuity.ResponseTransmitted:
		return authority.ResponsibilityTransmitter, 4, nil
	case continuity.ResponseAcknowledged:
		return authority.ResponsibilityAcknowledger, 3, nil
	case continuity.ResponseDraft:
		if from == continuity.ResponseRejected {
			return authority.ResponsibilityProposer, 2, nil
		}
		return authority.ResponsibilityReviewer, 3, nil
	case continuity.ResponseWithdrawn:
		if from == continuity.ResponseApproved {
			return authority.ResponsibilitySignatory, 4, nil
		}
		return authority.ResponsibilityProposer, 2, nil
	default:
		return "", 0, fmt.Errorf("%w: unsupported response lifecycle target %s", continuity.ErrInvalidState, target)
	}
}
