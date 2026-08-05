package autonomy

import (
	"context"
	"sort"
	"sync"
)

type MemoryRepository struct {
	mu      sync.RWMutex
	signals map[string]Signal
	drifts  map[string]Drift
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{signals: map[string]Signal{}, drifts: map[string]Drift{}}
}
func (r *MemoryRepository) InsertSignal(_ context.Context, value Signal) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := value.TenantID + "|" + value.DedupeKey
	if _, ok := r.signals[key]; ok {
		return false, nil
	}
	r.signals[key] = value
	return true, nil
}
func (r *MemoryRepository) UpsertDrift(_ context.Context, value Drift) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.drifts[value.TenantID+"|"+value.Dimension+"|"+value.SubjectType+"|"+value.SubjectID] = value
	return nil
}
func (r *MemoryRepository) ListDrifts(_ context.Context, tenant string) ([]Drift, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := []Drift{}
	for _, value := range r.drifts {
		if value.TenantID == tenant && value.State == "ACTIVE" {
			values = append(values, value)
		}
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].Severity == values[j].Severity {
			return values[i].DetectedAt.After(values[j].DetectedAt)
		}
		return values[i].Severity > values[j].Severity
	})
	return values, nil
}
