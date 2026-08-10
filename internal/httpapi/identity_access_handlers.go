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
	PolicyID       string   `json:"policy_id"`
	SequenceID     string   `json:"sequence_id"`
	DepartmentPath []string `json:"department_path"`
}

type escalationPreviewStep struct {
	Index          int      `json:"index"`
	After          string   `json:"after"`
	Responsibility string   `json:"responsibility"`
	Scope          string   `json:"scope"`
	DepartmentPath []string `json:"department_path,omitempty"`
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
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"sign_in": map[string]any{
			"mode": a.deps.IdentityMode, "issuer": a.deps.OIDCIssuer,
			"authentication": actor.AuthenticationMethod, "assurance_level": actor.AssuranceLevel,
		},
		"can_configure": identity.HasPermission(actor, identity.PermissionIdentityConfigure),
		"sources": overview.Sources, "people": overview.People, "groups": overview.Groups,
		"roles": overview.Roles, "legal_entities": overview.LegalEntities, "bindings": overview.Bindings,
		"escalation": overview.Escalation, "escalation_policies": policies,
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
	if err := a.deps.AccessAdmin.RetireGroupRoleBinding(r.Context(), actor.TenantID, actor.LegalEntityID, r.PathValue("id"), actor.PrincipalID); err != nil {
		writeIdentityAccessError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
		sequences, parseErr := governance.ParseEscalationSequences(policy.Definition)
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
				preview := escalationPreviewStep{Index: index, After: step.After.String(), Responsibility: step.Responsibility, Scope: "LEGAL_ENTITY"}
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
				"policy_id": policy.ID, "policy_code": policy.Code, "policy_version": policy.CurrentVersion,
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
		result = append(result, map[string]any{
			"policy_id": policy.ID, "code": policy.Code, "name": policy.Name,
			"version": policy.CurrentVersion, "sequences": sequences,
		})
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
