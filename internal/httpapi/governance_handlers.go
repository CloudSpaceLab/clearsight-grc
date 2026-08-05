package httpapi

import (
	"errors"
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/governance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) listGovernancePolicies(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requiredQuery(w, r, "tenant_id")
	if !ok {
		return
	}
	values, err := a.deps.Governance.ListPolicies(r.Context(), tenantID)
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
	tenantID, ok := requiredQuery(w, r, "tenant_id")
	if !ok {
		return
	}
	values, err := a.deps.Governance.ListDelegations(r.Context(), tenantID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "governance_failed", "Delegations could not be loaded.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": values})
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
