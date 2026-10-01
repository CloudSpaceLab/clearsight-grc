package oversight

import (
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/metric"
)

func TestHeadlineMetricsPreserveCoverageAndAttentionState(t *testing.T) {
	unknown, excluded := 0, 2
	generated := time.Date(2026, time.October, 1, 16, 30, 0, 0, time.UTC)
	snapshot := Snapshot{
		GeneratedAt:       generated,
		ProjectionVersion: ProjectionVersion,
		Freshness:         FreshnessCurrent,
		Coverage:          Coverage{Population: 42, Excluded: &excluded, Unknown: &unknown},
		Counts: Counts{
			CriticalHigh:    2,
			Overdue:         0,
			RoutingFailures: 1,
			OutcomeFailures: 0,
		},
	}

	items := headlineMetrics(snapshot)
	if len(items) != 4 {
		t.Fatalf("metric count = %d", len(items))
	}
	assertHeadlineMetric(t, items[0], "critical_high_open", 2, metric.StateCritical, true)
	assertHeadlineMetric(t, items[1], "overdue_open", 0, metric.StateClear, true)
	assertHeadlineMetric(t, items[2], "routing_gaps", 1, metric.StateWarning, true)
	assertHeadlineMetric(t, items[3], "outcome_failures", 0, metric.StateClear, true)
	for _, item := range items {
		if item.Population != 42 || item.Excluded == nil || *item.Excluded != 2 || item.Unknown == nil || *item.Unknown != 0 {
			t.Fatalf("%s coverage = %#v", item.Code, item)
		}
		if item.GeneratedAt != generated || item.ProjectionVersion != ProjectionVersion {
			t.Fatalf("%s provenance = %#v", item.Code, item)
		}
	}
}

func TestHeadlineMetricsNeverPresentIncompleteZeroAsClear(t *testing.T) {
	unknown := 3
	items := headlineMetrics(Snapshot{
		GeneratedAt:       time.Now().UTC(),
		ProjectionVersion: ProjectionVersion,
		Freshness:         FreshnessCurrent,
		Coverage:          Coverage{Population: 12, Unknown: &unknown},
	})
	for _, item := range items {
		if item.Value != 0 || item.State != metric.StateUnknown || item.StateLabel != "Coverage incomplete" || item.Complete {
			t.Fatalf("incomplete zero metric = %#v", item)
		}
	}
}

func TestHeadlineMetricsKeepKnownFailureVisibleWhenCoverageIsIncomplete(t *testing.T) {
	unknown := 2
	items := headlineMetrics(Snapshot{
		GeneratedAt:       time.Now().UTC(),
		ProjectionVersion: ProjectionVersion,
		Freshness:         FreshnessCurrent,
		Coverage:          Coverage{Population: 20, Unknown: &unknown},
		Counts:            Counts{OutcomeFailures: 1},
	})
	item := items[3]
	if item.State != metric.StateCritical || item.Value != 1 || item.Complete {
		t.Fatalf("known failure with incomplete coverage = %#v", item)
	}
}

func TestHeadlineMetricsTreatStaleZeroAsUnknown(t *testing.T) {
	unknown := 0
	items := headlineMetrics(Snapshot{
		GeneratedAt:       time.Now().UTC().Add(-time.Hour),
		ProjectionVersion: ProjectionVersion,
		Freshness:         FreshnessStale,
		Coverage:          Coverage{Population: 9, Unknown: &unknown},
	})
	for _, item := range items {
		if item.State != metric.StateUnknown || item.Complete {
			t.Fatalf("stale metric = %#v", item)
		}
	}
}

func assertHeadlineMetric(t *testing.T, item metric.Snapshot, code string, value int64, state metric.State, complete bool) {
	t.Helper()
	if item.Code != code || item.Value != value || item.State != state || item.Complete != complete {
		t.Fatalf("metric %s = %#v", code, item)
	}
	if item.Unit != "COUNT" || item.DrillKey == "" || item.Direction != metric.DirectionUnknown {
		t.Fatalf("metric %s contract = %#v", code, item)
	}
}
