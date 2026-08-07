package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

type Maintainer interface {
	Maintain(context.Context, time.Time, int) (int, error)
}

type Service struct {
	repo         Repository
	lifecycle    DelegationLifecycle
	publisher    Publisher
	maintainers  []namedMaintainer
	classOptions map[string]WorkClassOptions
	workerID     string
	now          func() time.Time
	logger       *slog.Logger
	healthMu     sync.RWMutex
	health       map[string]WorkClassHealth
}

func NewService(repo Repository, lifecycle DelegationLifecycle, publisher Publisher, workerID string) *Service {
	if workerID == "" {
		workerID = "worker"
	}
	service := &Service{
		repo:         repo,
		lifecycle:    lifecycle,
		publisher:    publisher,
		classOptions: map[string]WorkClassOptions{},
		workerID:     workerID,
		now:          time.Now,
		health:       map[string]WorkClassHealth{},
	}
	if lifecycle != nil {
		service.health[WorkClassDelegationLifecycle] = WorkClassHealth{Name: WorkClassDelegationLifecycle, State: WorkClassStarting}
	}
	service.health[WorkClassWorkflowTimers] = WorkClassHealth{Name: WorkClassWorkflowTimers, State: WorkClassStarting}
	service.health[WorkClassOutboxDelivery] = WorkClassHealth{Name: WorkClassOutboxDelivery, State: WorkClassStarting}
	return service
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
	timer.ID, timer.State = valueID, TimerReady
	return s.repo.ScheduleTimer(ctx, timer)
}

func (s *Service) maintainDelegations(ctx context.Context, run classRun) (int, error) {
	if s.lifecycle == nil {
		return 0, nil
	}
	activated, activateErr := s.lifecycle.ActivateDueDelegations(ctx, run.now, run.batch)
	if ctx.Err() != nil {
		return activated, ctx.Err()
	}
	expired, expireErr := s.lifecycle.ExpireDueDelegations(ctx, run.now, run.batch)
	return activated + expired, errors.Join(activateErr, expireErr)
}

func (s *Service) maintainTimers(ctx context.Context, run classRun) (int, error) {
	timers, err := s.repo.ClaimDueTimers(ctx, run.workerID, run.now, run.lease, run.batch)
	if err != nil {
		return 0, err
	}
	processed := 0
	var persistenceErrors []error
	for _, timer := range timers {
		if ctx.Err() != nil {
			return processed, ctx.Err()
		}
		eventID, eventErr := id.NewUUIDv7()
		if eventErr == nil {
			event := OutboxEvent{ID: eventID, TenantID: timer.TenantID, AggregateType: "WORKFLOW", AggregateID: timer.WorkflowID, EventType: "WorkflowTimerFired", Payload: timer.Payload, OccurredAt: run.now}
			eventErr = s.repo.CompleteTimer(ctx, timer, event, run.now)
		}
		if eventErr != nil {
			terminal, failErr := s.repo.FailTimer(ctx, timer, run.maxAttempts, eventErr.Error(), run.now, run.now.Add(itemBackoff(timer.Attempts, run.maxBackoff)))
			if failErr != nil {
				persistenceErrors = append(persistenceErrors, fmt.Errorf("fail timer %s: %w", timer.ID, failErr))
			} else if terminal && s.logger != nil {
				s.logger.Warn("workflow timer moved to terminal failure", "work_class", WorkClassWorkflowTimers, "timer_id", timer.ID, "attempts", timer.Attempts)
			}
		}
		processed++
	}
	return processed, errors.Join(persistenceErrors...)
}

func (s *Service) maintainOutbox(ctx context.Context, run classRun) (int, error) {
	events, err := s.repo.ClaimOutbox(ctx, run.workerID, run.now, run.lease, run.batch)
	if err != nil {
		return 0, err
	}
	processed := 0
	var persistenceErrors []error
	for _, event := range events {
		if ctx.Err() != nil {
			return processed, ctx.Err()
		}
		publishErr := publishSafely(ctx, s.publisher, event)
		if publishErr != nil {
			terminal, failErr := s.repo.MarkFailed(ctx, event, run.maxAttempts, publishErr.Error(), run.now, run.now.Add(itemBackoff(event.Attempts, run.maxBackoff)))
			if failErr != nil {
				persistenceErrors = append(persistenceErrors, fmt.Errorf("fail outbox event %s: %w", event.ID, failErr))
			} else if terminal && s.logger != nil {
				s.logger.Warn("outbox event moved to dead letter", "work_class", WorkClassOutboxDelivery, "event_id", event.ID, "event_type", event.EventType, "attempts", event.Attempts)
			}
			processed++
			continue
		}
		if err := s.repo.MarkPublished(ctx, event, run.now); err != nil {
			persistenceErrors = append(persistenceErrors, fmt.Errorf("mark outbox event %s published: %w", event.ID, err))
		}
		processed++
	}
	return processed, errors.Join(persistenceErrors...)
}

func publishSafely(ctx context.Context, publisher Publisher, event OutboxEvent) (err error) {
	if publisher == nil {
		return errors.New("outbox publisher is not configured")
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("publisher panic: %v", recovered)
		}
	}()
	return publisher.Publish(ctx, event)
}

func itemBackoff(attempt int, maximum time.Duration) time.Duration {
	if maximum <= 0 {
		maximum = 5 * time.Minute
	}
	seconds := math.Pow(2, float64(max(attempt, 0)))
	delay := time.Duration(seconds * float64(time.Second))
	if delay <= 0 || delay > maximum {
		return maximum
	}
	return delay
}

type LogPublisher struct{ Logger *slog.Logger }

func (p LogPublisher) Publish(_ context.Context, event OutboxEvent) error {
	if p.Logger != nil {
		p.Logger.Info("outbox event published", "event_id", event.ID, "type", event.EventType, "aggregate_type", event.AggregateType, "aggregate_id", event.AggregateID)
	}
	return nil
}
