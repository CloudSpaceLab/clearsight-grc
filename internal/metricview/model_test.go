package metricview

import (
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
)

func TestFromOversightPreservesPopulationAndSafeDataQuality(t *testing.T) {
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	unknown := 2
	excluded := 3
	bundle := FromOversight(oversight.Snapshot{
		LegalEntityID:     "entity-ng",
		GeneratedAt:       now,
		PeriodStart:       now.Add(-24 * time.Hour),
		PeriodEnd:         now,
		ProjectionVersion: oversight.ProjectionVersion,
		Freshness:         oversight.FreshnessCurrent,
		Coverage:          oversight.Coverage{Population: 100, Unknown: &unknown, Excluded: &excluded},
		Counts: oversight.Counts{
			CriticalHigh:    7,
			Overdue:         0,
			RoutingFailures: 1,
			OutcomeFailures: 2,
		},
	})

	if bundle.DefinitionRevision != HomeDefinitionRevision || bundle.SourceRevision != oversight.ProjectionVersion {
		t.Fatalf("unexpected revisions: %#v", bundle)
	}
	if bundle.ScopeID != "entity-ng" || bundle.ScopeKind != "LEGAL_ENTITY" {
		t.Fatalf("unexpected scope: %#v", bundle)
	}
	if bundle.Completeness != CompletenessPartial {
		t.Fatalf("completeness = %q", bundle.Completeness)
	}
	if len(bundle.Items) != 4 {
		t.Fatalf("metric count = %d", len(bundle.Items))
	}
	if bundle.Items[0].ID != "critical_high_open" || bundle.Items[0].Value != 7 || bundle.Items[0].Condition != ConditionAttention {
		t.Fatalf("critical metric = %#v", bundle.Items[0])
	}
	if bundle.Items[1].Value != 0 || bundle.Items[1].Condition != ConditionClear || bundle.Items[1].Completeness != CompletenessPartial {
		t.Fatalf("zero metric lost partial coverage: %#v", bundle.Items[1])
	}
	for _, item := range bundle.Items {
		if item.Population != 100 || item.Unknown == nil || *item.Unknown != 2 || item.Excluded == nil || *item.Excluded != 3 {
			t.Fatalf("metric coverage drifted from bundle: %#v", item)
		}
		if item.Drill.Consistency != DrillCurrentState || item.Drill.Workspace != "oversight" {
			t.Fatalf("unexpected drill contract: %#v", item.Drill)
		}
	}
}

func TestFromOversightDoesNotTreatMissingUnknownCountAsComplete(t *testing.T) {
	bundle := FromOversight(oversight.Snapshot{
		LegalEntityID:     "entity-ng",
		GeneratedAt:       time.Now().UTC(),
		ProjectionVersion: oversight.ProjectionVersion,
		Freshness:         oversight.FreshnessStale,
		Coverage:          oversight.Coverage{Population: 0, Unknown: nil},
	})

	if bundle.Completeness != CompletenessUnknown {
		t.Fatalf("bundle completeness = %q", bundle.Completeness)
	}
	for _, item := range bundle.Items {
		if item.Completeness != CompletenessUnknown {
			t.Fatalf("metric completeness = %q for %s", item.Completeness, item.ID)
		}
		if item.Freshness != oversight.FreshnessStale {
			t.Fatalf("metric freshness = %q for %s", item.Freshness, item.ID)
		}
	}
}
