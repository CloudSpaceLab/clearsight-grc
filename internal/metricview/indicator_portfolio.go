package metricview

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/risk"
)

var ErrIndicatorPortfolioInvalid = errors.New("indicator portfolio request is invalid")

type IndicatorPortfolioState string

const (
	IndicatorPortfolioNormal  IndicatorPortfolioState = "NORMAL"
	IndicatorPortfolioWatch   IndicatorPortfolioState = "WATCH"
	IndicatorPortfolioBreach  IndicatorPortfolioState = "BREACH"
	IndicatorPortfolioUnknown IndicatorPortfolioState = "UNKNOWN"
)

type IndicatorRiskRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type IndicatorPortfolioItem struct {
	Kind                risk.IndicatorKind            `json:"kind"`
	ProgramID           string                        `json:"program_id"`
	ProgramName         string                        `json:"program_name"`
	CheckID             string                        `json:"check_id"`
	CheckCode           string                        `json:"check_code"`
	CheckName           string                        `json:"check_name"`
	Claim               string                        `json:"claim"`
	CheckStatus         monitoring.LifecycleStatus    `json:"check_status"`
	CheckVersion        int64                         `json:"check_version"`
	InputKind           monitoring.InputKind          `json:"input_kind"`
	OwnerDisplayName    string                        `json:"owner_display_name,omitempty"`
	ReviewerDisplayName string                        `json:"reviewer_display_name,omitempty"`
	NativeMeasurement   *monitoring.NativeMeasurement `json:"native_measurement,omitempty"`
	State               IndicatorPortfolioState       `json:"state"`
	Reason              string                        `json:"reason"`
	Score               *float64                      `json:"score,omitempty"`
	Band                monitoring.RiskBand           `json:"band,omitempty"`
	Coverage            *float64                      `json:"coverage,omitempty"`
	MinimumCoverage     float64                       `json:"minimum_coverage"`
	FreshnessMinutes    int                           `json:"freshness_minutes"`
	ResultID            string                        `json:"result_id,omitempty"`
	EvaluatedAt         *time.Time                    `json:"evaluated_at,omitempty"`
	Risks               []IndicatorRiskRef            `json:"risks"`
}

type IndicatorPortfolioPage struct {
	GeneratedAt time.Time                `json:"generated_at"`
	Items       []IndicatorPortfolioItem `json:"items"`
	NextCursor  string                   `json:"next_cursor,omitempty"`
}

type IndicatorPortfolioFilter struct {
	Kind   risk.IndicatorKind
	State  IndicatorPortfolioState
	Search string
	Cursor string
	Limit  int
}

type indicatorPortfolioCursor struct {
	Name    string             `json:"name"`
	CheckID string             `json:"check_id"`
	Version int64              `json:"version"`
	Kind    risk.IndicatorKind `json:"kind"`
}

func ValidateIndicatorPortfolioFilter(value IndicatorPortfolioFilter) error {
	_, _, err := normalizeIndicatorPortfolioFilter(value)
	return err
}

func normalizeIndicatorPortfolioFilter(value IndicatorPortfolioFilter) (IndicatorPortfolioFilter, indicatorPortfolioCursor, error) {
	value.Search = strings.TrimSpace(value.Search)
	value.Cursor = strings.TrimSpace(value.Cursor)
	if len(value.Search) > 200 {
		return IndicatorPortfolioFilter{}, indicatorPortfolioCursor{}, ErrIndicatorPortfolioInvalid
	}
	switch value.Kind {
	case "", risk.IndicatorKRI, risk.IndicatorKCI:
	default:
		return IndicatorPortfolioFilter{}, indicatorPortfolioCursor{}, ErrIndicatorPortfolioInvalid
	}
	switch value.State {
	case "", IndicatorPortfolioNormal, IndicatorPortfolioWatch, IndicatorPortfolioBreach, IndicatorPortfolioUnknown:
	default:
		return IndicatorPortfolioFilter{}, indicatorPortfolioCursor{}, ErrIndicatorPortfolioInvalid
	}
	if value.Limit <= 0 {
		value.Limit = 50
	}
	if value.Limit > 100 {
		value.Limit = 100
	}
	cursor, err := decodeIndicatorPortfolioCursor(value.Cursor)
	if err != nil {
		return IndicatorPortfolioFilter{}, indicatorPortfolioCursor{}, err
	}
	return value, cursor, nil
}

func encodeIndicatorPortfolioCursor(value IndicatorPortfolioItem) (string, error) {
	payload, err := json.Marshal(indicatorPortfolioCursor{
		Name: strings.ToLower(value.CheckName), CheckID: value.CheckID, Version: value.CheckVersion, Kind: value.Kind,
	})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeIndicatorPortfolioCursor(value string) (indicatorPortfolioCursor, error) {
	if value == "" {
		return indicatorPortfolioCursor{}, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return indicatorPortfolioCursor{}, ErrIndicatorPortfolioInvalid
	}
	var cursor indicatorPortfolioCursor
	if err := json.Unmarshal(raw, &cursor); err != nil || strings.TrimSpace(cursor.Name) == "" || strings.TrimSpace(cursor.CheckID) == "" || cursor.Version < 1 {
		return indicatorPortfolioCursor{}, ErrIndicatorPortfolioInvalid
	}
	switch cursor.Kind {
	case risk.IndicatorKRI, risk.IndicatorKCI:
	default:
		return indicatorPortfolioCursor{}, ErrIndicatorPortfolioInvalid
	}
	return cursor, nil
}

type IndicatorPortfolioReader interface {
	ListIndicators(context.Context, string, string, IndicatorPortfolioFilter) (IndicatorPortfolioPage, error)
}
