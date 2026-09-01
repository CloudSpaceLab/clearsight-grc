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

func TestServiceDoesNotSubstituteMetricsWhenProjectionIsMissing(t *testing.T) {
	service := NewService(NewMemoryRepository(nil))
	_, err := service.Get(context.Background(), Scope{TenantID: "bank", LegalEntityID: "bank-ng"})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err=%v", err)
	}
}
