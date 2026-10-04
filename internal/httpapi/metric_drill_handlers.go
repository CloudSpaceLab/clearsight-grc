package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/metricview"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) homeMetricDrill(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	if a.deps.MetricDrills == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "metric_drill_unavailable", "Metric detail is unavailable. Try again.")
		return
	}
	if strings.TrimSpace(actor.TenantID) == "" || strings.TrimSpace(actor.LegalEntityID) == "" || actor.LegalEntityID == "*" {
		httpx.WriteError(w, http.StatusForbidden, "metric_drill_scope_unavailable", "Choose an eligible legal entity before opening metric detail.")
		return
	}
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		value, parseErr := strconv.Atoi(raw)
		if parseErr != nil || value < 1 || value > 100 {
			httpx.WriteError(w, http.StatusBadRequest, "metric_drill_filter_invalid", "The metric page size must be between 1 and 100.")
			return
		}
		limit = value
	}
	page, err := a.deps.MetricDrills.ListExactDrill(r.Context(), metricview.DrillQuery{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID,
		SourceID: strings.TrimSpace(r.URL.Query().Get("source_id")),
		MetricID: strings.TrimSpace(r.URL.Query().Get("metric_id")),
		Cursor: strings.TrimSpace(r.URL.Query().Get("cursor")),
		Limit: limit,
	})
	switch {
	case errors.Is(err, metricview.ErrDrillInvalid):
		httpx.WriteError(w, http.StatusBadRequest, "metric_drill_filter_invalid", "Review the metric detail request.")
	case errors.Is(err, metricview.ErrDrillNotFound):
		httpx.WriteError(w, http.StatusNotFound, "metric_drill_not_found", "This exact metric population is not available in your legal entity.")
	case errors.Is(err, metricview.ErrDrillMismatch):
		httpx.WriteError(w, http.StatusServiceUnavailable, "metric_drill_inconsistent", "Metric detail does not match the retained count. Refresh before relying on it.")
	case err != nil:
		httpx.WriteError(w, http.StatusServiceUnavailable, "metric_drill_unavailable", "Metric detail is unavailable. Try again.")
	default:
		httpx.WriteJSON(w, http.StatusOK, page)
	}
}
