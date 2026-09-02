package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/access"
	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) getMatterOperations(w http.ResponseWriter, r *http.Request) {
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
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to view issue responsibilities.")
		return
	}
	aggregate, err := service.GetMatter(r.Context(), tenant, r.PathValue("id"))
	if err != nil || !canReadMatter(r.Context(), aggregate.Matter) {
		writeContinuityError(w, continuity.ErrNotFound)
		return
	}
	actor, err = a.exactRecordActor(r.Context(), actor, tenant, aggregate.Matter.TenantID, aggregate.Matter.LegalEntityID)
	if err != nil {
		writeContinuityError(w, continuity.ErrNotFound)
		return
	}
	ctx := identity.WithActor(r.Context(), actor)
	value := a.buildMatterOperations(ctx, actor, aggregate, time.Now().UTC())
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) buildMatterOperations(ctx context.Context, actor identity.Actor, aggregate continuity.MatterAggregate, now time.Time) matterOperationsResponse {
	response := matterOperationsResponse{
		MatterID: aggregate.Matter.ID, MatterVersion: aggregate.Matter.Version,
		AuthorityAvailable: a.deps.Authority != nil, ResponsibilityLabelsComplete: true, Operations: []RecordOperation{}, GeneratedAt: now.UTC(),
	}
	exactActor, err := a.exactRecordActor(ctx, actor, aggregate.Matter.TenantID, aggregate.Matter.TenantID, aggregate.Matter.LegalEntityID)
	if err != nil {
		response.AuthorityAvailable = false
		response.ResponsibilityLabelsComplete = false
		response.ResponsibleParties = nil
		return response
	}
	actor = exactActor
	ctx = identity.WithActor(ctx, actor)
	ctx, response.ResponsibilityLabelsComplete = a.withPrincipalLabels(ctx, actor, matterResponsibilityPrincipalIDs(aggregate))
	response.ResponsibleParties = a.matterResponsibleParties(ctx, actor, aggregate)
	specs := make([]recordOperationSpec, 0, 16+len(aggregate.Actions)*3+len(aggregate.VerificationContracts)*3)
	add := func(spec recordOperationSpec) {
		specs = append(specs, spec)
	}
	if aggregate.Matter.Status == continuity.MatterCancelled {
		return response
	}
	if aggregate.Matter.Status == continuity.MatterClosed {
		targets := continuity.AllowedMatterTargets(aggregate.Matter.Status)
		allowed := make([]string, len(targets))
		for index := range targets {
			allowed[index] = string(targets[index])
		}
		if len(allowed) > 0 {
			add(recordOperationSpec{Command: "matter.transition", Label: "Reopen issue", Responsibility: authority.ResponsibilityAuthorizer, Materiality: max(4, aggregate.Matter.Priority), AllowedTargets: allowed})
		}
		response.Operations, response.AuthorityAvailable = a.resolveMatterOperations(ctx, actor, aggregate.Matter, specs, response.AuthorityAvailable, now)
		return response
	}
	ownerID := aggregate.Matter.OwnerPrincipalID
	assignmentResponsibility := authority.ResponsibilityOwner
	assignmentLabel := "Change issue owner"
	if strings.TrimSpace(ownerID) == "" {
		assignmentResponsibility = authority.ResponsibilityAuthorizer
		assignmentLabel = "Assign issue owner"
	}
	matterTargets := continuity.AllowedMatterTargets(aggregate.Matter.Status)
	for _, spec := range []recordOperationSpec{
		{Command: "matter.details.update", Label: "Edit issue details", Responsibility: authority.ResponsibilityOwner, Materiality: max(2, aggregate.Matter.Priority), RequiredPrincipalID: ownerID},
		{Command: "matter.context.change", Label: "Update facts and missing information", Responsibility: authority.ResponsibilityOwner, Materiality: max(2, aggregate.Matter.Priority), RequiredPrincipalID: ownerID},
		{Command: "matter.assign", Label: assignmentLabel, Responsibility: assignmentResponsibility, CandidateResponsibility: authority.ResponsibilityOwner, Materiality: max(3, aggregate.Matter.Priority), RequiredPrincipalID: ownerID, ReassignmentPrincipalID: ownerID, IncludeCandidates: true},
		{Command: "matter.action.add", Label: "Add an action", Responsibility: authority.ResponsibilityOwner, CandidateResponsibility: authority.ResponsibilityPerformer, Materiality: max(2, aggregate.Matter.Priority), RequiredPrincipalID: ownerID, IncludeCandidates: true},
		{Command: "matter.link", Label: "Link this issue", Responsibility: authority.ResponsibilityOwner, Materiality: max(2, aggregate.Matter.Priority), RequiredPrincipalID: ownerID},
		{Command: "matter.outcome.define", Label: "Define an outcome check", Responsibility: authority.ResponsibilityReviewer, Materiality: max(3, aggregate.Matter.Priority), IncludeCandidates: true},
	} {
		add(spec)
	}
	for _, link := range aggregate.Links {
		add(recordOperationSpec{
			Command: "matter.unlink", SubresourceID: link.ID, Label: "Remove linked Program",
			Responsibility: authority.ResponsibilityOwner, Materiality: max(3, aggregate.Matter.Priority), RequiredPrincipalID: ownerID,
		})
	}
	ordinaryMatterTargets := make([]string, 0, len(matterTargets))
	governedMatterTargets := make([]string, 0, len(matterTargets))
	for _, target := range matterTargets {
		if governedMatterTransition(aggregate.Matter.Status, target) {
			governedMatterTargets = append(governedMatterTargets, string(target))
		} else {
			ordinaryMatterTargets = append(ordinaryMatterTargets, string(target))
		}
	}
	if len(ordinaryMatterTargets) > 0 {
		add(recordOperationSpec{Command: "matter.transition", Label: "Change issue status", Responsibility: authority.ResponsibilityOwner, Materiality: max(3, aggregate.Matter.Priority), RequiredPrincipalID: ownerID, AllowedTargets: ordinaryMatterTargets})
	}
	if len(governedMatterTargets) > 0 {
		add(recordOperationSpec{Command: "matter.transition", Label: "Authorize issue status", Responsibility: authority.ResponsibilityAuthorizer, Materiality: max(4, aggregate.Matter.Priority), AllowedTargets: governedMatterTargets})
	}
	if aggregate.Matter.Status == continuity.MatterDecisionRequired && len(aggregate.Decisions) == 0 {
		add(recordOperationSpec{Command: "matter.decision.record", Label: "Propose decision", Responsibility: authority.ResponsibilityProposer, Materiality: max(2, aggregate.Matter.Priority), AllowedTargets: []string{string(continuity.DecisionProposed)}})
	}
	if aggregate.Matter.Type == continuity.MatterAuthorityRequest || aggregate.Matter.Status == continuity.MatterResponsePreparation {
		add(recordOperationSpec{Command: "matter.response.add", Label: "Prepare response", Responsibility: authority.ResponsibilityProposer, Materiality: max(2, aggregate.Matter.Priority)})
	}

	for _, action := range aggregate.Actions {
		if action.Status == continuity.ActionImplemented || action.Status == continuity.ActionCancelled {
			continue
		}
		actionResponsibility := authority.Responsibility(continuity.ActionResponsibility(action))
		add(recordOperationSpec{Command: "matter.action.update", SubresourceID: action.ID, ObjectType: "ACTION", ObjectID: action.ID, Label: "Edit action", Responsibility: authority.ResponsibilityOwner, Materiality: max(2, aggregate.Matter.Priority), RequiredPrincipalID: ownerID})
		add(recordOperationSpec{Command: "matter.action.assign", SubresourceID: action.ID, ObjectType: "ACTION", ObjectID: action.ID, Label: "Change action owner", Responsibility: authority.ResponsibilityOwner, CandidateResponsibility: authority.ResponsibilityPerformer, Materiality: max(3, aggregate.Matter.Priority), RequiredPrincipalID: ownerID, ReassignmentPrincipalID: action.OwnerPrincipalID, IncludeCandidates: true})
		targets := continuity.AllowedActionTargets(action.Status)
		allowed := make([]string, len(targets))
		for index := range targets {
			allowed[index] = string(targets[index])
		}
		if len(allowed) > 0 {
			add(recordOperationSpec{Command: "matter.action.transition", SubresourceID: action.ID, ObjectType: "ACTION", ObjectID: action.ID, Label: "Update action status", Responsibility: actionResponsibility, Materiality: max(2, aggregate.Matter.Priority), RequiredPrincipalID: action.OwnerPrincipalID, AllowedTargets: allowed})
		}
	}
	for _, contract := range aggregate.VerificationContracts {
		if contract.Status != continuity.VerificationActive {
			continue
		}
		add(recordOperationSpec{
			Command: "matter.outcome.supersede", SubresourceID: contract.ID, ObjectType: "VERIFICATION_CONTRACT", ObjectID: contract.ID, Label: "Replace outcome check",
			Responsibility: authority.ResponsibilityReviewer, CandidateResponsibility: authority.ResponsibilityReviewer,
			Materiality: max(3, aggregate.Matter.Priority), RequiredPrincipalID: contract.AuthorityPrincipalID, IncludeCandidates: true,
		})
		add(recordOperationSpec{
			Command: "matter.outcome.retire", SubresourceID: contract.ID, ObjectType: "VERIFICATION_CONTRACT", ObjectID: contract.ID, Label: "End outcome check",
			Responsibility: authority.ResponsibilityReviewer, Materiality: max(3, aggregate.Matter.Priority), RequiredPrincipalID: contract.AuthorityPrincipalID,
		})
	}

	requirements, ambiguities := continuity.CompileMatterWork(aggregate, now)
	for _, requirement := range requirements {
		add(recordOperationSpec{
			Command: requirement.CommandName, SubresourceID: requirement.SubresourceID,
			ObjectType: matterOperationObjectType(requirement.CommandName, requirement.SubresourceType), ObjectID: requirement.SubresourceID,
			Label: requirement.PrimaryAction, Responsibility: authority.Responsibility(requirement.Responsibility),
			Materiality: requirement.Materiality, RequiredPrincipalID: requirement.RequiredPrincipalID,
			AllowedTargets: append([]string(nil), requirement.AllowedTargets...),
		})
	}
	responsibilities := []authority.Responsibility{
		authority.ResponsibilityProposer,
		authority.ResponsibilityReviewer,
		authority.ResponsibilityChallenger,
		authority.ResponsibilityAuthorizer,
		authority.ResponsibilitySignatory,
		authority.ResponsibilityTransmitter,
		authority.ResponsibilityAcknowledger,
	}
	for _, ambiguity := range ambiguities {
		byResponsibility := map[authority.Responsibility][]string{}
		materiality := map[authority.Responsibility]int{}
		for _, target := range ambiguity.AllowedTargets {
			policy, err := continuity.WorkAmbiguityTargetPolicy(ambiguity, target)
			if err != nil {
				continue
			}
			responsibility := authority.Responsibility(policy.Responsibility)
			byResponsibility[responsibility] = append(byResponsibility[responsibility], target)
			materiality[responsibility] = max(materiality[responsibility], policy.Materiality, aggregate.Matter.Priority)
		}
		for _, responsibility := range responsibilities {
			targets := byResponsibility[responsibility]
			if len(targets) == 0 {
				continue
			}
			add(recordOperationSpec{
				Command: ambiguity.CommandName, SubresourceID: ambiguity.SubresourceID,
				ObjectType: matterOperationObjectType(ambiguity.CommandName, ambiguity.SubresourceType), ObjectID: ambiguity.SubresourceID,
				Label: ambiguity.Title, Responsibility: responsibility,
				Materiality: materiality[responsibility], AllowedTargets: targets,
			})
		}
	}
	response.Operations, response.AuthorityAvailable = a.resolveMatterOperations(ctx, actor, aggregate.Matter, specs, response.AuthorityAvailable, now)
	return response
}

type recordOperationSpec struct {
	Command                 string
	SubresourceID           string
	ObjectType              string
	ObjectID                string
	Label                   string
	Responsibility          authority.Responsibility
	CandidateResponsibility authority.Responsibility
	Materiality             int
	RequiredPrincipalID     string
	ReassignmentPrincipalID string
	IncludeCandidates       bool
	AllowedTargets          []string
}

type recordOperationResolution struct {
	primary   authority.ResolveOutcome
	candidate *authority.ResolveOutcome
}

func (a *API) resolveMatterOperations(ctx context.Context, actor identity.Actor, matter continuity.Matter, specs []recordOperationSpec, authorityAvailable bool, at time.Time) ([]RecordOperation, bool) {
	operations := make([]RecordOperation, 0, len(specs))
	if len(specs) == 0 {
		return operations, authorityAvailable
	}
	batch, ok := a.deps.Authority.(authority.BatchResolver)
	if !ok {
		for _, spec := range specs {
			operation, _ := a.resolveMatterOperation(ctx, actor, matter, spec, recordOperationResolution{
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
	inputFor := func(spec recordOperationSpec, responsibility authority.Responsibility) authority.ResolveInput {
		objectType, objectID := strings.TrimSpace(spec.ObjectType), strings.TrimSpace(spec.ObjectID)
		if objectType == "" || objectID == "" {
			objectType, objectID = "MATTER", matter.ID
		}
		return authority.ResolveInput{
			TenantID: actor.TenantID, LegalEntityID: matter.LegalEntityID, ObjectType: objectType, ObjectID: objectID,
			Responsibility: responsibility, DecisionType: spec.Command, Materiality: spec.Materiality, At: at.UTC(),
		}
	}
	for index, spec := range specs {
		primaryIndexes[index] = len(inputs)
		inputs = append(inputs, inputFor(spec, spec.Responsibility))
		if spec.IncludeCandidates && spec.CandidateResponsibility != "" && spec.CandidateResponsibility != spec.Responsibility {
			candidateIndexes[index] = len(inputs)
			inputs = append(inputs, inputFor(spec, spec.CandidateResponsibility))
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
		resolved := recordOperationResolution{primary: outcomes[primaryIndexes[index]]}
		if candidateIndexes[index] >= 0 {
			resolved.candidate = &outcomes[candidateIndexes[index]]
		}
		operation, available := a.resolveMatterOperation(ctx, actor, matter, spec, resolved)
		authorityAvailable = authorityAvailable && available
		operations = append(operations, operation)
	}
	return operations, authorityAvailable
}

func matterOperationObjectType(command, subresourceType string) string {
	switch command {
	case "matter.action.update", "matter.action.assign", "matter.action.transition":
		return "ACTION"
	case "matter.outcome.supersede", "matter.outcome.retire", "matter.outcome.record":
		return "VERIFICATION_CONTRACT"
	case "matter.response.transition":
		return "RESPONSE_PACKAGE"
	case "matter.decision.record":
		return "DECISION"
	}
	switch strings.ToUpper(strings.TrimSpace(subresourceType)) {
	case "ACTION":
		return "ACTION"
	case "VERIFICATION_CONTRACT":
		return "VERIFICATION_CONTRACT"
	case "RESPONSE", "RESPONSE_PACKAGE":
		return "RESPONSE_PACKAGE"
	case "DECISION":
		return "DECISION"
	default:
		return ""
	}
}

func (a *API) resolveMatterOperation(ctx context.Context, actor identity.Actor, matter continuity.Matter, spec recordOperationSpec, resolved recordOperationResolution) (RecordOperation, bool) {
	operation := RecordOperation{
		Command: spec.Command, SubresourceID: spec.SubresourceID, Label: spec.Label,
		Responsibility: string(spec.Responsibility), AllowedTargets: spec.AllowedTargets,
	}
	requiresStoredPrincipal := matterOperationRequiresStoredPrincipal(spec)
	if strings.TrimSpace(spec.RequiredPrincipalID) != "" {
		operation.AssignedTo = a.assignedPrincipal(ctx, actor, authority.Resolution{}, spec.RequiredPrincipalID)
	}
	if a.deps.Authority == nil {
		operation.Reason = "Responsibility could not be checked. No change is available until the authority route is restored."
		return operation, false
	}
	resolution, err := resolved.primary.Resolution, resolved.primary.Err
	if err != nil {
		if errors.Is(err, authority.ErrNoRoute) {
			operation.Reason = fmt.Sprintf("No current %s route is available for this issue. Ask a GRC administrator to restore the route.", responsibilityLabel(spec.Responsibility))
			return operation, true
		}
		operation.Reason = "Responsibility could not be checked. No change is available until the authority route is restored."
		return operation, false
	}
	if strings.TrimSpace(spec.RequiredPrincipalID) != "" || !requiresStoredPrincipal {
		operation.AssignedTo = a.assignedPrincipal(ctx, actor, resolution, spec.RequiredPrincipalID)
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
		operation.Candidates = visibleCandidates(candidateResolution, matter)
	}
	allowed := resolution.AllowsPrincipal(actor.PrincipalID)
	if required := strings.TrimSpace(spec.RequiredPrincipalID); required != "" {
		allowed = resolution.AllowsPrincipalFor(actor.PrincipalID, required)
	} else if requiresStoredPrincipal && spec.Command != "matter.assign" {
		allowed = false
	}
	operation.CanAct = allowed
	if !operation.CanAct && strings.TrimSpace(spec.ReassignmentPrincipalID) != "" {
		if decision, checked := a.canReassignStoredResponsibility(ctx, actor, matter.LegalEntityID, spec.ReassignmentPrincipalID); checked && decision.Allowed {
			operation.CanAct = true
			if decision.Basis == "CURRENT_ASSIGNEE" {
				operation.Reason = "You are the current assignee and can hand this work to another eligible person."
			} else {
				operation.Reason = "You are in the current assignee's reporting line and can hand this work to another eligible person."
			}
		}
	}
	if allowed {
		operation.Reason = "You hold the current responsibility for this issue and can complete this action."
	} else if !operation.CanAct && operation.AssignedTo != nil {
		operation.Reason = fmt.Sprintf("Assigned to %s for the current issue state.", operation.AssignedTo.DisplayName)
	} else if strings.TrimSpace(spec.RequiredPrincipalID) != "" {
		operation.Reason = "This action has a recorded assignee, but their name is unavailable. Your signed-in responsibility does not match the stored assignment."
	} else if requiresStoredPrincipal {
		operation.Reason = "Assign an owner before this issue action can be completed."
	} else {
		operation.Reason = fmt.Sprintf("The current %s route does not include your signed-in role.", responsibilityLabel(spec.Responsibility))
	}
	return operation, true
}

func (a *API) canReassignStoredResponsibility(ctx context.Context, actor identity.Actor, legalEntityID, currentOwnerID string) (access.ReassignmentDecision, bool) {
	currentOwnerID = strings.TrimSpace(currentOwnerID)
	if currentOwnerID == "" {
		return access.ReassignmentDecision{}, false
	}
	if actor.PrincipalID == currentOwnerID {
		return access.ReassignmentDecision{Allowed: true, Basis: "CURRENT_ASSIGNEE"}, true
	}
	resolver, ok := a.deps.Access.(access.ReassignmentResolver)
	if !ok {
		return access.ReassignmentDecision{}, false
	}
	decision, err := resolver.CanReassign(ctx, access.ReassignmentRequest{
		TenantID: actor.TenantID, LegalEntityID: legalEntityID, ActorPrincipalID: actor.PrincipalID, CurrentOwnerPrincipalID: currentOwnerID,
	})
	return decision, err == nil
}

func matterOperationRequiresStoredPrincipal(spec recordOperationSpec) bool {
	if matterOwnerBoundCommand(spec.Command) {
		return true
	}
	return (spec.Command == "matter.transition" && spec.Responsibility == authority.ResponsibilityOwner) || spec.Command == "matter.action.transition" || spec.Command == "matter.outcome.supersede" || spec.Command == "matter.outcome.retire"
}

func (a *API) assignedPrincipal(ctx context.Context, actor identity.Actor, resolution authority.Resolution, requiredID string) *authority.Principal {
	requiredID = strings.TrimSpace(requiredID)
	if requiredID == "" {
		if resolution.Principal.ID == "" {
			return nil
		}
		value := resolution.Principal
		return &value
	}
	if resolution.Principal.ID == requiredID {
		value := resolution.Principal
		return &value
	}
	for _, candidate := range resolution.CandidatePrincipals {
		if candidate.ID == requiredID {
			value := candidate
			return &value
		}
	}
	if principal, cached := cachedPrincipalLabel(ctx, requiredID); cached {
		return principal
	}
	if a.deps.Access != nil {
		resolved, err := a.deps.Access.ResolvePrincipal(ctx, actor.TenantID, requiredID, actor.LegalEntityID)
		if err == nil {
			return &authority.Principal{ID: resolved.PrincipalID, DisplayName: resolved.DisplayName, Kind: resolved.Kind}
		}
	}
	return nil
}

func matterResponsibilityPrincipalIDs(aggregate continuity.MatterAggregate) []string {
	values := make([]string, 0, len(aggregate.Actions)+len(aggregate.VerificationContracts)+len(aggregate.VerificationResults)+1)
	values = append(values, aggregate.Matter.OwnerPrincipalID)
	for _, action := range aggregate.Actions {
		values = append(values, action.OwnerPrincipalID)
	}
	for _, contract := range aggregate.VerificationContracts {
		values = append(values, contract.AuthorityPrincipalID)
	}
	for _, result := range aggregate.VerificationResults {
		values = append(values, result.ReviewerPrincipalID)
	}
	return values
}

func (a *API) matterResponsibleParties(ctx context.Context, actor identity.Actor, aggregate continuity.MatterAggregate) []RecordResponsibleParty {
	parties := make([]RecordResponsibleParty, 0, len(aggregate.Actions)+len(aggregate.VerificationContracts)+len(aggregate.VerificationResults)+1)
	if owner := a.storedResponsibleParty(ctx, actor, aggregate.Matter.OwnerPrincipalID, "RECORD", "", authority.ResponsibilityOwner); owner != nil {
		parties = append(parties, *owner)
	}
	for _, action := range aggregate.Actions {
		responsibility := authority.Responsibility(continuity.ActionResponsibility(action))
		if owner := a.storedResponsibleParty(ctx, actor, action.OwnerPrincipalID, "ACTION", action.ID, responsibility); owner != nil {
			parties = append(parties, *owner)
		}
	}
	for _, contract := range aggregate.VerificationContracts {
		if reviewer := a.storedResponsibleParty(ctx, actor, contract.AuthorityPrincipalID, "OUTCOME_CHECK", contract.ID, authority.ResponsibilityReviewer); reviewer != nil {
			parties = append(parties, *reviewer)
		}
	}
	for _, result := range aggregate.VerificationResults {
		if reviewer := a.storedResponsibleParty(ctx, actor, result.ReviewerPrincipalID, "OUTCOME_RESULT", result.ID, authority.ResponsibilityReviewer); reviewer != nil {
			parties = append(parties, *reviewer)
		}
	}
	return parties
}

func (a *API) storedResponsibleParty(ctx context.Context, actor identity.Actor, principalID, scope, subresourceID string, responsibility authority.Responsibility) *RecordResponsibleParty {
	if strings.TrimSpace(principalID) == "" {
		return nil
	}
	principal := a.assignedPrincipal(ctx, actor, authority.Resolution{}, principalID)
	if principal == nil || strings.TrimSpace(principal.DisplayName) == "" {
		return &RecordResponsibleParty{Scope: scope, SubresourceID: subresourceID, Responsibility: string(responsibility), DisplayName: unavailableResponsiblePartyName}
	}
	return &RecordResponsibleParty{
		Scope: scope, SubresourceID: subresourceID, Responsibility: string(responsibility),
		DisplayName: principal.DisplayName, Kind: principal.Kind,
	}
}

func visibleCandidates(resolution authority.Resolution, matter continuity.Matter) []authority.Principal {
	values := append([]authority.Principal{resolution.Principal}, resolution.CandidatePrincipals...)
	result := make([]authority.Principal, 0, len(values))
	seen := map[string]bool{}
	for _, candidate := range values {
		if candidate.ID == "" || seen[candidate.ID] || !continuity.MatterVisibleTo(matter, candidate.ID) {
			continue
		}
		seen[candidate.ID] = true
		result = append(result, candidate)
	}
	return result
}

func responsibilityLabel(value authority.Responsibility) string {
	switch value {
	case authority.ResponsibilityOwner:
		return "accountable owner"
	case authority.ResponsibilityPerformer:
		return "performer"
	case authority.ResponsibilityReviewer:
		return "reviewer"
	case authority.ResponsibilityAuthorizer:
		return "authorizer"
	case authority.ResponsibilitySignatory:
		return "signatory"
	default:
		return strings.ToLower(strings.ReplaceAll(string(value), "_", " "))
	}
}
