package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

type Service struct {
	repo      Repository
	lifecycle DelegationLifecycle
	publisher Publisher
	workerID  string
	lease     time.Duration
	batch     int
	now       func() time.Time
}

func NewService(repo Repository, lifecycle DelegationLifecycle, publisher Publisher, workerID string) *Service {
	if workerID == "" {
		workerID = "worker"
	}
	return &Service{repo: repo, lifecycle: lifecycle, publisher: publisher, workerID: workerID, lease: 30 * time.Second, batch: 50, now: time.Now}
}
func (s *Service) Schedule(ctx context.Context, timer Timer) (Timer, error) {
	if strings.TrimSpace(timer.TenantID) == "" || strings.TrimSpace(timer.WorkflowID) == "" || strings.TrimSpace(timer.Type) == "" || strings.TrimSpace(timer.DedupeKey) == "" || timer.DueAt.IsZero() {
		return Timer{}, fmt.Errorf("tenant, workflow, type, due_at and dedupe_key are required")
	}
	if len(timer.Payload) == 0 {
		timer.Payload = json.RawMessage(`{}`)
	}
	valueID, err := id.NewUUIDv7()
	if err != nil {
		return Timer{}, err
	}
	timer.ID = valueID
	timer.State = TimerReady
	return s.repo.ScheduleTimer(ctx, timer)
}
func (s *Service) Tick(ctx context.Context) error {
	now := s.now().UTC()
	if s.lifecycle != nil {
		if _, err := s.lifecycle.ActivateDueDelegations(ctx, now, s.batch); err != nil {
			return fmt.Errorf("activate delegations: %w", err)
		}
		if _, err := s.lifecycle.ExpireDueDelegations(ctx, now, s.batch); err != nil {
			return fmt.Errorf("expire delegations: %w", err)
		}
	}
	timers, err := s.repo.ClaimDueTimers(ctx, s.workerID, now, s.lease, s.batch)
	if err != nil {
		return err
	}
	for _, timer := range timers {
		eventID, err := id.NewUUIDv7()
		if err != nil {
			return err
		}
		event := OutboxEvent{ID: eventID, TenantID: timer.TenantID, AggregateType: "WORKFLOW", AggregateID: timer.WorkflowID, EventType: "WorkflowTimerFired", Payload: timer.Payload, OccurredAt: now}
		if err := s.repo.CompleteTimer(ctx, timer, event, now); err != nil {
			_ = s.repo.FailTimer(ctx, timer, err.Error(), now.Add(backoff(timer.Attempts)))
			continue
		}
	}
	events, err := s.repo.ClaimOutbox(ctx, s.workerID, now, s.lease, s.batch)
	if err != nil {
		return err
	}
	for _, event := range events {
		if err := s.publisher.Publish(ctx, event); err != nil {
			_ = s.repo.MarkFailed(ctx, event, err.Error(), now.Add(backoff(event.Attempts)))
			continue
		}
		if err := s.repo.MarkPublished(ctx, event, now); err != nil {
			return err
		}
	}
	return nil
}
func (s *Service) Run(ctx context.Context, poll time.Duration) error {
	if poll <= 0 {
		poll = time.Second
	}
	ticker := time.NewTicker(poll)
	defer ticker.Stop()
	for {
		if err := s.Tick(ctx); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
func backoff(attempt int) time.Duration {
	seconds := math.Min(300, math.Pow(2, float64(attempt)))
	return time.Duration(seconds) * time.Second
}

type LogPublisher struct{ Logger *slog.Logger }

func (p LogPublisher) Publish(_ context.Context, event OutboxEvent) error {
	p.Logger.Info("outbox event published", "event_id", event.ID, "type", event.EventType, "aggregate_type", event.AggregateType, "aggregate_id", event.AggregateID)
	return nil
}
