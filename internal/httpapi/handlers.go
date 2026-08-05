package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/capture"
	"github.com/CloudSpaceLab/clearsight-grc/internal/onboarding"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
)

func (a *API) live(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "live"})
}
func (a *API) ready(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready", "mode": a.deps.Mode})
}
func (a *API) context(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"tenant": map[string]string{"id": "bank-demo", "name": "ClearSight Demonstration Bank"}, "legal_entity": map[string]string{"id": "bank-ng", "name": "Demonstration Bank Nigeria"}, "actor": map[string]string{"id": "user-demo", "name": "Amaka Okafor", "role": "Control Assurance Lead"}, "mode": a.deps.Mode})
}
func (a *API) today(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": a.deps.Today.List(), "generated_at": time.Now().UTC()})
}
func (a *API) resolveAuthority(w http.ResponseWriter, r *http.Request) {
	var input authority.ResolveInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	resolution, err := a.deps.Authority.Resolve(r.Context(), input)
	if errors.Is(err, authority.ErrNoRoute) {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "routing_failed", "No eligible route exists for the supplied scope and responsibility.")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "resolution_failed", "Authority could not be resolved.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, resolution)
}
func (a *API) simulateAuthority(w http.ResponseWriter, r *http.Request) {
	var input authority.ResolveInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := a.deps.Authority.Simulate(r.Context(), input)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "simulation_failed", "Routing could not be simulated.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}
func (a *API) authorityIntegrity(w http.ResponseWriter, r *http.Request) {
	findings, err := a.deps.Authority.Integrity(r.Context(), r.URL.Query().Get("tenant_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "integrity_failed", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"findings": findings, "checked_at": time.Now().UTC()})
}
func (a *API) authorityPolicies(w http.ResponseWriter, r *http.Request) {
	values, err := a.deps.Authority.Policies(r.Context(), r.URL.Query().Get("tenant_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "policies_failed", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": values})
}
func (a *API) getCaptureRequest(w http.ResponseWriter, r *http.Request) {
	request, err := a.deps.Capture.Get(r.PathValue("id"))
	if errors.Is(err, capture.ErrRequestNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "The request was not found.")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "request_failed", "The request could not be loaded.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, request)
}
func (a *API) submitCaptureRequest(w http.ResponseWriter, r *http.Request) {
	var submission capture.Submission
	if err := httpx.DecodeJSON(w, r, &submission); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	receipt, err := a.deps.Capture.Submit(r.PathValue("id"), submission)
	var validation capture.ValidationError
	switch {
	case errors.As(err, &validation):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_failed", validation.Error())
	case errors.Is(err, capture.ErrRequestNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "The request was not found.")
	case errors.Is(err, capture.ErrVersionConflict):
		httpx.WriteError(w, http.StatusConflict, "version_conflict", "The request changed. Reload before submitting.")
	case errors.Is(err, capture.ErrRequestClosed):
		httpx.WriteError(w, http.StatusConflict, "request_closed", "The request is no longer open.")
	case err != nil:
		httpx.WriteError(w, http.StatusInternalServerError, "submission_failed", "The response could not be submitted.")
	default:
		httpx.WriteJSON(w, http.StatusOK, receipt)
	}
}
func (a *API) redeemInvitation(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
	}
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	session, err := a.deps.Invitations.Redeem(body.Token)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "invitation_unavailable", "This invitation is unavailable. Request a new invitation from the sender.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, session)
}
func (a *API) listWorkflowTasks(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	values, err := a.deps.Workflow.List(r.Context(), workflow.ListFilter{TenantID: r.URL.Query().Get("tenant_id"), PrincipalID: r.URL.Query().Get("principal_id"), Status: workflow.Status(r.URL.Query().Get("status")), Limit: limit})
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "workflow_failed", "Tasks could not be loaded.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": values})
}
func (a *API) createWorkflowTask(w http.ResponseWriter, r *http.Request) {
	var input workflow.CreateInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := a.deps.Workflow.Create(r.Context(), input)
	if err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "workflow_invalid", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, value)
}
func (a *API) transitionWorkflowTask(w http.ResponseWriter, r *http.Request) {
	var input workflow.TransitionInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := a.deps.Workflow.Transition(r.Context(), r.PathValue("id"), input)
	switch {
	case errors.Is(err, workflow.ErrTaskNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Task not found.")
	case errors.Is(err, workflow.ErrVersionConflict):
		httpx.WriteError(w, http.StatusConflict, "version_conflict", "Task changed. Reload before updating.")
	case errors.Is(err, workflow.ErrInvalidTransition):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "invalid_transition", "The requested transition is not allowed.")
	case err != nil:
		httpx.WriteError(w, http.StatusInternalServerError, "workflow_failed", "Task could not be updated.")
	default:
		httpx.WriteJSON(w, http.StatusOK, value)
	}
}
func (a *API) onboardingGuide(w http.ResponseWriter, r *http.Request) {
	guide, err := a.deps.Onboarding.Guide(r.URL.Query().Get("role"), r.URL.Query().Get("code"))
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Guide not found.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, guide)
}
func (a *API) onboardingState(w http.ResponseWriter, r *http.Request) {
	value, err := a.deps.Onboarding.State(r.Context(), r.URL.Query().Get("tenant_id"), r.URL.Query().Get("principal_id"), r.URL.Query().Get("guide_code"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "state_failed", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}
func (a *API) updateOnboardingState(w http.ResponseWriter, r *http.Request) {
	var input onboarding.UpdateInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := a.deps.Onboarding.Update(r.Context(), r.URL.Query().Get("tenant_id"), r.URL.Query().Get("principal_id"), r.URL.Query().Get("guide_code"), input)
	if errors.Is(err, onboarding.ErrVersionConflict) {
		httpx.WriteError(w, http.StatusConflict, "version_conflict", "Onboarding state changed. Reload before updating.")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "state_failed", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}
func (a *API) readiness(w http.ResponseWriter, r *http.Request) {
	value, err := a.deps.Autonomy.Readiness(r.Context(), r.URL.Query().Get("tenant_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "readiness_failed", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}
func (a *API) ingestSignal(w http.ResponseWriter, r *http.Request) {
	var input autonomy.Signal
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	drift, inserted, err := a.deps.Autonomy.Ingest(r.Context(), input)
	if err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "signal_failed", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, map[string]any{"inserted": inserted, "drift": drift})
}
