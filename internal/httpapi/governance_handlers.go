package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/CloudSpaceLab/clearsight-grc/internal/governance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) listGovernancePolicies(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	limit, ok := governanceInventoryLimit(w, r)
	if !ok {
		return
	}
	values, err := a.deps.Governance.ListPoliciesForEntity(r.Context(), actor.TenantID, actor.LegalEntityID, limit)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "governance_failed", "Routing policies could not be loaded.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": values})
}

func (a *API) createGovernancePolicy(w http.ResponseWriter, r *http.Request) {
	var input governance.CreatePolicyInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := a.deps.Governance.CreatePolicy(r.Context(), input)
	if err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "governance_invalid", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, value)
}

func (a *API) transitionGovernancePolicy(w http.ResponseWriter, r *http.Request) {
	var input governance.TransitionInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.ID = r.PathValue("id")
	var value governance.RoutingPolicy
	var err error
	switch r.PathValue("action") {
	case "submit":
		value, err = a.deps.Governance.SubmitPolicy(r.Context(), input)
	case "approve":
		value, err = a.deps.Governance.ApprovePolicy(r.Context(), input)
	case "reject":
		value, err = a.deps.Governance.RejectPolicy(r.Context(), input)
	case "retire":
		value, err = a.deps.Governance.RetirePolicy(r.Context(), input)
	default:
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Unknown policy action.")
		return
	}
	writeGovernanceResult(w, value, err)
}

func (a *API) listGovernanceDelegations(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	limit, ok := governanceInventoryLimit(w, r)
	if !ok {
		return
	}
	values, err := a.deps.Governance.ListDelegationsForEntity(r.Context(), actor.TenantID, actor.LegalEntityID, limit)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "governance_failed", "Delegations could not be loaded.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": values})
}

func (a *API) listGovernanceDelegationCandidates(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 50 {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_limit", "Limit must be between 1 and 50.")
			return
		}
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if utf8.RuneCountInString(query) > 100 {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_search", "Search must be 100 characters or fewer.")
		return
	}
	page, err := a.deps.Governance.SearchDelegationCandidates(r.Context(), actor.TenantID, actor.LegalEntityID, r.URL.Query().Get("responsibility"), query, limit)
	switch {
	case errors.Is(err, governance.ErrDelegationCandidateSearchInvalid):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_candidate_scope", "Choose a supported responsibility and a search of 100 characters or fewer.")
	case errors.Is(err, governance.ErrDelegationCandidatesUnavailable):
		httpx.WriteError(w, http.StatusServiceUnavailable, "delegation_candidates_unavailable", "Eligible people could not be confirmed from current responsibility records.")
	case err != nil:
		httpx.WriteError(w, http.StatusInternalServerError, "delegation_candidates_failed", "Eligible people could not be loaded.")
	default:
		httpx.WriteJSON(w, http.StatusOK, page)
	}
}

func governanceInventoryLimit(w http.ResponseWriter, r *http.Request) (int, bool) {
	value := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 200 {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_limit", "Limit must be between 1 and 200.")
			return 0, false
		}
		value = parsed
	}
	return value, true
}

func (a *API) createGovernanceDelegation(w http.ResponseWriter, r *http.Request) {
	var input governance.CreateDelegationInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := a.deps.Governance.CreateDelegation(r.Context(), input)
	if err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "governance_invalid", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, value)
}

func (a *API) transitionGovernanceDelegation(w http.ResponseWriter, r *http.Request) {
	var input governance.TransitionInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.ID = r.PathValue("id")
	var value governance.Delegation
	var err error
	switch r.PathValue("action") {
	case "submit":
		value, err = a.deps.Governance.SubmitDelegation(r.Context(), input)
	case "approve":
		value, err = a.deps.Governance.ApproveDelegation(r.Context(), input)
	case "revoke":
		value, err = a.deps.Governance.RevokeDelegation(r.Context(), input)
	default:
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Unknown delegation action.")
		return
	}
	writeGovernanceResult(w, value, err)
}

func writeGovernanceResult[T any](w http.ResponseWriter, value T, err error) {
	switch {
	case errors.Is(err, governance.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Governance object not found.")
	case errors.Is(err, governance.ErrVersionConflict):
		httpx.WriteError(w, http.StatusConflict, "version_conflict", "Governance object changed. Reload before updating.")
	case errors.Is(err, governance.ErrMakerChecker):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "maker_checker_failed", "The maker cannot perform the checker action.")
	case errors.Is(err, governance.ErrConflict):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "governance_conflict", err.Error())
	case errors.Is(err, governance.ErrInvalidTransition):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "invalid_transition", "The requested governance transition is not allowed.")
	case err != nil:
		httpx.WriteError(w, http.StatusUnprocessableEntity, "governance_failed", err.Error())
	default:
		httpx.WriteJSON(w, http.StatusOK, value)
	}
}
