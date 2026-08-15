package runtime

import (
	"context"
	"errors"
	"sync"
	"time"
)

type MemoryRepository struct {
	mu     sync.Mutex
	timers map[string]Timer
	outbox map[string]OutboxEvent
	inbox  map[string]struct{}
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{timers: map[string]Timer{}, outbox: map[string]OutboxEvent{}, inbox: map[string]struct{}{}}
}
func (r *MemoryRepository) ScheduleTimer(_ context.Context, v Timer) (Timer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, x := range r.timers {
		if x.TenantID == v.TenantID && x.DedupeKey == v.DedupeKey {
			return x, nil
		}
	}
	r.timers[v.ID] = v
	return v, nil
}
func (r *MemoryRepository) ClaimDueTimers(_ context.Context, worker string, now time.Time, lease time.Duration, limit int) ([]Timer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []Timer{}
	for id, v := range r.timers {
		if len(out) >= limit {
			break
		}
		if (v.State == TimerReady || (v.State == TimerClaimed && v.LeaseUntil != nil && v.LeaseUntil.Before(now))) && !v.DueAt.After(now) {
			until := now.Add(lease)
			v.State = TimerClaimed
			v.LeaseUntil = &until
			v.Attempts++
			v.LockedBy = worker
			r.timers[id] = v
			out = append(out, v)
		}
	}
	return out, nil
}
func (r *MemoryRepository) CompleteTimer(_ context.Context, t Timer, e OutboxEvent, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.timers[t.ID]
	if !ok {
		return errors.New("timer not found")
	}
	if v.State != TimerClaimed || v.LockedBy != t.LockedBy || !sameLeaseGeneration(v.LeaseUntil, t.LeaseUntil) || v.LeaseUntil.Before(now) {
		return errors.New("timer claim lost")
	}
	v.State = TimerFired
	v.LeaseUntil = nil
	v.LockedBy = ""
	v.LastError = ""
	r.timers[t.ID] = v
	r.outbox[e.ID] = e
	return nil
}
func (r *MemoryRepository) FailTimer(_ context.Context, t Timer, maxAttempts int, message string, at, next time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.timers[t.ID]
	if !ok || v.State != TimerClaimed || v.LockedBy != t.LockedBy || !sameLeaseGeneration(v.LeaseUntil, t.LeaseUntil) || v.LeaseUntil.Before(at) {
		return false, errors.New("timer claim lost")
	}
	terminal := maxAttempts > 0 && v.Attempts >= maxAttempts
	v.LastError = message
	v.LeaseUntil = nil
	v.LockedBy = ""
	if terminal {
		v.State = TimerFailed
		v.FailedAt = timePtr(at)
	} else {
		v.State = TimerReady
		v.DueAt = next
	}
	r.timers[t.ID] = v
	return terminal, nil
}
func (r *MemoryRepository) CancelPendingTaskTimers(_ context.Context, tenant, taskID, timerType string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cancelled := 0
	for id, timer := range r.timers {
		if timer.TenantID != tenant || timer.TaskID != taskID || timer.Type != timerType || timer.State != TimerReady {
			continue
		}
		timer.State = TimerCancelled
		r.timers[id] = timer
		cancelled++
	}
	return cancelled, nil
}
func (r *MemoryRepository) ClaimOutbox(_ context.Context, worker string, now time.Time, lease time.Duration, limit int) ([]OutboxEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []OutboxEvent{}
	for id, value := range r.outbox {
		if len(out) >= limit {
			break
		}
		if value.DeadLetteredAt != nil || (value.NextAttemptAt != nil && value.NextAttemptAt.After(now)) {
			continue
		}
		if value.LockedBy != "" && value.LeaseUntil != nil && !value.LeaseUntil.Before(now) {
			continue
		}
		until := now.Add(lease)
		value.Attempts++
		value.LockedBy = worker
		value.LeaseUntil = &until
		r.outbox[id] = value
		out = append(out, value)
	}
	return out, nil
}
func (r *MemoryRepository) MarkPublished(_ context.Context, e OutboxEvent, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.outbox[e.ID]
	if !ok || value.LockedBy != e.LockedBy || !sameLeaseGeneration(value.LeaseUntil, e.LeaseUntil) || value.LeaseUntil.Before(at) {
		return errors.New("outbox claim lost")
	}
	delete(r.outbox, e.ID)
	return nil
}
func (r *MemoryRepository) MarkFailed(_ context.Context, e OutboxEvent, maxAttempts int, message string, at, next time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.outbox[e.ID]
	if !ok || value.LockedBy != e.LockedBy || !sameLeaseGeneration(value.LeaseUntil, e.LeaseUntil) || value.LeaseUntil.Before(at) {
		return false, errors.New("outbox claim lost")
	}
	terminal := maxAttempts > 0 && value.Attempts >= maxAttempts
	value.LastError = message
	value.LockedBy = ""
	value.LeaseUntil = nil
	if terminal {
		value.DeadLetteredAt = timePtr(at)
		value.NextAttemptAt = nil
	} else {
		value.NextAttemptAt = timePtr(next)
	}
	r.outbox[e.ID] = value
	return terminal, nil
}
func (r *MemoryRepository) InboxProcessed(_ context.Context, tenant, consumer, eventID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.inbox[tenant+":"+consumer+":"+eventID]
	return ok, nil
}
func (r *MemoryRepository) RecordInbox(_ context.Context, tenant, consumer, eventID string, at time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := tenant + ":" + consumer + ":" + eventID
	if _, ok := r.inbox[k]; ok {
		return false, nil
	}
	r.inbox[k] = struct{}{}
	return true, nil
}
func (r *MemoryRepository) RecordInboxWithOutbox(_ context.Context, receipts []InboxReceipt, event OutboxEvent, _ time.Time) (bool, error) {
	if len(receipts) == 0 {
		return false, errors.New("at least one inbox receipt is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	first := receipts[0].TenantID + ":" + receipts[0].Consumer + ":" + receipts[0].EventID
	if _, exists := r.inbox[first]; exists {
		return false, nil
	}
	keys := make([]string, len(receipts))
	for index, receipt := range receipts {
		if receipt.TenantID != event.TenantID || receipt.TenantID == "" || receipt.Consumer == "" || receipt.EventID == "" {
			return false, errors.New("inbox receipt does not match outbox tenant")
		}
		keys[index] = receipt.TenantID + ":" + receipt.Consumer + ":" + receipt.EventID
		if index > 0 {
			if _, exists := r.inbox[keys[index]]; exists {
				return false, ErrInboxReceiptConflict
			}
		}
	}
	if _, exists := r.outbox[event.ID]; exists {
		return false, errors.New("outbox event already exists")
	}
	for _, key := range keys {
		r.inbox[key] = struct{}{}
	}
	r.outbox[event.ID] = event
	return true, nil
}
func (r *MemoryRepository) TimerQueueHealth(context.Context) (QueueHealth, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var health QueueHealth
	for _, timer := range r.timers {
		switch timer.State {
		case TimerReady, TimerClaimed:
			health.Pending++
			if timer.Attempts > health.HighestAttempts {
				health.HighestAttempts = timer.Attempts
			}
			if health.OldestPending == nil || timer.DueAt.Before(*health.OldestPending) {
				health.OldestPending = timePtr(timer.DueAt)
			}
		case TimerFailed:
			health.Terminal++
			if timer.Attempts > health.HighestAttempts {
				health.HighestAttempts = timer.Attempts
			}
		}
	}
	return health, nil
}
func (r *MemoryRepository) OutboxQueueHealth(context.Context) (QueueHealth, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var health QueueHealth
	for _, event := range r.outbox {
		if event.Attempts > health.HighestAttempts {
			health.HighestAttempts = event.Attempts
		}
		if event.DeadLetteredAt != nil {
			health.Terminal++
			continue
		}
		health.Pending++
		pendingAt := event.OccurredAt
		if health.OldestPending == nil || pendingAt.Before(*health.OldestPending) {
			health.OldestPending = timePtr(pendingAt)
		}
	}
	return health, nil
}

func sameLeaseGeneration(current, claimed *time.Time) bool {
	return current != nil && claimed != nil && current.Equal(*claimed)
}
