package runtime

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

type countingPublisher struct{ count int }

func (p *countingPublisher) Publish(context.Context, OutboxEvent) error {
	p.count++
	return nil
}

func TestTimerBecomesPublishedOutboxEvent(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	publisher := &countingPublisher{}
	svc := NewService(repo, nil, publisher, "w1")
	now := time.Date(2026, 8, 5, 8, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	_, err := svc.Schedule(ctx, Timer{TenantID: "t1", WorkflowID: "wf1", Type: "ESCALATION", DueAt: now, DedupeKey: "wf1:escalate", Payload: json.RawMessage(`{"task":"a"}`)})
	if err != nil {
		t.Fatal(err)
	}
	if err = svc.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if publisher.count != 1 {
		t.Fatalf("expected one event, got %d", publisher.count)
	}
}

func TestExpiredTimerLeaseCanBeReclaimedAndStaleWorkerCannotComplete(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	now := time.Date(2026, 8, 5, 8, 0, 0, 0, time.UTC)
	timer := Timer{ID: "timer-1", TenantID: "t1", WorkflowID: "wf1", Type: "REMINDER", DueAt: now, State: TimerReady, DedupeKey: "wf1:reminder", Payload: json.RawMessage(`{}`)}
	if _, err := repo.ScheduleTimer(ctx, timer); err != nil {
		t.Fatal(err)
	}
	first, err := repo.ClaimDueTimers(ctx, "worker-a", now, time.Second, 1)
	if err != nil || len(first) != 1 {
		t.Fatalf("first claim: %v %#v", err, first)
	}
	second, err := repo.ClaimDueTimers(ctx, "worker-b", now.Add(2*time.Second), time.Second, 1)
	if err != nil || len(second) != 1 || second[0].LockedBy != "worker-b" {
		t.Fatalf("reclaim: %v %#v", err, second)
	}
	event := OutboxEvent{ID: "event-1", TenantID: "t1", AggregateType: "WORKFLOW", AggregateID: "wf1", EventType: "Timer", Payload: json.RawMessage(`{}`), OccurredAt: now}
	if err := repo.CompleteTimer(ctx, first[0], event, now); err == nil {
		t.Fatal("expected stale worker completion to fail")
	}
	if err := repo.CompleteTimer(ctx, second[0], event, now); err != nil {
		t.Fatal(err)
	}
}

func TestOutboxClaimOwnershipIsEnforced(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	now := time.Now().UTC()
	timer := Timer{ID: "timer-2", TenantID: "t1", WorkflowID: "wf1", Type: "REMINDER", DueAt: now, State: TimerClaimed, DedupeKey: "wf1:r2", LockedBy: "timer-worker"}
	repo.timers[timer.ID] = timer
	event := OutboxEvent{ID: "event-2", TenantID: "t1", AggregateType: "WORKFLOW", AggregateID: "wf1", EventType: "Timer", Payload: json.RawMessage(`{}`), OccurredAt: now}
	if err := repo.CompleteTimer(ctx, timer, event, now); err != nil {
		t.Fatal(err)
	}
	claimed, err := repo.ClaimOutbox(ctx, "publisher-a", now, time.Second, 1)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim: %v %#v", err, claimed)
	}
	stale := claimed[0]
	stale.LockedBy = "publisher-b"
	if err := repo.MarkPublished(ctx, stale, now); err == nil {
		t.Fatal("expected stale outbox claim to fail")
	}
	if err := repo.MarkPublished(ctx, claimed[0], now); err != nil {
		t.Fatal(err)
	}
}

func TestInboxIsIdempotent(t *testing.T) {
	repo := NewMemoryRepository()
	first, _ := repo.RecordInbox(context.Background(), "t", "c", "e", time.Now())
	second, _ := repo.RecordInbox(context.Background(), "t", "c", "e", time.Now())
	if !first || second {
		t.Fatal("expected first insert only")
	}
}
