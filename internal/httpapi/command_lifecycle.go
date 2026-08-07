package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

// lifecycleCommandPolicy resolves authority from the route-bound object and its
// current lifecycle state. Route identifiers are canonical: redundant body IDs
// may match them, but may never redirect authority evaluation to another object.
func (a *API) lifecycleCommandPolicy(ctx context.Context, r *http.Request, tenant, name string, payload map[string]any, policy commandPolicy) (commandPolicy, error) {
	matterID, err := lifecycleMatterID(r, payload)
	if err != nil {
		return policy, err
	}

	var aggregate *continuity.MatterAggregate
	matterPriority := 0
	if existingMatterCommand(name) && matterID != "" {
		if a.deps.Continuity == nil {
			return policy, fmt.Errorf("continuity service is unavailable")
		}
		current, loadErr := a.deps.Continuity.GetMatter(ctx, tenant, matterID)
		if loadErr != nil {
			return policy, loadErr
		}
		aggregate = &current
		matterPriority = current.Matter.Priority
	}

	switch name {
	case "matter.transition":
		if aggregate == nil {
			return policy, nil
		}
		target := continuity.MatterStatus(strings.ToUpper(stringValue(payload["to"])))
		if target == continuity.MatterDecisionRequired || target == continuity.MatterClosed || target == continuity.MatterCancelled || aggregate.Matter.Status == continuity.MatterClosed {
			policy.Responsibility = authority.ResponsibilityAuthorizer
			policy.Materiality = max(4, matterPriority)
		} else {
			policy.Materiality = max(policy.Materiality, matterPriority)
		}
		return policy, nil

	case "matter.decision.record":
		if aggregate == nil {
			return policy, nil
		}
		decisionType := stringValue(payload["type"])
		target := continuity.DecisionStatus(strings.ToUpper(stringValue(payload["status"])))
		if err := continuity.ValidateDecisionLifecycle(aggregate.Decisions, decisionType, target); err != nil {
			return policy, err
		}
		responsibility, materiality, err := decisionLifecyclePolicy(target)
		if err != nil {
			return policy, err
		}
		policy.ActorField = "authority_principal_id"
		policy.Responsibility = responsibility
		policy.Materiality = max(materiality, matterPriority)
		return policy, nil

	case "matter.action.add":
		if aggregate != nil {
			policy.Materiality = max(policy.Materiality, matterPriority)
			ownerID := stringValue(payload["owner_principal_id"])
			if ownerID != "" && !continuity.MatterVisibleTo(aggregate.Matter, ownerID) {
				return policy, fmt.Errorf("%w: action owner is not permitted to view this issue", continuity.ErrInvalidState)
			}
		}
		return policy, nil

	case "matter.action.transition":
		if _, err := lifecycleSubresourceID(r, payload, "action_id"); err != nil {
			return policy, err
		}
		policy.Materiality = max(policy.Materiality, matterPriority)
		return policy, nil

	case "matter.response.add":
		policy.Responsibility = authority.ResponsibilityProposer
		policy.Materiality = max(2, matterPriority)
		return policy, nil

	case "matter.response.transition":
		if aggregate == nil {
			return policy, nil
		}
		responseID, err := lifecycleSubresourceID(r, payload, "response_id")
		if err != nil {
			return policy, err
		}
		if responseID == "" {
			return policy, nil
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
		policy.Materiality = max(materiality, matterPriority)
		return policy, nil

	default:
		if aggregate != nil {
			policy.Materiality = max(policy.Materiality, matterPriority)
		}
		return policy, nil
	}
}

func existingMatterCommand(name string) bool {
	return strings.HasPrefix(name, "matter.") && name != "matter.create"
}

func lifecycleMatterID(r *http.Request, payload map[string]any) (string, error) {
	pathID := ""
	if r != nil {
		pathID = strings.TrimSpace(r.PathValue("id"))
	}
	return boundLifecycleID(pathID, stringValue(payload["matter_id"]), "matter_id")
}

func lifecycleSubresourceID(r *http.Request, payload map[string]any, field string) (string, error) {
	pathID := ""
	if r != nil {
		pathID = strings.TrimSpace(r.PathValue(field))
	}
	return boundLifecycleID(pathID, stringValue(payload[field]), field)
}

func boundLifecycleID(routeID, bodyID, field string) (string, error) {
	routeID = strings.TrimSpace(routeID)
	bodyID = strings.TrimSpace(bodyID)
	if routeID != "" && bodyID != "" && routeID != bodyID {
		return "", fmt.Errorf("%w: %s conflicts with the route identifier", continuity.ErrInvalidState, field)
	}
	if routeID != "" {
		return routeID, nil
	}
	return bodyID, nil
}

func decisionLifecyclePolicy(target continuity.DecisionStatus) (authority.Responsibility, int, error) {
	switch target {
	case continuity.DecisionProposed:
		return authority.ResponsibilityProposer, 2, nil
	case continuity.DecisionInReview, continuity.DecisionReturned:
		return authority.ResponsibilityReviewer, 3, nil
	case continuity.DecisionChallenged:
		return authority.ResponsibilityChallenger, 3, nil
	case continuity.DecisionApproved, continuity.DecisionConditionallyApproved, continuity.DecisionRejected, continuity.DecisionExpired, continuity.DecisionSuperseded:
		return authority.ResponsibilityAuthorizer, 4, nil
	default:
		return "", 0, fmt.Errorf("%w: unsupported decision lifecycle target %s", continuity.ErrInvalidState, target)
	}
}

func responseLifecyclePolicy(from, target continuity.ResponseStatus) (authority.Responsibility, int, error) {
	allowed := map[continuity.ResponseStatus]map[continuity.ResponseStatus]bool{
		continuity.ResponseDraft:       {continuity.ResponseInReview: true, continuity.ResponseWithdrawn: true},
		continuity.ResponseInReview:    {continuity.ResponseApproved: true, continuity.ResponseRejected: true, continuity.ResponseDraft: true, continuity.ResponseWithdrawn: true},
		continuity.ResponseApproved:    {continuity.ResponseTransmitted: true, continuity.ResponseWithdrawn: true},
		continuity.ResponseTransmitted: {continuity.ResponseAcknowledged: true},
		continuity.ResponseRejected:    {continuity.ResponseDraft: true},
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
