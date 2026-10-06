package risk

import (
	"context"
	"sort"
	"strconv"
)

func (r *MemoryRepository) IndicatorPortfolio(ctx context.Context, scope Scope, filter IndicatorPortfolioFilter) (IndicatorPortfolioPage, error) {
	if err := ctx.Err(); err != nil {
		return IndicatorPortfolioPage{}, err
	}
	scope, err := normalizeScope(scope)
	if err != nil {
		return IndicatorPortfolioPage{}, err
	}
	filter, err = normalizeIndicatorPortfolioFilter(filter)
	if err != nil {
		return IndicatorPortfolioPage{}, err
	}
	cursor, err := decodeIndicatorPortfolioCursor(filter.Cursor)
	if err != nil {
		return IndicatorPortfolioPage{}, err
	}

	type grouped struct {
		item  IndicatorPortfolioItem
		risks map[string]struct{}
	}
	groups := map[string]*grouped{}

	r.mu.RLock()
	for key, currentRisk := range r.risks {
		if currentRisk.TenantID != scope.TenantID || currentRisk.LegalEntityID != scope.LegalEntityID || currentRisk.Status != StatusActive {
			continue
		}
		latest := make(map[string]IndicatorLink)
		for _, link := range r.indicators[key] {
			existing, ok := latest[link.MonitoringCheckID]
			if !ok || link.RiskVersion > existing.RiskVersion || (link.RiskVersion == existing.RiskVersion && link.ID > existing.ID) {
				latest[link.MonitoringCheckID] = link
			}
		}
		for _, link := range latest {
			if filter.Kind != "" && link.Kind != filter.Kind {
				continue
			}
			groupKey := string(link.Kind) + "|" + link.MonitoringCheckID + "|" + strconv.FormatInt(link.MonitoringCheckVersion, 10) + "|" + link.ProgramID
			value := groups[groupKey]
			if value == nil {
				value = &grouped{
					item: IndicatorPortfolioItem{
						ProgramID: link.ProgramID, MonitoringCheckID: link.MonitoringCheckID,
						MonitoringCheckVersion: link.MonitoringCheckVersion, Kind: link.Kind,
					},
					risks: map[string]struct{}{},
				}
				groups[groupKey] = value
			}
			value.risks[currentRisk.ID] = struct{}{}
		}
	}
	r.mu.RUnlock()

	items := make([]IndicatorPortfolioItem, 0, len(groups))
	for _, value := range groups {
		value.item.RiskCount = len(value.risks)
		items = append(items, value.item)
	}
	sort.Slice(items, func(i, j int) bool { return compareIndicatorPortfolioItem(items[i], items[j]) < 0 })

	if cursor.MonitoringCheckID != "" {
		cursorItem := indicatorPortfolioCursorItem(cursor)
		filtered := items[:0]
		for _, item := range items {
			if compareIndicatorPortfolioItem(item, cursorItem) > 0 {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	page := IndicatorPortfolioPage{Items: items}
	if len(items) > filter.Limit {
		page.Items = items[:filter.Limit]
		page.NextCursor, err = encodeIndicatorPortfolioCursor(page.Items[len(page.Items)-1])
		if err != nil {
			return IndicatorPortfolioPage{}, err
		}
	}
	return page, nil
}
