package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) backgroundJobs(w http.ResponseWriter, r *http.Request) {
	if a.deps.BackgroundJobs == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "background_jobs_unavailable", "Background job operations are unavailable.")
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to continue.")
		return
	}
	limit := 100
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, parseErr := strconv.Atoi(raw)
		if parseErr != nil || parsed < 1 || parsed > 200 {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_limit", "limit must be between 1 and 200.")
			return
		}
		limit = parsed
	}
	snapshot, err := a.deps.BackgroundJobs.Snapshot(r.Context(), actor.TenantID, limit)
	if err != nil {
		a.deps.Logger.Error("background job snapshot failed", "error", err, "tenant_id", actor.TenantID)
		httpx.WriteError(w, http.StatusInternalServerError, "background_jobs_failed", "Background job state could not be loaded.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, snapshot)
}
