package monitoring

import (
	"context"
	"strings"
)

func (r *MemoryRepository) ListAdvancedFormLibrary(_ context.Context, filter FormLibraryFilter, includeStatusFacets bool) (FormTemplatePage, error) {
	if filter.TenantID == "" || filter.LegalEntityID == "" {
		return FormTemplatePage{}, ErrInvalid
	}
	cursor, err := decodeFormLibraryCursor(filter.Cursor)
	if err != nil {
		return FormTemplatePage{}, err
	}
	order, err := normalizedFormLibrarySort(filter.Sort)
	if err != nil {
		return FormTemplatePage{}, err
	}
	expression, err := combinedFormFilterExpression(filter)
	if err != nil {
		return FormTemplatePage{}, err
	}
	facetExpression := formFilterExpressionWithoutField(expression, FormFilterStatus)
	search := strings.ToLower(strings.TrimSpace(filter.Search))
	limit := boundedFormLibraryLimit(filter.Limit)

	r.mu.RLock()
	defer r.mu.RUnlock()
	latest := make(map[string]FormTemplate)
	activeVersions := make(map[string]FormTemplate)
	for _, value := range r.forms {
		if value.TenantID != filter.TenantID || value.LegalEntityID != filter.LegalEntityID {
			continue
		}
		if prior, exists := latest[value.ID]; !exists || value.Version > prior.Version {
			latest[value.ID] = value
		}
		if value.IsCurrent && (value.Status == LifecycleActive || value.Status == LifecyclePaused) {
			activeVersions[value.ID] = value
		}
	}

	matchesSearch := func(value FormTemplate) bool {
		return search == "" || strings.Contains(strings.ToLower(value.Code+" "+value.Name+" "+value.Purpose), search)
	}
	items := make([]FormLibraryItem, 0, len(latest))
	total := 0
	facets := FormLibraryFacets{}
	if includeStatusFacets {
		facets.Status = make(map[LifecycleStatus]int)
	}
	for _, value := range latest {
		if !matchesSearch(value) {
			continue
		}
		if includeStatusFacets && formFilterExpressionMatches(facetExpression, value) {
			facets.Status[value.Status]++
		}
		if !formFilterExpressionMatches(expression, value) {
			continue
		}
		total++
		if !formLibraryItemBeyondCursor(value, cursor, order) {
			continue
		}
		item := FormLibraryItem{Template: cloneValue(value)}
		if active, exists := activeVersions[value.ID]; exists {
			item.ActiveVersion = active.Version
			item.ActiveStatus = active.Status
		}
		items = append(items, item)
	}
	sortFormLibraryItems(items, order)
	page := FormTemplatePage{Items: items, Total: &total}
	if includeStatusFacets {
		page.Facets = &facets
	}
	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		last := page.Items[len(page.Items)-1].Template
		page.NextCursor = encodeFormLibraryCursor(formLibraryCursor{UpdatedAt: last.UpdatedAt, ID: last.ID})
	}
	return page, nil
}
