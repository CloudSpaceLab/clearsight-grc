package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/metricview"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) lossPeriodMetrics(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	if a == nil || a.deps.LossPeriodMetrics == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "loss_metrics_unavailable", "Loss metrics are unavailable. Try again.")
		return
	}
	if strings.TrimSpace(actor.TenantID) == "" || strings.TrimSpace(actor.LegalEntityID) == "" ||
		actor.LegalEntityID == "*" || strings.TrimSpace(actor.PrincipalID) == "" {
		httpx.WriteError(w, http.StatusForbidden, "loss_metrics_scope_unavailable", "Choose an eligible legal entity before viewing Loss metrics.")
		return
	}

	selection, scopeErr := a.resolveOrganizationScopeSelection(
		r.Context(), actor, strings.TrimSpace(r.URL.Query().Get("organization_scope_id")), true,
	)
	if scopeErr != nil {
		writeOrganizationScopeRequestError(w, scopeErr, "This organization scope is not available for Loss metrics.")
		return
	}

	now := time.Now().UTC()
	start, end, ok := lossMetricPeriod(w, r, now)
	if !ok {
		return
	}
	bundle, err := a.deps.LossPeriodMetrics.CurrentLossPeriod(
		r.Context(),
		actor.TenantID,
		actor.LegalEntityID,
		selection.ID,
		selection.IDs,
		start,
		end,
		now,
	)
	switch {
	case errors.Is(err, metricview.ErrLossPeriodInvalid):
		httpx.WriteError(w, http.StatusBadRequest, "loss_metrics_filter_invalid", "Review the Loss metric period and scope.")
	case errors.Is(err, metricview.ErrLossPeriodNotFound):
		httpx.WriteError(w, http.StatusNotFound, "loss_metrics_not_found", "Loss metrics are not available for this period.")
	case err != nil:
		httpx.WriteError(w, http.StatusServiceUnavailable, "loss_metrics_unavailable", "Loss metrics are unavailable. Try again.")
	default:
		httpx.WriteJSON(w, http.StatusOK, bundle)
	}
}

func (a *API) lossPeriodMetricMembers(w http.ResponseWriter, r *http.Request) {
	a.metricMembers(w, r, true)
}

func lossMetricPeriod(w http.ResponseWriter, r *http.Request, now time.Time) (time.Time, time.Time, bool) {
	startRaw := strings.TrimSpace(r.URL.Query().Get("start_date"))
	endRaw := strings.TrimSpace(r.URL.Query().Get("end_date"))
	if startRaw == "" {
		httpx.WriteError(w, http.StatusBadRequest, "loss_metrics_filter_invalid", "Start date is required.")
		return time.Time{}, time.Time{}, false
	}
	start, err := time.Parse("2006-01-02", startRaw)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "loss_metrics_filter_invalid", "Start date is invalid.")
		return time.Time{}, time.Time{}, false
	}
	now = now.UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	end := now
	if endRaw != "" {
		parsed, parseErr := time.Parse("2006-01-02", endRaw)
		if parseErr != nil || parsed.After(today) {
			httpx.WriteError(w, http.StatusBadRequest, "loss_metrics_filter_invalid", "End date is invalid.")
			return time.Time{}, time.Time{}, false
		}
		if !parsed.Equal(today) {
			end = parsed.Add(24*time.Hour - time.Nanosecond)
		}
	}
	start = start.UTC()
	if !start.Before(end) || end.Sub(start) >= (metricview.TrendMaxDays+1)*24*time.Hour {
		httpx.WriteError(w, http.StatusBadRequest, "loss_metrics_filter_invalid", "Choose a period of up to 365 days.")
		return time.Time{}, time.Time{}, false
	}
	return start, end, true
}
