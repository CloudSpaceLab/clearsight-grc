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
	if v.State != TimerClaimed || v.LockedBy != t.LockedBy {
		return errors.New("timer claim lost")
	}
	v.State = TimerFired
	v.LeaseUntil = nil
	v.LockedBy = ""
	r.timers[t.ID] = v
	r.outbox[e.ID] = e
	return nil
}
func (r *MemoryRepository) FailTimer(_ context.Context, t Timer, message string, next time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	v := r.timers[t.ID]
	v.State = TimerReady
	v.DueAt = next
	v.LeaseUntil = nil
	v.LockedBy = ""
	r.timers[t.ID] = v
	return nil
}
func (r *MemoryRepository) ClaimOutbox(_ context.Context, worker string, now time.Time, lease time.Duration, limit int) ([]OutboxEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []OutboxEvent{}
	for id, value := range r.outbox {
		if len(out) >= limit {
			break
		}
		if value.NextAttemptAt != nil && value.NextAttemptAt.After(now) {
			continue
		}
		value.Attempts++
		value.LockedBy = worker
		r.outbox[id] = value
		out = append(out, value)
	}
	return out, nil
}
func (r *MemoryRepository) MarkPublished(_ context.Context, e OutboxEvent, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.outbox[e.ID]
	if !ok || value.LockedBy != e.LockedBy {
		return errors.New("outbox claim lost")
	}
	delete(r.outbox, e.ID)
	return nil
}
func (r *MemoryRepository) MarkFailed(_ context.Context, e OutboxEvent, message string, next time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.outbox[e.ID]
	if !ok || value.LockedBy != e.LockedBy {
		return errors.New("outbox claim lost")
	}
	value.LockedBy = ""
	value.NextAttemptAt = &next
	r.outbox[e.ID] = value
	return nil
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
