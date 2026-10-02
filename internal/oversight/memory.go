package oversight

import (
	"context"
	"sync"
	"time"
)

type PeriodBuilder func(context.Context, Scope, time.Time, time.Time) (Snapshot, error)

type MemoryRepository struct {
	mu            sync.RWMutex
	snapshots     []Snapshot
	periodBuilder PeriodBuilder
}

func NewMemoryRepository(values []Snapshot) *MemoryRepository {
	return &MemoryRepository{snapshots: append([]Snapshot(nil), values...)}
}

func (r *MemoryRepository) WithPeriodBuilder(builder PeriodBuilder) *MemoryRepository {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.periodBuilder = builder
	return r
}

func (r *MemoryRepository) Latest(_ context.Context, scope Scope) (Snapshot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var latest Snapshot
	found := false
	for _, value := range r.snapshots {
		if value.TenantID != scope.TenantID || value.LegalEntityID != scope.LegalEntityID {
			continue
		}
		if !found || value.GeneratedAt.After(latest.GeneratedAt) {
			latest, found = value, true
		}
	}
	if !found {
		return Snapshot{}, ErrNotFound
	}
	return latest, nil
}

func (r *MemoryRepository) BuildPeriod(ctx context.Context, scope Scope, start, end time.Time) (Snapshot, error) {
	r.mu.RLock()
	builder := r.periodBuilder
	r.mu.RUnlock()
	if builder == nil {
		return Snapshot{}, ErrReportingPeriodUnavailable
	}
	return builder(ctx, scope, start.UTC(), end.UTC())
}

func (r *MemoryRepository) Put(value Snapshot) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.snapshots = append(r.snapshots, value)
}
