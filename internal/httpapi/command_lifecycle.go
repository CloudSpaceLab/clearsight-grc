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
	matterID := ""
	if existingMatterCommand(name) {
		var err error
		matterID, err = lifecycleMatterID(r, payload)
		if err != nil {
			return policy, err
		}
	}
	programID := ""
	if existingProgramCommand(name) {
		var err error
		programID, err = lifecycleProgramID(r, payload)
		if err != nil {
			return policy, err
		}
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
		actor, actorOK := identity.FromContext(ctx)
		if !actorOK || !continuity.MatterVisibleTo(current.Matter, actor.PrincipalID) {
			return policy, continuity.ErrNotFound
		}
		aggregate = &current
		matterPriority = current.Matter.Priority
	}
	var programAggregate *continuity.ProgramAggregate
	if existingProgramCommand(name) && programID != "" {
		if a.deps.Continuity != nil {
			current, loadErr := a.deps.Continuity.GetProgram(ctx, tenant, programID)
			if loadErr != nil {
				return policy, loadErr
			}
			programAggregate = &current
		}
	}

	switch name {
	case "program.assign":
		if programAggregate == nil {
			return policy, nil
		}
		candidateID := stringValue(payload["owner_principal_id"])
		if err := a.validateProgramAssignmentCandidate(ctx, tenant, name, *programAggregate, candidateID, authority.ResponsibilityOwner, policy.Materiality); err != nil {
			return policy, err
		}
		return policy, nil

	case "program.safeguard.define":
		if programAggregate == nil || stringValue(payload["owner_principal_id"]) == "" {
			return policy, nil
		}
		if err := a.validateProgramAssignmentCandidate(ctx, tenant, name, *programAggregate, stringValue(payload["owner_principal_id"]), authority.ResponsibilityPerformer, policy.Materiality); err != nil {
			return policy, err
		}
		return policy, nil

	case "program.requirement.supersede":
		requirementID, err := lifecycleSubresourceID(r, payload, "requirement_id")
		if err != nil {
			return policy, err
		}
		if programAggregate != nil && !programHasRequirement(*programAggregate, requirementID) {
			return policy, continuity.ErrNotFound
		}
		return policy, nil

	case "matter.transition":
		if aggregate == nil {
			return policy, nil
		}
		target := continuity.MatterStatus(strings.ToUpper(stringValue(payload["to"])))
		if governedMatterTransition(aggregate.Matter.Status, target) {
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
			if ownerID != "" {
				if err := a.validateMatterAssignmentCandidate(ctx, tenant, name, *aggregate, ownerID, authority.ResponsibilityPerformer, policy.Materiality); err != nil {
					return policy, err
				}
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

func governedMatterTransition(from, target continuity.MatterStatus) bool {
	return target == continuity.MatterDecisionRequired || target == continuity.MatterClosed || target == continuity.MatterCancelled || from == continuity.MatterClosed
}

func (a *API) validateProgramAssignmentCandidate(ctx context.Context, tenant, commandName string, aggregate continuity.ProgramAggregate, candidateID string, candidateResponsibility authority.Responsibility, materiality int) error {
	candidateID = strings.TrimSpace(candidateID)
	if candidateID == "" {
		return fmt.Errorf("%w: owner_principal_id is required", continuity.ErrInvalidState)
	}
	if a.deps.Authority == nil {
		return fmt.Errorf("%w: assignment route is unavailable", commandauth.ErrGuardUnavailable)
	}
	actor, err := identity.Require(ctx)
	if err != nil {
		return fmt.Errorf("%w: verified identity is required for assignment", commandauth.ErrIdentityRequired)
	}
	ownerResolution, err := a.deps.Authority.Resolve(ctx, authority.ResolveInput{
		TenantID: tenant, LegalEntityID: actor.LegalEntityID, ObjectType: "PROGRAM", ObjectID: aggregate.Program.ID,
		Responsibility: authority.ResponsibilityOwner, DecisionType: commandName, Materiality: materiality,
	})
	if err != nil {
		return fmt.Errorf("%w: assignment route could not be checked", commandauth.ErrGuardUnavailable)
	}
	if !ownerResolution.AllowsPrincipal(actor.PrincipalID) {
		return fmt.Errorf("%w: signed-in person does not hold the current Program owner route", continuity.ErrInvalidState)
	}
	candidateResolution := ownerResolution
	if candidateResponsibility != authority.ResponsibilityOwner {
		candidateResolution, err = a.deps.Authority.Resolve(ctx, authority.ResolveInput{
			TenantID: tenant, LegalEntityID: actor.LegalEntityID, ObjectType: "PROGRAM", ObjectID: aggregate.Program.ID,
			Responsibility: candidateResponsibility, DecisionType: commandName, Materiality: materiality,
		})
		if err != nil {
			return fmt.Errorf("%w: assignment candidate route could not be checked", commandauth.ErrGuardUnavailable)
		}
	}
	if !candidateResolution.AllowsPrincipal(candidateID) {
		return fmt.Errorf("%w: selected person is not eligible for Program ownership", continuity.ErrInvalidState)
	}
	return nil
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

func programHasRequirement(aggregate continuity.ProgramAggregate, requirementID string) bool {
	for _, requirement := range aggregate.Requirements {
		if requirement.ID == requirementID {
			return true
		}
	}
	return false
}

func existingMatterCommand(name string) bool {
	return strings.HasPrefix(name, "matter.") && name != "matter.create"
}

func existingProgramCommand(name string) bool {
	return strings.HasPrefix(name, "program.") && name != "program.create"
}

func lifecycleMatterID(r *http.Request, payload map[string]any) (string, error) {
	pathID := ""
	if r != nil {
		pathID = strings.TrimSpace(r.PathValue("id"))
	}
	return boundLifecycleID(pathID, stringValue(payload["matter_id"]), "matter_id")
}

func lifecycleProgramID(r *http.Request, payload map[string]any) (string, error) {
	pathID := ""
	if r != nil {
		pathID = strings.TrimSpace(r.PathValue("id"))
	}
	return boundLifecycleID(pathID, stringValue(payload["program_id"]), "program_id")
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
