package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
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
		lifecycle, err := continuity.DecisionLifecyclePolicy(target)
		if err != nil {
			return policy, err
		}
		policy.ActorField = "authority_principal_id"
		policy.Responsibility = authority.Responsibility(lifecycle.Responsibility)
		policy.Materiality = max(lifecycle.Materiality, matterPriority)
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

	case "matter.assign":
		if aggregate == nil {
			return policy, nil
		}
		candidateID := stringValue(payload["owner_principal_id"])
		if err := a.validateMatterAssignmentCandidate(ctx, tenant, name, *aggregate, candidateID, authority.ResponsibilityOwner, max(policy.Materiality, matterPriority)); err != nil {
			return policy, err
		}
		policy.Materiality = max(policy.Materiality, matterPriority)
		return policy, nil

	case "matter.action.update":
		if aggregate == nil {
			return policy, nil
		}
		actionID, err := lifecycleSubresourceID(r, payload, "action_id")
		if err != nil {
			return policy, err
		}
		if !matterHasAction(*aggregate, actionID) {
			return policy, continuity.ErrNotFound
		}
		policy.Materiality = max(policy.Materiality, matterPriority)
		return policy, nil

	case "matter.action.assign":
		if aggregate == nil {
			return policy, nil
		}
		actionID, err := lifecycleSubresourceID(r, payload, "action_id")
		if err != nil {
			return policy, err
		}
		if !matterHasAction(*aggregate, actionID) {
			return policy, continuity.ErrNotFound
		}
		candidateID := stringValue(payload["owner_principal_id"])
		if err := a.validateMatterAssignmentCandidate(ctx, tenant, name, *aggregate, candidateID, authority.ResponsibilityPerformer, max(policy.Materiality, matterPriority)); err != nil {
			return policy, err
		}
		policy.Materiality = max(policy.Materiality, matterPriority)
		return policy, nil

	case "matter.action.transition":
		actionID, err := lifecycleSubresourceID(r, payload, "action_id")
		if err != nil {
			return policy, err
		}
		if aggregate != nil && !matterHasAction(*aggregate, actionID) {
			return policy, continuity.ErrNotFound
		}
		policy.Responsibility = authority.ResponsibilityPerformer
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
		lifecycle, err := continuity.ResponseLifecyclePolicy(current.Status, target)
		if err != nil {
			return policy, err
		}
		policy.Responsibility = authority.Responsibility(lifecycle.Responsibility)
		policy.Materiality = max(lifecycle.Materiality, matterPriority)
		return policy, nil

	default:
		if aggregate != nil {
			policy.Materiality = max(policy.Materiality, matterPriority)
		}
		return policy, nil
	}
}

func (a *API) validateMatterAssignmentCandidate(ctx context.Context, tenant, commandName string, aggregate continuity.MatterAggregate, candidateID string, responsibility authority.Responsibility, materiality int) error {
	candidateID = strings.TrimSpace(candidateID)
	if candidateID == "" {
		return fmt.Errorf("%w: owner_principal_id is required", continuity.ErrInvalidState)
	}
	if !continuity.MatterVisibleTo(aggregate.Matter, candidateID) {
		return fmt.Errorf("%w: assigned person is not permitted to view this issue", continuity.ErrInvalidState)
	}
	if a.deps.Authority == nil {
		return fmt.Errorf("%w: assignment route is unavailable", commandauth.ErrGuardUnavailable)
	}
	actor, err := identity.Require(ctx)
	if err != nil {
		return fmt.Errorf("%w: verified identity is required for assignment", commandauth.ErrIdentityRequired)
	}
	resolution, err := a.deps.Authority.Resolve(ctx, authority.ResolveInput{
		TenantID:       tenant,
		LegalEntityID:  actor.LegalEntityID,
		ObjectType:     "MATTER",
		ObjectID:       aggregate.Matter.ID,
		Responsibility: responsibility,
		DecisionType:   commandName,
		Materiality:    materiality,
	})
	if err != nil {
		return fmt.Errorf("%w: assignment route could not be checked", commandauth.ErrGuardUnavailable)
	}
	if !resolution.AllowsPrincipal(candidateID) {
		return fmt.Errorf("%w: selected person is not eligible for this responsibility", continuity.ErrInvalidState)
	}
	return nil
}

func matterHasAction(aggregate continuity.MatterAggregate, actionID string) bool {
	for _, action := range aggregate.Actions {
		if action.ID == actionID {
			return true
		}
	}
	return false
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
