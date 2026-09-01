package oversight

import (
	"context"
	"sync"
)

type MemoryRepository struct {
	mu        sync.RWMutex
	snapshots []Snapshot
}

func NewMemoryRepository(values []Snapshot) *MemoryRepository {
	return &MemoryRepository{snapshots: append([]Snapshot(nil), values...)}
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

func (r *MemoryRepository) Put(value Snapshot) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.snapshots = append(r.snapshots, value)
}
