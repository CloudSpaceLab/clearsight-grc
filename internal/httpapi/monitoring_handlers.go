package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

type createFormTemplateRequest struct {
	monitoring.CreateFormInput
	TenantID  string `json:"tenant_id,omitempty"`
	CreatedBy string `json:"created_by,omitempty"`
	ActorID   string `json:"actor_id,omitempty"`
}

type transitionFormTemplateRequest struct {
	monitoring.TransitionInput
	TenantID string `json:"tenant_id,omitempty"`
	ActorID  string `json:"actor_id,omitempty"`
}

type startFormCollectionRequest struct {
	monitoring.StartCollectionInput
	TenantID string `json:"tenant_id,omitempty"`
	ActorID  string `json:"actor_id,omitempty"`
}

type createMonitoringCheckRequest struct {
	monitoring.CreateCheckInput
	TenantID string `json:"tenant_id,omitempty"`
}

type transitionMonitoringCheckRequest struct {
	monitoring.TransitionInput
	TenantID string `json:"tenant_id,omitempty"`
}

type evaluateMonitoringSourceRequest struct {
	monitoring.EvaluateSourceInput
	TenantID string `json:"tenant_id,omitempty"`
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
	return monitoring.Actor{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PrincipalID: actor.PrincipalID}, nil
}

func (a *API) bindMonitoringProgram(r *http.Request, programID string) (monitoring.Actor, continuity.ProgramAggregate, error) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		return monitoring.Actor{}, continuity.ProgramAggregate{}, err
	}
	if a.deps.Continuity == nil {
		return monitoring.Actor{}, continuity.ProgramAggregate{}, fmt.Errorf("continuity service is unavailable")
	}
	aggregate, err := a.deps.Continuity.GetProgram(r.Context(), actor.TenantID, programID)
	if err != nil {
		return monitoring.Actor{}, continuity.ProgramAggregate{}, err
	}
	exactActor, err := a.exactRecordActor(r.Context(), actor, actor.TenantID, aggregate.Program.TenantID, aggregate.Program.LegalEntityID)
	if err != nil {
		return monitoring.Actor{}, continuity.ProgramAggregate{}, continuity.ErrNotFound
	}
	*r = *r.WithContext(identity.WithActor(r.Context(), exactActor))
	return monitoring.Actor{TenantID: exactActor.TenantID, LegalEntityID: exactActor.LegalEntityID, PrincipalID: exactActor.PrincipalID}, aggregate, nil
}

func (a *API) bindMonitoringCheck(r *http.Request, service *monitoring.Service, checkID string, version int64) (monitoring.Actor, monitoring.MonitoringCheck, continuity.ProgramAggregate, error) {
	actor, err := monitoringActor(r)
	if err != nil {
		return monitoring.Actor{}, monitoring.MonitoringCheck{}, continuity.ProgramAggregate{}, err
	}
	var check monitoring.MonitoringCheck
	if version > 0 {
		check, err = service.Check(r.Context(), actor, checkID, version)
	} else {
		check, err = service.LatestCheck(r.Context(), actor, checkID)
	}
	if err != nil {
		return monitoring.Actor{}, monitoring.MonitoringCheck{}, continuity.ProgramAggregate{}, err
	}
	exactActor, aggregate, err := a.bindMonitoringProgram(r, check.ProgramID)
	if err != nil {
		return monitoring.Actor{}, monitoring.MonitoringCheck{}, continuity.ProgramAggregate{}, err
	}
	return exactActor, check, aggregate, nil
}

func writeMonitoringScopeError(w http.ResponseWriter, err error) {
	if errors.Is(err, identity.ErrMissingIdentity) || errors.Is(err, identity.ErrInvalidIdentity) || errors.Is(err, identity.ErrExpiredIdentity) {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "Sign in is required.")
		return
	}
	if errors.Is(err, continuity.ErrNotFound) || errors.Is(err, monitoring.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "The monitoring record was not found.")
		return
	}
	httpx.WriteError(w, http.StatusServiceUnavailable, "monitoring_scope_unavailable", "The Program scope could not be checked. No monitoring data was returned.")
}

func (a *API) listFormTemplates(w http.ResponseWriter, r *http.Request) {
	service, ok := a.monitoringService(w)
	if !ok {
		return
	}
	actor, _, err := a.bindMonitoringProgram(r, r.PathValue("id"))
	if err != nil {
		writeMonitoringScopeError(w, err)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	values, err := service.ListForms(r.Context(), actor, r.PathValue("id"), limit)
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
	actor, aggregate, err := a.bindMonitoringProgram(r, r.PathValue("id"))
	if err != nil {
		writeMonitoringScopeError(w, err)
		return
	}
	var request createFormTemplateRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	request.ProgramID = aggregate.Program.ID
	request.LegalEntityID = aggregate.Program.LegalEntityID
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
	actor, aggregate, err := a.bindMonitoringProgram(r, r.PathValue("id"))
	if err != nil {
		writeMonitoringScopeError(w, err)
		return
	}
	var request transitionFormTemplateRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input := request.TransitionInput
	input.ID = r.PathValue("form_id")
	input.ProgramID = aggregate.Program.ID
	input.LegalEntityID = aggregate.Program.LegalEntityID
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
	actor, aggregate, err := a.bindMonitoringProgram(r, r.PathValue("id"))
	if err != nil {
		writeMonitoringScopeError(w, err)
		return
	}
	var request startFormCollectionRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input := request.StartCollectionInput
	input.FormTemplateID = r.PathValue("form_id")
	input.ProgramID = aggregate.Program.ID
	input.LegalEntityID = aggregate.Program.LegalEntityID
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
	actor, _, err := a.bindMonitoringProgram(r, r.PathValue("id"))
	if err != nil {
		writeMonitoringScopeError(w, err)
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
	actor, _, err := a.bindMonitoringProgram(r, r.PathValue("id"))
	if err != nil {
		writeMonitoringScopeError(w, err)
		return
	}
	var request createMonitoringCheckRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input := request.CreateCheckInput
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
	var request transitionMonitoringCheckRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input := request.TransitionInput
	input.ID = r.PathValue("id")
	actor, _, _, err := a.bindMonitoringCheck(r, service, input.ID, input.ExpectedVersion)
	if err != nil {
		writeMonitoringScopeError(w, err)
		return
	}
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
	actor, _, _, err := a.bindMonitoringCheck(r, service, r.PathValue("id"), 0)
	if err != nil {
		writeMonitoringScopeError(w, err)
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
	var request evaluateMonitoringSourceRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input := request.EvaluateSourceInput
	input.CheckID = r.PathValue("id")
	actor, _, _, err := a.bindMonitoringCheck(r, service, input.CheckID, input.CheckVersion)
	if err != nil {
		writeMonitoringScopeError(w, err)
		return
	}
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
