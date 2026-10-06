package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/metricview"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/risk"
)

func (a *API) indicatorPortfolio(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	reader, ok := a.deps.DomainMetrics.(metricview.IndicatorPortfolioReader)
	if !ok || reader == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "indicator_insights_unavailable", "Indicator insights are unavailable. Try again.")
		return
	}
	if strings.TrimSpace(actor.TenantID) == "" || strings.TrimSpace(actor.LegalEntityID) == "" || actor.LegalEntityID == "*" {
		httpx.WriteError(w, http.StatusForbidden, "indicator_insights_scope_unavailable", "Choose an eligible legal entity before viewing indicators.")
		return
	}
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, parseErr := strconv.Atoi(raw)
		if parseErr != nil || parsed < 1 {
			httpx.WriteError(w, http.StatusBadRequest, "indicator_insights_filter_invalid", "Indicator filters are invalid.")
			return
		}
		limit = parsed
	}
	filter := metricview.IndicatorPortfolioFilter{
		Kind:   risk.IndicatorKind(strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("kind")))),
		State:  metricview.IndicatorPortfolioState(strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("state")))),
		Search: strings.TrimSpace(r.URL.Query().Get("search")),
		Cursor: strings.TrimSpace(r.URL.Query().Get("cursor")),
		Limit:  limit,
	}
	page, err := reader.ListIndicators(r.Context(), actor.TenantID, actor.LegalEntityID, filter)
	switch {
	case errors.Is(err, metricview.ErrIndicatorPortfolioInvalid):
		httpx.WriteError(w, http.StatusBadRequest, "indicator_insights_filter_invalid", "Indicator filters are invalid.")
	case err != nil:
		httpx.WriteError(w, http.StatusServiceUnavailable, "indicator_insights_unavailable", "Indicator insights are unavailable. Try again.")
	default:
		httpx.WriteJSON(w, http.StatusOK, page)
	}
}
