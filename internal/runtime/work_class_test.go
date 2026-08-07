package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type panicMaintainer struct{ calls atomic.Int32 }

func (m *panicMaintainer) Maintain(context.Context, time.Time, int) (int, error) {
	m.calls.Add(1)
	panic("isolated maintainer failure")
}

type channelPublisher struct{ events chan OutboxEvent }

func (p *channelPublisher) Publish(_ context.Context, event OutboxEvent) error {
	select {
	case p.events <- event:
	default:
	}
	return nil
}

type failingPublisher struct{}

func (failingPublisher) Publish(context.Context, OutboxEvent) error {
	return errors.New("publisher unavailable")
}

type failingTimerRepository struct{ *MemoryRepository }

func (r *failingTimerRepository) CompleteTimer(context.Context, Timer, OutboxEvent, time.Time) error {
	return errors.New("timer completion failed")
}

func TestRunIsolatesPanickingMaintainerFromTimerAndOutbox(t *testing.T) {
	repo := NewMemoryRepository()
	publisher := &channelPublisher{events: make(chan OutboxEvent, 1)}
	service := NewService(repo, nil, publisher, "worker-test")
	service.AddMaintainerClass("broken-maintainer", &panicMaintainer{})
	service.ConfigureClass("broken-maintainer", WorkClassOptions{Poll: time.Millisecond, Timeout: 20 * time.Millisecond})
	service.ConfigureClass(WorkClassWorkflowTimers, WorkClassOptions{Poll: time.Millisecond, Timeout: 20 * time.Millisecond})
	service.ConfigureClass(WorkClassOutboxDelivery, WorkClassOptions{Poll: time.Millisecond, Timeout: 20 * time.Millisecond})

	now := time.Now().UTC()
	if _, err := service.Schedule(context.Background(), Timer{TenantID: "bank", WorkflowID: "11111111-1111-7111-8111-111111111111", Type: "REMINDER", DueAt: now, DedupeKey: "isolation", Payload: json.RawMessage(`{}`)}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- service.Run(ctx, time.Millisecond) }()

	select {
	case <-publisher.events:
	case <-time.After(2 * time.Second):
		cancel()
		t.Fatal("timer/outbox work stopped behind the panicking maintainer")
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("unexpected run result: %v", err)
	}
	health, err := service.Health(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	states := map[string]string{}
	for _, item := range health {
		states[item.Name] = item.State
	}
	if states["broken-maintainer"] != WorkClassDegraded {
		t.Fatalf("expected broken class to be degraded, got %#v", states)
	}
	if states[WorkClassOutboxDelivery] != WorkClassCurrent {
		t.Fatalf("expected outbox class to remain current, got %#v", states)
	}
}

func TestTimerFailureBudgetMovesPoisonItemToTerminalState(t *testing.T) {
	base := NewMemoryRepository()
	repo := &failingTimerRepository{MemoryRepository: base}
	service := NewService(repo, nil, &countingPublisher{}, "worker-test")
	service.ConfigureClass(WorkClassWorkflowTimers, WorkClassOptions{MaxAttempts: 2})

	now := time.Date(2026, 8, 7, 2, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	timer, err := service.Schedule(context.Background(), Timer{TenantID: "bank", WorkflowID: "11111111-1111-7111-8111-111111111111", Type: "REMINDER", DueAt: now, DedupeKey: "poison-timer", Payload: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	now = now.Add(3 * time.Second)
	if err := service.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	stored := base.timers[timer.ID]
	if stored.State != TimerFailed || stored.FailedAt == nil || stored.Attempts != 2 {
		t.Fatalf("unexpected terminal timer: %#v", stored)
	}
	health, err := base.TimerQueueHealth(context.Background())
	if err != nil || health.Terminal != 1 || health.Pending != 0 {
		t.Fatalf("unexpected timer queue health: %#v err=%v", health, err)
	}
}

func TestOutboxFailureBudgetDeadLettersPoisonEvent(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Date(2026, 8, 7, 2, 0, 0, 0, time.UTC)
	repo.outbox["event-1"] = OutboxEvent{ID: "event-1", TenantID: "bank", AggregateType: "WORKFLOW", AggregateID: "11111111-1111-7111-8111-111111111111", EventType: "Test", Payload: json.RawMessage(`{}`), OccurredAt: now}
	service := NewService(repo, nil, failingPublisher{}, "worker-test")
	service.ConfigureClass(WorkClassOutboxDelivery, WorkClassOptions{MaxAttempts: 2})
	service.now = func() time.Time { return now }

	if err := service.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	now = now.Add(3 * time.Second)
	if err := service.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	stored := repo.outbox["event-1"]
	if stored.DeadLetteredAt == nil || stored.Attempts != 2 || stored.NextAttemptAt != nil {
		t.Fatalf("unexpected dead-letter event: %#v", stored)
	}
	health, err := service.Health(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range health {
		if item.Name == WorkClassOutboxDelivery {
			if item.State != WorkClassNeedsAttention || item.Queue == nil || item.Queue.Terminal != 1 {
				t.Fatalf("unexpected outbox health: %#v", item)
			}
			return
		}
	}
	t.Fatal("outbox work-class health missing")
}

func TestMemoryOutboxHonorsLeaseBeforeReclaim(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Now().UTC()
	repo.outbox["event-lease"] = OutboxEvent{ID: "event-lease", TenantID: "bank", OccurredAt: now}
	first, err := repo.ClaimOutbox(context.Background(), "worker-a", now, time.Minute, 1)
	if err != nil || len(first) != 1 {
		t.Fatalf("first claim: %v %#v", err, first)
	}
	second, err := repo.ClaimOutbox(context.Background(), "worker-b", now.Add(time.Second), time.Minute, 1)
	if err != nil || len(second) != 0 {
		t.Fatalf("active lease was reclaimed: %v %#v", err, second)
	}
	third, err := repo.ClaimOutbox(context.Background(), "worker-b", now.Add(2*time.Minute), time.Minute, 1)
	if err != nil || len(third) != 1 || third[0].LockedBy != "worker-b" {
		t.Fatalf("expired lease was not reclaimed: %v %#v", err, third)
	}
}

type selectivePanicPublisher struct{ published atomic.Int32 }

func (p *selectivePanicPublisher) Publish(_ context.Context, event OutboxEvent) error {
	if event.ID == "bad-event" {
		panic("bad publisher event")
	}
	p.published.Add(1)
	return nil
}

func TestPublisherPanicIsContainedToOneOutboxItem(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Now().UTC()
	repo.outbox["bad-event"] = OutboxEvent{ID: "bad-event", TenantID: "bank", AggregateType: "WORKFLOW", AggregateID: "11111111-1111-7111-8111-111111111111", EventType: "Bad", Payload: json.RawMessage(`{}`), OccurredAt: now}
	repo.outbox["good-event"] = OutboxEvent{ID: "good-event", TenantID: "bank", AggregateType: "WORKFLOW", AggregateID: "11111111-1111-7111-8111-111111111112", EventType: "Good", Payload: json.RawMessage(`{}`), OccurredAt: now}
	publisher := &selectivePanicPublisher{}
	service := NewService(repo, nil, publisher, "worker-test")
	service.now = func() time.Time { return now }

	if err := service.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if publisher.published.Load() != 1 {
		t.Fatalf("unrelated outbox item did not publish: %d", publisher.published.Load())
	}
	if _, exists := repo.outbox["good-event"]; exists {
		t.Fatal("successfully published outbox item remained pending")
	}
	bad := repo.outbox["bad-event"]
	if bad.NextAttemptAt == nil || bad.LastError == "" || bad.LockedBy != "" {
		t.Fatalf("publisher panic was not converted into an item retry: %#v", bad)
	}
}
