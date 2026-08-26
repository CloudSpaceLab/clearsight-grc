package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) continuityService(w http.ResponseWriter) (*continuity.Service, bool) {
	if a.deps.Continuity == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "continuity_unavailable", "Programs and issue tracking are unavailable.")
		return nil, false
	}
	return a.deps.Continuity, true
}

func (a *API) listPrograms(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	tenant, ok := requiredQuery(w, r, "tenant_id")
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	values, err := service.ListPrograms(r.Context(), tenant, limit)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "programs_failed", "Programs could not be loaded.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": values, "generated_at": time.Now().UTC()})
}

func (a *API) createProgram(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var request struct {
		continuity.CreateProgramInput
		OwnerCandidateID             string `json:"owner_candidate_id"`
		ApprovalAuthorityCandidateID string `json:"approval_authority_candidate_id"`
	}
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	owner, approval, err := a.resolveProgramSetupSelections(r.Context(), request.LegalEntityID, request.OwnerCandidateID, request.ApprovalAuthorityCandidateID)
	if err != nil {
		if errors.Is(err, commandauth.ErrIdentityRequired) || errors.Is(err, commandauth.ErrGuardUnavailable) || errors.Is(err, commandauth.ErrLegalEntityMismatch) {
			writeCommandAuthorizationError(w, err)
			return
		}
		writeContinuityError(w, err)
		return
	}
	request.OwnerPrincipalID = owner.ID
	request.AuthorityPrincipalID = approval.ID
	value, err := service.CreateProgram(r.Context(), request.CreateProgramInput)
	writeContinuityResult(w, value, err, http.StatusCreated)
}

type programSetupCandidatesResponse struct {
	OwnerCandidates             []authority.Principal `json:"owner_candidates"`
	ApprovalAuthorityCandidates []authority.Principal `json:"approval_authority_candidates"`
	HasMore                     bool                  `json:"has_more"`
	GeneratedAt                 time.Time             `json:"generated_at"`
}

func (a *API) listProgramSetupCandidates(w http.ResponseWriter, r *http.Request) {
	if _, ok := requiredQuery(w, r, "tenant_id"); !ok {
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to choose Program responsibilities.")
		return
	}
	actor, err = exactProgramCandidateActor(actor, r.URL.Query().Get("scope_legal_entity_id"))
	if err != nil {
		writeCommandAuthorizationError(w, err)
		return
	}
	ownerResolution, approvalResolution, err := a.resolveProgramSetupRoutes(r.Context(), actor)
	if err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "program_responsibilities_unavailable", "Current Program ownership and approval responsibilities could not be confirmed.")
		return
	}
	if !ownerResolution.AllowsPrincipal(actor.PrincipalID) {
		httpx.WriteError(w, http.StatusForbidden, "program_creation_not_allowed", "You do not hold the current responsibility to create a Program in this legal entity.")
		return
	}
	owners := visibleProgramCandidates(ownerResolution)
	approvers := visibleProgramCandidates(approvalResolution)
	hasMore := len(owners) > 50 || len(approvers) > 50
	owners = boundedPrincipals(owners, 50)
	approvers = boundedPrincipals(approvers, 50)
	httpx.WriteJSON(w, http.StatusOK, programSetupCandidatesResponse{OwnerCandidates: owners, ApprovalAuthorityCandidates: approvers, HasMore: hasMore, GeneratedAt: time.Now().UTC()})
}

func boundedPrincipals(values []authority.Principal, limit int) []authority.Principal {
	if len(values) <= limit {
		return values
	}
	return values[:limit]
}

func (a *API) resolveProgramSetupRoutes(ctx context.Context, actor identity.Actor) (authority.Resolution, authority.Resolution, error) {
	if a.deps.Authority == nil {
		return authority.Resolution{}, authority.Resolution{}, commandauth.ErrGuardUnavailable
	}
	owner, err := a.deps.Authority.Resolve(ctx, authority.ResolveInput{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ObjectType: "PROGRAM", ObjectID: "*", Responsibility: authority.ResponsibilityOwner, DecisionType: "program.create", Materiality: 2})
	if err != nil {
		return authority.Resolution{}, authority.Resolution{}, err
	}
	approval, err := a.deps.Authority.Resolve(ctx, authority.ResolveInput{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ObjectType: "PROGRAM", ObjectID: "*", Responsibility: authority.ResponsibilityAuthorizer, DecisionType: "program.transition", Materiality: 3})
	if err != nil {
		return authority.Resolution{}, authority.Resolution{}, err
	}
	return owner, approval, nil
}

func (a *API) resolveProgramSetupSelections(ctx context.Context, legalEntityID, ownerCandidateID, approvalCandidateID string) (authority.Principal, authority.Principal, error) {
	actor, err := identity.Require(ctx)
	if err != nil {
		return authority.Principal{}, authority.Principal{}, fmt.Errorf("%w: verified identity is required", commandauth.ErrIdentityRequired)
	}
	actor, err = exactProgramCandidateActor(actor, legalEntityID)
	if err != nil {
		return authority.Principal{}, authority.Principal{}, err
	}
	ownerResolution, approvalResolution, err := a.resolveProgramSetupRoutes(ctx, actor)
	if err != nil {
		return authority.Principal{}, authority.Principal{}, fmt.Errorf("%w: Program responsibilities could not be checked", commandauth.ErrGuardUnavailable)
	}
	if !ownerResolution.AllowsPrincipal(actor.PrincipalID) {
		return authority.Principal{}, authority.Principal{}, fmt.Errorf("%w: signed-in person does not hold the current Program creation responsibility", continuity.ErrInvalidState)
	}
	owner, ownerOK := selectedPrincipal(ownerResolution, ownerCandidateID)
	approval, approvalOK := selectedPrincipal(approvalResolution, approvalCandidateID)
	if !ownerOK || !approvalOK {
		return authority.Principal{}, authority.Principal{}, fmt.Errorf("%w: choose current eligible Program responsibilities", continuity.ErrInvalidState)
	}
	if owner.ID == approval.ID {
		return authority.Principal{}, authority.Principal{}, fmt.Errorf("%w: Program owner and approval authority must be different people", continuity.ErrInvalidState)
	}
	return owner, approval, nil
}

func exactProgramCandidateActor(actor identity.Actor, requestedEntity string) (identity.Actor, error) {
	requestedEntity = strings.TrimSpace(requestedEntity)
	currentEntity := strings.TrimSpace(actor.LegalEntityID)
	if currentEntity == "*" {
		if requestedEntity == "" || requestedEntity == "*" {
			return identity.Actor{}, commandauth.ErrLegalEntityMismatch
		}
		actor.LegalEntityID = requestedEntity
		return actor, nil
	}
	if currentEntity == "" || (requestedEntity != "" && requestedEntity != currentEntity) {
		return identity.Actor{}, commandauth.ErrLegalEntityMismatch
	}
	actor.LegalEntityID = currentEntity
	return actor, nil
}

func selectedPrincipal(resolution authority.Resolution, candidateID string) (authority.Principal, bool) {
	candidateID = strings.TrimSpace(candidateID)
	for _, candidate := range visibleProgramCandidates(resolution) {
		if candidate.ID == candidateID {
			return candidate, true
		}
	}
	return authority.Principal{}, false
}

func (a *API) getProgram(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	tenant, ok := requiredQuery(w, r, "tenant_id")
	if !ok {
		return
	}
	value, err := service.GetProgram(r.Context(), tenant, r.PathValue("id"))
	writeContinuityResult(w, value, err, http.StatusOK)
}

func (a *API) getProgramHistory(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	tenant, ok := requiredQuery(w, r, "tenant_id")
	if !ok {
		return
	}
	at, ok := requiredTimeQuery(w, r, "at")
	if !ok {
		return
	}
	value, err := service.ProgramAt(r.Context(), tenant, r.PathValue("id"), at)
	writeContinuityResult(w, value, err, http.StatusOK)
}

func (a *API) transitionProgram(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.ProgramTransitionInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.ID = r.PathValue("id")
	value, err := service.TransitionProgram(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusOK)
}

func (a *API) updateProgramDetails(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.UpdateProgramDetailsInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.ProgramID = r.PathValue("id")
	value, err := service.UpdateProgramDetails(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusOK)
}

func (a *API) assignProgram(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.AssignProgramInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.ProgramID = r.PathValue("id")
	value, err := service.AssignProgram(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusOK)
}

func (a *API) assignProgramApprovalAuthority(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var request struct {
		TenantID                      string `json:"tenant_id"`
		ExpectedVersion               int64  `json:"expected_version"`
		CandidateID                   string `json:"candidate_id"`
		UntrustedAuthorityPrincipalID string `json:"authority_principal_id,omitempty"`
		ActorID                       string `json:"actor_id,omitempty"`
		Rationale                     string `json:"rationale"`
	}
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := service.AssignProgramApprovalAuthority(r.Context(), continuity.AssignProgramApprovalAuthorityInput{
		TenantID: request.TenantID, ProgramID: r.PathValue("id"), ExpectedVersion: request.ExpectedVersion,
		AuthorityPrincipalID: request.CandidateID, ActorID: request.ActorID, Rationale: request.Rationale,
	})
	writeContinuityResult(w, value, err, http.StatusOK)
}

func (a *API) addProgramRequirement(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.AddRequirementInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.ProgramID = r.PathValue("id")
	value, err := service.AddRequirement(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusCreated)
}

func (a *API) supersedeProgramRequirement(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.SupersedeRequirementInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.ProgramID = r.PathValue("id")
	input.RequirementID = r.PathValue("requirement_id")
	value, err := service.SupersedeRequirement(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusOK)
}

func (a *API) determineProgramApplicability(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.DetermineApplicabilityInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.ProgramID = r.PathValue("id")
	value, err := service.DetermineApplicability(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusCreated)
}

func (a *API) addProgramControlObjective(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.AddControlObjectiveInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.ProgramID = r.PathValue("id")
	value, err := service.AddControlObjective(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusCreated)
}

func (a *API) addProgramControlImplementation(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.AddControlImplementationInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.ProgramID = r.PathValue("id")
	value, err := service.AddControlImplementation(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusCreated)
}

func (a *API) linkProgramRequirementControl(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.LinkRequirementControlInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.ProgramID = r.PathValue("id")
	value, err := service.LinkRequirementControl(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusCreated)
}

func (a *API) addProgramEvidenceContract(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.AddEvidenceContractInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.ProgramID = r.PathValue("id")
	value, err := service.AddEvidenceContract(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusCreated)
}

func (a *API) recordProgramEvidenceAssessment(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.RecordEvidenceAssessmentInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.ProgramID = r.PathValue("id")
	value, err := service.RecordEvidenceAssessment(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusCreated)
}

func (a *API) applyProgramTrigger(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.Trigger
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.ProgramID = r.PathValue("id")
	program, matter, inserted, err := service.ApplyTrigger(r.Context(), input)
	if err != nil {
		writeContinuityError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, map[string]any{"program": program, "matter": matter, "inserted": inserted})
}

func (a *API) listMatters(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	tenant, ok := requiredQuery(w, r, "tenant_id")
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	values, err := service.ListMatters(r.Context(), tenant, r.URL.Query().Get("status"), limit)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "matters_failed", "Issues and changes could not be loaded.")
		return
	}
	values = filterMatterAggregates(r.Context(), values)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": values, "generated_at": time.Now().UTC()})
}

func (a *API) createMatter(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.CreateMatterInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := service.CreateMatter(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusCreated)
}

func (a *API) getMatter(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	tenant, ok := requiredQuery(w, r, "tenant_id")
	if !ok {
		return
	}
	value, err := service.GetMatter(r.Context(), tenant, r.PathValue("id"))
	if err == nil && !canReadMatter(r.Context(), value.Matter) {
		err = continuity.ErrNotFound
	}
	writeContinuityResult(w, value, err, http.StatusOK)
}

func (a *API) getMatterHistory(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	tenant, ok := requiredQuery(w, r, "tenant_id")
	if !ok {
		return
	}
	at, ok := requiredTimeQuery(w, r, "at")
	if !ok {
		return
	}
	value, err := service.MatterAt(r.Context(), tenant, r.PathValue("id"), at)
	if err == nil && !canReadMatter(r.Context(), value.Matter) {
		err = continuity.ErrNotFound
	}
	writeContinuityResult(w, value, err, http.StatusOK)
}

func (a *API) addMatterLink(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.AddMatterLinkInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.MatterID = r.PathValue("id")
	value, err := service.AddMatterLink(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusCreated)
}

func (a *API) transitionMatter(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.TransitionInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.ID = r.PathValue("id")
	value, err := service.TransitionMatter(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusOK)
}

func (a *API) updateMatterDetails(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.UpdateMatterDetailsInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.MatterID = r.PathValue("id")
	value, err := service.UpdateMatterDetails(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusOK)
}

func (a *API) changeMatterContext(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.ChangeMatterContextInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.MatterID = r.PathValue("id")
	value, err := service.ChangeMatterContext(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusOK)
}

func (a *API) assignMatter(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.AssignMatterInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.MatterID = r.PathValue("id")
	value, err := service.AssignMatter(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusOK)
}

func (a *API) addMatterDecision(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.AddDecisionInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.MatterID = r.PathValue("id")
	value, err := service.RecordDecisionLifecycle(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusCreated)
}

func (a *API) addMatterAction(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.AddActionInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.MatterID = r.PathValue("id")
	value, err := service.AddAction(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusCreated)
}

func (a *API) transitionMatterAction(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.TransitionActionInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.MatterID = r.PathValue("id")
	input.ActionID = r.PathValue("action_id")
	value, err := service.TransitionAction(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusOK)
}

func (a *API) updateMatterAction(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.UpdateActionInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.MatterID = r.PathValue("id")
	input.ActionID = r.PathValue("action_id")
	value, err := service.UpdateAction(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusOK)
}

func (a *API) assignMatterAction(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.AssignActionInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.MatterID = r.PathValue("id")
	input.ActionID = r.PathValue("action_id")
	value, err := service.AssignAction(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusOK)
}

func (a *API) addMatterVerificationContract(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.AddVerificationContractInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.MatterID = r.PathValue("id")
	value, err := service.AddVerificationContract(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusCreated)
}

func (a *API) recordMatterVerificationResult(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.RecordVerificationResultInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.MatterID = r.PathValue("id")
	value, err := service.RecordVerificationResult(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusCreated)
}

func (a *API) addMatterResponse(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.AddResponsePackageInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.MatterID = r.PathValue("id")
	value, err := service.AddResponsePackage(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusCreated)
}

func (a *API) transitionMatterResponse(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.TransitionResponseInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.MatterID = r.PathValue("id")
	input.ResponseID = r.PathValue("response_id")
	value, err := service.TransitionResponsePackage(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusOK)
}

func (a *API) getMatterResponseHistory(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	tenant, ok := requiredQuery(w, r, "tenant_id")
	if !ok {
		return
	}
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil && r.URL.Query().Get("limit") != "" {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_limit", "History limit must be a number.")
		return
	}
	value, err := service.ResponsePackageHistory(r.Context(), tenant, r.PathValue("id"), r.PathValue("response_id"), limit)
	writeContinuityResult(w, value, err, http.StatusOK)
}

func requiredTimeQuery(w http.ResponseWriter, r *http.Request, name string) (time.Time, bool) {
	value := strings.TrimSpace(r.URL.Query().Get(name))
	if value == "" {
		httpx.WriteError(w, http.StatusBadRequest, "missing_query", name+" is required")
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_time", name+" must be an RFC3339 timestamp")
		return time.Time{}, false
	}
	return parsed, true
}

func writeContinuityResult[T any](w http.ResponseWriter, value T, err error, success int) {
	if err != nil {
		writeContinuityError(w, err)
		return
	}
	httpx.WriteJSON(w, success, value)
}

func writeContinuityError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, continuity.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "The requested program or issue was not found in this bank scope.")
	case errors.Is(err, continuity.ErrVersionConflict):
		httpx.WriteError(w, http.StatusConflict, "version_conflict", "This record changed. Reload it before saving your update.")
	case errors.Is(err, continuity.ErrDuplicate), errors.Is(err, continuity.ErrTriggerDuplicate):
		httpx.WriteError(w, http.StatusConflict, "duplicate", "An active record already exists for this code or trigger.")
	case errors.Is(err, continuity.ErrInvalidState):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "state_not_allowed", "That status change is not allowed from the current state.")
	case errors.Is(err, continuity.ErrClosureBlocked):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "closure_blocked", err.Error())
	case err != nil:
		httpx.WriteError(w, http.StatusUnprocessableEntity, "continuity_invalid", err.Error())
	}
}
