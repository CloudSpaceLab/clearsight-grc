package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formpolicy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

type formPolicyCreateRequest struct {
	formpolicy.CreateInput
	TenantID      string `json:"tenant_id,omitempty"`
	LegalEntityID string `json:"legal_entity_id,omitempty"`
	MakerID       string `json:"maker_id,omitempty"`
	CheckerID     string `json:"checker_id,omitempty"`
	ActorID       string `json:"actor_id,omitempty"`
}

type formPolicyActionRequest struct {
	ExpectedVersion int64  `json:"expected_version"`
	SimulationID    string `json:"simulation_id,omitempty"`
	TargetPolicyID  string `json:"target_policy_id,omitempty"`
	TenantID        string `json:"tenant_id,omitempty"`
	LegalEntityID   string `json:"legal_entity_id,omitempty"`
	MakerID         string `json:"maker_id,omitempty"`
	CheckerID       string `json:"checker_id,omitempty"`
	ActorID         string `json:"actor_id,omitempty"`
}

func (a *API) listFormPolicies(w http.ResponseWriter, r *http.Request) {
	service, actor, ok := a.formPolicyContext(w, r)
	if !ok {
		return
	}
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 200 {
			httpx.WriteError(w, http.StatusUnprocessableEntity, "policy_filters_invalid", "Limit must be between 1 and 200.")
			return
		}
		limit = parsed
	}
	values, err := service.List(r.Context(), actor, limit)
	if err != nil {
		writeFormPolicyError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": values})
}

func (a *API) getFormPolicy(w http.ResponseWriter, r *http.Request) {
	service, actor, ok := a.formPolicyContext(w, r)
	if !ok {
		return
	}
	value, err := service.Get(r.Context(), actor, r.PathValue("id"))
	if err != nil {
		writeFormPolicyError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) createFormPolicy(w http.ResponseWriter, r *http.Request) {
	service, actor, ok := a.formPolicyContext(w, r)
	if !ok {
		return
	}
	var request formPolicyCreateRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "The policy definition must be valid JSON.")
		return
	}
	value, err := service.Create(r.Context(), actor, request.CreateInput)
	if err != nil {
		writeFormPolicyError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, value)
}

func (a *API) simulateFormPolicy(w http.ResponseWriter, r *http.Request) {
	a.formPolicyAction(w, r, "simulate")
}
func (a *API) submitFormPolicy(w http.ResponseWriter, r *http.Request) {
	a.formPolicyAction(w, r, "submit")
}
func (a *API) approveFormPolicy(w http.ResponseWriter, r *http.Request) {
	a.formPolicyAction(w, r, "approve")
}
func (a *API) activateFormPolicy(w http.ResponseWriter, r *http.Request) {
	a.formPolicyAction(w, r, "activate")
}
func (a *API) suspendFormPolicy(w http.ResponseWriter, r *http.Request) {
	a.formPolicyAction(w, r, "suspend")
}
func (a *API) rollbackFormPolicy(w http.ResponseWriter, r *http.Request) {
	a.formPolicyAction(w, r, "rollback")
}

func (a *API) formPolicyAction(w http.ResponseWriter, r *http.Request, action string) {
	service, actor, ok := a.formPolicyContext(w, r)
	if !ok {
		return
	}
	var request formPolicyActionRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "The policy command must be valid JSON.")
		return
	}
	policyID := r.PathValue("id")
	switch action {
	case "simulate":
		value, err := service.Simulate(r.Context(), actor, policyID, request.ExpectedVersion)
		if err != nil {
			writeFormPolicyError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, value)
	case "submit":
		value, err := service.Submit(r.Context(), actor, policyID, request.ExpectedVersion, request.SimulationID)
		writeFormPolicyResult(w, value, err)
	case "approve":
		value, err := service.Approve(r.Context(), actor, policyID, request.ExpectedVersion, request.SimulationID)
		writeFormPolicyResult(w, value, err)
	case "activate":
		value, err := service.Activate(r.Context(), actor, policyID, request.ExpectedVersion)
		writeFormPolicyResult(w, value, err)
	case "suspend":
		value, err := service.Suspend(r.Context(), actor, policyID, request.ExpectedVersion)
		writeFormPolicyResult(w, value, err)
	case "rollback":
		value, err := service.Rollback(r.Context(), actor, policyID, request.ExpectedVersion, request.TargetPolicyID)
		writeFormPolicyResult(w, value, err)
	default:
		httpx.WriteError(w, http.StatusNotFound, "policy_action_not_found", "The requested policy action was not found.")
	}
}

func (a *API) formPolicyContext(w http.ResponseWriter, r *http.Request) (*formpolicy.Service, formpolicy.Actor, bool) {
	if a.deps.FormPolicies == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "policy_service_unavailable", "Response policies cannot be checked right now. No change was made.")
		return nil, formpolicy.Actor{}, false
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to continue.")
		return nil, formpolicy.Actor{}, false
	}
	if strings.TrimSpace(actor.LegalEntityID) == "" || actor.LegalEntityID == "*" {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "legal_entity_required", "Choose one legal entity before working with response policies.")
		return nil, formpolicy.Actor{}, false
	}
	return a.deps.FormPolicies, formpolicy.Actor{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PrincipalID: actor.PrincipalID}, true
}

func writeFormPolicyResult(w http.ResponseWriter, value formpolicy.Policy, err error) {
	if err != nil {
		writeFormPolicyError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func writeFormPolicyError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, formpolicy.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "policy_not_found", "The response policy was not found in this legal entity.")
	case errors.Is(err, formpolicy.ErrConflict):
		httpx.WriteError(w, http.StatusConflict, "policy_changed", "The response policy changed. Reload it before continuing.")
	case errors.Is(err, formpolicy.ErrMakerChecker):
		httpx.WriteError(w, http.StatusConflict, "independent_approval_required", "A different authorized person must approve this policy.")
	case errors.Is(err, formpolicy.ErrInvalid), errors.Is(err, formpolicy.ErrInvalidTransition), errors.Is(err, formpolicy.ErrPreviewRequired), errors.Is(err, formpolicy.ErrPreviewStale), errors.Is(err, formpolicy.ErrFormInactive), errors.Is(err, formpolicy.ErrShadowRequired):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "policy_not_ready", "Check the policy definition, current form revision, simulation, and rollout state before continuing.")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "policy_update_failed", "The response policy could not be updated. No change was made.")
	}
}
