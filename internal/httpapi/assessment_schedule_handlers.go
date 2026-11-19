package httpapi

import (
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) listAssessmentSchedules(w http.ResponseWriter, r *http.Request) {
	service, ok := a.monitoringService(w)
	if !ok {
		return
	}
	actor, program, err := a.bindMonitoringProgram(r, r.PathValue("id"))
	if err != nil {
		writeMonitoringScopeError(w, err)
		return
	}
	if _, err = a.programForActor(r.Context(), program, nil); err != nil {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Program not found.")
		return
	}
	var items []monitoring.AssessmentSchedule
	if id := r.URL.Query().Get("schedule_id"); id != "" {
		var selected monitoring.AssessmentSchedule
		selected, err = service.FindAssessmentSchedule(r.Context(), actor, program.Program.ID, id)
		if err == nil {
			items = []monitoring.AssessmentSchedule{selected}
		}
	} else {
		items, err = service.ListAssessmentSchedules(r.Context(), actor, program.Program.ID)
	}
	if err != nil {
		writeMonitoringError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": items, "can_propose": actor.PrincipalID == program.Program.OwnerPrincipalID && program.Program.Status == "ACTIVE",
		"can_approve": actor.PrincipalID == program.Program.AuthorityPrincipalID && program.Program.Status == "ACTIVE",
		"can_stop":    actor.PrincipalID == program.Program.OwnerPrincipalID,
	})
}

func (a *API) proposeAssessmentSchedule(w http.ResponseWriter, r *http.Request) {
	service, ok := a.monitoringService(w)
	if !ok {
		return
	}
	actor, program, err := a.bindMonitoringProgram(r, r.PathValue("id"))
	if err != nil {
		writeMonitoringScopeError(w, err)
		return
	}
	var input struct {
		TenantID      string `json:"tenant_id,omitempty"`
		LegalEntityID string `json:"legal_entity_id,omitempty"`
		monitoring.AssessmentScheduleInput
	}
	if err = httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Check the schedule dates and assignment.")
		return
	}
	schedule, err := service.ProposeAssessmentSchedule(r.Context(), actor, program.Program.ID, input.AssessmentScheduleInput)
	if err != nil {
		writeMonitoringError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, schedule)
}

func (a *API) approveAssessmentSchedule(w http.ResponseWriter, r *http.Request) {
	a.changeAssessmentSchedule(w, r, "ACTIVE")
}

func (a *API) stopAssessmentSchedule(w http.ResponseWriter, r *http.Request) {
	a.changeAssessmentSchedule(w, r, "STOPPED")
}

func (a *API) changeAssessmentSchedule(w http.ResponseWriter, r *http.Request, state string) {
	service, ok := a.monitoringService(w)
	if !ok {
		return
	}
	actor, program, err := a.bindMonitoringProgram(r, r.PathValue("id"))
	if err != nil {
		writeMonitoringScopeError(w, err)
		return
	}
	var input struct {
		TenantID      string `json:"tenant_id,omitempty"`
		LegalEntityID string `json:"legal_entity_id,omitempty"`
		Version       int64  `json:"expected_version"`
		Reason        string `json:"reason"`
	}
	if err = httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Enter a reason for the schedule decision.")
		return
	}
	schedule, err := service.ChangeAssessmentSchedule(r.Context(), actor, program.Program.ID, r.PathValue("schedule_id"), state, input.Reason, input.Version)
	if err != nil {
		writeMonitoringError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, schedule)
}
