package monitoring

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMemoryCollectionCyclesAreTenantScopedAndSummarizedAfterScope(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	for _, cycle := range []CollectionCycle{
		collectionCycleFixture("cycle-a-1", "bank-a", "program-a", "check-a", 1, now.Add(time.Hour)),
		collectionCycleFixture("cycle-a-2", "bank-a", "program-a", "check-b", 1, now.Add(2*time.Hour)),
		collectionCycleFixture("cycle-b-1", "bank-b", "program-b", "check-c", 1, now.Add(-time.Hour)),
	} {
		if _, err := repo.UpsertCollectionCycle(context.Background(), cycle); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := repo.CollectionCycle(context.Background(), "bank-b", "cycle-a-1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-tenant cycle read error = %v", err)
	}
	summaries, err := repo.ListCollectionSummaries(context.Background(), "bank-a", "program-a", 1)
	if err != nil || len(summaries) != 1 || summaries[0].TenantID != "bank-a" || summaries[0].ProgramID != "program-a" {
		t.Fatalf("scoped summaries = %#v, err = %v", summaries, err)
	}
	all, err := repo.ListCollectionSummaries(context.Background(), "bank-a", "program-a", 10)
	if err != nil || len(all) != 2 {
		t.Fatalf("all summaries = %#v, err = %v", all, err)
	}
}

func TestMemoryCollectionCycleClaimOrdersDueWorkAndFencesStaleLease(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	later := collectionCycleFixture("cycle-later", "bank-a", "program-a", "check-a", 1, now.Add(-time.Minute))
	earlier := collectionCycleFixture("cycle-earlier", "bank-a", "program-a", "check-b", 1, now.Add(-2*time.Minute))
	for _, cycle := range []CollectionCycle{later, earlier} {
		if _, err := repo.UpsertCollectionCycle(context.Background(), cycle); err != nil {
			t.Fatal(err)
		}
	}
	claims, err := repo.ClaimDueCollectionCycles(context.Background(), "worker-a", now, time.Minute, 1)
	if err != nil || len(claims) != 1 || claims[0].ID != earlier.ID || claims[0].LeaseToken == "" {
		t.Fatalf("first claim = %#v, err = %v", claims, err)
	}
	reclaimed, err := repo.ClaimDueCollectionCycles(context.Background(), "worker-b", now.Add(2*time.Minute), time.Minute, 1)
	if err != nil || len(reclaimed) != 1 || reclaimed[0].ID != earlier.ID || reclaimed[0].LeaseToken == claims[0].LeaseToken {
		t.Fatalf("reclaimed = %#v, err = %v", reclaimed, err)
	}
	completion := CollectionActionCompletion{State: CycleAwaitingResponse, NextActionAt: timePtr(now.Add(3 * time.Minute)), At: now.Add(2*time.Minute + 10*time.Second)}
	if _, err := repo.CompleteCollectionAction(context.Background(), claims[0], completion); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale completion error = %v, want conflict", err)
	}
	completed, err := repo.CompleteCollectionAction(context.Background(), reclaimed[0], completion)
	if err != nil || completed.State != CycleAwaitingResponse || completed.LeaseToken != "" {
		t.Fatalf("completed = %#v, err = %v", completed, err)
	}
}

func TestMemoryCollectionCycleRetryBecomesTerminalAndCancellationStopsWork(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	cycle := collectionCycleFixture("cycle-1", "bank-a", "program-a", "check-a", 1, now.Add(-time.Minute))
	if _, err := repo.UpsertCollectionCycle(context.Background(), cycle); err != nil {
		t.Fatal(err)
	}
	claim := claimOneCollectionCycle(t, repo, now)
	retryAt := now.Add(time.Minute)
	retried, err := repo.FailCollectionAction(context.Background(), claim, "Delivery service unavailable", &retryAt, 2, now)
	if err != nil || retried.State != CycleScheduled || retried.Attempts != 1 || retried.NextActionAt == nil {
		t.Fatalf("retry = %#v, err = %v", retried, err)
	}
	claim = claimOneCollectionCycle(t, repo, retryAt)
	failed, err := repo.FailCollectionAction(context.Background(), claim, "Delivery service unavailable", nil, 2, retryAt)
	if err != nil || failed.State != CycleFailed || failed.Attempts != 2 || failed.NextActionAt != nil {
		t.Fatalf("terminal failure = %#v, err = %v", failed, err)
	}

	second := collectionCycleFixture("cycle-2", "bank-a", "program-a", "check-a", 2, now.Add(time.Hour))
	if _, err := repo.UpsertCollectionCycle(context.Background(), second); err != nil {
		t.Fatal(err)
	}
	cancelled, err := repo.CancelCollectionCyclesByCheck(context.Background(), "bank-a", "check-a", now)
	if err != nil || cancelled != 1 {
		t.Fatalf("cancelled = %d, err = %v", cancelled, err)
	}
	stored, err := repo.CollectionCycle(context.Background(), "bank-a", second.ID)
	if err != nil || stored.State != CycleCancelled || stored.NextActionAt != nil {
		t.Fatalf("cancelled cycle = %#v, err = %v", stored, err)
	}
}

func collectionCycleFixture(id, tenant, program, check string, sequence int64, next time.Time) CollectionCycle {
	expires := next.Add(30 * 24 * time.Hour)
	return CollectionCycle{
		ID: id, TenantID: tenant, ProgramID: program, MonitoringCheckID: check, MonitoringCheckVersion: 1, Sequence: sequence,
		Policy:    CollectionPolicy{ValidityMonths: 12, RenewalWindowDays: 30, ReminderCount: 3},
		ExpiresAt: expires, RenewalOpensAt: expires.Add(-30 * 24 * time.Hour), NextActionAt: &next,
		Recipient:     RecipientRoute{Type: RouteInternalPrincipal, PrincipalID: "principal-1"},
		DeliveryState: DeliveryNotDispatched, State: CycleScheduled, CreatedAt: next.Add(-time.Hour), UpdatedAt: next.Add(-time.Hour),
	}
}

func claimOneCollectionCycle(t *testing.T, repo *MemoryRepository, now time.Time) CollectionCycle {
	t.Helper()
	claims, err := repo.ClaimDueCollectionCycles(context.Background(), "worker", now, time.Minute, 1)
	if err != nil || len(claims) != 1 {
		t.Fatalf("claim = %#v, err = %v", claims, err)
	}
	return claims[0]
}

func timePtr(value time.Time) *time.Time { return &value }
