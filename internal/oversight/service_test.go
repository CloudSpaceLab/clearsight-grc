package oversight

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestServicePreservesUnknownCoverageAndMarksStaleSnapshot(t *testing.T) {
	now := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	unknown := 3
	repo := NewMemoryRepository([]Snapshot{{
		TenantID: "bank", LegalEntityID: "bank-ng", GeneratedAt: now.Add(-20 * time.Minute), ProjectionVersion: "oversight-v1",
		Coverage: Coverage{Population: 12, Unknown: &unknown},
	}})
	service := NewService(repo)
	service.Now = func() time.Time { return now }

	value, err := service.Get(context.Background(), Scope{TenantID: "bank", LegalEntityID: "bank-ng"})
	if err != nil {
		t.Fatal(err)
	}
	if value.Freshness != FreshnessStale || value.Coverage.Unknown == nil || *value.Coverage.Unknown != 3 {
		t.Fatalf("snapshot=%#v", value)
	}
}

func TestServiceMarksPriorProjectionVersionStaleEvenWhenRecentlyGenerated(t *testing.T) {
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository([]Snapshot{{
		TenantID: "bank", LegalEntityID: "bank-ng", GeneratedAt: now, ProjectionVersion: "oversight-v2",
	}})
	service := NewService(repo)
	service.Now = func() time.Time { return now }

	value, err := service.Get(context.Background(), Scope{TenantID: "bank", LegalEntityID: "bank-ng"})
	if err != nil {
		t.Fatal(err)
	}
	if value.Freshness != FreshnessStale {
		t.Fatalf("freshness=%s", value.Freshness)
	}
}

func TestServiceDoesNotSubstituteMetricsWhenProjectionIsMissing(t *testing.T) {
	service := NewService(NewMemoryRepository(nil))
	_, err := service.Get(context.Background(), Scope{TenantID: "bank", LegalEntityID: "bank-ng"})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err=%v", err)
	}
}

func TestServiceBuildsBoundedCurrentWindowWithoutChangingPostureMeaning(t *testing.T) {
	now := time.Date(2026, 10, 2, 15, 30, 0, 0, time.UTC)
	var gotStart, gotEnd time.Time
	repo := NewMemoryRepository(nil).WithPeriodBuilder(func(_ context.Context, scope Scope, start, end time.Time) (Snapshot, error) {
		gotStart, gotEnd = start, end
		return Snapshot{
			TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID,
			GeneratedAt: end, PeriodStart: start, PeriodEnd: end, PostureAsOf: end,
			ProjectionVersion: ProjectionVersion, Counts: Counts{CriticalHigh: 7},
		}, nil
	})
	service := NewService(repo)
	service.Now = func() time.Time { return now }

	value, err := service.GetForPeriod(context.Background(), Scope{TenantID: "bank", LegalEntityID: "bank-ng"}, PeriodRequest{
		StartDate: "2026-08-01", EndDate: "2026-10-02",
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotStart.Format(ReportingDateLayout) != "2026-08-01" || !gotEnd.Equal(now) {
		t.Fatalf("builder period start=%s end=%s", gotStart, gotEnd)
	}
	if value.Counts.CriticalHigh != 7 || value.PostureAsOf != now {
		t.Fatalf("current posture drifted: %#v", value)
	}
	if value.ReportingPeriod.StartDate != "2026-08-01" || value.ReportingPeriod.EndDate != "2026-10-02" || value.ReportingPeriod.HistoricalEndSupported {
		t.Fatalf("reporting period=%#v", value.ReportingPeriod)
	}
}

func TestServiceRejectsHistoricalEndAndUnboundedReportingPeriod(t *testing.T) {
	now := time.Date(2026, 10, 2, 15, 30, 0, 0, time.UTC)
	service := NewService(NewMemoryRepository(nil).WithPeriodBuilder(func(_ context.Context, scope Scope, start, end time.Time) (Snapshot, error) {
		return Snapshot{TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID, GeneratedAt: end, PeriodStart: start, PeriodEnd: end, ProjectionVersion: ProjectionVersion}, nil
	}))
	service.Now = func() time.Time { return now }
	scope := Scope{TenantID: "bank", LegalEntityID: "bank-ng"}

	if _, err := service.GetForPeriod(context.Background(), scope, PeriodRequest{StartDate: "2026-09-01", EndDate: "2026-10-01"}); !errors.Is(err, ErrHistoricalEndUnsupported) {
		t.Fatalf("historical end err=%v", err)
	}
	if _, err := service.GetForPeriod(context.Background(), scope, PeriodRequest{StartDate: "2025-10-02", EndDate: "2026-10-02"}); !errors.Is(err, ErrInvalidReportingPeriod) {
		t.Fatalf("unbounded period err=%v", err)
	}
}
