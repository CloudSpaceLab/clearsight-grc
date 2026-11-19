package httpapi

import (
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"net/http"
	"strconv"
)

func (a *API) getProgramActivity(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	_, aggregate, err := a.bindMonitoringProgram(r, r.PathValue("id"))
	if err != nil {
		writeMonitoringScopeError(w, err)
		return
	}
	if _, err = a.programForActor(r.Context(), aggregate, nil); err != nil {
		httpx.WriteError(w, 404, "not_found", "Program not found.")
		return
	}
	params := r.URL.Query()
	limit := 25
	if params.Has("limit") {
		limit, err = strconv.Atoi(params.Get("limit"))
		if err != nil || limit < 1 || limit > 100 {
			httpx.WriteError(w, 400, "invalid_query", "Choose between 1 and 100 activity records per page.")
			return
		}
	}
	page, err := service.ProgramActivity(r.Context(), aggregate.Program.ID, evidence.ProgramActivityQuery{Search: params.Get("search"), EventType: params.Get("event_type"), Category: params.Get("category"), From: params.Get("from"), To: params.Get("to"), Cursor: params.Get("cursor"), Limit: limit})
	if err == evidence.ErrDraftInvalid {
		httpx.WriteError(w, 400, "invalid_query", "Check the activity filters or restart the history search.")
		return
	}
	if writeAssessmentResponseError(w, err) {
		return
	}
	httpx.WriteJSON(w, 200, page)
}
