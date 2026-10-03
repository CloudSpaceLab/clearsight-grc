package oversight

import (
	"errors"
	"testing"
	"time"
)

func TestBuildGroupSnapshotSumsDisjointChildrenAndRetainsContributors(t *testing.T) {
	now := time.Date(2026, 10, 3, 20, 0, 0, 0, time.UTC)
	zero := 0
	entities := []GroupEntity{
		{ID: "entity-b", Code: "B", Name: "Beta"},
		{ID: "entity-a", Code: "A", Name: "Alpha"},
	}
	snapshots := []Snapshot{
		{
			SnapshotID: "snap-a", LegalEntityID: "entity-a", GeneratedAt: now.Add(-time.Minute),
			PostureAsOf: now.Add(-time.Minute), ProjectionVersion: ProjectionVersion, Freshness: FreshnessCurrent,
			Coverage: Coverage{Population: 4, Excluded: &zero, Unknown: &zero},
			Counts: Counts{CriticalHigh: 2, Overdue: 1},
		},
		{
			SnapshotID: "snap-b", LegalEntityID: "entity-b", GeneratedAt: now,
			PostureAsOf: now, ProjectionVersion: ProjectionVersion, Freshness: FreshnessCurrent,
			Coverage: Coverage{Population: 6, Excluded: &zero, Unknown: &zero},
			Counts: Counts{CriticalHigh: 3, RoutingFailures: 2},
		},
	}
	value, err := BuildGroupSnapshot("tenant-1", "Clear Bank", entities, snapshots)
	if err != nil {
		t.Fatal(err)
	}
	if value.Counts.CriticalHigh != 5 || value.Counts.Overdue != 1 || value.Counts.RoutingFailures != 2 {
		t.Fatalf("counts=%#v", value.Counts)
	}
	if value.Coverage.AuthorizedChildren != 2 || value.Coverage.ContributingChildren != 2 ||
		value.Coverage.MissingChildren != 0 || value.Coverage.Population != 10 {
		t.Fatalf("coverage=%#v", value.Coverage)
	}
	if value.Coverage.Unknown == nil || *value.Coverage.Unknown != 0 || value.Freshness != FreshnessCurrent {
		t.Fatalf("quality coverage=%#v freshness=%s", value.Coverage, value.Freshness)
	}
	if value.PostureAsOf != snapshots[0].PostureAsOf || len(value.Contributors) != 2 ||
		value.Contributors[0].LegalEntityID != "entity-a" || value.Contributors[1].LegalEntityID != "entity-b" ||
		value.ContributorRevision == "" {
		t.Fatalf("provenance=%#v", value)
	}
}

func TestBuildGroupSnapshotMakesMissingChildExplicit(t *testing.T) {
	now := time.Date(2026, 10, 3, 20, 0, 0, 0, time.UTC)
	zero := 0
	value, err := BuildGroupSnapshot("tenant-1", "Clear Bank", []GroupEntity{
		{ID: "entity-a", Name: "Alpha"},
		{ID: "entity-b", Name: "Beta"},
	}, []Snapshot{{
		SnapshotID: "snap-a", LegalEntityID: "entity-a", GeneratedAt: now,
		PostureAsOf: now, ProjectionVersion: ProjectionVersion, Freshness: FreshnessCurrent,
		Coverage: Coverage{Population: 3, Excluded: &zero, Unknown: &zero},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if value.Freshness != FreshnessStale || value.Coverage.MissingChildren != 1 ||
		value.Coverage.Unknown != nil || value.Children[1].State != GroupChildMissing {
		t.Fatalf("missing child was not explicit: %#v", value)
	}
}

func TestBuildGroupSnapshotRejectsDuplicateOrUnauthorizedContributors(t *testing.T) {
	now := time.Date(2026, 10, 3, 20, 0, 0, 0, time.UTC)
	entities := []GroupEntity{{ID: "entity-a", Name: "Alpha"}, {ID: "entity-b", Name: "Beta"}}
	snapshot := Snapshot{SnapshotID: "snap-a", LegalEntityID: "entity-a", GeneratedAt: now, Freshness: FreshnessCurrent}
	_, err := BuildGroupSnapshot("tenant-1", "Clear Bank", entities, []Snapshot{snapshot, snapshot})
	if !errors.Is(err, ErrInvalidGroupAggregate) {
		t.Fatalf("duplicate contributor error=%v", err)
	}
	snapshot.LegalEntityID = "entity-c"
	_, err = BuildGroupSnapshot("tenant-1", "Clear Bank", entities, []Snapshot{snapshot})
	if !errors.Is(err, ErrInvalidGroupAggregate) {
		t.Fatalf("unauthorized contributor error=%v", err)
	}
}
