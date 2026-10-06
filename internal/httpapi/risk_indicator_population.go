package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/risk"
)

type riskIndicatorPopulationItemRead struct {
	Indicator      riskIndicatorRead             `json:"indicator"`
	Risks          []risk.IndicatorRiskReference `json:"risks"`
	RiskCount      int                           `json:"risk_count"`
	RisksTruncated bool                          `json:"risks_truncated,omitempty"`
	KindConflict   bool                          `json:"kind_conflict,omitempty"`
}

type riskIndicatorPopulationRead struct {
	Items               []riskIndicatorPopulationItemRead `json:"items"`
	Truncated           bool                              `json:"truncated,omitempty"`
	Complete            bool                              `json:"complete"`
	OrganizationScopeID string                            `json:"organization_scope_id,omitempty"`
	NextCursor          string                            `json:"next_cursor,omitempty"`
}

func (a *API) listRiskIndicators(w http.ResponseWriter, r *http.Request) {
	service, ok := a.riskService(w)
	if !ok {
		return
	}
	actor, scope, ok := a.riskActorScope(w, r)
	if !ok {
		return
	}
	selection, err := a.resolveOrganizationScopeSelection(r.Context(), actor, r.URL.Query().Get("organization_scope_id"), true)
	if err != nil {
		writeOrganizationScopeRequestError(w, err, "This organization scope is not available for Indicators.")
		return
	}
	limit, ok := riskIndicatorPopulationLimit(w, r)
	if !ok {
		return
	}
	kind := risk.IndicatorKind(strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("kind"))))
	page, err := service.ListIndicatorPopulation(r.Context(), scope, risk.IndicatorPopulationFilter{
		Kind: kind, MonitoringCheckID: strings.TrimSpace(r.URL.Query().Get("check_id")), OrganizationScopeID: selection.ID, OrganizationScopeIDs: selection.IDs,
		Cursor: strings.TrimSpace(r.URL.Query().Get("cursor")), Limit: limit,
	})
	if err != nil {
		writeRiskError(w, err)
		return
	}
	response := riskIndicatorPopulationRead{
		Items: []riskIndicatorPopulationItemRead{}, Truncated: page.Truncated, Complete: true,
		OrganizationScopeID: page.OrganizationScopeID, NextCursor: page.NextCursor,
	}
	if len(page.Items) == 0 {
		httpx.WriteJSON(w, http.StatusOK, response)
		return
	}
	if a == nil || a.deps.Monitoring == nil || a.deps.Continuity == nil {
		response.Complete = false
		response.Truncated = false
		httpx.WriteJSON(w, http.StatusOK, response)
		return
	}

	builder := newRiskIndicatorReadBuilder(a, r.Context(), actor)
	details := make([]riskIndicatorRead, 0, len(page.Items))
	labels := make([]riskIndicatorLabelPair, 0, len(page.Items))
	visible := make([]risk.IndicatorPopulationItem, 0, len(page.Items))
	for _, item := range page.Items {
		detail, labelPair, isVisible, complete := builder.build(item.Link)
		if !isVisible {
			response.Complete = false
			continue
		}
		if !complete {
			response.Complete = false
		}
		details = append(details, detail)
		labels = append(labels, labelPair)
		visible = append(visible, item)
	}
	a.applyRiskIndicatorLabels(r.Context(), actor, details, labels)
	for index, item := range visible {
		response.Items = append(response.Items, riskIndicatorPopulationItemRead{
			Indicator: details[index], Risks: item.Risks, RiskCount: item.RiskCount,
			RisksTruncated: item.RisksTruncated, KindConflict: item.KindConflict,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

func riskIndicatorPopulationLimit(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if raw == "" {
		return 50, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 || value > 100 {
		httpx.WriteError(w, http.StatusBadRequest, "indicator_filter_invalid", "The indicator page size must be between 1 and 100.")
		return 0, false
	}
	return value, true
}
