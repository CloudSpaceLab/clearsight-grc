package risk

import (
	"encoding/base64"
	"encoding/json"
	"strings"
)

type IndicatorPortfolioFilter struct {
	Kind   IndicatorKind
	Cursor string
	Limit  int
}

type IndicatorPortfolioItem struct {
	ProgramID              string        `json:"program_id"`
	MonitoringCheckID      string        `json:"monitoring_check_id"`
	MonitoringCheckVersion int64         `json:"monitoring_check_version"`
	Kind                   IndicatorKind `json:"kind"`
	RiskCount              int           `json:"risk_count"`
}

type IndicatorPortfolioPage struct {
	Items      []IndicatorPortfolioItem `json:"items"`
	NextCursor string                   `json:"next_cursor,omitempty"`
}

type indicatorPortfolioCursor struct {
	Kind                   IndicatorKind `json:"k"`
	MonitoringCheckID      string        `json:"c"`
	MonitoringCheckVersion int64         `json:"v"`
	ProgramID              string        `json:"p"`
}

func normalizeIndicatorPortfolioFilter(filter IndicatorPortfolioFilter) (IndicatorPortfolioFilter, error) {
	filter.Kind = IndicatorKind(strings.ToUpper(strings.TrimSpace(string(filter.Kind))))
	filter.Cursor = strings.TrimSpace(filter.Cursor)
	if filter.Kind != "" && !validIndicatorKind(filter.Kind) {
		return IndicatorPortfolioFilter{}, ErrInvalid
	}
	if filter.Limit <= 0 {
		filter.Limit = 50
	} else if filter.Limit > 100 {
		filter.Limit = 100
	}
	if _, err := decodeIndicatorPortfolioCursor(filter.Cursor); err != nil {
		return IndicatorPortfolioFilter{}, err
	}
	return filter, nil
}

func encodeIndicatorPortfolioCursor(item IndicatorPortfolioItem) (string, error) {
	payload, err := json.Marshal(indicatorPortfolioCursor{
		Kind: item.Kind, MonitoringCheckID: item.MonitoringCheckID,
		MonitoringCheckVersion: item.MonitoringCheckVersion, ProgramID: item.ProgramID,
	})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeIndicatorPortfolioCursor(value string) (indicatorPortfolioCursor, error) {
	if strings.TrimSpace(value) == "" {
		return indicatorPortfolioCursor{}, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return indicatorPortfolioCursor{}, ErrInvalid
	}
	var cursor indicatorPortfolioCursor
	if err := json.Unmarshal(payload, &cursor); err != nil {
		return indicatorPortfolioCursor{}, ErrInvalid
	}
	if !validIndicatorKind(cursor.Kind) || strings.TrimSpace(cursor.MonitoringCheckID) == "" ||
		cursor.MonitoringCheckVersion < 1 || strings.TrimSpace(cursor.ProgramID) == "" {
		return indicatorPortfolioCursor{}, ErrInvalid
	}
	return cursor, nil
}

func compareIndicatorPortfolioItem(left, right IndicatorPortfolioItem) int {
	if left.Kind != right.Kind {
		if left.Kind < right.Kind {
			return -1
		}
		return 1
	}
	if left.MonitoringCheckID != right.MonitoringCheckID {
		if left.MonitoringCheckID < right.MonitoringCheckID {
			return -1
		}
		return 1
	}
	if left.MonitoringCheckVersion != right.MonitoringCheckVersion {
		if left.MonitoringCheckVersion < right.MonitoringCheckVersion {
			return -1
		}
		return 1
	}
	if left.ProgramID < right.ProgramID {
		return -1
	}
	if left.ProgramID > right.ProgramID {
		return 1
	}
	return 0
}

func indicatorPortfolioCursorItem(cursor indicatorPortfolioCursor) IndicatorPortfolioItem {
	return IndicatorPortfolioItem{
		ProgramID: cursor.ProgramID, MonitoringCheckID: cursor.MonitoringCheckID,
		MonitoringCheckVersion: cursor.MonitoringCheckVersion, Kind: cursor.Kind,
	}
}
