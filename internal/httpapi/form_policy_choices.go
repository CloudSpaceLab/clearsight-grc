package httpapi

import (
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"net/http"
	"strconv"
)

func (a *API) listFormPolicyExecutions(w http.ResponseWriter, r *http.Request) {
	service, actor, ok := a.formPolicyContext(w, r)
	if !ok {
		return
	}
	values, err := service.ExecutionHistory(r.Context(), actor, r.PathValue("id"))
	if err != nil {
		writeFormPolicyError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": values, "limit": 50})
}

func (a *API) listFormPolicyAutomationChoices(w http.ResponseWriter, r *http.Request) {
	service, actor, ok := a.formPolicyContext(w, r)
	if !ok {
		return
	}
	version, err := strconv.ParseInt(r.URL.Query().Get("form_template_version"), 10, 64)
	if err != nil || version < 1 {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "form_revision_required", "Choose an approved form revision to check its automation policies.")
		return
	}
	values, err := service.AutomationChoices(r.Context(), actor, r.URL.Query().Get("form_template_id"), version)
	if err != nil {
		writeFormPolicyError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": values})
}
