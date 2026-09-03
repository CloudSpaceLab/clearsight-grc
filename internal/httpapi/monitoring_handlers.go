package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

type createFormTemplateRequest struct {
	monitoring.CreateFormInput
	TenantID  string `json:"tenant_id,omitempty"`
	CreatedBy string `json:"created_by,omitempty"`
}

type updateCollectionPolicyRequest struct {
	monitoring.UpdateCollectionPolicyInput
	TenantID string `json:"tenant_id,omitempty"`
	ActorID  string `json:"actor_id,omitempty"`
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

func (a *API) updateCollectionPolicy(w http.ResponseWriter, r *http.Request) {
	service, ok := a.monitoringService(w)
	if !ok {
		return
	}
	actor, err := monitoringActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "Sign in is required.")
		return
	}
	var request updateCollectionPolicyRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input := request.UpdateCollectionPolicyInput
	input.ID = r.PathValue("id")
	value, err := service.UpdateCollectionPolicy(r.Context(), actor, input)
	if err != nil {
		writeMonitoringError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

type collectionSummaryResponse struct {
	MonitoringCheckID       string                   `json:"monitoring_check_id"`
	LatestRequestID         string                   `json:"latest_request_id,omitempty"`
	LatestSubmissionID      string                   `json:"latest_submission_id,omitempty"`
	LatestSubmissionAt      *time.Time               `json:"latest_submission_at,omitempty"`
	RespondentLabel         string                   `json:"respondent_label,omitempty"`
	RecipientHint           string                   `json:"recipient_hint,omitempty"`
	ExpiresAt               time.Time                `json:"expires_at"`
	RenewalOpensAt          time.Time                `json:"renewal_opens_at"`
	CurrencyState           string                   `json:"currency_state"`
	ActiveRequestDeadline   *time.Time               `json:"active_request_deadline,omitempty"`
	RemindersSent           int                      `json:"reminders_sent"`
	ReminderCount           int                      `json:"reminder_count"`
	DeliveryState           monitoring.DeliveryState `json:"delivery_state"`
	LastErrorSafe           string                   `json:"last_error_safe,omitempty"`
	ProjectionGeneratedAt   time.Time                `json:"projection_generated_at"`
	ProjectionSourceVersion int64                    `json:"projection_source_version"`
}

func (a *API) listCollectionSummaries(w http.ResponseWriter, r *http.Request) {
	service, ok := a.monitoringService(w)
	if !ok {
		return
	}
	programs, ok := a.continuityService(w)
	if !ok {
		return
	}
	actor, err := monitoringActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "Sign in is required.")
		return
	}
	programID := r.PathValue("id")
	if _, err := programs.GetProgram(r.Context(), actor.TenantID, programID); err != nil {
		writeContinuityError(w, err)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	values, err := service.ListCollectionSummaries(r.Context(), actor, programID, limit)
	if err != nil {
		writeMonitoringError(w, err)
		return
	}
	now := time.Now().UTC()
	items := make([]collectionSummaryResponse, len(values))
	for index, value := range values {
		items[index] = collectionSummaryView(value, now)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func collectionSummaryView(value monitoring.CollectionSummary, now time.Time) collectionSummaryResponse {
	view := collectionSummaryResponse{
		MonitoringCheckID: value.MonitoringCheckID, LatestRequestID: value.CurrentRequestID,
		LatestSubmissionID: value.LatestSubmissionID, LatestSubmissionAt: value.LatestSubmittedAt,
		RespondentLabel: value.LatestRespondentLabel, ExpiresAt: value.ExpiresAt, RenewalOpensAt: value.RenewalOpensAt,
		CurrencyState: collectionCurrencyState(value, now), RemindersSent: value.RemindersSent, ReminderCount: value.Policy.ReminderCount,
		DeliveryState: value.DeliveryState, LastErrorSafe: value.SafeError, ProjectionGeneratedAt: value.GeneratedAt,
		ProjectionSourceVersion: value.MonitoringCheckVersion,
	}
	if value.Recipient.Type == monitoring.RouteExternalContact {
		view.RecipientHint = value.Recipient.SafeHint
	}
	if value.State == monitoring.CycleAwaitingResponse && value.CurrentRequestID != "" {
		deadline := value.ExpiresAt
		view.ActiveRequestDeadline = &deadline
	}
	return view
}

func collectionCurrencyState(value monitoring.CollectionSummary, now time.Time) string {
	if value.State == monitoring.CycleBlocked || value.State == monitoring.CycleFailed || value.DeliveryState == monitoring.DeliveryBlocked || value.DeliveryState == monitoring.DeliveryFailed {
		return "RENEWAL_BLOCKED"
	}
	if value.State == monitoring.CycleAwaitingResponse {
		return "AWAITING_RESPONSE"
	}
	if value.LatestSubmittedAt == nil {
		return "NO_RESPONSE_SUBMITTED"
	}
	if !now.Before(value.ExpiresAt) {
		return "RESPONSE_POTENTIALLY_EXPIRED"
	}
	if !now.Before(value.RenewalOpensAt) {
		return "RENEWAL_DUE"
	}
	return "CURRENT"
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
