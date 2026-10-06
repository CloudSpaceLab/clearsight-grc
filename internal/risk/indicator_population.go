package risk

import (
	"context"
	"strings"
)

const indicatorPopulationRiskPreviewLimit = 20

type IndicatorRiskReference struct {
	ID                  string `json:"id"`
	Code                string `json:"code"`
	Name                string `json:"name"`
	OrganizationScopeID string `json:"organization_scope_id,omitempty"`
	Status              Status `json:"status"`
}

type IndicatorPopulationItem struct {
	Link           IndicatorLink            `json:"link"`
	Risks          []IndicatorRiskReference `json:"risks"`
	RiskCount      int                      `json:"risk_count"`
	RisksTruncated bool                     `json:"risks_truncated,omitempty"`
	KindConflict   bool                     `json:"kind_conflict,omitempty"`
}

type IndicatorPopulationPage struct {
	Items               []IndicatorPopulationItem `json:"items"`
	Truncated           bool                      `json:"truncated,omitempty"`
	OrganizationScopeID string                    `json:"organization_scope_id,omitempty"`
}

type IndicatorPopulationFilter struct {
	Kind                 IndicatorKind
	OrganizationScopeID  string
	OrganizationScopeIDs []string
	Limit                int
}

type IndicatorPopulationRepository interface {
	ListIndicatorPopulation(context.Context, Scope, IndicatorPopulationFilter) (IndicatorPopulationPage, error)
}

func (s *Service) ListIndicatorPopulation(ctx context.Context, scope Scope, filter IndicatorPopulationFilter) (IndicatorPopulationPage, error) {
	if s == nil || s.repository == nil {
		return IndicatorPopulationPage{}, ErrInvalid
	}
	repository, ok := s.repository.(IndicatorPopulationRepository)
	if !ok {
		return IndicatorPopulationPage{}, ErrInvalid
	}
	normalizedScope, err := normalizeScope(scope)
	if err != nil {
		return IndicatorPopulationPage{}, err
	}
	filter.Kind = IndicatorKind(strings.ToUpper(strings.TrimSpace(string(filter.Kind))))
	filter.OrganizationScopeID = strings.TrimSpace(filter.OrganizationScopeID)
	filter.OrganizationScopeIDs = normalizeOrganizationScopeIDs(filter.OrganizationScopeID, filter.OrganizationScopeIDs)
	if filter.Kind != "" && !validIndicatorKind(filter.Kind) {
		return IndicatorPopulationPage{}, ErrInvalid
	}
	if filter.Limit <= 0 {
		filter.Limit = 50
	} else if filter.Limit > 100 {
		filter.Limit = 100
	}
	return repository.ListIndicatorPopulation(ctx, normalizedScope, filter)
}

func indicatorPopulationCanonical(left, right IndicatorLink) bool {
	if left.MonitoringCheckVersion != right.MonitoringCheckVersion {
		return left.MonitoringCheckVersion > right.MonitoringCheckVersion
	}
	if left.RiskVersion != right.RiskVersion {
		return left.RiskVersion > right.RiskVersion
	}
	if !left.CreatedAt.Equal(right.CreatedAt) {
		return left.CreatedAt.After(right.CreatedAt)
	}
	return left.ID > right.ID
}
