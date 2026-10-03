package oploss

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestLossRecoveryTotalsAreDerivedAndReversible(t *testing.T) {
	ctx := context.Background()
	service := NewService(NewMemoryRepository())
	now := time.Date(2026, 10, 3, 18, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }

	created, err := service.Create(ctx, CreateInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "LOSS-1", Title: "Payment processing loss",
		EventType: EventExecutionDeliveryProcess, Cause: "Duplicate settlement instruction.",
		Description: "Duplicate debit was settled before reversal.", GrossAmountMinor: 5_000_000_00, Currency: "NGN",
		OccurredAt: now.Add(-2 * time.Hour), DiscoveredAt: now.Add(-time.Hour),
		OwnerPrincipalID: "owner-1", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Version != 1 {
		t.Fatalf("created=%#v", created)
	}

	now = now.Add(time.Minute)
	afterRecovery, recovery, err := service.AddRecovery(ctx, RecoveryInput{
		TenantID: "bank", LegalEntityID: "entity-a", LossID: created.ID, ExpectedVersion: 1,
		Kind: RecoveryCash, AmountMinor: 2_000_000_00, Reference: "Insurance payment", RecoveredAt: now,
		ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if afterRecovery.Version != 2 || recovery.LossVersion != 2 {
		t.Fatalf("recovery version loss=%#v recovery=%#v", afterRecovery, recovery)
	}
	aggregate, err := service.Get(ctx, Scope{TenantID: "bank", LegalEntityID: "entity-a"}, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if aggregate.Totals.GrossAmountMinor != 5_000_000_00 ||
		aggregate.Totals.RecoveredAmountMinor != 2_000_000_00 ||
		aggregate.Totals.NetLossMinor != 3_000_000_00 ||
		aggregate.Totals.RecoveryStatus != "PARTIAL" {
		t.Fatalf("totals=%#v", aggregate.Totals)
	}

	now = now.Add(time.Minute)
	afterReversal, _, err := service.AddRecovery(ctx, RecoveryInput{
		TenantID: "bank", LegalEntityID: "entity-a", LossID: created.ID, ExpectedVersion: 2,
		Kind: RecoveryReversal, AmountMinor: 500_000_00, Reference: "Recovery correction", RecoveredAt: now,
		ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	aggregate, err = service.Get(ctx, Scope{TenantID: "bank", LegalEntityID: "entity-a"}, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if afterReversal.Version != 3 || aggregate.Totals.RecoveredAmountMinor != 1_500_000_00 || aggregate.Totals.NetLossMinor != 3_500_000_00 {
		t.Fatalf("reversed aggregate=%#v", aggregate)
	}
}

func TestLossRejectsOverRecoveryAndCurrencyMutation(t *testing.T) {
	ctx := context.Background()
	service := NewService(NewMemoryRepository())
	now := time.Date(2026, 10, 3, 18, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }
	created, err := service.Create(ctx, CreateInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "LOSS-2", Title: "Fraud loss",
		EventType: EventExternalFraud, Cause: "External fraud.", GrossAmountMinor: 100_00, Currency: "USD",
		OccurredAt: now.Add(-time.Hour), DiscoveredAt: now, OwnerPrincipalID: "owner-1", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = service.AddRecovery(ctx, RecoveryInput{
		TenantID: "bank", LegalEntityID: "entity-a", LossID: created.ID, ExpectedVersion: 1,
		Kind: RecoveryCash, AmountMinor: 101_00, ActorID: "owner-1",
	})
	if !errors.Is(err, ErrRecoveryLimit) {
		t.Fatalf("over recovery error=%v", err)
	}

	_, err = service.Update(ctx, UpdateInput{
		TenantID: "bank", LegalEntityID: "entity-a", LossID: created.ID, ExpectedVersion: 1,
		Title: created.Title, EventType: created.EventType, Cause: created.Cause,
		GrossAmountMinor: created.GrossAmountMinor, Currency: "EUR",
		OccurredAt: created.OccurredAt, DiscoveredAt: created.DiscoveredAt, Status: StatusActive,
		OwnerPrincipalID: created.OwnerPrincipalID, ActorID: "owner-1",
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("currency mutation error=%v", err)
	}
}

func TestLossRejectsRecoveryBeforeOccurrence(t *testing.T) {
	ctx := context.Background()
	service := NewService(NewMemoryRepository())
	now := time.Date(2026, 10, 3, 18, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }
	created, err := service.Create(ctx, CreateInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "LOSS-DATE", Title: "Timed loss",
		EventType: EventOther, Cause: "Operational event.", GrossAmountMinor: 100_00, Currency: "NGN",
		OccurredAt: now.Add(-time.Hour), DiscoveredAt: now, OwnerPrincipalID: "owner-1", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = service.AddRecovery(ctx, RecoveryInput{
		TenantID: "bank", LegalEntityID: "entity-a", LossID: created.ID, ExpectedVersion: 1,
		Kind: RecoveryCash, AmountMinor: 10_00, RecoveredAt: created.OccurredAt.Add(-time.Minute), ActorID: "owner-1",
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("recovery before occurrence error=%v", err)
	}
}

func TestLossCodeIsScopedPerLegalEntity(t *testing.T) {
	ctx := context.Background()
	service := NewService(NewMemoryRepository())
	now := time.Date(2026, 10, 3, 18, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }
	input := CreateInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "LOSS-3", Title: "Loss",
		EventType: EventOther, Cause: "Cause.", GrossAmountMinor: 1_00, Currency: "NGN",
		OccurredAt: now.Add(-time.Hour), DiscoveredAt: now, OwnerPrincipalID: "owner-1", ActorID: "owner-1",
	}
	if _, err := service.Create(ctx, input); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Create(ctx, input); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("duplicate code error=%v", err)
	}
	input.LegalEntityID = "entity-b"
	if _, err := service.Create(ctx, input); err != nil {
		t.Fatalf("same code in another entity rejected: %v", err)
	}
}
