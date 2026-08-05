package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) projectionHealth(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	tenant, ok := requiredQuery(w, r, "tenant_id")
	if !ok {
		return
	}
	values, err := service.ProjectionHealth(r.Context(), tenant)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "status_updates_failed", "Program status update health could not be loaded.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": values})
}

func (a *API) reconcileProgramState(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input struct {
		TenantID string `json:"tenant_id"`
		Limit    int    `json:"limit"`
	}
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := service.ReconcileProgramState(r.Context(), input.TenantID, input.Limit)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "reconcile_failed", "Program status records could not be checked. No existing status was changed.")
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, result)
}

func (a *API) rebuildProgramState(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input struct {
		TenantID  string `json:"tenant_id"`
		ProgramID string `json:"program_id"`
		Reason    string `json:"reason"`
	}
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	actor, _ := identity.FromContext(r.Context())
	job, err := service.QueueProgramStateRebuild(r.Context(), input.TenantID, input.ProgramID, actor.PrincipalID, strings.TrimSpace(input.Reason))
	if err != nil {
		writeContinuityError(w, err)
		return
	}
	w.Header().Set("Retry-After", strconv.Itoa(1))
	httpx.WriteJSON(w, http.StatusAccepted, map[string]any{"job": job, "message": "The Program status will be recalculated."})
}
