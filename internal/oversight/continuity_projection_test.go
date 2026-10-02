package oversight

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

func TestMatterAggregateProjectionExcludesRestrictedAndUnknownScopes(t *testing.T) {
	now := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	aggregates := []continuity.MatterAggregate{
		{Matter: oversightMatter("visible", json.RawMessage(`{"access":"INTERNAL"}`), now)},
		{Matter: oversightMatter("restricted", json.RawMessage(`{"access":"RESTRICTED"}`), now)},
		{Matter: oversightMatter("unknown", json.RawMessage(`{"access":{"unexpected":true}}`), now)},
	}

	value := FromMatterAggregates("bank", "bank-ng", aggregates, now)
	if value.Coverage.Population != 3 || value.Coverage.Excluded == nil || *value.Coverage.Excluded != 1 || value.Coverage.Unknown == nil || *value.Coverage.Unknown != 1 {
		t.Fatalf("coverage=%#v", value.Coverage)
	}
	if value.Counts.CriticalHigh != 1 || len(value.Interventions) != 1 || value.Interventions[0].TargetID != "visible" {
		t.Fatalf("restricted records leaked into projection: counts=%#v interventions=%#v", value.Counts, value.Interventions)
	}
}

func TestMatterAggregateProjectionDoesNotInventLifecycleHistoryMeasures(t *testing.T) {
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	aggregates := make([]continuity.MatterAggregate, 0, 5)
	for index, hours := range []int{30, 42, 55, 74, 96} {
		created := now.Add(-time.Duration(index+10) * 24 * time.Hour)
		closed := created.Add(time.Duration(hours) * time.Hour)
		due := created.Add(60 * time.Hour)
		aggregates = append(aggregates, continuity.MatterAggregate{Matter: continuity.Matter{
			ID: string(rune('a' + index)), TenantID: "bank", LegalEntityID: "bank-ng", Type: continuity.MatterVendorReview,
			Status: continuity.MatterClosed, Priority: 3, Title: "Vendor review", Scope: json.RawMessage(`{"access":"INTERNAL"}`),
			OwnerPrincipalID: "owner-1", CreatedAt: created, UpdatedAt: closed, ClosedAt: &closed, DueAt: &due,
		}})
	}

	value := FromMatterAggregates("bank", "bank-ng", aggregates, now)
	if len(value.Estimates) != 0 {
		t.Fatalf("aggregate-only projection invented resolution estimates = %#v", value.Estimates)
	}
	if len(value.Performance) != 1 || value.Performance[0].Completed != 5 || value.Performance[0].MeasurementSamples != 0 || value.Performance[0].MedianHours != nil {
		t.Fatalf("performance = %#v", value.Performance)
	}
	if value.HistoryQuality.CompletedPopulation != 5 || value.HistoryQuality.ExcludedFromDurations != 5 {
		t.Fatalf("history quality = %#v", value.HistoryQuality)
	}
	if value.Performance[0].Reassigned != nil || value.Performance[0].Returned != nil {
		t.Fatalf("aggregate-only projection invented event history: %#v", value.Performance[0])
	}
}

func oversightMatter(id string, scope json.RawMessage, now time.Time) continuity.Matter {
	return continuity.Matter{
		ID: id, TenantID: "bank", LegalEntityID: "bank-ng", Type: continuity.MatterControlGap,
		Status: continuity.MatterInitialReview, Priority: 5, Title: id, Scope: scope,
		CreatedAt: now.Add(-24 * time.Hour), UpdatedAt: now,
	}
}

func TestMatterAggregateReportingPeriodChangesHistoryNotCurrentPosture(t *testing.T) {
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	closedAt := now.Add(-60 * 24 * time.Hour)
	openedAt := now.Add(-70 * 24 * time.Hour)
	dueAt := now.Add(-1 * time.Hour)
	aggregates := []continuity.MatterAggregate{
		{Matter: continuity.Matter{
			ID: "current-high", TenantID: "bank", LegalEntityID: "bank-ng", Type: continuity.MatterControlGap,
			Status: continuity.MatterInitialReview, Priority: 5, Title: "Current high risk",
			Scope: json.RawMessage(`{"access":"INTERNAL"}`), CreatedAt: now.Add(-10 * 24 * time.Hour), UpdatedAt: now, DueAt: &dueAt,
		}},
		{Matter: continuity.Matter{
			ID: "closed-history", TenantID: "bank", LegalEntityID: "bank-ng", Type: continuity.MatterVendorReview,
			Status: continuity.MatterClosed, Priority: 3, Title: "Historical closure",
			Scope: json.RawMessage(`{"access":"INTERNAL"}`), OwnerPrincipalID: "owner-1",
			CreatedAt: openedAt, UpdatedAt: closedAt, ClosedAt: &closedAt,
		}},
	}

	short := FromMatterAggregatesForPeriod("bank", "bank-ng", aggregates, now.Add(-30*24*time.Hour), now)
	long := FromMatterAggregatesForPeriod("bank", "bank-ng", aggregates, now.Add(-90*24*time.Hour), now)

	if short.Counts != long.Counts || short.Counts.CriticalHigh != 1 || short.Counts.Overdue != 1 {
		t.Fatalf("current posture changed with history window: short=%#v long=%#v", short.Counts, long.Counts)
	}
	if short.HistoryQuality.CompletedPopulation != 0 || long.HistoryQuality.CompletedPopulation != 1 {
		t.Fatalf("history did not follow selected period: short=%#v long=%#v", short.HistoryQuality, long.HistoryQuality)
	}
}
