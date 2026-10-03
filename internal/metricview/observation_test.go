package metricview

import (
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
)

func TestObservationsFromBundlePreservesHomeMetricTruth(t *testing.T) {
	now := time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC)
	excluded, unknown := 2, 3
	snapshot := oversight.Snapshot{
		LegalEntityID: "entity-1", GeneratedAt: now,
		PeriodStart: now.Add(-90 * 24 * time.Hour), PeriodEnd: now, PostureAsOf: now,
		ProjectionVersion: oversight.ProjectionVersion, Freshness: oversight.FreshnessCurrent,
		SourceHighWater: map[string]time.Time{"matters": now.Add(-time.Minute)},
		Coverage: oversight.Coverage{Population: 100, Excluded: &excluded, Unknown: &unknown},
		Counts: oversight.Counts{CriticalHigh: 7, Overdue: 4, RoutingFailures: 2, OutcomeFailures: 1},
	}
	bundle := FromOversight(snapshot)
	values, err := ObservationsFromBundle("tenant-1", "entity-1", "snapshot-1", snapshot.SourceHighWater, bundle)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != len(HomeDefinitions) {
		t.Fatalf("observation count=%d", len(values))
	}
	byID := map[string]Observation{}
	for _, value := range values {
		byID[value.MetricID] = value
		if value.SourceKind != ObservationSourceOversightSnapshot ||
			value.SourceID != "snapshot-1" ||
			value.DefinitionRevision != HomeDefinitionRevision ||
			value.SourceRevision != oversight.ProjectionVersion ||
			value.Population != 100 ||
			value.Excluded == nil || *value.Excluded != 2 ||
			value.Unknown == nil || *value.Unknown != 3 ||
			value.SourceHighWater["matters"] != snapshot.SourceHighWater["matters"] {
			t.Fatalf("observation drifted: %#v", value)
		}
	}
	if byID["critical_high_open"].Value != 7 || byID["critical_high_open"].Condition != ConditionAttention {
		t.Fatalf("critical observation=%#v", byID["critical_high_open"])
	}
	if byID["outcome_failures"].Value != 1 {
		t.Fatalf("outcome observation=%#v", byID["outcome_failures"])
	}
}

func TestObservationsFromBundleRejectsScopeOrDefinitionDrift(t *testing.T) {
	now := time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC)
	bundle := FromOversight(oversight.Snapshot{
		LegalEntityID: "entity-1", GeneratedAt: now,
		PeriodStart: now.Add(-time.Hour), PeriodEnd: now,
		ProjectionVersion: oversight.ProjectionVersion, Freshness: oversight.FreshnessCurrent,
		Coverage: oversight.Coverage{Population: 1, Unknown: intPointer(0)},
	})
	if _, err := ObservationsFromBundle("tenant-1", "entity-2", "snapshot-1", nil, bundle); !errors.Is(err, ErrInvalidObservation) {
		t.Fatalf("scope drift error=%v", err)
	}

	bundle.Items[0].DefinitionRevision = "unexpected"
	if _, err := ObservationsFromBundle("tenant-1", "entity-1", "snapshot-1", nil, bundle); !errors.Is(err, ErrInvalidObservation) {
		t.Fatalf("definition drift error=%v", err)
	}
}

func TestHomeDefinitionsAreUniqueAndExplicitlyAdditive(t *testing.T) {
	seen := map[string]struct{}{}
	for _, definition := range HomeDefinitions {
		if _, exists := seen[definition.ID]; exists {
			t.Fatalf("duplicate metric definition %q", definition.ID)
		}
		seen[definition.ID] = struct{}{}
		if definition.Revision != HomeDefinitionRevision ||
			definition.AggregationRule != AggregationSumDisjointCounts ||
			definition.ConditionRule != ConditionRuleZeroClear ||
			definition.Drill.Consistency != DrillCurrentState {
			t.Fatalf("definition=%#v", definition)
		}
	}
	if len(seen) != 4 {
		t.Fatalf("definition count=%d", len(seen))
	}
}

func intPointer(value int) *int { return &value }
