package onboarding

import (
	"context"
	"sync"
	"time"
)

type MemoryRepository struct {
	mu     sync.RWMutex
	states map[string]State
	now    func() time.Time
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{states: map[string]State{}, now: time.Now}
}
func key(tenant, principal, guide string) string { return tenant + "|" + principal + "|" + guide }
func (r *MemoryRepository) Get(_ context.Context, tenant, principal, guide string) (State, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.states[key(tenant, principal, guide)]
	if !ok {
		return State{}, ErrStateNotFound
	}
	return value, nil
}
func (r *MemoryRepository) Upsert(_ context.Context, value State, expected int64) (State, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := key(value.TenantID, value.PrincipalID, value.GuideCode)
	current, ok := r.states[k]
	if ok && expected != current.Version {
		return State{}, ErrVersionConflict
	}
	if !ok && expected != 0 {
		return State{}, ErrVersionConflict
	}
	value.Version = current.Version + 1
	if !ok {
		value.Version = 1
	}
	value.UpdatedAt = r.now().UTC()
	r.states[k] = value
	return value, nil
}
