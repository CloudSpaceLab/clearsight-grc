package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) getProgramOperations(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	tenant, ok := requiredQuery(w, r, "tenant_id")
	if !ok {
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to view Program responsibilities.")
		return
	}
	aggregate, err := service.GetProgram(r.Context(), tenant, r.PathValue("id"))
	if err != nil {
		writeContinuityError(w, err)
		return
	}
	actor, err = a.exactRecordActor(r.Context(), actor, tenant, aggregate.Program.TenantID, aggregate.Program.LegalEntityID)
	if err != nil {
		writeContinuityError(w, continuity.ErrNotFound)
		return
	}
	ctx := identity.WithActor(r.Context(), actor)
	httpx.WriteJSON(w, http.StatusOK, a.buildProgramOperations(ctx, actor, aggregate, time.Now().UTC()))
}

func (a *API) buildProgramOperations(ctx context.Context, actor identity.Actor, aggregate continuity.ProgramAggregate, now time.Time) programOperationsResponse {
	response := programOperationsResponse{
		ProgramID: aggregate.Program.ID, ProgramVersion: aggregate.Program.Version,
		AuthorityAvailable: a.deps.Authority != nil, Operations: []RecordOperation{}, GeneratedAt: now.UTC(),
	}
	exactActor, err := a.exactRecordActor(ctx, actor, aggregate.Program.TenantID, aggregate.Program.TenantID, aggregate.Program.LegalEntityID)
	if err != nil {
		response.AuthorityAvailable = false
		return response
	}
	actor = exactActor
	ctx = identity.WithActor(ctx, actor)
	if owner := a.storedProgramResponsibleParty(ctx, actor, aggregate.Program.OwnerPrincipalID, authority.ResponsibilityOwner); owner != nil {
		response.ResponsibleParties = append(response.ResponsibleParties, *owner)
	}
	if authorizer := a.storedProgramResponsibleParty(ctx, actor, aggregate.Program.AuthorityPrincipalID, authority.ResponsibilityAuthorizer); authorizer != nil {
		response.ResponsibleParties = append(response.ResponsibleParties, *authorizer)
	}
	for _, safeguard := range aggregate.ControlImplementations {
		if owner := a.storedProgramSubresourceParty(ctx, actor, safeguard.OwnerPrincipalID, "SAFEGUARD", safeguard.ID, authority.ResponsibilityPerformer); owner != nil {
			response.ResponsibleParties = append(response.ResponsibleParties, *owner)
		}
	}
	for _, assessment := range aggregate.EvidenceAssessments {
		if reviewer := a.storedProgramSubresourceParty(ctx, actor, assessment.AssessedBy, "EVIDENCE_ASSESSMENT", assessment.ID, authority.ResponsibilityReviewer); reviewer != nil {
			response.ResponsibleParties = append(response.ResponsibleParties, *reviewer)
		}
	}
	if aggregate.Program.Status == continuity.ProgramRetired {
		return response
	}
	specs := make([]programOperationSpec, 0, 16)
	add := func(spec programOperationSpec) {
		specs = append(specs, spec)
	}
	ownerID := aggregate.Program.OwnerPrincipalID
	approvalAuthorityID := aggregate.Program.AuthorityPrincipalID
	for _, spec := range []programOperationSpec{
		{Command: "program.details.update", Label: "Edit Program details", Responsibility: authority.ResponsibilityOwner, Materiality: 2, AssignedPrincipalID: ownerID},
		{Command: "program.assign", Label: "Change Program owner", Responsibility: authority.ResponsibilityOwner, Materiality: 3, AssignedPrincipalID: ownerID, IncludeCandidates: true},
		{Command: "program.approval-authority.assign", DecisionType: "program.transition", Label: "Change approval authority", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 4, AssignedPrincipalID: approvalAuthorityID, IncludeCandidates: true},
		{Command: "program.requirement.add", Label: "Add a requirement", Responsibility: authority.ResponsibilityOwner, Materiality: 2, AssignedPrincipalID: ownerID},
		{Command: "program.safeguard.define", Label: "Define safeguards", Responsibility: authority.ResponsibilityOwner, CandidateResponsibility: authority.ResponsibilityPerformer, Materiality: 2, AssignedPrincipalID: ownerID, IncludeCandidates: true},
		{Command: "program.evidence.define", Label: "Define an evidence check", Responsibility: authority.ResponsibilityOwner, Materiality: 2, AssignedPrincipalID: ownerID},
		{Command: "program.monitoring.form.define", Label: "Create a collection form", Responsibility: authority.ResponsibilityOwner, Materiality: 2, AssignedPrincipalID: ownerID},
		{Command: "program.monitoring.define", Label: "Add a monitoring check", Responsibility: authority.ResponsibilityOwner, Materiality: 2, AssignedPrincipalID: ownerID},
		{Command: "program.applicability.decide", Label: "Decide whether requirements apply", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 3, AssignedPrincipalID: approvalAuthorityID},
		{Command: "program.evidence.assess", Label: "Record evidence check results", Responsibility: authority.ResponsibilityReviewer, Materiality: 3},
		{Command: "program.review.accept", Label: "Confirm the Program review", Responsibility: authority.ResponsibilityReviewer, Materiality: 3},
	} {
		add(spec)
	}
	if a.deps.Monitoring != nil {
		forms, err := a.deps.Monitoring.ListForms(ctx, monitoring.Actor{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PrincipalID: actor.PrincipalID}, aggregate.Program.ID, 500)
		if err == nil {
			seen := map[string]bool{}
			for _, form := range forms {
				if seen[form.ID] {
					continue
				}
				seen[form.ID] = true
				if targets := monitoringTransitionTargets(form.Status); len(targets) > 0 {
					responsibility := authority.ResponsibilityReviewer
					assignedID := ""
					materiality := 3
					if form.Status == monitoring.LifecycleDraft {
						responsibility = authority.ResponsibilityOwner
						assignedID = ownerID
						materiality = 2
					}
					add(programOperationSpec{Command: "program.monitoring.form.transition", SubresourceID: form.ID, Label: "Change " + form.Name + " status", Responsibility: responsibility, Materiality: materiality, AssignedPrincipalID: assignedID, AllowedTargets: targets})
				}
				if form.Status == monitoring.LifecycleActive && form.IsCurrent {
					add(programOperationSpec{Command: "program.monitoring.collect", SubresourceID: form.ID, Label: "Collect " + form.Name + " responses", Responsibility: authority.ResponsibilityOwner, Materiality: 2, AssignedPrincipalID: ownerID})
				}
			}
		}
		checks, err := a.deps.Monitoring.ListChecks(ctx, monitoring.Actor{TenantID: actor.TenantID, PrincipalID: actor.PrincipalID}, aggregate.Program.ID, 500)
		if err == nil {
			latest := make([]monitoring.MonitoringCheck, 0, len(checks))
			seen := map[string]bool{}
			for _, check := range checks {
				if !seen[check.ID] {
					latest = append(latest, check)
					seen[check.ID] = true
				}
			}
			for _, check := range latest {
				if targets := monitoringTransitionTargets(check.Status); len(targets) > 0 {
					responsibility := authority.ResponsibilityReviewer
					assignedID := check.ReviewerPrincipalID
					materiality := 3
					if check.Status == monitoring.LifecycleDraft {
						responsibility = authority.ResponsibilityOwner
						assignedID = check.OwnerPrincipalID
						materiality = 2
					}
					add(programOperationSpec{
						Command: "program.monitoring.transition", SubresourceID: check.ID, Label: "Change " + check.Name + " status",
						ObjectType: "MONITORING_CHECK", ObjectID: check.ID, Responsibility: responsibility, Materiality: materiality,
						AssignedPrincipalID: assignedID, AllowedTargets: targets,
					})
				}
				if check.Status == monitoring.LifecycleActive && check.IsCurrent && check.InputKind == monitoring.InputSource {
					add(programOperationSpec{
						Command: "program.monitoring.evaluate", SubresourceID: check.ID, Label: "Check " + check.Name + " now",
						ObjectType: "MONITORING_CHECK", ObjectID: check.ID, Responsibility: authority.ResponsibilityPerformer, Materiality: 2,
					})
				}
				if check.Status == monitoring.LifecycleActive && check.IsCurrent && check.FailureAction == monitoring.FailureRecommendMatter {
					add(programOperationSpec{
						Command: "program.monitoring.issue.create", SubresourceID: check.ID, Label: "Create linked issue for " + check.Name,
						ObjectType: "MONITORING_CHECK", ObjectID: check.ID, Responsibility: authority.ResponsibilityReviewer, Materiality: 4,
						AssignedPrincipalID: check.ReviewerPrincipalID,
					})
				}
			}
		}
	}
	targets := continuity.AllowedProgramTargets(aggregate.Program.Status)
	allowedTargets := make([]string, len(targets))
	for index := range targets {
		allowedTargets[index] = string(targets[index])
	}
	if len(allowedTargets) > 0 {
		add(programOperationSpec{Command: "program.transition", Label: "Change Program status", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 3, AssignedPrincipalID: approvalAuthorityID, AllowedTargets: allowedTargets})
	}
	for _, requirement := range aggregate.Requirements {
		if requirement.Status == continuity.RequirementApproved && requirement.EffectiveUntil == nil {
			add(programOperationSpec{
				Command: "program.requirement.supersede", SubresourceID: requirement.ID,
				Label: "Replace " + requirement.Title, Responsibility: authority.ResponsibilityOwner,
				Materiality: 2, AssignedPrincipalID: ownerID,
			})
		}
	}
	for _, link := range aggregate.RequirementControlLinks {
		requirementName := "requirement"
		for _, requirement := range aggregate.Requirements {
			if requirement.ID == link.RequirementID {
				requirementName = requirement.Title
				break
			}
		}
		safeguardName := "safeguard"
		for _, safeguard := range aggregate.ControlImplementations {
			if safeguard.ID == link.ImplementationID {
				safeguardName = safeguard.Name
				break
			}
		}
		add(programOperationSpec{
			Command: "program.safeguard.unlink", SubresourceID: link.ID,
			Label:          "Remove " + requirementName + " from " + safeguardName,
			Responsibility: authority.ResponsibilityOwner, Materiality: 3, AssignedPrincipalID: ownerID,
		})
	}
	for _, safeguard := range aggregate.ControlImplementations {
		if safeguard.Status == continuity.ImplementationRetired {
			continue
		}
		add(programOperationSpec{
			Command: "program.safeguard.update", SubresourceID: safeguard.ID,
			Label: "Edit " + safeguard.Name, Responsibility: authority.ResponsibilityOwner,
			Materiality: 2, AssignedPrincipalID: ownerID,
		})
		add(programOperationSpec{
			Command: "program.safeguard.assign", SubresourceID: safeguard.ID,
			Label: "Change the owner of " + safeguard.Name, Responsibility: authority.ResponsibilityOwner,
			CandidateResponsibility: authority.ResponsibilityPerformer, Materiality: 3,
			CandidateObjectType: "CONTROL_IMPLEMENTATION", CandidateObjectID: safeguard.ID, CandidateDecisionType: "program.safeguard.transition",
			AssignedPrincipalID: ownerID, IncludeCandidates: true,
		})
		if transitionTargets := safeguardTransitionTargets(safeguard.Status); len(transitionTargets) > 0 {
			add(programOperationSpec{
				Command: "program.safeguard.transition", SubresourceID: safeguard.ID,
				Label: "Change " + safeguard.Name + " status", ObjectType: "CONTROL_IMPLEMENTATION", ObjectID: safeguard.ID,
				Responsibility: authority.ResponsibilityPerformer, Materiality: 3,
				AssignedPrincipalID: safeguard.OwnerPrincipalID, AllowedTargets: transitionTargets,
			})
		}
	}
	for _, contract := range aggregate.EvidenceContracts {
		if contract.Status == continuity.EvidenceContractRetired {
			continue
		}
		add(programOperationSpec{
			Command: "program.evidence.revise", SubresourceID: contract.ID,
			Label: "Edit " + contract.Name, Responsibility: authority.ResponsibilityOwner,
			Materiality: 2, AssignedPrincipalID: ownerID,
		})
		if transitionTargets := evidenceContractTransitionTargets(contract.Status); len(transitionTargets) > 0 {
			add(programOperationSpec{
				Command: "program.evidence.transition", SubresourceID: contract.ID,
				Label: "Review " + contract.Name + " status", Responsibility: authority.ResponsibilityReviewer,
				Materiality: 3, AllowedTargets: transitionTargets,
			})
		}
	}
	response.Operations, response.AuthorityAvailable = a.resolveProgramOperations(ctx, actor, aggregate.Program, specs, response.AuthorityAvailable, now)
	return response
}

func (a *API) storedProgramResponsibleParty(ctx context.Context, actor identity.Actor, principalID string, responsibility authority.Responsibility) *RecordResponsibleParty {
	return a.storedProgramSubresourceParty(ctx, actor, principalID, "RECORD", "", responsibility)
}

func (a *API) storedProgramSubresourceParty(ctx context.Context, actor identity.Actor, principalID, scope, subresourceID string, responsibility authority.Responsibility) *RecordResponsibleParty {
	principal := a.assignedPrincipal(ctx, actor, authority.Resolution{}, principalID)
	if principal == nil || strings.TrimSpace(principal.DisplayName) == "" {
		return nil
	}
	return &RecordResponsibleParty{Scope: scope, SubresourceID: subresourceID, Responsibility: string(responsibility), DisplayName: principal.DisplayName, Kind: principal.Kind}
}

type programOperationSpec struct {
	Command                 string
	ObjectType              string
	ObjectID                string
	DecisionType            string
	SubresourceID           string
	Label                   string
	Responsibility          authority.Responsibility
	CandidateResponsibility authority.Responsibility
	CandidateObjectType     string
	CandidateObjectID       string
	CandidateDecisionType   string
	Materiality             int
	AssignedPrincipalID     string
	IncludeCandidates       bool
	AllowedTargets          []string
}

type programOperationResolution struct {
	primary   authority.ResolveOutcome
	candidate *authority.ResolveOutcome
}

func (a *API) resolveProgramOperations(ctx context.Context, actor identity.Actor, program continuity.Program, specs []programOperationSpec, authorityAvailable bool, at time.Time) ([]RecordOperation, bool) {
	operations := make([]RecordOperation, 0, len(specs))
	if len(specs) == 0 {
		return operations, authorityAvailable
	}
	batch, ok := a.deps.Authority.(authority.BatchResolver)
	if !ok {
		for _, spec := range specs {
			operation, _ := a.resolveProgramOperation(ctx, actor, program, spec, programOperationResolution{
				primary: authority.ResolveOutcome{Err: errors.New("batch authority resolution is unavailable")},
			})
			operations = append(operations, operation)
		}
		return operations, false
	}

	inputs := make([]authority.ResolveInput, 0, len(specs)*2)
	primaryIndexes := make([]int, len(specs))
	candidateIndexes := make([]int, len(specs))
	for index := range candidateIndexes {
		candidateIndexes[index] = -1
	}
	inputFor := func(spec programOperationSpec, responsibility authority.Responsibility, candidate bool) authority.ResolveInput {
		decisionType := spec.DecisionType
		if decisionType == "" {
			decisionType = spec.Command
		}
		objectType := strings.TrimSpace(spec.ObjectType)
		if objectType == "" {
			objectType = "PROGRAM"
		}
		objectID := strings.TrimSpace(spec.ObjectID)
		if objectID == "" {
			objectID = program.ID
		}
		if candidate {
			if value := strings.TrimSpace(spec.CandidateObjectType); value != "" {
				objectType = value
			}
			if value := strings.TrimSpace(spec.CandidateObjectID); value != "" {
				objectID = value
			}
			if value := strings.TrimSpace(spec.CandidateDecisionType); value != "" {
				decisionType = value
			}
		}
		return authority.ResolveInput{
			TenantID: actor.TenantID, LegalEntityID: program.LegalEntityID, ObjectType: objectType, ObjectID: objectID,
			Responsibility: responsibility, DecisionType: decisionType, Materiality: spec.Materiality, At: at.UTC(),
		}
	}
	for index, spec := range specs {
		primaryIndexes[index] = len(inputs)
		inputs = append(inputs, inputFor(spec, spec.Responsibility, false))
		if spec.IncludeCandidates && spec.CandidateResponsibility != "" && spec.CandidateResponsibility != spec.Responsibility {
			candidateIndexes[index] = len(inputs)
			inputs = append(inputs, inputFor(spec, spec.CandidateResponsibility, true))
		}
	}
	outcomes, err := batch.ResolveMany(ctx, inputs)
	if err != nil || len(outcomes) != len(inputs) {
		outcomes = make([]authority.ResolveOutcome, len(inputs))
		failure := err
		if failure == nil {
			failure = errors.New("batch authority resolution returned incomplete results")
		}
		for index := range outcomes {
			outcomes[index].Err = failure
		}
	}
	for index, spec := range specs {
		resolved := programOperationResolution{primary: outcomes[primaryIndexes[index]]}
		if candidateIndexes[index] >= 0 {
			resolved.candidate = &outcomes[candidateIndexes[index]]
		}
		operation, available := a.resolveProgramOperation(ctx, actor, program, spec, resolved)
		authorityAvailable = authorityAvailable && available
		operations = append(operations, operation)
	}
	return operations, authorityAvailable
}

func (a *API) resolveProgramOperation(ctx context.Context, actor identity.Actor, program continuity.Program, spec programOperationSpec, resolved programOperationResolution) (RecordOperation, bool) {
	operation := RecordOperation{
		Command: spec.Command, SubresourceID: spec.SubresourceID, Label: spec.Label,
		Responsibility: string(spec.Responsibility), AllowedTargets: spec.AllowedTargets,
	}
	requiresStoredPrincipal := programOperationRequiresStoredPrincipal(spec.Command)
	if strings.TrimSpace(spec.AssignedPrincipalID) != "" {
		operation.AssignedTo = a.assignedPrincipal(ctx, actor, authority.Resolution{}, spec.AssignedPrincipalID)
	}
	if a.deps.Authority == nil {
		operation.Reason = "Responsibility could not be checked. No Program change is available until the authority route is restored."
		return operation, false
	}
	resolution, err := resolved.primary.Resolution, resolved.primary.Err
	if err != nil {
		if errors.Is(err, authority.ErrNoRoute) {
			operation.Reason = fmt.Sprintf("No current %s route is available for this Program. Ask a GRC administrator to restore the route.", responsibilityLabel(spec.Responsibility))
			return operation, true
		}
		operation.Reason = "Responsibility could not be checked. No Program change is available until the authority route is restored."
		return operation, false
	}
	if strings.TrimSpace(spec.AssignedPrincipalID) != "" {
		operation.AssignedTo = a.assignedPrincipal(ctx, actor, resolution, spec.AssignedPrincipalID)
	} else if !requiresStoredPrincipal {
		operation.AssignedTo = a.assignedPrincipal(ctx, actor, resolution, "")
	}
	if spec.IncludeCandidates {
		candidateResolution := resolution
		if resolved.candidate != nil {
			candidateResolution, err = resolved.candidate.Resolution, resolved.candidate.Err
			if err != nil {
				operation.Reason = "Eligible assignment candidates could not be checked. No assignment change is available."
				return operation, !errors.Is(err, authority.ErrNoRoute)
			}
		}
		operation.Candidates = visibleProgramCandidates(candidateResolution)
	}
	operation.CanAct = resolution.AllowsPrincipal(actor.PrincipalID)
	if assigned := strings.TrimSpace(spec.AssignedPrincipalID); assigned != "" {
		operation.CanAct = resolution.AllowsPrincipalFor(actor.PrincipalID, assigned)
	} else if requiresStoredPrincipal && spec.Command != "program.assign" {
		operation.CanAct = false
	}
	if operation.CanAct {
		operation.Reason = "You hold the current responsibility for this Program and can complete this action."
	} else if operation.AssignedTo != nil {
		operation.Reason = fmt.Sprintf("Assigned to %s for the current Program state.", operation.AssignedTo.DisplayName)
	} else if requiresStoredPrincipal {
		operation.Reason = "Assign a Program owner before this action can be completed."
	} else {
		operation.Reason = fmt.Sprintf("The current %s route does not include your signed-in role.", responsibilityLabel(spec.Responsibility))
	}
	return operation, true
}

func monitoringTransitionTargets(status monitoring.LifecycleStatus) []string {
	switch status {
	case monitoring.LifecycleDraft:
		return []string{string(monitoring.LifecyclePendingApproval)}
	case monitoring.LifecyclePendingApproval:
		return []string{string(monitoring.LifecycleActive), string(monitoring.LifecycleRejected)}
	case monitoring.LifecycleActive:
		return []string{string(monitoring.LifecyclePaused), string(monitoring.LifecycleRetired)}
	case monitoring.LifecyclePaused:
		return []string{string(monitoring.LifecycleActive), string(monitoring.LifecycleRetired)}
	default:
		return nil
	}
}

func safeguardTransitionTargets(status continuity.ControlImplementationStatus) []string {
	switch status {
	case continuity.ImplementationPlanned:
		return []string{string(continuity.ImplementationInProgress), string(continuity.ImplementationRetired)}
	case continuity.ImplementationInProgress:
		return []string{string(continuity.ImplementationImplemented), string(continuity.ImplementationInactive), string(continuity.ImplementationRetired)}
	case continuity.ImplementationImplemented:
		return []string{string(continuity.ImplementationInactive), string(continuity.ImplementationRetired)}
	case continuity.ImplementationInactive:
		return []string{string(continuity.ImplementationInProgress), string(continuity.ImplementationRetired)}
	default:
		return nil
	}
}

func evidenceContractTransitionTargets(status continuity.EvidenceContractStatus) []string {
	switch status {
	case continuity.EvidenceContractDraft:
		return []string{string(continuity.EvidenceContractActive), string(continuity.EvidenceContractRetired)}
	case continuity.EvidenceContractActive:
		return []string{string(continuity.EvidenceContractRetired)}
	default:
		return nil
	}
}

func programOperationRequiresStoredPrincipal(command string) bool {
	if programOwnerBoundCommand(command) {
		return true
	}
	switch command {
	case "program.transition", "program.applicability.decide", "program.approval-authority.assign", "program.safeguard.transition", "program.monitoring.issue.create":
		return true
	default:
		return false
	}
}

func visibleProgramCandidates(resolution authority.Resolution) []authority.Principal {
	values := append([]authority.Principal{resolution.Principal}, resolution.CandidatePrincipals...)
	result := make([]authority.Principal, 0, len(values))
	seen := map[string]bool{}
	for _, candidate := range values {
		if candidate.ID == "" || seen[candidate.ID] {
			continue
		}
		seen[candidate.ID] = true
		result = append(result, candidate)
	}
	return result
}
