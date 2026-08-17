package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

type createFormTemplateRequest struct {
	monitoring.CreateFormInput
	TenantID  string `json:"tenant_id,omitempty"`
	CreatedBy string `json:"created_by,omitempty"`
}

func (a *API) monitoringService(w http.ResponseWriter) (*monitoring.Service, bool) {
	if a.deps.Monitoring == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "monitoring_unavailable", "Monitoring is temporarily unavailable.")
		return nil, false
	}
	return a.deps.Monitoring, true
}

func monitoringActor(r *http.Request) (monitoring.Actor, error) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		return monitoring.Actor{}, err
	}
	return monitoring.Actor{TenantID: actor.TenantID, PrincipalID: actor.PrincipalID}, nil
}

func (a *API) listFormTemplates(w http.ResponseWriter, r *http.Request) {
	service, ok := a.monitoringService(w)
	if !ok {
		return
	}
	actor, err := monitoringActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "Sign in is required.")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	values, err := service.ListForms(r.Context(), actor, limit)
	if err != nil {
		writeMonitoringError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": values})
}

func (a *API) createFormTemplate(w http.ResponseWriter, r *http.Request) {
	service, ok := a.monitoringService(w)
	if !ok {
		return
	}
	actor, err := monitoringActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "Sign in is required.")
		return
	}
	var request createFormTemplateRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := service.CreateForm(r.Context(), actor, request.CreateFormInput)
	if err != nil {
		writeMonitoringError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, value)
}

func (a *API) transitionFormTemplate(w http.ResponseWriter, r *http.Request) {
	service, ok := a.monitoringService(w)
	if !ok {
		return
	}
	actor, err := monitoringActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "Sign in is required.")
		return
	}
	var input monitoring.TransitionInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.ID = r.PathValue("id")
	value, err := service.TransitionForm(r.Context(), actor, input)
	if err != nil {
		writeMonitoringError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) startFormCollection(w http.ResponseWriter, r *http.Request) {
	service, ok := a.monitoringService(w)
	if !ok {
		return
	}
	actor, err := monitoringActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "Sign in is required.")
		return
	}
	var input monitoring.StartCollectionInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.FormTemplateID = r.PathValue("id")
	value, err := service.StartCollection(r.Context(), actor, input)
	if err != nil {
		writeMonitoringError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, value)
}

func (a *API) listMonitoringChecks(w http.ResponseWriter, r *http.Request) {
	service, ok := a.monitoringService(w)
	if !ok {
		return
	}
	actor, err := monitoringActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "Sign in is required.")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	values, err := service.ListChecks(r.Context(), actor, r.PathValue("id"), limit)
	if err != nil {
		writeMonitoringError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": values})
}

func (a *API) createMonitoringCheck(w http.ResponseWriter, r *http.Request) {
	service, ok := a.monitoringService(w)
	if !ok {
		return
	}
	actor, err := monitoringActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "Sign in is required.")
		return
	}
	var input monitoring.CreateCheckInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.ProgramID = r.PathValue("id")
	value, err := service.CreateCheck(r.Context(), actor, input)
	if err != nil {
		writeMonitoringError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, value)
}

func (a *API) transitionMonitoringCheck(w http.ResponseWriter, r *http.Request) {
	service, ok := a.monitoringService(w)
	if !ok {
		return
	}
	actor, err := monitoringActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "Sign in is required.")
		return
	}
	var input monitoring.TransitionInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.ID = r.PathValue("id")
	value, err := service.TransitionCheck(r.Context(), actor, input)
	if err != nil {
		writeMonitoringError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) listMonitoringResults(w http.ResponseWriter, r *http.Request) {
	service, ok := a.monitoringService(w)
	if !ok {
		return
	}
	actor, err := monitoringActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "Sign in is required.")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	values, err := service.ListResults(r.Context(), actor, r.PathValue("id"), limit)
	if err != nil {
		writeMonitoringError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": values})
}

func (a *API) evaluateMonitoringSource(w http.ResponseWriter, r *http.Request) {
	service, ok := a.monitoringService(w)
	if !ok {
		return
	}
	actor, err := monitoringActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "Sign in is required.")
		return
	}
	var input monitoring.EvaluateSourceInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.CheckID = r.PathValue("id")
	value, err := service.EvaluateSource(r.Context(), actor, input)
	if err != nil {
		writeMonitoringError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, value)
}

func writeMonitoringError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, monitoring.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "The monitoring record was not found.")
	case errors.Is(err, monitoring.ErrConflict), errors.Is(err, monitoring.ErrMakerChecker), errors.Is(err, monitoring.ErrInactive):
		httpx.WriteError(w, http.StatusConflict, "monitoring_conflict", err.Error())
	case errors.Is(err, monitoring.ErrInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "monitoring_invalid", err.Error())
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "monitoring_failed", "The monitoring change could not be completed.")
	}
}
