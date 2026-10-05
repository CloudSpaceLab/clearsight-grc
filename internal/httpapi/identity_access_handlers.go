package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/access"
	"github.com/CloudSpaceLab/clearsight-grc/internal/governance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
)

type escalationPreviewInput struct {
	PolicyID        string   `json:"policy_id"`
	SequenceID      string   `json:"sequence_id"`
	DepartmentPath  []string `json:"department_path"`
	RevisionVersion int      `json:"revision_version,omitempty"`
}

type escalationPreviewStep struct {
	Index             int      `json:"index"`
	After             string   `json:"after"`
	Responsibility    string   `json:"responsibility"`
	Scope             string   `json:"scope"`
	DepartmentPath    []string `json:"department_path,omitempty"`
	SourceRoles       []string `json:"source_roles,omitempty"`
	TargetRoles       []string `json:"target_roles,omitempty"`
	TargetGroupIDs    []string `json:"target_group_ids,omitempty"`
	TargetPositionIDs []string `json:"target_position_ids,omitempty"`
}

type approveEscalationGuardInput struct {
	ExpectedPolicyVersion int64  `json:"expected_policy_version"`
	Rationale             string `json:"rationale"`
}

func (a *API) identityAccessOverview(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	if a.deps.AccessAdmin == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "identity_access_unavailable", "Identity and access administration is unavailable in this runtime.")
		return
	}
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, parseErr := strconv.Atoi(raw)
		if parseErr != nil || parsed < 1 || parsed > 100 {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_limit", "limit must be between 1 and 100.")
			return
		}
		limit = parsed
	}
	overview, err := a.deps.AccessAdmin.Overview(r.Context(), actor.TenantID, actor.LegalEntityID, limit)
	if err != nil {
		writeIdentityAccessError(w, err)
		return
	}
	policies := identityEscalationPolicies(r, a.deps.Governance, actor.TenantID, actor.LegalEntityID)
	canConfigure := identity.HasPermission(actor, identity.PermissionIdentityConfigure)
	dataBoundary := overview.DataBoundary
	if dataBoundary.DetailTransferMode == "" {
		dataBoundary.LegalEntityID = actor.LegalEntityID
		dataBoundary.DetailTransferMode = access.DetailTransferAggregateOnly
		dataBoundary.AllowedDestinationRegions = []string{}
	}
	payload := map[string]any{
		"sign_in": map[string]any{
			"mode": a.deps.IdentityMode, "issuer": a.deps.OIDCIssuer,
			"authentication": actor.AuthenticationMethod, "assurance_level": actor.AssuranceLevel,
		},
		"actor_principal_id":         actor.PrincipalID,
		"can_configure":              canConfigure,
		"can_configure_organization": canConfigure && identity.HasPermission(actor, identity.PermissionConfigWrite),
		"can_configure_escalation":   canConfigure && identity.HasPermission(actor, identity.PermissionConfigWrite),
		"sources":                    overview.Sources,
		"people":                     overview.People,
		"groups":                     overview.Groups,
		"roles":                      overview.Roles,
		"legal_entities":             overview.LegalEntities,
		"bindings":                   overview.Bindings,
		"positions":                  overview.Positions,
		"escalation":                 overview.Escalation,
		"escalation_policies":        policies,
	}
	payload["organization_scopes"] = overview.OrganizationScopes
	payload["organization_scopes_truncated"] = overview.OrganizationScopesTruncated
	payload["organization_scope_revisions"] = overview.OrganizationScopeRevisions
	payload["organization_position_revisions"] = overview.OrganizationPositionRevisions
	payload["organization_position_history"] = overview.OrganizationPositionHistory
	payload["organization_position_role_revisions"] = overview.OrganizationPositionRoleRevisions
	payload["data_boundary"] = dataBoundary
	payload["data_boundary_revisions"] = overview.DataBoundaryRevisions
	httpx.WriteJSON(w, http.StatusOK, payload)
}

type decideOrganizationScopeInput struct {
	Rationale string `json:"rationale"`
}

type decideLegalEntityDataBoundaryInput struct {
	Rationale string `json:"rationale"`
}

func (a *API) proposeLegalEntityDataBoundary(w http.ResponseWriter, r *http.Request) {
	actor, admin, ok := dataBoundaryAdminActor(w, r, a.deps.AccessAdmin)
	if !ok {
		return
	}
	var input access.ProposeLegalEntityDataBoundaryInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.TenantID = actor.TenantID
	input.LegalEntityID = actor.LegalEntityID
	input.ActorID = actor.PrincipalID
	revision, err := admin.ProposeLegalEntityDataBoundary(r.Context(), input)
	if err != nil {
		writeIdentityAccessError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, revision)
}

func (a *API) approveLegalEntityDataBoundary(w http.ResponseWriter, r *http.Request) {
	a.decideLegalEntityDataBoundary(w, r, true)
}

func (a *API) rejectLegalEntityDataBoundary(w http.ResponseWriter, r *http.Request) {
	a.decideLegalEntityDataBoundary(w, r, false)
}

func (a *API) decideLegalEntityDataBoundary(w http.ResponseWriter, r *http.Request, approve bool) {
	actor, admin, ok := dataBoundaryAdminActor(w, r, a.deps.AccessAdmin)
	if !ok {
		return
	}
	var input decideLegalEntityDataBoundaryInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	decision := access.DecideLegalEntityDataBoundaryInput{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID,
		RevisionID: r.PathValue("id"), ActorID: actor.PrincipalID, Rationale: input.Rationale,
	}
	var err error
	if approve {
		err = admin.ApproveLegalEntityDataBoundary(r.Context(), decision)
	} else {
		err = admin.RejectLegalEntityDataBoundary(r.Context(), decision)
	}
	if err != nil {
		writeIdentityAccessError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) proposeOrganizationScope(w http.ResponseWriter, r *http.Request) {
	actor, ok := organizationScopeAdminActor(w, r, a.deps.AccessAdmin)
	if !ok {
		return
	}
	var input access.ProposeOrganizationScopeInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.TenantID = actor.TenantID
	input.LegalEntityID = actor.LegalEntityID
	input.ActorID = actor.PrincipalID
	revision, err := a.deps.AccessAdmin.ProposeOrganizationScope(r.Context(), input)
	if err != nil {
		writeIdentityAccessError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, revision)
}

func (a *API) approveOrganizationScope(w http.ResponseWriter, r *http.Request) {
	a.decideOrganizationScope(w, r, true)
}

func (a *API) rejectOrganizationScope(w http.ResponseWriter, r *http.Request) {
	a.decideOrganizationScope(w, r, false)
}

func (a *API) decideOrganizationScope(w http.ResponseWriter, r *http.Request, approve bool) {
	actor, ok := organizationScopeAdminActor(w, r, a.deps.AccessAdmin)
	if !ok {
		return
	}
	var input decideOrganizationScopeInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	decision := access.DecideOrganizationScopeInput{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID,
		RevisionID: r.PathValue("id"), ActorID: actor.PrincipalID, Rationale: input.Rationale,
	}
	var err error
	if approve {
		err = a.deps.AccessAdmin.ApproveOrganizationScope(r.Context(), decision)
	} else {
		err = a.deps.AccessAdmin.RejectOrganizationScope(r.Context(), decision)
	}
	if err != nil {
		writeIdentityAccessError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type decideOrganizationPositionInput struct {
	Rationale string `json:"rationale"`
}

func (a *API) proposeOrganizationPosition(w http.ResponseWriter, r *http.Request) {
	actor, ok := organizationPositionAdminActor(w, r, a.deps.AccessAdmin)
	if !ok {
		return
	}
	var input access.ProposeOrganizationPositionInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.TenantID = actor.TenantID
	input.LegalEntityID = actor.LegalEntityID
	input.ActorID = actor.PrincipalID
	revision, err := a.deps.AccessAdmin.ProposeOrganizationPosition(r.Context(), input)
	if err != nil {
		writeIdentityAccessError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, revision)
}

func (a *API) simulateOrganizationPosition(w http.ResponseWriter, r *http.Request) {
	actor, ok := organizationPositionAdminActor(w, r, a.deps.AccessAdmin)
	if !ok {
		return
	}
	value, err := a.deps.AccessAdmin.SimulateOrganizationPosition(r.Context(), access.SimulateOrganizationPositionInput{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID,
		RevisionID: r.PathValue("id"), ActorID: actor.PrincipalID,
	})
	if err != nil {
		writeIdentityAccessError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) restoreOrganizationPosition(w http.ResponseWriter, r *http.Request) {
	actor, ok := organizationPositionAdminActor(w, r, a.deps.AccessAdmin)
	if !ok {
		return
	}
	revision, err := a.deps.AccessAdmin.RestoreOrganizationPosition(r.Context(), access.RestoreOrganizationPositionInput{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID,
		RevisionID: r.PathValue("id"), ActorID: actor.PrincipalID,
	})
	if err != nil {
		writeIdentityAccessError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, revision)
}

func (a *API) approveOrganizationPosition(w http.ResponseWriter, r *http.Request) {
	a.decideOrganizationPosition(w, r, true)
}

func (a *API) rejectOrganizationPosition(w http.ResponseWriter, r *http.Request) {
	a.decideOrganizationPosition(w, r, false)
}

func (a *API) decideOrganizationPosition(w http.ResponseWriter, r *http.Request, approve bool) {
	actor, ok := organizationPositionAdminActor(w, r, a.deps.AccessAdmin)
	if !ok {
		return
	}
	var input decideOrganizationPositionInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	decision := access.DecideOrganizationPositionInput{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID,
		RevisionID: r.PathValue("id"), ActorID: actor.PrincipalID, Rationale: input.Rationale,
	}
	var err error
	if approve {
		err = a.deps.AccessAdmin.ApproveOrganizationPosition(r.Context(), decision)
	} else {
		err = a.deps.AccessAdmin.RejectOrganizationPosition(r.Context(), decision)
	}
	if err != nil {
		writeIdentityAccessError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) proposeOrganizationPositionRole(w http.ResponseWriter, r *http.Request) {
	actor, ok := organizationPositionAdminActor(w, r, a.deps.AccessAdmin)
	if !ok {
		return
	}
	var input access.ProposeOrganizationPositionRoleInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.TenantID = actor.TenantID
	input.LegalEntityID = actor.LegalEntityID
	input.ActorID = actor.PrincipalID
	revision, err := a.deps.AccessAdmin.ProposeOrganizationPositionRole(r.Context(), input)
	if err != nil {
		writeIdentityAccessError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, revision)
}

func (a *API) approveOrganizationPositionRole(w http.ResponseWriter, r *http.Request) {
	a.decideOrganizationPositionRole(w, r, true)
}

func (a *API) rejectOrganizationPositionRole(w http.ResponseWriter, r *http.Request) {
	a.decideOrganizationPositionRole(w, r, false)
}

func (a *API) decideOrganizationPositionRole(w http.ResponseWriter, r *http.Request, approve bool) {
	actor, ok := organizationPositionAdminActor(w, r, a.deps.AccessAdmin)
	if !ok {
		return
	}
	var input decideOrganizationPositionInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	decision := access.DecideOrganizationPositionRoleInput{
		TenantID:      actor.TenantID,
		LegalEntityID: actor.LegalEntityID,
		RevisionID:    r.PathValue("id"),
		ActorID:       actor.PrincipalID,
		Rationale:     input.Rationale,
	}
	var err error
	if approve {
		err = a.deps.AccessAdmin.ApproveOrganizationPositionRole(r.Context(), decision)
	} else {
		err = a.deps.AccessAdmin.RejectOrganizationPositionRole(r.Context(), decision)
	}
	if err != nil {
		writeIdentityAccessError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) createSCIMSource(w http.ResponseWriter, r *http.Request) {
	actor, ok := identityAdminActor(w, r, a.deps.AccessAdmin)
	if !ok {
		return
	}
	var input access.CreateSCIMSourceInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.TenantID, input.ActorID = actor.TenantID, actor.PrincipalID
	token, digest, err := access.NewProvisioningToken()
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "token_generation_failed", "A provisioning token could not be generated.")
		return
	}
	source, err := a.deps.AccessAdmin.CreateSCIMSource(r.Context(), input, digest[:])
	if err != nil {
		writeIdentityAccessError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"source": source, "token": token})
}

func (a *API) rotateSCIMSourceToken(w http.ResponseWriter, r *http.Request) {
	actor, ok := identityAdminActor(w, r, a.deps.AccessAdmin)
	if !ok {
		return
	}
	token, digest, err := access.NewProvisioningToken()
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "token_generation_failed", "A provisioning token could not be generated.")
		return
	}
	if err := a.deps.AccessAdmin.RotateSCIMSourceToken(r.Context(), actor.TenantID, r.PathValue("id"), actor.PrincipalID, digest[:]); err != nil {
		writeIdentityAccessError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"token": token})
}

func (a *API) revokeSCIMSource(w http.ResponseWriter, r *http.Request) {
	actor, ok := identityAdminActor(w, r, a.deps.AccessAdmin)
	if !ok {
		return
	}
	if err := a.deps.AccessAdmin.RevokeSCIMSource(r.Context(), actor.TenantID, r.PathValue("id"), actor.PrincipalID); err != nil {
		writeIdentityAccessError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) createDirectoryGroupRoleBinding(w http.ResponseWriter, r *http.Request) {
	actor, ok := identityAdminActor(w, r, a.deps.AccessAdmin)
	if !ok {
		return
	}
	var input access.CreateGroupRoleBindingInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.TenantID, input.ActorID, input.LegalEntityID = actor.TenantID, actor.PrincipalID, actor.LegalEntityID
	value, err := a.deps.AccessAdmin.CreateGroupRoleBinding(r.Context(), input)
	if err != nil {
		writeIdentityAccessError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, value)
}

func (a *API) retireDirectoryGroupRoleBinding(w http.ResponseWriter, r *http.Request) {
	actor, ok := identityAdminActor(w, r, a.deps.AccessAdmin)
	if !ok {
		return
	}
	bindingID := strings.TrimSpace(r.PathValue("id"))
	overview, err := a.deps.AccessAdmin.Overview(r.Context(), actor.TenantID, actor.LegalEntityID, 100)
	if err != nil {
		writeIdentityAccessError(w, err)
		return
	}
	found := false
	for _, binding := range overview.Bindings {
		if binding.ID == bindingID {
			found = true
			break
		}
	}
	if !found {
		writeIdentityAccessError(w, access.ErrAdminNotFound)
		return
	}
	if err := a.deps.AccessAdmin.RetireGroupRoleBinding(r.Context(), actor.TenantID, bindingID, actor.PrincipalID); err != nil {
		writeIdentityAccessError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) proposeEscalationGuardRevision(w http.ResponseWriter, r *http.Request) {
	actor, ok := escalationGuardAdminActor(w, r, a.deps.Governance)
	if !ok {
		return
	}
	var input governance.EscalationGuardRevisionInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.TenantID, input.LegalEntityID, input.ActorID = actor.TenantID, actor.LegalEntityID, actor.PrincipalID
	revision, err := a.deps.Governance.ProposeEscalationGuardRevision(r.Context(), input)
	if err != nil {
		writeEscalationGuardError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, revision)
}

func (a *API) approveEscalationGuardRevision(w http.ResponseWriter, r *http.Request) {
	actor, ok := escalationGuardAdminActor(w, r, a.deps.Governance)
	if !ok {
		return
	}
	revisionVersion, err := strconv.Atoi(strings.TrimSpace(r.PathValue("version")))
	if err != nil || revisionVersion < 1 {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_revision_version", "A positive revision version is required.")
		return
	}
	var input approveEscalationGuardInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	policy, err := a.deps.Governance.ApprovePolicyRevision(r.Context(), governance.ApprovePolicyRevisionInput{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PolicyID: strings.TrimSpace(r.PathValue("policy_id")), RevisionVersion: revisionVersion,
		ActorID: actor.PrincipalID, ExpectedPolicyVersion: input.ExpectedPolicyVersion, Rationale: input.Rationale,
	})
	if err != nil {
		writeEscalationGuardError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, policy)
}

func (a *API) proposeEscalationSequenceRevision(w http.ResponseWriter, r *http.Request) {
	actor, ok := escalationGuardAdminActor(w, r, a.deps.Governance)
	if !ok {
		return
	}
	var input governance.EscalationSequenceRevisionInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.TenantID, input.LegalEntityID, input.ActorID = actor.TenantID, actor.LegalEntityID, actor.PrincipalID
	revision, err := a.deps.Governance.ProposeEscalationSequenceRevision(r.Context(), input)
	if err != nil {
		writeEscalationGuardError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, revision)
}

func (a *API) proposeEscalationRollback(w http.ResponseWriter, r *http.Request) {
	actor, ok := escalationGuardAdminActor(w, r, a.deps.Governance)
	if !ok {
		return
	}
	var input governance.EscalationRollbackInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.TenantID, input.LegalEntityID, input.PolicyID, input.ActorID = actor.TenantID, actor.LegalEntityID, strings.TrimSpace(r.PathValue("policy_id")), actor.PrincipalID
	revision, err := a.deps.Governance.ProposeEscalationRollback(r.Context(), input)
	if err != nil {
		writeEscalationGuardError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, revision)
}

type escalationSimulationInput struct {
	PolicyID        string `json:"policy_id"`
	SequenceID      string `json:"sequence_id"`
	RevisionVersion int    `json:"revision_version,omitempty"`
	Limit           int    `json:"limit,omitempty"`
}

func (a *API) simulateEscalation(w http.ResponseWriter, r *http.Request) {
	actor, ok := escalationGuardAdminActor(w, r, a.deps.Governance)
	if !ok {
		return
	}
	if a.deps.EscalationSimulation == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "escalation_simulation_unavailable", "Escalation simulation is unavailable in this runtime.")
		return
	}
	var input escalationSimulationInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := a.deps.EscalationSimulation.SimulateEscalation(r.Context(), workflow.EscalationSimulationInput{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID,
		PolicyID: input.PolicyID, SequenceID: input.SequenceID, RevisionVersion: input.RevisionVersion, Limit: input.Limit,
	})
	if err != nil {
		writeEscalationGuardError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

func (a *API) previewEscalation(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	if a.deps.Governance == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "governance_unavailable", "Escalation policy preview is unavailable.")
		return
	}
	var input escalationPreviewInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	path, err := identity.NormalizeDepartmentPath(input.DepartmentPath)
	if err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "invalid_department_path", err.Error())
		return
	}
	policies, err := a.deps.Governance.ListPoliciesForEntity(r.Context(), actor.TenantID, actor.LegalEntityID, 100)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "governance_failed", "Escalation policies could not be loaded.")
		return
	}
	for _, policy := range policies {
		if policy.ID != strings.TrimSpace(input.PolicyID) || policy.Status != governance.PolicyActive {
			continue
		}
		definition := policy.Definition
		version := policy.CurrentVersion
		if input.RevisionVersion > 0 {
			revision, revisionErr := a.deps.Governance.PendingPolicyRevision(r.Context(), actor.TenantID, policy.ID)
			if revisionErr != nil || revision.Version != input.RevisionVersion {
				httpx.WriteError(w, http.StatusNotFound, "escalation_revision_not_found", "The pending escalation guard revision was not found.")
				return
			}
			definition, version = revision.Definition, revision.Version
		}
		sequences, parseErr := governance.ParseEscalationSequences(definition)
		if parseErr != nil {
			httpx.WriteError(w, http.StatusUnprocessableEntity, "invalid_escalation_policy", parseErr.Error())
			return
		}
		for _, sequence := range sequences {
			if sequence.ID != strings.TrimSpace(input.SequenceID) {
				continue
			}
			steps := make([]escalationPreviewStep, 0, len(sequence.Steps))
			for index, step := range sequence.Steps {
				preview := escalationPreviewStep{
					Index: index, After: step.After.String(), Responsibility: step.Responsibility, Scope: "LEGAL_ENTITY",
					SourceRoles: append([]string(nil), step.SourceRoles...), TargetRoles: append([]string(nil), step.TargetRoles...), TargetGroupIDs: append([]string(nil), step.TargetGroupIDs...), TargetPositionIDs: append([]string(nil), step.TargetPositionIDs...),
				}
				if step.DepartmentLevelsUp != nil {
					preview.Scope = "DEPARTMENT"
					if scoped, exists := governance.DepartmentScope(path, step.DepartmentLevelsUp); exists {
						preview.DepartmentPath = scoped
					} else {
						preview.Scope = "OUT_OF_RANGE"
					}
				}
				steps = append(steps, preview)
			}
			httpx.WriteJSON(w, http.StatusOK, map[string]any{
				"policy_id": policy.ID, "policy_code": policy.Code, "policy_version": version,
				"sequence_id": sequence.ID, "trigger": sequence.Trigger, "steps": steps,
			})
			return
		}
	}
	httpx.WriteError(w, http.StatusNotFound, "escalation_sequence_not_found", "The active escalation sequence was not found.")
}

func identityAdminActor(w http.ResponseWriter, r *http.Request, admin access.Administrator) (identity.Actor, bool) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return identity.Actor{}, false
	}
	if admin == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "identity_access_unavailable", "Identity and access administration is unavailable in this runtime.")
		return identity.Actor{}, false
	}
	return actor, true
}

func dataBoundaryAdminActor(
	w http.ResponseWriter,
	r *http.Request,
	admin access.Administrator,
) (identity.Actor, access.DataBoundaryAdministrator, bool) {
	actor, ok := organizationScopeAdminActor(w, r, admin)
	if !ok {
		return identity.Actor{}, nil, false
	}
	boundaryAdmin, ok := admin.(access.DataBoundaryAdministrator)
	if !ok {
		httpx.WriteError(w, http.StatusServiceUnavailable, "data_boundary_unavailable", "Data-boundary administration is unavailable in this runtime.")
		return identity.Actor{}, nil, false
	}
	return actor, boundaryAdmin, true
}

func organizationScopeAdminActor(w http.ResponseWriter, r *http.Request, admin access.Administrator) (identity.Actor, bool) {
	actor, ok := identityAdminActor(w, r, admin)
	if !ok {
		return identity.Actor{}, false
	}
	if !identity.HasPermission(actor, identity.PermissionIdentityConfigure) || !identity.HasPermission(actor, identity.PermissionConfigWrite) {
		httpx.WriteError(w, http.StatusForbidden, "organization_scope_governance_required", "Identity and governance configuration permissions are required.")
		return identity.Actor{}, false
	}
	return actor, true
}

func organizationPositionAdminActor(w http.ResponseWriter, r *http.Request, admin access.Administrator) (identity.Actor, bool) {
	actor, ok := identityAdminActor(w, r, admin)
	if !ok {
		return identity.Actor{}, false
	}
	if !identity.HasPermission(actor, identity.PermissionIdentityConfigure) || !identity.HasPermission(actor, identity.PermissionConfigWrite) {
		httpx.WriteError(w, http.StatusForbidden, "organization_position_governance_required", "Identity and governance configuration permissions are required.")
		return identity.Actor{}, false
	}
	return actor, true
}

func escalationGuardAdminActor(w http.ResponseWriter, r *http.Request, service *governance.Service) (identity.Actor, bool) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return identity.Actor{}, false
	}
	if service == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "governance_unavailable", "Escalation guard administration is unavailable.")
		return identity.Actor{}, false
	}
	if !identity.HasPermission(actor, identity.PermissionIdentityConfigure) || !identity.HasPermission(actor, identity.PermissionConfigWrite) {
		httpx.WriteError(w, http.StatusForbidden, "escalation_governance_required", "Identity configuration and governance configuration permissions are both required.")
		return identity.Actor{}, false
	}
	return actor, true
}

func identityEscalationPolicies(r *http.Request, service *governance.Service, tenant, legalEntityID string) []map[string]any {
	if service == nil {
		return []map[string]any{}
	}
	policies, err := service.ListPoliciesForEntity(r.Context(), tenant, legalEntityID, 100)
	if err != nil {
		return []map[string]any{}
	}
	result := make([]map[string]any, 0, len(policies))
	for _, policy := range policies {
		if policy.Status != governance.PolicyActive {
			continue
		}
		sequences, err := governance.ParseEscalationSequences(policy.Definition)
		if err != nil {
			continue
		}
		item := map[string]any{
			"policy_id": policy.ID, "code": policy.Code, "name": policy.Name,
			"version": policy.CurrentVersion, "record_version": policy.Version, "effective_from": policy.EffectiveFrom, "sequences": sequences,
		}
		if revision, revisionErr := service.PendingPolicyRevision(r.Context(), tenant, policy.ID); revisionErr == nil {
			if pendingSequences, parseErr := governance.ParseEscalationSequences(revision.Definition); parseErr == nil {
				item["pending_revision"] = map[string]any{
					"version": revision.Version, "maker_id": revision.MakerID, "created_at": revision.CreatedAt,
					"sequences": pendingSequences,
				}
			}
		}
		result = append(result, item)
	}
	return result
}

func writeIdentityAccessError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, access.ErrAdminNotFound):
		httpx.WriteError(w, http.StatusNotFound, "identity_access_not_found", "The identity or access object was not found in this scope.")
	case errors.Is(err, access.ErrAdminMakerChecker):
		httpx.WriteError(w, http.StatusConflict, "identity_access_maker_checker", "A different administrator must approve this change.")
	case errors.Is(err, access.ErrAdminConflict):
		httpx.WriteError(w, http.StatusConflict, "identity_access_conflict", "The current state changed or the requested change conflicts with existing configuration.")
	case errors.Is(err, access.ErrAdminInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "identity_access_invalid", "The identity or access configuration is invalid.")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "identity_access_failed", "Identity and access configuration could not be updated.")
	}
}

func writeEscalationGuardError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, governance.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "escalation_guard_not_found", "The routing policy or pending guard revision was not found.")
	case errors.Is(err, governance.ErrVersionConflict), errors.Is(err, governance.ErrRevisionStale):
		httpx.WriteError(w, http.StatusConflict, "escalation_guard_stale", "The routing policy changed. Reload the current configuration before continuing.")
	case errors.Is(err, governance.ErrMakerChecker):
		httpx.WriteError(w, http.StatusConflict, "escalation_guard_maker_checker", "A different authorized principal must approve the latest guard revision, and another maker's pending revision cannot be overwritten.")
	case errors.Is(err, governance.ErrConflict):
		httpx.WriteError(w, http.StatusConflict, "escalation_guard_conflict", err.Error())
	case errors.Is(err, governance.ErrInvalidTransition):
		httpx.WriteError(w, http.StatusConflict, "escalation_guard_policy_state", "Escalation guard revisions require an active routing policy.")
	default:
		httpx.WriteError(w, http.StatusUnprocessableEntity, "escalation_guard_invalid", err.Error())
	}
}
