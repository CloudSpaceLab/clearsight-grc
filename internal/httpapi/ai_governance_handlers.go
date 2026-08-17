package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/aigovernance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) listAIGovernancePolicies(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to continue.")
		return
	}
	items, err := a.deps.AIGovernance.ListPolicies(r.Context(), actor.TenantID, queryLimit(r, 50))
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "ai_governance_failed", "AI governance policies could not be loaded.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}
func (a *API) getAIGovernancePolicy(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		return
	}
	v, err := a.deps.AIGovernance.GetPolicy(r.Context(), actor.TenantID, r.PathValue("id"))
	writeAIGovernanceResult(w, v, err, http.StatusOK)
}
func (a *API) createAIGovernancePolicy(w http.ResponseWriter, r *http.Request) {
	var in aigovernance.CreatePolicyInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	v, err := a.deps.AIGovernance.CreatePolicy(r.Context(), in)
	writeAIGovernanceResult(w, v, err, http.StatusCreated)
}
func (a *API) aiGovernancePolicyAction(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, err := identity.Require(r.Context())
		if err != nil {
			return
		}
		var in aigovernance.TransitionInput
		if err := httpx.DecodeJSON(w, r, &in); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		in.ID = r.PathValue("id")
		in.TenantID = actor.TenantID
		in.ActorID = actor.PrincipalID
		v, err := a.deps.AIGovernance.TransitionPolicy(r.Context(), action, in)
		writeAIGovernanceResult(w, v, err, http.StatusOK)
	}
}

func (a *API) listAIGovernanceWorkloads(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		return
	}
	items, err := a.deps.AIGovernance.ListWorkloads(r.Context(), actor.TenantID, queryLimit(r, 50))
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "ai_governance_failed", "AI workloads could not be loaded.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}
func (a *API) getAIGovernanceWorkload(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		return
	}
	v, err := a.deps.AIGovernance.GetWorkload(r.Context(), actor.TenantID, r.PathValue("id"))
	writeAIGovernanceResult(w, v, err, http.StatusOK)
}
func (a *API) createAIGovernanceWorkload(w http.ResponseWriter, r *http.Request) {
	var in aigovernance.CreateWorkloadInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	v, err := a.deps.AIGovernance.CreateWorkload(r.Context(), in)
	writeAIGovernanceResult(w, v, err, http.StatusCreated)
}
func (a *API) aiGovernanceWorkloadAction(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, err := identity.Require(r.Context())
		if err != nil {
			return
		}
		var in aigovernance.TransitionInput
		if err := httpx.DecodeJSON(w, r, &in); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		in.ID = r.PathValue("id")
		in.TenantID = actor.TenantID
		in.ActorID = actor.PrincipalID
		v, err := a.deps.AIGovernance.TransitionWorkload(r.Context(), action, in)
		writeAIGovernanceResult(w, v, err, http.StatusOK)
	}
}

func (a *API) ingestAIGovernanceReceipt(w http.ResponseWriter, r *http.Request) {
	var in aigovernance.DecisionReceipt
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	inserted, err := a.deps.AIGovernance.IngestReceipt(r.Context(), in)
	if err != nil {
		writeAIGovernanceResult(w, map[string]any{}, err, http.StatusAccepted)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, map[string]any{"accepted": inserted, "duplicate": !inserted})
}
func (a *API) createAIGovernanceGrant(w http.ResponseWriter, r *http.Request) {
	var in aigovernance.CreateGrantInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	v, err := a.deps.AIGovernance.CreateExecutionGrant(r.Context(), in)
	writeAIGovernanceResult(w, v, err, http.StatusCreated)
}

func writeAIGovernanceResult[T any](w http.ResponseWriter, v T, err error, status int) {
	switch {
	case errors.Is(err, aigovernance.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "AI governance object not found.")
	case errors.Is(err, aigovernance.ErrConflict):
		httpx.WriteError(w, http.StatusConflict, "version_conflict", "AI governance state changed. Reload before updating.")
	case errors.Is(err, aigovernance.ErrMakerChecker):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "maker_checker_failed", "The maker cannot perform the checker action.")
	case errors.Is(err, aigovernance.ErrInvalidTransition):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "invalid_transition", "The requested AI governance transition is not allowed.")
	case errors.Is(err, aigovernance.ErrGrantInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "grant_invalid", "The execution grant is not backed by the exact approved Matter decision/action.")
	case err != nil:
		httpx.WriteError(w, http.StatusUnprocessableEntity, "ai_governance_invalid", err.Error())
	default:
		httpx.WriteJSON(w, status, v)
	}
}
func queryLimit(r *http.Request, fallback int) int {
	v := strings.TrimSpace(r.URL.Query().Get("limit"))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return fallback
	}
	if n > 200 {
		return 200
	}
	return n
}
