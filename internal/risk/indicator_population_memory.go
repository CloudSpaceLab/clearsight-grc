package risk

import (
	"context"
	"sort"
)

func (r *MemoryRepository) ListIndicatorPopulation(ctx context.Context, scope Scope, filter IndicatorPopulationFilter) (IndicatorPopulationPage, error) {
	if err := ctx.Err(); err != nil {
		return IndicatorPopulationPage{}, err
	}
	scope, err := normalizeScope(scope)
	if err != nil {
		return IndicatorPopulationPage{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	allowedScopes := make(map[string]struct{}, len(filter.OrganizationScopeIDs))
	for _, id := range filter.OrganizationScopeIDs {
		allowedScopes[id] = struct{}{}
	}
	type grouped struct {
		item      IndicatorPopulationItem
		riskIndex map[string]struct{}
	}
	groups := make(map[string]*grouped)
	for key, riskValue := range r.risks {
		if riskValue.TenantID != scope.TenantID || riskValue.LegalEntityID != scope.LegalEntityID {
			continue
		}
		if filter.OrganizationScopeID != "" {
			if _, ok := allowedScopes[riskValue.OrganizationScopeID]; !ok {
				continue
			}
		}
		current := make(map[string]IndicatorLink)
		for _, link := range r.indicators[key] {
			if existing, ok := current[link.MonitoringCheckID]; !ok || indicatorPopulationCanonical(link, existing) {
				current[link.MonitoringCheckID] = link
			}
		}
		for _, link := range current {
			group := groups[link.MonitoringCheckID]
			if group == nil {
				group = &grouped{item: IndicatorPopulationItem{Link: link}, riskIndex: map[string]struct{}{}}
				groups[link.MonitoringCheckID] = group
			} else {
				if group.item.Link.Kind != link.Kind {
					group.item.KindConflict = true
				}
				if indicatorPopulationCanonical(link, group.item.Link) {
					group.item.Link = link
				}
			}
			if _, exists := group.riskIndex[riskValue.ID]; exists {
				continue
			}
			group.riskIndex[riskValue.ID] = struct{}{}
			group.item.Risks = append(group.item.Risks, IndicatorRiskReference{
				ID: riskValue.ID, Code: riskValue.Code, Name: riskValue.Name,
				OrganizationScopeID: riskValue.OrganizationScopeID, Status: riskValue.Status,
			})
		}
	}

	items := make([]IndicatorPopulationItem, 0, len(groups))
	for _, group := range groups {
		if filter.Kind != "" && group.item.Link.Kind != filter.Kind {
			continue
		}
		sort.Slice(group.item.Risks, func(i, j int) bool {
			if group.item.Risks[i].Code != group.item.Risks[j].Code {
				return group.item.Risks[i].Code < group.item.Risks[j].Code
			}
			return group.item.Risks[i].ID < group.item.Risks[j].ID
		})
		group.item.RiskCount = len(group.item.Risks)
		if len(group.item.Risks) > indicatorPopulationRiskPreviewLimit {
			group.item.Risks = group.item.Risks[:indicatorPopulationRiskPreviewLimit]
			group.item.RisksTruncated = true
		}
		items = append(items, group.item)
	}
	sort.Slice(items, func(i, j int) bool {
		if !items[i].Link.CreatedAt.Equal(items[j].Link.CreatedAt) {
			return items[i].Link.CreatedAt.After(items[j].Link.CreatedAt)
		}
		return items[i].Link.MonitoringCheckID > items[j].Link.MonitoringCheckID
	})
	page := IndicatorPopulationPage{OrganizationScopeID: filter.OrganizationScopeID}
	if len(items) > filter.Limit {
		page.Items = items[:filter.Limit]
		page.Truncated = true
	} else {
		page.Items = items
	}
	return page, nil
}

var _ IndicatorPopulationRepository = (*MemoryRepository)(nil)
