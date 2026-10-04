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

func (a *API) homeMetricMembers(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	if a == nil || a.deps.MetricMembership == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "metric_drill_unavailable", "Exact metric detail is unavailable. Try again.")
		return
	}
	if strings.TrimSpace(actor.TenantID) == "" || strings.TrimSpace(actor.LegalEntityID) == "" ||
		actor.LegalEntityID == "*" || strings.TrimSpace(actor.PrincipalID) == "" {
		httpx.WriteError(w, http.StatusForbidden, "metric_drill_scope_unavailable", "Choose an eligible legal entity before opening metric detail.")
		return
	}

	metricID := strings.TrimSpace(r.PathValue("metric_id"))
	organizationScopeID := strings.TrimSpace(r.URL.Query().Get("organization_scope_id"))
	if _, err := a.resolveOrganizationScopeSelection(r.Context(), actor, organizationScopeID, true); err != nil {
		writeOrganizationScopeRequestError(w, err, "This organization scope is not available for metric detail.")
		return
	}
	sourceID := strings.TrimSpace(r.URL.Query().Get("source_id"))
	revision := strings.TrimSpace(r.URL.Query().Get("definition_revision"))
	cursor := strings.TrimSpace(r.URL.Query().Get("cursor"))
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		value, parseErr := strconv.Atoi(raw)
		if parseErr != nil || value < 1 || value > 100 {
			httpx.WriteError(w, http.StatusBadRequest, "metric_drill_filter_invalid", "The metric detail page size must be between 1 and 100.")
			return
		}
		limit = value
	}
	if metricID == "" || sourceID == "" || revision == "" {
		httpx.WriteError(w, http.StatusBadRequest, "metric_drill_filter_invalid", "Metric, source snapshot and definition revision are required.")
		return
	}

	page, err := a.deps.MetricMembership.ListSnapshotMembers(
		r.Context(),
		actor.TenantID,
		actor.LegalEntityID,
		organizationScopeID,
		sourceID,
		metricID,
		revision,
		actor.PrincipalID,
		cursor,
		limit,
	)
	switch {
	case errors.Is(err, metricview.ErrMetricMembershipInvalid):
		httpx.WriteError(w, http.StatusBadRequest, "metric_drill_filter_invalid", "Review the metric detail request and try again.")
	case errors.Is(err, metricview.ErrMetricMembershipNotFound):
		httpx.WriteError(w, http.StatusNotFound, "metric_drill_not_found", "This metric snapshot is not available in your legal entity.")
	case err != nil:
		httpx.WriteError(w, http.StatusServiceUnavailable, "metric_drill_unavailable", "Exact metric detail is unavailable. Try again.")
	default:
		httpx.WriteJSON(w, http.StatusOK, page)
	}
}
