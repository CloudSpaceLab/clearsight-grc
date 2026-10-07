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

func (a *API) homeMetricTrend(w http.ResponseWriter, r *http.Request) {
	a.metricTrend(w, r)
}

func (a *API) domainMetricTrend(w http.ResponseWriter, r *http.Request) {
	a.metricTrend(w, r)
}

func (a *API) domainMetricOrganizationTrend(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	if strings.TrimSpace(actor.TenantID) == "" || strings.TrimSpace(actor.LegalEntityID) == "" || actor.LegalEntityID == "*" {
		httpx.WriteError(w, http.StatusForbidden, "metric_trend_scope_unavailable", "Choose an eligible legal entity before viewing metric history.")
		return
	}
	organizationScopeID := strings.TrimSpace(r.URL.Query().Get("organization_scope_id"))
	if organizationScopeID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "metric_trend_scope_invalid", "Choose an organization area.")
		return
	}
	selection, scopeErr := a.resolveOrganizationScopeSelection(r.Context(), actor, organizationScopeID, false)
	if scopeErr != nil {
		writeOrganizationScopeRequestError(w, scopeErr, "This organization scope is not available for risk history.")
		return
	}
	reader, ok := a.deps.MetricTrends.(metricview.OrganizationTrendReader)
	if !ok {
		httpx.WriteError(w, http.StatusServiceUnavailable, "metric_trend_unavailable", "Organization risk history is unavailable. Try again.")
		return
	}
	start, end, valid := metricTrendPeriod(w, r, time.Now().UTC())
	if !valid {
		return
	}
	series, err := reader.OrganizationTrend(
		r.Context(),
		actor.TenantID,
		actor.LegalEntityID,
		selection.ID,
		strings.TrimSpace(r.PathValue("metric_id")),
		start,
		end,
	)
	switch {
	case errors.Is(err, metricview.ErrTrendInvalid):
		httpx.WriteError(w, http.StatusBadRequest, "metric_trend_filter_invalid", "Choose a valid risk metric and a period of up to 365 days.")
	case errors.Is(err, metricview.ErrTrendNotFound):
		httpx.WriteError(w, http.StatusNotFound, "metric_trend_not_found", "No retained organization risk history is available for this period.")
	case err != nil:
		httpx.WriteError(w, http.StatusServiceUnavailable, "metric_trend_unavailable", "Organization risk history is unavailable. Try again.")
	default:
		httpx.WriteJSON(w, http.StatusOK, series)
	}
}

func (a *API) metricTrend(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	if a == nil || a.deps.MetricTrends == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "metric_trend_unavailable", "Metric history is unavailable. Try again.")
		return
	}
	if strings.TrimSpace(actor.TenantID) == "" || strings.TrimSpace(actor.LegalEntityID) == "" || actor.LegalEntityID == "*" {
		httpx.WriteError(w, http.StatusForbidden, "metric_trend_scope_unavailable", "Choose an eligible legal entity before viewing metric history.")
		return
	}
	if strings.TrimSpace(r.URL.Query().Get("organization_scope_id")) != "" {
		httpx.WriteError(w, http.StatusBadRequest, "metric_trend_scope_invalid", "Metric history is currently available at legal-entity scope.")
		return
	}
	start, end, ok := metricTrendPeriod(w, r, time.Now().UTC())
	if !ok {
		return
	}
	series, err := a.deps.MetricTrends.Trend(
		r.Context(), actor.TenantID, actor.LegalEntityID, strings.TrimSpace(r.PathValue("metric_id")), start, end,
	)
	switch {
	case errors.Is(err, metricview.ErrTrendInvalid):
		httpx.WriteError(w, http.StatusBadRequest, "metric_trend_filter_invalid", "Choose a valid metric and a period of up to 365 days.")
	case errors.Is(err, metricview.ErrTrendNotFound):
		httpx.WriteError(w, http.StatusNotFound, "metric_trend_not_found", "No retained metric history is available for this period.")
	case err != nil:
		httpx.WriteError(w, http.StatusServiceUnavailable, "metric_trend_unavailable", "Metric history is unavailable. Try again.")
	default:
		httpx.WriteJSON(w, http.StatusOK, series)
	}
}

func metricTrendPeriod(w http.ResponseWriter, r *http.Request, now time.Time) (time.Time, time.Time, bool) {
	startRaw := strings.TrimSpace(r.URL.Query().Get("start_date"))
	endRaw := strings.TrimSpace(r.URL.Query().Get("end_date"))
	if startRaw == "" {
		httpx.WriteError(w, http.StatusBadRequest, "metric_trend_filter_invalid", "Start date is required.")
		return time.Time{}, time.Time{}, false
	}
	start, err := time.Parse("2006-01-02", startRaw)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "metric_trend_filter_invalid", "Start date is invalid.")
		return time.Time{}, time.Time{}, false
	}
	now = now.UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	end := now
	if endRaw != "" {
		parsed, parseErr := time.Parse("2006-01-02", endRaw)
		if parseErr != nil || parsed.After(today) {
			httpx.WriteError(w, http.StatusBadRequest, "metric_trend_filter_invalid", "End date is invalid.")
			return time.Time{}, time.Time{}, false
		}
		if !parsed.Equal(today) {
			end = parsed.Add(24*time.Hour - time.Nanosecond)
		}
	}
	start = start.UTC()
	if !start.Before(end) || end.Sub(start) >= (metricview.TrendMaxDays+1)*24*time.Hour {
		httpx.WriteError(w, http.StatusBadRequest, "metric_trend_filter_invalid", "Choose a period of up to 365 days.")
		return time.Time{}, time.Time{}, false
	}
	return start, end, true
}
