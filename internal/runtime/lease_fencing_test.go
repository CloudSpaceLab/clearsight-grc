package runtime

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMemoryTimerLeaseGenerationFencesStaleSameWorker(t *testing.T) {
	ctx := context.Background()
	repository := NewMemoryRepository()
	now := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	timer := Timer{
		ID: "timer-1", TenantID: "bank", WorkflowID: "workflow-1", Type: "source-pull",
		DueAt: now, State: TimerReady, DedupeKey: "source-pull:binding-1", Payload: []byte(`{"binding_id":"binding-1"}`),
	}
	if _, err := repository.ScheduleTimer(ctx, timer); err != nil {
		t.Fatal(err)
	}
	first, err := repository.ClaimDueTimers(ctx, "worker-a", now, time.Minute, 1)
	if err != nil || len(first) != 1 {
		t.Fatalf("first claim=%#v err=%v", first, err)
	}
	second, err := repository.ClaimDueTimers(ctx, "worker-a", now.Add(2*time.Minute), time.Minute, 1)
	if err != nil || len(second) != 1 {
		t.Fatalf("same worker did not reclaim expired timer: claims=%#v err=%v", second, err)
	}
	if first[0].LeaseUntil == nil || second[0].LeaseUntil == nil || first[0].LeaseUntil.Equal(*second[0].LeaseUntil) {
		t.Fatalf("timer lease generation did not change: first=%#v second=%#v", first[0], second[0])
	}
	event := OutboxEvent{ID: "event-stale", TenantID: "bank", AggregateType: "SOURCE_BINDING", AggregateID: "binding-1", EventType: "SourcePull", OccurredAt: now.Add(2 * time.Minute)}
	if err := repository.CompleteTimer(ctx, first[0], event, now.Add(2*time.Minute+10*time.Second)); err == nil {
		t.Fatal("stale same-worker timer claim completed the newer lease")
	}
	if _, err := repository.FailTimer(ctx, first[0], 3, "STALE", now.Add(2*time.Minute+10*time.Second), now.Add(4*time.Minute)); err == nil {
		t.Fatal("stale same-worker timer claim failed the newer lease")
	}
	event.ID = "event-current"
	if err := repository.CompleteTimer(ctx, second[0], event, now.Add(2*time.Minute+10*time.Second)); err != nil {
		t.Fatalf("current timer lease could not complete: %v", err)
	}
}

func TestMemoryOutboxLeaseGenerationFencesStaleSameWorker(t *testing.T) {
	ctx := context.Background()
	repository := NewMemoryRepository()
	now := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	timer := Timer{ID: "timer-1", TenantID: "bank", WorkflowID: "workflow-1", Type: "source-pull", DueAt: now, State: TimerReady, DedupeKey: "outbox-source", Payload: []byte(`{}`)}
	if _, err := repository.ScheduleTimer(ctx, timer); err != nil {
		t.Fatal(err)
	}
	claim, err := repository.ClaimDueTimers(ctx, "timer-worker", now, time.Minute, 1)
	if err != nil || len(claim) != 1 {
		t.Fatalf("timer claim=%#v err=%v", claim, err)
	}
	event := OutboxEvent{ID: "event-1", TenantID: "bank", AggregateType: "SOURCE_BINDING", AggregateID: "binding-1", EventType: "SourcePull", OccurredAt: now}
	if err := repository.CompleteTimer(ctx, claim[0], event, now.Add(10*time.Second)); err != nil {
		t.Fatal(err)
	}
	first, err := repository.ClaimOutbox(ctx, "worker-a", now.Add(20*time.Second), time.Minute, 1)
	if err != nil || len(first) != 1 {
		t.Fatalf("first outbox claim=%#v err=%v", first, err)
	}
	second, err := repository.ClaimOutbox(ctx, "worker-a", now.Add(2*time.Minute), time.Minute, 1)
	if err != nil || len(second) != 1 {
		t.Fatalf("same worker did not reclaim expired outbox event: claims=%#v err=%v", second, err)
	}
	if first[0].LeaseUntil == nil || second[0].LeaseUntil == nil || first[0].LeaseUntil.Equal(*second[0].LeaseUntil) {
		t.Fatalf("outbox lease generation did not change: first=%#v second=%#v", first[0], second[0])
	}
	at := now.Add(2*time.Minute + 10*time.Second)
	if err := repository.MarkPublished(ctx, first[0], at); err == nil {
		t.Fatal("stale same-worker outbox claim published the newer lease")
	}
	if _, err := repository.MarkFailed(ctx, first[0], 3, "STALE", at, now.Add(4*time.Minute)); err == nil {
		t.Fatal("stale same-worker outbox claim failed the newer lease")
	}
	if err := repository.MarkPublished(ctx, second[0], at); err != nil {
		t.Fatalf("current outbox lease could not publish: %v", err)
	}
	if err := repository.MarkPublished(ctx, second[0], at); !errors.Is(err, errors.New("outbox claim lost")) && err == nil {
		t.Fatal("published event remained claimable")
	}
}

func TestMemoryRuntimeRejectsCompletionAfterLeaseExpiry(t *testing.T) {
	ctx := context.Background()
	repository := NewMemoryRepository()
	now := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	timer := Timer{ID: "timer-1", TenantID: "bank", WorkflowID: "workflow-1", Type: "source-pull", DueAt: now, State: TimerReady, DedupeKey: "lease-expiry", Payload: []byte(`{}`)}
	if _, err := repository.ScheduleTimer(ctx, timer); err != nil {
		t.Fatal(err)
	}
	claims, err := repository.ClaimDueTimers(ctx, "worker-a", now, time.Minute, 1)
	if err != nil || len(claims) != 1 {
		t.Fatalf("claims=%#v err=%v", claims, err)
	}
	event := OutboxEvent{ID: "event-1", TenantID: "bank", AggregateType: "SOURCE_BINDING", AggregateID: "binding-1", EventType: "SourcePull", OccurredAt: now}
	if err := repository.CompleteTimer(ctx, claims[0], event, now.Add(time.Minute+time.Second)); err == nil {
		t.Fatal("timer completed after its lease expired")
	}
}
