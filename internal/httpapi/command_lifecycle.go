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
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
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
	var monitoringCheck *monitoring.MonitoringCheck
	var monitoringForm *monitoring.FormTemplate
	var monitoringResult *monitoring.MonitoringResult
	if name == "program.monitoring.issue.create" {
		if a.deps.Monitoring == nil {
			return policy, fmt.Errorf("%w: monitoring service is unavailable", commandauth.ErrGuardUnavailable)
		}
		_, result, check, _, loadErr := a.bindEligibleMonitoringResult(r, a.deps.Monitoring, r.PathValue("result_id"))
		if loadErr != nil {
			return policy, loadErr
		}
		monitoringResult = &result
		monitoringCheck = &check
		programID = check.ProgramID
		payload["program_id"] = check.ProgramID
		payload["monitoring_check_id"] = check.ID
	}
	if name == "program.monitoring.transition" || name == "program.monitoring.evaluate" {
		if a.deps.Monitoring == nil {
			return policy, fmt.Errorf("%w: monitoring service is unavailable", commandauth.ErrGuardUnavailable)
		}
		actor, actorErr := identity.Require(ctx)
		if actorErr != nil {
			return policy, fmt.Errorf("%w: verified identity is required", commandauth.ErrIdentityRequired)
		}
		versionField := "expected_version"
		if name == "program.monitoring.evaluate" {
			versionField = "check_version"
		}
		version, ok := int64Value(payload[versionField])
		if !ok || version < 1 {
			return policy, monitoring.ErrInvalid
		}
		check, loadErr := a.deps.Monitoring.Check(ctx, monitoring.Actor{TenantID: tenant, PrincipalID: actor.PrincipalID}, r.PathValue("id"), version)
		if loadErr != nil {
			return policy, loadErr
		}
		monitoringCheck = &check
		programID = check.ProgramID
	}
	if name == "program.monitoring.form.transition" || name == "program.monitoring.collect" {
		if a.deps.Monitoring == nil {
			return policy, fmt.Errorf("%w: monitoring service is unavailable", commandauth.ErrGuardUnavailable)
		}
		actor, actorErr := identity.Require(ctx)
		if actorErr != nil {
			return policy, fmt.Errorf("%w: verified identity is required", commandauth.ErrIdentityRequired)
		}
		versionField := "expected_version"
		if name == "program.monitoring.collect" {
			versionField = "form_template_version"
		}
		version, ok := int64Value(payload[versionField])
		if !ok || version < 1 {
			return policy, monitoring.ErrInvalid
		}
		form, loadErr := a.deps.Monitoring.Form(ctx, monitoring.Actor{TenantID: tenant, LegalEntityID: actor.LegalEntityID, PrincipalID: actor.PrincipalID}, r.PathValue("id"), r.PathValue("form_id"), version)
		if loadErr != nil {
			return policy, loadErr
		}
		monitoringForm = &form
		boundProgramID, bindErr := lifecycleProgramID(r, payload)
		if bindErr != nil {
			return policy, bindErr
		}
		if boundProgramID == "" || boundProgramID != form.ProgramID {
			return policy, continuity.ErrNotFound
		}
		programID = boundProgramID
	}
	if existingProgramCommand(name) && programID == "" {
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
		actor, actorErr := identity.Require(ctx)
		if actorErr != nil {
			return policy, fmt.Errorf("%w: verified identity is required", commandauth.ErrIdentityRequired)
		}
		if err := validateRequestedRecordEntity(actor, stringValue(payload["legal_entity_id"]), current.Matter.LegalEntityID); err != nil {
			return policy, err
		}
		exactActor, err := a.exactRecordActor(ctx, actor, tenant, current.Matter.TenantID, current.Matter.LegalEntityID)
		if err != nil {
			return policy, err
		}
		ctx = identity.WithActor(ctx, exactActor)
		if r != nil {
			*r = *r.WithContext(ctx)
		}
		delete(payload, "legal_entity_id")
	}
	var programAggregate *continuity.ProgramAggregate
	if existingProgramCommand(name) && programID != "" {
		if a.deps.Continuity != nil {
			current, loadErr := a.deps.Continuity.GetProgram(ctx, tenant, programID)
			if loadErr != nil {
				return policy, loadErr
			}
			programAggregate = &current
			actor, actorErr := identity.Require(ctx)
			if actorErr != nil {
				return policy, fmt.Errorf("%w: verified identity is required", commandauth.ErrIdentityRequired)
			}
			if err := validateRequestedRecordEntity(actor, stringValue(payload["legal_entity_id"]), current.Program.LegalEntityID); err != nil {
				return policy, err
			}
			exactActor, err := a.exactRecordActor(ctx, actor, tenant, current.Program.TenantID, current.Program.LegalEntityID)
			if err != nil {
				return policy, err
			}
			ctx = identity.WithActor(ctx, exactActor)
			if r != nil {
				*r = *r.WithContext(ctx)
			}
			delete(payload, "legal_entity_id")
		}
	}
	if programAggregate != nil && programOwnerBoundCommand(name) {
		storedOwner := strings.TrimSpace(programAggregate.Program.OwnerPrincipalID)
		var err error
		if name == "program.assign" && storedOwner == "" {
			err = a.validateCurrentResponsibilityRouteActor(ctx, tenant, programAggregate.Program.LegalEntityID, "PROGRAM", programAggregate.Program.ID, name, policy.Materiality, authority.ResponsibilityOwner)
		} else {
			err = a.validateStoredResponsibilityActor(ctx, tenant, programAggregate.Program.LegalEntityID, "PROGRAM", programAggregate.Program.ID, name, policy.Materiality, authority.ResponsibilityOwner, storedOwner)
		}
		if err != nil {
			return policy, err
		}
	}
	if aggregate != nil && matterOwnerBoundCommand(name) {
		storedOwner := strings.TrimSpace(aggregate.Matter.OwnerPrincipalID)
		var err error
		if name == "matter.assign" && storedOwner == "" {
			err = a.validateCurrentResponsibilityRouteActor(ctx, tenant, aggregate.Matter.LegalEntityID, "MATTER", aggregate.Matter.ID, name, max(policy.Materiality, matterPriority), authority.ResponsibilityOwner)
		} else {
			err = a.validateStoredResponsibilityActor(ctx, tenant, aggregate.Matter.LegalEntityID, "MATTER", aggregate.Matter.ID, name, max(policy.Materiality, matterPriority), authority.ResponsibilityOwner, storedOwner)
		}
		if err != nil {
			return policy, err
		}
	}

	switch name {
	case "program.monitoring.define":
		if programAggregate == nil {
			return policy, nil
		}
		if err := a.requireMonitoringResponsibility(ctx, tenant, programAggregate.Program, "PROGRAM", programAggregate.Program.ID, authority.ResponsibilityOwner, programAggregate.Program.OwnerPrincipalID, name, policy.Materiality); err != nil {
			return policy, err
		}
		if a.deps.Authority == nil {
			return policy, fmt.Errorf("%w: monitoring reviewer route is unavailable", commandauth.ErrGuardUnavailable)
		}
		reviewer, err := a.deps.Authority.Resolve(ctx, authority.ResolveInput{
			TenantID: tenant, LegalEntityID: programAggregate.Program.LegalEntityID, ObjectType: "PROGRAM", ObjectID: programAggregate.Program.ID,
			Responsibility: authority.ResponsibilityReviewer, DecisionType: "program.monitoring.transition", Materiality: 3,
		})
		if err != nil || strings.TrimSpace(reviewer.Principal.ID) == "" {
			return policy, fmt.Errorf("%w: monitoring reviewer route is unavailable", commandauth.ErrGuardUnavailable)
		}
		if reviewer.Principal.ID == programAggregate.Program.OwnerPrincipalID {
			return policy, fmt.Errorf("%w: monitoring owner and reviewer must be different people", continuity.ErrInvalidState)
		}
		payload["owner_principal_id"] = programAggregate.Program.OwnerPrincipalID
		payload["reviewer_principal_id"] = reviewer.Principal.ID
		payload["program_id"] = programAggregate.Program.ID
		return policy, nil

	case "program.monitoring.form.define":
		if programAggregate == nil {
			return policy, nil
		}
		if err := a.requireMonitoringResponsibility(ctx, tenant, programAggregate.Program, "PROGRAM", programAggregate.Program.ID, authority.ResponsibilityOwner, programAggregate.Program.OwnerPrincipalID, name, policy.Materiality); err != nil {
			return policy, err
		}
		payload["program_id"] = programAggregate.Program.ID
		payload["legal_entity_id"] = programAggregate.Program.LegalEntityID
		return policy, nil

	case "program.monitoring.form.transition":
		if programAggregate == nil || monitoringForm == nil || monitoringForm.ProgramID != programAggregate.Program.ID || monitoringForm.LegalEntityID != programAggregate.Program.LegalEntityID {
			return policy, continuity.ErrNotFound
		}
		policy.Responsibility = authority.ResponsibilityReviewer
		policy.Materiality = 3
		assignedID := ""
		if monitoringForm.Status == monitoring.LifecycleDraft {
			policy.Responsibility = authority.ResponsibilityOwner
			policy.Materiality = 2
			assignedID = programAggregate.Program.OwnerPrincipalID
		}
		if err := a.requireMonitoringResponsibility(ctx, tenant, programAggregate.Program, "PROGRAM", programAggregate.Program.ID, policy.Responsibility, assignedID, name, policy.Materiality); err != nil {
			return policy, err
		}
		payload["program_id"] = programAggregate.Program.ID
		payload["legal_entity_id"] = programAggregate.Program.LegalEntityID
		return policy, nil

	case "program.monitoring.collect":
		if programAggregate == nil || monitoringForm == nil || monitoringForm.ProgramID != programAggregate.Program.ID || monitoringForm.LegalEntityID != programAggregate.Program.LegalEntityID {
			return policy, continuity.ErrNotFound
		}
		if err := a.requireMonitoringResponsibility(ctx, tenant, programAggregate.Program, "PROGRAM", programAggregate.Program.ID, authority.ResponsibilityOwner, programAggregate.Program.OwnerPrincipalID, name, policy.Materiality); err != nil {
			return policy, err
		}
		respondentID, err := a.resolveMonitoringAssignee(ctx, tenant, programAggregate.Program, authority.ResponsibilityPerformer, name, 2)
		if err != nil {
			return policy, err
		}
		reviewerID, err := a.resolveMonitoringAssignee(ctx, tenant, programAggregate.Program, authority.ResponsibilityReviewer, name, 3)
		if err != nil {
			return policy, err
		}
		if respondentID == reviewerID {
			return policy, fmt.Errorf("%w: monitoring respondent and reviewer must be different people", continuity.ErrInvalidState)
		}
		payload["program_id"] = programAggregate.Program.ID
		payload["legal_entity_id"] = programAggregate.Program.LegalEntityID
		payload["respondent_principal_id"] = respondentID
		payload["reviewer_principal_id"] = reviewerID
		return policy, nil

	case "program.monitoring.transition":
		if programAggregate == nil || monitoringCheck == nil {
			return policy, nil
		}
		assignedID := monitoringCheck.ReviewerPrincipalID
		policy.Responsibility = authority.ResponsibilityReviewer
		if monitoringCheck.Status == monitoring.LifecycleDraft {
			assignedID = monitoringCheck.OwnerPrincipalID
			policy.Responsibility = authority.ResponsibilityOwner
			policy.Materiality = 2
		}
		if err := a.requireMonitoringResponsibility(ctx, tenant, programAggregate.Program, "MONITORING_CHECK", monitoringCheck.ID, policy.Responsibility, assignedID, name, policy.Materiality); err != nil {
			return policy, err
		}
		return policy, nil

	case "program.monitoring.evaluate":
		if programAggregate == nil || monitoringCheck == nil {
			return policy, nil
		}
		policy.Responsibility = authority.ResponsibilityPerformer
		if err := a.requireMonitoringResponsibility(ctx, tenant, programAggregate.Program, "MONITORING_CHECK", monitoringCheck.ID, policy.Responsibility, "", name, policy.Materiality); err != nil {
			return policy, err
		}
		return policy, nil

	case "program.monitoring.issue.create":
		if programAggregate == nil || monitoringCheck == nil || monitoringResult == nil || monitoringCheck.ProgramID != programAggregate.Program.ID || monitoringResult.ProgramID != programAggregate.Program.ID {
			return policy, continuity.ErrNotFound
		}
		policy.ObjectType = "MONITORING_CHECK"
		policy.Responsibility = authority.ResponsibilityReviewer
		policy.Materiality = 4
		if err := a.requireMonitoringResponsibility(ctx, tenant, programAggregate.Program, "MONITORING_CHECK", monitoringCheck.ID, policy.Responsibility, monitoringCheck.ReviewerPrincipalID, name, policy.Materiality); err != nil {
			return policy, err
		}
		return policy, nil

	case "program.transition", "program.applicability.decide":
		if programAggregate == nil {
			return policy, nil
		}
		actor, err := identity.Require(ctx)
		if err != nil {
			return policy, fmt.Errorf("%w: verified identity is required", commandauth.ErrIdentityRequired)
		}
		storedAuthority := strings.TrimSpace(programAggregate.Program.AuthorityPrincipalID)
		if storedAuthority == "" || a.deps.Authority == nil {
			return policy, fmt.Errorf("%w: current Program approval authority could not be checked", commandauth.ErrGuardUnavailable)
		}
		resolution, resolveErr := a.deps.Authority.Resolve(ctx, authority.ResolveInput{
			TenantID: tenant, LegalEntityID: programAggregate.Program.LegalEntityID,
			ObjectType: "PROGRAM", ObjectID: programAggregate.Program.ID,
			Responsibility: authority.ResponsibilityAuthorizer, DecisionType: name, Materiality: policy.Materiality,
		})
		if resolveErr != nil || !resolution.AllowsPrincipalFor(actor.PrincipalID, storedAuthority) {
			return policy, fmt.Errorf("%w: signed-in person is not the current Program approval authority", commandauth.ErrNotAuthorized)
		}
		return policy, nil

	case "program.assign":
		if programAggregate == nil {
			return policy, nil
		}
		candidateID := stringValue(payload["owner_principal_id"])
		if err := a.validateProgramAssignmentCandidate(ctx, tenant, name, *programAggregate, candidateID, authority.ResponsibilityOwner, policy.Materiality); err != nil {
			return policy, err
		}
		return policy, nil

	case "program.approval-authority.assign":
		if programAggregate == nil {
			return policy, nil
		}
		candidateID := stringValue(payload["candidate_id"])
		if candidateID == programAggregate.Program.OwnerPrincipalID {
			return policy, fmt.Errorf("%w: Program owner and approval authority must be different people", continuity.ErrInvalidState)
		}
		if err := a.validateProgramApprovalAuthorityCandidate(ctx, tenant, *programAggregate, candidateID, policy.Materiality); err != nil {
			return policy, err
		}
		return policy, nil

	case "program.safeguard.define", "program.safeguard.assign":
		if programAggregate == nil || stringValue(payload["owner_principal_id"]) == "" {
			return policy, nil
		}
		if err := a.validateProgramAssignmentCandidate(ctx, tenant, name, *programAggregate, stringValue(payload["owner_principal_id"]), authority.ResponsibilityPerformer, policy.Materiality); err != nil {
			return policy, err
		}
		return policy, nil

	case "program.safeguard.update":
		implementationID, err := lifecycleSubresourceID(r, payload, "implementation_id")
		if err != nil {
			return policy, err
		}
		if programAggregate != nil && !programHasImplementation(*programAggregate, implementationID) {
			return policy, continuity.ErrNotFound
		}
		payload["implementation_id"] = implementationID
		return policy, nil

	case "program.safeguard.transition":
		implementationID, err := lifecycleSubresourceID(r, payload, "implementation_id")
		if err != nil {
			return policy, err
		}
		if programAggregate == nil {
			return policy, continuity.ErrNotFound
		}
		var implementation *continuity.ControlImplementation
		for index := range programAggregate.ControlImplementations {
			if programAggregate.ControlImplementations[index].ID == implementationID {
				implementation = &programAggregate.ControlImplementations[index]
				break
			}
		}
		if implementation == nil {
			return policy, continuity.ErrNotFound
		}
		if err := a.validateStoredResponsibilityActor(ctx, tenant, programAggregate.Program.LegalEntityID, "CONTROL_IMPLEMENTATION", implementation.ID, name, policy.Materiality, authority.ResponsibilityPerformer, implementation.OwnerPrincipalID); err != nil {
			return policy, err
		}
		payload["implementation_id"] = implementation.ID
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
			if err := a.validateStoredResponsibilityActor(ctx, tenant, aggregate.Matter.LegalEntityID, "MATTER", aggregate.Matter.ID, name, policy.Materiality, authority.ResponsibilityOwner, aggregate.Matter.OwnerPrincipalID); err != nil {
				return policy, err
			}
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
		action := matterActionByID(*aggregate, actionID)
		if action == nil {
			return policy, continuity.ErrNotFound
		}
		candidateID := stringValue(payload["owner_principal_id"])
		responsibility := authority.Responsibility(continuity.ActionResponsibility(*action))
		if err := a.validateMatterAssignmentCandidate(ctx, tenant, name, *aggregate, candidateID, responsibility, max(policy.Materiality, matterPriority)); err != nil {
			return policy, err
		}
		policy.Materiality = max(policy.Materiality, matterPriority)
		return policy, nil

	case "matter.action.transition":
		actionID, err := lifecycleSubresourceID(r, payload, "action_id")
		if err != nil {
			return policy, err
		}
		if aggregate != nil {
			action := matterActionByID(*aggregate, actionID)
			if action == nil {
				return policy, continuity.ErrNotFound
			}
			responsibility := authority.Responsibility(continuity.ActionResponsibility(*action))
			if err := a.validateStoredResponsibilityActor(ctx, tenant, aggregate.Matter.LegalEntityID, "MATTER", aggregate.Matter.ID, name, max(policy.Materiality, matterPriority), responsibility, action.OwnerPrincipalID); err != nil {
				return policy, err
			}
			policy.Responsibility = responsibility
		}
		policy.Materiality = max(policy.Materiality, matterPriority)
		return policy, nil

	case "matter.outcome.define":
		if aggregate == nil {
			return policy, nil
		}
		candidateID := stringValue(payload["reviewer_candidate_id"])
		policy.Materiality = max(policy.Materiality, matterPriority)
		if err := a.validateMatterAssignmentCandidate(ctx, tenant, name, *aggregate, candidateID, authority.ResponsibilityReviewer, policy.Materiality); err != nil {
			return policy, err
		}
		// The stored outcome reviewer comes only from the current server-resolved
		// route. Any authority field supplied by the browser is overwritten.
		payload["authority_principal_id"] = candidateID
		return policy, nil

	case "matter.outcome.record":
		if aggregate == nil {
			return policy, nil
		}
		contractID := stringValue(payload["contract_id"])
		var contract *continuity.VerificationContract
		for index := range aggregate.VerificationContracts {
			if aggregate.VerificationContracts[index].ID == contractID {
				contract = &aggregate.VerificationContracts[index]
				break
			}
		}
		if contract == nil || contract.Status != continuity.VerificationActive {
			return policy, continuity.ErrNotFound
		}
		policy.Materiality = max(policy.Materiality, matterPriority)
		if err := a.validateStoredResponsibilityActor(ctx, tenant, aggregate.Matter.LegalEntityID, "MATTER", aggregate.Matter.ID, name, policy.Materiality, authority.ResponsibilityReviewer, contract.AuthorityPrincipalID); err != nil {
			return policy, err
		}
		payload["reviewer_authority_principal_id"] = contract.AuthorityPrincipalID
		delete(payload, "escalation_principal_id")
		if continuity.VerificationResultStatus(strings.ToUpper(stringValue(payload["result"]))) != continuity.VerificationFailed || contract.FailureResponse != "ESCALATE" {
			return policy, nil
		}
		if a.deps.Authority == nil {
			return policy, fmt.Errorf("%w: escalation route is unavailable", commandauth.ErrGuardUnavailable)
		}
		escalation, err := a.deps.Authority.Resolve(ctx, authority.ResolveInput{
			TenantID: tenant, LegalEntityID: aggregate.Matter.LegalEntityID, ObjectType: "MATTER", ObjectID: aggregate.Matter.ID,
			Responsibility: authority.ResponsibilityEscalation, DecisionType: name, Materiality: policy.Materiality,
		})
		if err != nil || strings.TrimSpace(escalation.Principal.ID) == "" {
			return policy, fmt.Errorf("%w: current escalation owner could not be resolved", commandauth.ErrGuardUnavailable)
		}
		if !continuity.MatterVisibleTo(aggregate.Matter, escalation.Principal.ID) {
			return policy, fmt.Errorf("%w: current escalation owner cannot access this issue", commandauth.ErrGuardUnavailable)
		}
		payload["escalation_principal_id"] = escalation.Principal.ID
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

func (a *API) requireMonitoringResponsibility(ctx context.Context, tenant string, program continuity.Program, objectType, objectID string, responsibility authority.Responsibility, assignedID, decisionType string, materiality int) error {
	actor, err := identity.Require(ctx)
	if err != nil {
		return fmt.Errorf("%w: verified identity is required", commandauth.ErrIdentityRequired)
	}
	assignedID = strings.TrimSpace(assignedID)
	if a.deps.Authority == nil || assignedID == "" && responsibility != authority.ResponsibilityPerformer {
		return fmt.Errorf("%w: monitoring responsibility could not be checked", commandauth.ErrGuardUnavailable)
	}
	resolution, err := a.deps.Authority.Resolve(ctx, authority.ResolveInput{
		TenantID: tenant, LegalEntityID: program.LegalEntityID, ObjectType: objectType, ObjectID: objectID,
		Responsibility: responsibility, DecisionType: decisionType, Materiality: materiality,
	})
	if err != nil {
		return fmt.Errorf("%w: monitoring responsibility could not be checked", commandauth.ErrGuardUnavailable)
	}
	allowed := resolution.AllowsPrincipal(actor.PrincipalID)
	if assignedID != "" {
		allowed = resolution.AllowsPrincipalFor(actor.PrincipalID, assignedID)
	}
	if !allowed {
		return fmt.Errorf("%w: signed-in person does not hold the current monitoring responsibility", commandauth.ErrNotAuthorized)
	}
	return nil
}

func (a *API) resolveMonitoringAssignee(ctx context.Context, tenant string, program continuity.Program, responsibility authority.Responsibility, decisionType string, materiality int) (string, error) {
	if a.deps.Authority == nil {
		return "", fmt.Errorf("%w: monitoring responsibility could not be resolved", commandauth.ErrGuardUnavailable)
	}
	resolution, err := a.deps.Authority.Resolve(ctx, authority.ResolveInput{
		TenantID: tenant, LegalEntityID: program.LegalEntityID, ObjectType: "PROGRAM", ObjectID: program.ID,
		Responsibility: responsibility, DecisionType: decisionType, Materiality: materiality,
	})
	if err != nil || strings.TrimSpace(resolution.Principal.ID) == "" {
		return "", fmt.Errorf("%w: monitoring responsibility could not be resolved", commandauth.ErrGuardUnavailable)
	}
	return strings.TrimSpace(resolution.Principal.ID), nil
}

func programOwnerBoundCommand(name string) bool {
	switch name {
	case "program.details.update", "program.assign", "program.requirement.add", "program.requirement.supersede", "program.safeguard.define", "program.safeguard.update", "program.safeguard.assign", "program.evidence.define", "program.evidence.revise", "program.monitoring.form.define", "program.monitoring.collect":
		return true
	default:
		return false
	}
}

func programHasImplementation(aggregate continuity.ProgramAggregate, implementationID string) bool {
	for _, implementation := range aggregate.ControlImplementations {
		if implementation.ID == implementationID {
			return true
		}
	}
	return false
}

func matterOwnerBoundCommand(name string) bool {
	switch name {
	case "matter.details.update", "matter.context.change", "matter.assign", "matter.link", "matter.action.add", "matter.action.update", "matter.action.assign":
		return true
	default:
		return false
	}
}

func (a *API) validateStoredResponsibilityActor(ctx context.Context, tenant, legalEntity, objectType, objectID, decisionType string, materiality int, responsibility authority.Responsibility, storedPrincipalID string) error {
	actor, err := identity.Require(ctx)
	if err != nil {
		return fmt.Errorf("%w: verified identity is required", commandauth.ErrIdentityRequired)
	}
	storedPrincipalID = strings.TrimSpace(storedPrincipalID)
	if storedPrincipalID == "" {
		return fmt.Errorf("%w: stored responsibility could not be checked", commandauth.ErrGuardUnavailable)
	}
	if a.deps.Authority == nil {
		if actor.PrincipalID == storedPrincipalID {
			return nil
		}
		return fmt.Errorf("%w: stored responsibility could not be checked", commandauth.ErrGuardUnavailable)
	}
	resolution, err := a.deps.Authority.Resolve(ctx, authority.ResolveInput{
		TenantID: tenant, LegalEntityID: legalEntity, ObjectType: objectType, ObjectID: objectID,
		Responsibility: responsibility, DecisionType: decisionType, Materiality: materiality,
	})
	if err != nil {
		return fmt.Errorf("%w: current responsibility route could not be checked", commandauth.ErrGuardUnavailable)
	}
	if !resolution.AllowsPrincipalFor(actor.PrincipalID, storedPrincipalID) {
		return fmt.Errorf("%w: signed-in person is not assigned to the current responsibility", commandauth.ErrNotAuthorized)
	}
	return nil
}

func (a *API) validateCurrentResponsibilityRouteActor(ctx context.Context, tenant, legalEntity, objectType, objectID, decisionType string, materiality int, responsibility authority.Responsibility) error {
	actor, err := identity.Require(ctx)
	if err != nil {
		return fmt.Errorf("%w: verified identity is required", commandauth.ErrIdentityRequired)
	}
	if a.deps.Authority == nil {
		return fmt.Errorf("%w: current responsibility route could not be checked", commandauth.ErrGuardUnavailable)
	}
	resolution, err := a.deps.Authority.Resolve(ctx, authority.ResolveInput{
		TenantID: tenant, LegalEntityID: legalEntity, ObjectType: objectType, ObjectID: objectID,
		Responsibility: responsibility, DecisionType: decisionType, Materiality: materiality,
	})
	if err != nil {
		return fmt.Errorf("%w: current responsibility route could not be checked", commandauth.ErrGuardUnavailable)
	}
	if !resolution.AllowsPrincipal(actor.PrincipalID) {
		return fmt.Errorf("%w: signed-in person does not hold the current responsibility route", commandauth.ErrNotAuthorized)
	}
	return nil
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
		TenantID: tenant, LegalEntityID: aggregate.Program.LegalEntityID, ObjectType: "PROGRAM", ObjectID: aggregate.Program.ID,
		Responsibility: authority.ResponsibilityOwner, DecisionType: commandName, Materiality: materiality,
	})
	if err != nil {
		return fmt.Errorf("%w: assignment route could not be checked", commandauth.ErrGuardUnavailable)
	}
	storedOwner := strings.TrimSpace(aggregate.Program.OwnerPrincipalID)
	actorAllowed := ownerResolution.AllowsPrincipal(actor.PrincipalID)
	if storedOwner != "" {
		actorAllowed = ownerResolution.AllowsPrincipalFor(actor.PrincipalID, storedOwner)
	}
	if !actorAllowed {
		return fmt.Errorf("%w: signed-in person does not hold the current Program owner route", continuity.ErrInvalidState)
	}
	candidateResolution := ownerResolution
	if candidateResponsibility != authority.ResponsibilityOwner {
		candidateResolution, err = a.deps.Authority.Resolve(ctx, authority.ResolveInput{
			TenantID: tenant, LegalEntityID: aggregate.Program.LegalEntityID, ObjectType: "PROGRAM", ObjectID: aggregate.Program.ID,
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

func (a *API) validateProgramApprovalAuthorityCandidate(ctx context.Context, tenant string, aggregate continuity.ProgramAggregate, candidateID string, materiality int) error {
	candidateID = strings.TrimSpace(candidateID)
	if candidateID == "" {
		return fmt.Errorf("%w: candidate_id is required", continuity.ErrInvalidState)
	}
	if a.deps.Authority == nil {
		return fmt.Errorf("%w: approval route is unavailable", commandauth.ErrGuardUnavailable)
	}
	actor, err := identity.Require(ctx)
	if err != nil {
		return fmt.Errorf("%w: verified identity is required", commandauth.ErrIdentityRequired)
	}
	resolution, err := a.deps.Authority.Resolve(ctx, authority.ResolveInput{
		TenantID: tenant, LegalEntityID: aggregate.Program.LegalEntityID, ObjectType: "PROGRAM", ObjectID: aggregate.Program.ID,
		Responsibility: authority.ResponsibilityAuthorizer, DecisionType: "program.transition", Materiality: materiality,
	})
	if err != nil {
		return fmt.Errorf("%w: approval route could not be checked", commandauth.ErrGuardUnavailable)
	}
	if !resolution.AllowsPrincipalFor(actor.PrincipalID, aggregate.Program.AuthorityPrincipalID) {
		return fmt.Errorf("%w: signed-in person does not hold the current Program approval route", continuity.ErrInvalidState)
	}
	if !resolution.AllowsPrincipal(candidateID) {
		return fmt.Errorf("%w: selected person is not eligible for Program approval authority", continuity.ErrInvalidState)
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
	if _, err := identity.Require(ctx); err != nil {
		return fmt.Errorf("%w: verified identity is required for assignment", commandauth.ErrIdentityRequired)
	}
	resolution, err := a.deps.Authority.Resolve(ctx, authority.ResolveInput{
		TenantID:       tenant,
		LegalEntityID:  aggregate.Matter.LegalEntityID,
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
	return matterActionByID(aggregate, actionID) != nil
}

func matterActionByID(aggregate continuity.MatterAggregate, actionID string) *continuity.Action {
	for index := range aggregate.Actions {
		if aggregate.Actions[index].ID == actionID {
			return &aggregate.Actions[index]
		}
	}
	return nil
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
