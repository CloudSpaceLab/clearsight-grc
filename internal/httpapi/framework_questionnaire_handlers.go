package httpapi

import (
	"errors"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"net/http"
)

func (a *API) frameworkQuestionnaireScope(w http.ResponseWriter, r *http.Request) bool {
	_, aggregate, err := a.bindMonitoringProgram(r, r.PathValue("id"))
	if err != nil {
		writeMonitoringScopeError(w, err)
		return false
	}
	visible, err := a.programForActor(r.Context(), aggregate, nil)
	if err != nil {
		writeContinuityError(w, continuity.ErrNotFound)
		return false
	}
	for _, item := range visible.FrameworkAdoptions {
		if item.ID == r.PathValue("adoption_id") && (item.Status == continuity.FrameworkAdoptionDraft || item.Status == continuity.FrameworkAdoptionPending || item.Status == continuity.FrameworkAdoptionActive) {
			return true
		}
	}
	writeContinuityError(w, continuity.ErrNotFound)
	return false
}

func (a *API) getFrameworkQuestionnaire(w http.ResponseWriter, r *http.Request) {
	if !a.frameworkQuestionnaireScope(w, r) {
		return
	}
	service, ok := a.monitoringService(w)
	if !ok {
		return
	}
	item, err := service.ReadFrameworkQuestionnaire(r.Context(), r.PathValue("id"), r.PathValue("adoption_id"))
	if errors.Is(err, monitoring.ErrNotFound) {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"item": nil})
		return
	}
	if err != nil {
		writeFormsError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (a *API) selectFrameworkQuestionnaire(w http.ResponseWriter, r *http.Request) {
	if !a.frameworkQuestionnaireScope(w, r) {
		return
	}
	service, ok := a.monitoringService(w)
	if !ok {
		return
	}
	var input struct {
		TenantID        string `json:"tenant_id,omitempty"`
		FormID          string `json:"form_template_id"`
		FormVersion     int64  `json:"form_template_version"`
		ExpectedVersion int64  `json:"expected_version"`
	}
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_questionnaire", "Check the questionnaire selection.")
		return
	}
	item, err := service.SelectFrameworkQuestionnaire(r.Context(), r.PathValue("id"), r.PathValue("adoption_id"), input.FormID, input.FormVersion, input.ExpectedVersion)
	if err != nil {
		writeFormsError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, item)
}
