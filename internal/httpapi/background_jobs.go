package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/operations"
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

func (a *API) retryBackgroundJob(w http.ResponseWriter, r *http.Request) {
	if a.deps.BackgroundJobs == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "background_jobs_unavailable", "Background job recovery is unavailable.")
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to continue.")
		return
	}
	var input struct {
		TenantID         string `json:"tenant_id"`
		Queue            string `json:"queue"`
		ExpectedAttempts int    `json:"expected_attempts"`
		Rationale        string `json:"rationale"`
	}
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		return
	}
	receipt, err := a.deps.BackgroundJobs.RetryTerminalJob(r.Context(), operations.RetryInput{
		TenantID: input.TenantID, Queue: input.Queue, JobID: r.PathValue("job_id"), ExpectedAttempts: input.ExpectedAttempts,
		ActorPrincipalID: actor.PrincipalID, Rationale: input.Rationale,
	})
	switch {
	case errors.Is(err, operations.ErrRecoveryInvalid):
		httpx.WriteError(w, http.StatusBadRequest, "background_job_recovery_invalid", "Select a terminal job, confirm its current attempt count and record why retry is safe.")
	case errors.Is(err, operations.ErrRecoveryConflict):
		httpx.WriteError(w, http.StatusConflict, "background_job_recovery_conflict", "The job changed or is no longer terminal. Reload system operations before retrying it.")
	case err != nil:
		a.deps.Logger.Error("background job recovery failed", "error", err, "tenant_id", actor.TenantID, "job_id", r.PathValue("job_id"))
		httpx.WriteError(w, http.StatusInternalServerError, "background_job_recovery_failed", "The job could not be scheduled for recovery.")
	default:
		httpx.WriteJSON(w, http.StatusOK, receipt)
	}
}
