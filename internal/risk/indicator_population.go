package risk

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"
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
	NextCursor          string                    `json:"next_cursor,omitempty"`
}

type IndicatorPopulationFilter struct {
	Kind                 IndicatorKind
	MonitoringCheckID    string
	OrganizationScopeID  string
	OrganizationScopeIDs []string
	Cursor               string
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
	filter.MonitoringCheckID = strings.TrimSpace(filter.MonitoringCheckID)
	filter.OrganizationScopeID = strings.TrimSpace(filter.OrganizationScopeID)
	filter.Cursor = strings.TrimSpace(filter.Cursor)
	filter.OrganizationScopeIDs = normalizeOrganizationScopeIDs(filter.OrganizationScopeID, filter.OrganizationScopeIDs)
	if filter.Kind != "" && !validIndicatorKind(filter.Kind) {
		return IndicatorPopulationPage{}, ErrInvalid
	}
	if _, err := decodeIndicatorPopulationCursor(filter.Cursor); err != nil {
		return IndicatorPopulationPage{}, err
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

type indicatorPopulationCursor struct {
	CreatedAt time.Time `json:"created_at"`
	CheckID   string    `json:"check_id"`
}

func encodeIndicatorPopulationCursor(item IndicatorPopulationItem) (string, error) {
	payload, err := json.Marshal(indicatorPopulationCursor{
		CreatedAt: item.Link.CreatedAt.UTC(),
		CheckID:   item.Link.MonitoringCheckID,
	})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeIndicatorPopulationCursor(value string) (indicatorPopulationCursor, error) {
	if value == "" {
		return indicatorPopulationCursor{}, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return indicatorPopulationCursor{}, ErrInvalid
	}
	var cursor indicatorPopulationCursor
	if err := json.Unmarshal(payload, &cursor); err != nil || cursor.CreatedAt.IsZero() || strings.TrimSpace(cursor.CheckID) == "" {
		return indicatorPopulationCursor{}, ErrInvalid
	}
	cursor.CreatedAt = cursor.CreatedAt.UTC()
	cursor.CheckID = strings.TrimSpace(cursor.CheckID)
	return cursor, nil
}

func indicatorPopulationAfterCursor(item IndicatorPopulationItem, cursor indicatorPopulationCursor) bool {
	if cursor.CreatedAt.IsZero() {
		return true
	}
	if !item.Link.CreatedAt.Equal(cursor.CreatedAt) {
		return item.Link.CreatedAt.Before(cursor.CreatedAt)
	}
	return item.Link.MonitoringCheckID < cursor.CheckID
}
