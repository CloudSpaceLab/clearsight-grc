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
	httpx.WriteJSON(w, http.StatusOK, a.buildProgramOperations(r.Context(), actor, aggregate, time.Now().UTC()))
}

func (a *API) buildProgramOperations(ctx context.Context, actor identity.Actor, aggregate continuity.ProgramAggregate, now time.Time) programOperationsResponse {
	response := programOperationsResponse{
		ProgramID: aggregate.Program.ID, ProgramVersion: aggregate.Program.Version,
		AuthorityAvailable: a.deps.Authority != nil, Operations: []RecordOperation{}, GeneratedAt: now.UTC(),
	}
	if aggregate.Program.Status == continuity.ProgramRetired {
		return response
	}
	add := func(spec programOperationSpec) {
		operation, available := a.resolveProgramOperation(ctx, actor, aggregate.Program, spec)
		response.AuthorityAvailable = response.AuthorityAvailable && available
		response.Operations = append(response.Operations, operation)
	}
	ownerID := aggregate.Program.OwnerPrincipalID
	for _, spec := range []programOperationSpec{
		{Command: "program.details.update", Label: "Edit Program details", Responsibility: authority.ResponsibilityOwner, Materiality: 2, AssignedPrincipalID: ownerID},
		{Command: "program.assign", Label: "Change Program owner", Responsibility: authority.ResponsibilityOwner, Materiality: 3, AssignedPrincipalID: ownerID, IncludeCandidates: true},
		{Command: "program.requirement.add", Label: "Add a requirement", Responsibility: authority.ResponsibilityOwner, Materiality: 2, AssignedPrincipalID: ownerID},
		{Command: "program.safeguard.define", Label: "Define safeguards", Responsibility: authority.ResponsibilityOwner, CandidateResponsibility: authority.ResponsibilityPerformer, Materiality: 2, AssignedPrincipalID: ownerID, IncludeCandidates: true},
		{Command: "program.evidence.define", Label: "Define an evidence check", Responsibility: authority.ResponsibilityOwner, Materiality: 2, AssignedPrincipalID: ownerID},
		{Command: "program.applicability.decide", Label: "Decide whether requirements apply", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 3},
		{Command: "program.evidence.assess", Label: "Record evidence check results", Responsibility: authority.ResponsibilityReviewer, Materiality: 3},
		{Command: "program.review.accept", Label: "Confirm the Program review", Responsibility: authority.ResponsibilityReviewer, Materiality: 3},
	} {
		add(spec)
	}
	targets := continuity.AllowedProgramTargets(aggregate.Program.Status)
	allowedTargets := make([]string, len(targets))
	for index := range targets {
		allowedTargets[index] = string(targets[index])
	}
	if len(allowedTargets) > 0 {
		add(programOperationSpec{Command: "program.transition", Label: "Change Program status", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 3, AllowedTargets: allowedTargets})
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
	return response
}

type programOperationSpec struct {
	Command                 string
	SubresourceID           string
	Label                   string
	Responsibility          authority.Responsibility
	CandidateResponsibility authority.Responsibility
	Materiality             int
	AssignedPrincipalID     string
	IncludeCandidates       bool
	AllowedTargets          []string
}

func (a *API) resolveProgramOperation(ctx context.Context, actor identity.Actor, program continuity.Program, spec programOperationSpec) (RecordOperation, bool) {
	operation := RecordOperation{
		Command: spec.Command, SubresourceID: spec.SubresourceID, Label: spec.Label,
		Responsibility: string(spec.Responsibility), AllowedTargets: spec.AllowedTargets,
	}
	if strings.TrimSpace(spec.AssignedPrincipalID) != "" {
		operation.AssignedTo = a.assignedPrincipal(ctx, actor, authority.Resolution{}, spec.AssignedPrincipalID)
	}
	if a.deps.Authority == nil {
		operation.Reason = "Responsibility could not be checked. No Program change is available until the authority route is restored."
		return operation, false
	}
	resolution, err := a.deps.Authority.Resolve(ctx, authority.ResolveInput{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ObjectType: "PROGRAM", ObjectID: program.ID,
		Responsibility: spec.Responsibility, DecisionType: spec.Command, Materiality: spec.Materiality,
	})
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
	} else {
		operation.AssignedTo = a.assignedPrincipal(ctx, actor, resolution, "")
	}
	if spec.IncludeCandidates {
		candidateResolution := resolution
		if spec.CandidateResponsibility != "" && spec.CandidateResponsibility != spec.Responsibility {
			candidateResolution, err = a.deps.Authority.Resolve(ctx, authority.ResolveInput{
				TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ObjectType: "PROGRAM", ObjectID: program.ID,
				Responsibility: spec.CandidateResponsibility, DecisionType: spec.Command, Materiality: spec.Materiality,
			})
			if err != nil {
				operation.Reason = "Eligible assignment candidates could not be checked. No assignment change is available."
				return operation, !errors.Is(err, authority.ErrNoRoute)
			}
		}
		operation.Candidates = visibleProgramCandidates(candidateResolution)
	}
	operation.CanAct = resolution.AllowsPrincipal(actor.PrincipalID)
	if operation.CanAct {
		operation.Reason = "You hold the current responsibility for this Program and can complete this action."
	} else if operation.AssignedTo != nil {
		operation.Reason = fmt.Sprintf("Assigned to %s for the current Program state.", operation.AssignedTo.DisplayName)
	} else {
		operation.Reason = fmt.Sprintf("The current %s route does not include your signed-in role.", responsibilityLabel(spec.Responsibility))
	}
	return operation, true
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
