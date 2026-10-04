package metricview

import (
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
)

func TestTrendResolutionIsBounded(t *testing.T) {
	end := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	if got, err := trendResolution(end.Add(-7*24*time.Hour), end); err != nil || got != TrendResolutionHour {
		t.Fatalf("7 day resolution=%q err=%v", got, err)
	}
	if got, err := trendResolution(end.Add(-30*24*time.Hour), end); err != nil || got != TrendResolutionDay {
		t.Fatalf("30 day resolution=%q err=%v", got, err)
	}
	if _, err := trendResolution(end.Add(-366*24*time.Hour), end); err != ErrTrendInvalid {
		t.Fatalf("366 day error=%v", err)
	}
}

func TestTrendComparisonDirectionAndQuality(t *testing.T) {
	zero := 0
	series := TrendSeries{
		Baseline: &TrendPoint{Value: 7, Freshness: oversight.FreshnessCurrent, Completeness: CompletenessComplete, Excluded: &zero, Unknown: &zero},
		Current:  &TrendPoint{Value: 4, Freshness: oversight.FreshnessCurrent, Completeness: CompletenessComplete, Excluded: &zero, Unknown: &zero},
	}
	decorateTrendComparison(&series)
	if series.Delta == nil || *series.Delta != -3 || series.Direction != TrendImproved || series.ComparisonQuality != ComparisonComplete {
		t.Fatalf("comparison=%#v", series)
	}

	unknown := 1
	series.Current.Unknown = &unknown
	decorateTrendComparison(&series)
	if series.Direction != TrendImproved || series.ComparisonQuality != ComparisonLimited {
		t.Fatalf("limited comparison=%#v", series)
	}

	series.Baseline = nil
	decorateTrendComparison(&series)
	if series.Delta != nil || series.Direction != TrendUnknown || series.ComparisonQuality != ComparisonMissing {
		t.Fatalf("missing comparison=%#v", series)
	}
}
