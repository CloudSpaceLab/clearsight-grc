package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) matterFormRemediationService(w http.ResponseWriter) (*continuity.MatterFormRemediationService, bool) {
	if a.deps.MatterFormRemediation == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "issue_form_workflow_unavailable", "Linked form requests are unavailable for this issue. Try again after the service is restored.")
		return nil, false
	}
	return a.deps.MatterFormRemediation, true
}

func (a *API) listMatterFormRemediations(w http.ResponseWriter, r *http.Request) {
	service, ok := a.matterFormRemediationService(w)
	if !ok {
		return
	}
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_limit", "Choose a limit from 1 to 100.")
			return
		}
		limit = parsed
	}
	values, err := service.List(r.Context(), r.PathValue("id"), limit)
	if err != nil {
		writeMatterFormRemediationError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": values})
}

func (a *API) createMatterFormRemediation(w http.ResponseWriter, r *http.Request) {
	service, ok := a.matterFormRemediationService(w)
	if !ok {
		return
	}
	var input continuity.CreateMatterFormBindingInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := service.CreateBinding(r.Context(), r.PathValue("id"), input)
	if err != nil {
		writeMatterFormRemediationError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, value)
}

func (a *API) sendMatterFormRemediation(w http.ResponseWriter, r *http.Request) {
	service, ok := a.matterFormRemediationService(w)
	if !ok {
		return
	}
	var input continuity.SendMatterFormInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := service.Send(r.Context(), r.PathValue("id"), r.PathValue("binding_id"), input)
	if err != nil {
		writeMatterFormRemediationError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) applyMatterFormRemediation(w http.ResponseWriter, r *http.Request) {
	service, ok := a.matterFormRemediationService(w)
	if !ok {
		return
	}
	var input continuity.ApplyMatterFormResponseInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	matter, application, err := service.Apply(r.Context(), r.PathValue("id"), r.PathValue("binding_id"), input)
	if err != nil {
		writeMatterFormRemediationError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"matter": matter, "application": application})
}

func writeMatterFormRemediationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, continuity.ErrNotFound), errors.Is(err, evidence.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "issue_form_workflow_not_found", "This linked form request is not available for the current issue and legal entity.")
	case errors.Is(err, continuity.ErrVersionConflict):
		httpx.WriteError(w, http.StatusConflict, "issue_changed", "This issue changed after the page loaded. Reload it before continuing.")
	case errors.Is(err, continuity.ErrMatterFormBindingInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "issue_form_binding_invalid", "Choose a current approved form and map each field to distinct missing information for this open issue.")
	case errors.Is(err, continuity.ErrMatterFormResponseRejected):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "issue_form_response_not_ready", "The latest final response does not yet meet this issue's mapped information and score requirements.")
	case errors.Is(err, evidence.ErrDistributionInvalid), errors.Is(err, evidence.ErrDistributionAccessUnavailable):
		httpx.WriteError(w, http.StatusServiceUnavailable, "issue_form_delivery_unavailable", "The linked form could not be sent. Check form access delivery and try again.")
	default:
		writeContinuityError(w, err)
	}
}
