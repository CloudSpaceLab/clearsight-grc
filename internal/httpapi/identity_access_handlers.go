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
)

type escalationPreviewInput struct {
	PolicyID        string   `json:"policy_id"`
	SequenceID      string   `json:"sequence_id"`
	DepartmentPath  []string `json:"department_path"`
	RevisionVersion int      `json:"revision_version,omitempty"`
}

type escalationPreviewStep struct {
	Index          int      `json:"index"`
	After          string   `json:"after"`
	Responsibility string   `json:"responsibility"`
	Scope          string   `json:"scope"`
	DepartmentPath []string `json:"department_path,omitempty"`
	SourceRoles    []string `json:"source_roles,omitempty"`
	TargetRoles    []string `json:"target_roles,omitempty"`
	TargetGroupIDs []string `json:"target_group_ids,omitempty"`
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
	policies := identityEscalationPolicies(r, a.deps.Governance, actor.TenantID)
	canConfigure := identity.HasPermission(actor, identity.PermissionIdentityConfigure)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"sign_in": map[string]any{
			"mode": a.deps.IdentityMode, "issuer": a.deps.OIDCIssuer,
			"authentication": actor.AuthenticationMethod, "assurance_level": actor.AssuranceLevel,
		},
		"actor_principal_id":       actor.PrincipalID,
		"can_configure":            canConfigure,
		"can_configure_escalation": canConfigure && identity.HasPermission(actor, identity.PermissionConfigWrite),
		"sources":                  overview.Sources,
		"people":                   overview.People,
		"groups":                   overview.Groups,
		"roles":                    overview.Roles,
		"legal_entities":           overview.LegalEntities,
		"bindings":                 overview.Bindings,
		"positions":                overview.Positions,
		"escalation":               overview.Escalation,
		"escalation_policies":      policies,
	})
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
	input.TenantID, input.ActorID = actor.TenantID, actor.PrincipalID
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
		TenantID: actor.TenantID, PolicyID: strings.TrimSpace(r.PathValue("policy_id")), RevisionVersion: revisionVersion,
		ActorID: actor.PrincipalID, ExpectedPolicyVersion: input.ExpectedPolicyVersion, Rationale: input.Rationale,
	})
	if err != nil {
		writeEscalationGuardError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, policy)
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
	policies, err := a.deps.Governance.ListPolicies(r.Context(), actor.TenantID)
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
					SourceRoles: append([]string(nil), step.SourceRoles...), TargetRoles: append([]string(nil), step.TargetRoles...), TargetGroupIDs: append([]string(nil), step.TargetGroupIDs...),
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

func identityEscalationPolicies(r *http.Request, service *governance.Service, tenant string) []map[string]any {
	if service == nil {
		return []map[string]any{}
	}
	policies, err := service.ListPolicies(r.Context(), tenant)
	if err != nil {
		return []map[string]any{}
	}
	result := make([]map[string]any, 0, len(policies))
	for _, policy := range policies {
		if policy.Status != governance.PolicyActive {
			continue
		}
		sequences, err := governance.ParseEscalationSequences(policy.Definition)
		if err != nil || len(sequences) == 0 {
			continue
		}
		item := map[string]any{
			"policy_id": policy.ID, "code": policy.Code, "name": policy.Name,
			"version": policy.CurrentVersion, "record_version": policy.Version, "sequences": sequences,
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
	case errors.Is(err, access.ErrAdminConflict):
		httpx.WriteError(w, http.StatusConflict, "identity_access_conflict", "The requested identity or access configuration already exists.")
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
