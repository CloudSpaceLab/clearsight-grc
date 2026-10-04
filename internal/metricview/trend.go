package metricview

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
)

const (
	RawObservationRetention = 14 * 24 * time.Hour
	DailyRollupRetention    = 730 * 24 * time.Hour
	TrendMaxDays            = 365
	TrendHourlyMaxDays      = 7
)

var (
	ErrTrendInvalid  = errors.New("metric trend request is invalid")
	ErrTrendNotFound = errors.New("metric trend is not available")
)

type TrendResolution string

const (
	TrendResolutionHour TrendResolution = "HOUR"
	TrendResolutionDay  TrendResolution = "DAY"
)

type TrendDirection string

const (
	TrendImproved  TrendDirection = "IMPROVED"
	TrendWorsened  TrendDirection = "WORSENED"
	TrendUnchanged TrendDirection = "UNCHANGED"
	TrendUnknown   TrendDirection = "UNKNOWN"
)

type ComparisonQuality string

const (
	ComparisonComplete ComparisonQuality = "COMPLETE"
	ComparisonLimited  ComparisonQuality = "LIMITED"
	ComparisonMissing  ComparisonQuality = "MISSING"
)

type TrendPoint struct {
	At             time.Time           `json:"at"`
	Value          int                 `json:"value"`
	Freshness      oversight.Freshness `json:"freshness"`
	Completeness   Completeness        `json:"completeness"`
	Population     int                 `json:"population"`
	Excluded       *int                `json:"excluded,omitempty"`
	Unknown        *int                `json:"unknown,omitempty"`
	SourceRevision string              `json:"source_revision"`
}

type TrendSeries struct {
	MetricID           string            `json:"metric_id"`
	DefinitionRevision string            `json:"definition_revision"`
	Start              time.Time         `json:"start"`
	End                time.Time         `json:"end"`
	Resolution         TrendResolution   `json:"resolution"`
	Points             []TrendPoint      `json:"points"`
	Current            *TrendPoint       `json:"current,omitempty"`
	Baseline           *TrendPoint       `json:"baseline,omitempty"`
	Delta              *int              `json:"delta,omitempty"`
	Direction          TrendDirection    `json:"direction"`
	ComparisonQuality  ComparisonQuality `json:"comparison_quality"`
}

type TrendReader interface {
	Trend(context.Context, string, string, string, time.Time, time.Time) (TrendSeries, error)
}

func trendResolution(start, end time.Time) (TrendResolution, error) {
	start, end = start.UTC(), end.UTC()
	if start.IsZero() || end.IsZero() || !start.Before(end) {
		return "", ErrTrendInvalid
	}
	startDay := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	endDay := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.UTC)
	daySpan := int(endDay.Sub(startDay) / (24 * time.Hour))
	if daySpan < 0 || daySpan > TrendMaxDays {
		return "", ErrTrendInvalid
	}
	if daySpan <= TrendHourlyMaxDays {
		return TrendResolutionHour, nil
	}
	return TrendResolutionDay, nil
}

func decorateTrendComparison(series *TrendSeries) {
	if series == nil || series.Current == nil || series.Baseline == nil {
		if series != nil {
			series.Direction = TrendUnknown
			series.ComparisonQuality = ComparisonMissing
		}
		return
	}
	delta := series.Current.Value - series.Baseline.Value
	series.Delta = &delta
	switch {
	case delta < 0:
		series.Direction = TrendImproved
	case delta > 0:
		series.Direction = TrendWorsened
	default:
		series.Direction = TrendUnchanged
	}
	if trendPointComplete(*series.Current) && trendPointComplete(*series.Baseline) && trendComparisonGapAcceptable(*series) {
		series.ComparisonQuality = ComparisonComplete
		return
	}
	series.ComparisonQuality = ComparisonLimited
}

func trendComparisonGapAcceptable(series TrendSeries) bool {
	if series.Current == nil || series.Baseline == nil {
		return false
	}
	maximumGap := 36 * time.Hour
	if series.Resolution == TrendResolutionHour {
		maximumGap = 2 * time.Hour
	}
	baselineGap := series.Start.Sub(series.Baseline.At)
	currentGap := series.End.Sub(series.Current.At)
	return baselineGap >= 0 && baselineGap <= maximumGap && currentGap >= 0 && currentGap <= maximumGap
}

func trendPointComplete(point TrendPoint) bool {
	return point.Freshness == oversight.FreshnessCurrent &&
		point.Completeness == CompletenessComplete &&
		point.Excluded != nil && *point.Excluded == 0 &&
		point.Unknown != nil && *point.Unknown == 0
}

func validHomeTrendMetric(metricID string) bool {
	metricID = strings.TrimSpace(metricID)
	_, ok := HomeDefinition(metricID)
	return ok
}
