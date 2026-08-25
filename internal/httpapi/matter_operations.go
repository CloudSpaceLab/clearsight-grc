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
	value := a.buildMatterOperations(r.Context(), actor, aggregate, time.Now().UTC())
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) buildMatterOperations(ctx context.Context, actor identity.Actor, aggregate continuity.MatterAggregate, now time.Time) matterOperationsResponse {
	response := matterOperationsResponse{
		MatterID: aggregate.Matter.ID, MatterVersion: aggregate.Matter.Version,
		AuthorityAvailable: a.deps.Authority != nil, Operations: []RecordOperation{}, GeneratedAt: now.UTC(),
	}
	if aggregate.Matter.Status == continuity.MatterClosed || aggregate.Matter.Status == continuity.MatterCancelled {
		return response
	}

	add := func(spec recordOperationSpec) {
		operation, available := a.resolveRecordOperation(ctx, actor, aggregate.Matter, spec)
		response.AuthorityAvailable = response.AuthorityAvailable && available
		response.Operations = append(response.Operations, operation)
	}
	ownerID := aggregate.Matter.OwnerPrincipalID
	for _, spec := range []recordOperationSpec{
		{Command: "matter.details.update", Label: "Edit issue details", Responsibility: authority.ResponsibilityOwner, Materiality: max(2, aggregate.Matter.Priority), RequiredPrincipalID: ownerID},
		{Command: "matter.context.change", Label: "Update facts and missing information", Responsibility: authority.ResponsibilityOwner, Materiality: max(2, aggregate.Matter.Priority), RequiredPrincipalID: ownerID},
		{Command: "matter.assign", Label: "Change issue owner", Responsibility: authority.ResponsibilityOwner, Materiality: max(3, aggregate.Matter.Priority), RequiredPrincipalID: ownerID, IncludeCandidates: true},
		{Command: "matter.action.add", Label: "Add an action", Responsibility: authority.ResponsibilityOwner, CandidateResponsibility: authority.ResponsibilityPerformer, Materiality: max(2, aggregate.Matter.Priority), RequiredPrincipalID: ownerID, IncludeCandidates: true},
	} {
		add(spec)
	}

	for _, action := range aggregate.Actions {
		if action.Status == continuity.ActionImplemented || action.Status == continuity.ActionCancelled {
			continue
		}
		add(recordOperationSpec{Command: "matter.action.update", SubresourceID: action.ID, Label: "Edit action", Responsibility: authority.ResponsibilityOwner, Materiality: max(2, aggregate.Matter.Priority), RequiredPrincipalID: ownerID})
		add(recordOperationSpec{Command: "matter.action.assign", SubresourceID: action.ID, Label: "Change action owner", Responsibility: authority.ResponsibilityOwner, CandidateResponsibility: authority.ResponsibilityPerformer, Materiality: max(3, aggregate.Matter.Priority), RequiredPrincipalID: ownerID, IncludeCandidates: true})
		targets := continuity.AllowedActionTargets(action.Status)
		allowed := make([]string, len(targets))
		for index := range targets {
			allowed[index] = string(targets[index])
		}
		if len(allowed) > 0 {
			add(recordOperationSpec{Command: "matter.action.transition", SubresourceID: action.ID, Label: "Update action status", Responsibility: authority.ResponsibilityPerformer, Materiality: max(2, aggregate.Matter.Priority), RequiredPrincipalID: action.OwnerPrincipalID, AllowedTargets: allowed})
		}
	}

	requirements, _ := continuity.CompileMatterWork(aggregate, now)
	for _, requirement := range requirements {
		add(recordOperationSpec{
			Command: requirement.CommandName, SubresourceID: requirement.SubresourceID,
			Label: requirement.PrimaryAction, Responsibility: authority.Responsibility(requirement.Responsibility),
			Materiality: requirement.Materiality, RequiredPrincipalID: requirement.RequiredPrincipalID,
			AllowedTargets: append([]string(nil), requirement.AllowedTargets...),
		})
	}
	return response
}

type recordOperationSpec struct {
	Command                 string
	SubresourceID           string
	Label                   string
	Responsibility          authority.Responsibility
	CandidateResponsibility authority.Responsibility
	Materiality             int
	RequiredPrincipalID     string
	IncludeCandidates       bool
	AllowedTargets          []string
}

func (a *API) resolveRecordOperation(ctx context.Context, actor identity.Actor, matter continuity.Matter, spec recordOperationSpec) (RecordOperation, bool) {
	operation := RecordOperation{
		Command: spec.Command, SubresourceID: spec.SubresourceID, Label: spec.Label,
		Responsibility: string(spec.Responsibility), AllowedTargets: spec.AllowedTargets,
	}
	if strings.TrimSpace(spec.RequiredPrincipalID) != "" {
		operation.AssignedTo = a.assignedPrincipal(ctx, actor, authority.Resolution{}, spec.RequiredPrincipalID)
	}
	if a.deps.Authority == nil {
		operation.Reason = "Responsibility could not be checked. No change is available until the authority route is restored."
		return operation, false
	}
	resolution, err := a.deps.Authority.Resolve(ctx, authority.ResolveInput{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ObjectType: "MATTER", ObjectID: matter.ID,
		Responsibility: spec.Responsibility, DecisionType: spec.Command, Materiality: spec.Materiality,
	})
	if err != nil {
		if errors.Is(err, authority.ErrNoRoute) {
			operation.Reason = fmt.Sprintf("No current %s route is available for this issue. Ask a GRC administrator to restore the route.", responsibilityLabel(spec.Responsibility))
			return operation, true
		}
		operation.Reason = "Responsibility could not be checked. No change is available until the authority route is restored."
		return operation, false
	}
	operation.AssignedTo = a.assignedPrincipal(ctx, actor, resolution, spec.RequiredPrincipalID)
	if spec.IncludeCandidates {
		candidateResolution := resolution
		if spec.CandidateResponsibility != "" && spec.CandidateResponsibility != spec.Responsibility {
			candidateResolution, err = a.deps.Authority.Resolve(ctx, authority.ResolveInput{
				TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ObjectType: "MATTER", ObjectID: matter.ID,
				Responsibility: spec.CandidateResponsibility, DecisionType: spec.Command, Materiality: spec.Materiality,
			})
			if err != nil {
				operation.Reason = "Eligible assignment candidates could not be checked. No assignment change is available."
				return operation, !errors.Is(err, authority.ErrNoRoute)
			}
		}
		operation.Candidates = visibleCandidates(candidateResolution, matter)
	}
	allowed := resolution.AllowsPrincipal(actor.PrincipalID)
	if required := strings.TrimSpace(spec.RequiredPrincipalID); required != "" {
		allowed = allowed && actor.PrincipalID == required
	}
	operation.CanAct = allowed
	if allowed {
		operation.Reason = "You hold the current responsibility for this issue and can complete this action."
	} else if operation.AssignedTo != nil {
		operation.Reason = fmt.Sprintf("Assigned to %s for the current issue state.", operation.AssignedTo.DisplayName)
	} else {
		operation.Reason = fmt.Sprintf("The current %s route does not include your signed-in role.", responsibilityLabel(spec.Responsibility))
	}
	return operation, true
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
	if a.deps.Access != nil {
		resolved, err := a.deps.Access.ResolvePrincipal(ctx, actor.TenantID, requiredID, actor.LegalEntityID)
		if err == nil {
			return &authority.Principal{ID: resolved.PrincipalID, DisplayName: resolved.DisplayName, Kind: resolved.Kind}
		}
	}
	return &authority.Principal{ID: requiredID, DisplayName: requiredID, Kind: "PRINCIPAL"}
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
