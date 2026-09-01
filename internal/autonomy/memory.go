package autonomy

import (
	"context"
	"sort"
	"sync"
	"time"
)

type MemoryRepository struct {
	mu       sync.RWMutex
	signals  map[string]Signal
	drifts   map[string]Drift
	policies []AutomationPolicy
}

func NewMemoryRepository(policies ...AutomationPolicy) *MemoryRepository {
	return &MemoryRepository{signals: map[string]Signal{}, drifts: map[string]Drift{}, policies: append([]AutomationPolicy(nil), policies...)}
}

func (r *MemoryRepository) Ingest(_ context.Context, signal Signal, drift Drift) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := signal.TenantID + "|" + signal.DedupeKey
	if _, ok := r.signals[key]; ok {
		return false, nil
	}
	r.signals[key] = signal
	r.drifts[drift.TenantID+"|"+drift.Dimension+"|"+drift.SubjectType+"|"+drift.SubjectID] = drift
	return true, nil
}

func (r *MemoryRepository) Resolve(_ context.Context, signal Signal, dimension string, _ time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	signalKey := signal.TenantID + "|" + signal.DedupeKey
	if _, ok := r.signals[signalKey]; ok {
		return false, nil
	}
	r.signals[signalKey] = signal
	driftKey := signal.TenantID + "|" + dimension + "|" + signal.SubjectType + "|" + signal.SubjectID
	if drift, ok := r.drifts[driftKey]; ok && drift.State == "ACTIVE" {
		drift.State = "RESOLVED"
		r.drifts[driftKey] = drift
	}
	return true, nil
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

func (r *MemoryRepository) ListAutomationPolicies(_ context.Context, tenant string) ([]AutomationPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	latest := map[string]AutomationPolicy{}
	for _, value := range r.policies {
		if value.TenantID != tenant {
			continue
		}
		current, ok := latest[value.Code]
		if !ok || value.Version > current.Version {
			latest[value.Code] = value
		}
	}
	values := make([]AutomationPolicy, 0, len(latest))
	for _, value := range latest {
		values = append(values, value)
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].Name == values[j].Name {
			return values[i].Code < values[j].Code
		}
		return values[i].Name < values[j].Name
	})
	return values, nil
}

func (r *MemoryRepository) GetAutomationPolicy(_ context.Context, tenant, id string, version int64) (AutomationPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, value := range r.policies {
		if value.TenantID == tenant && value.ID == id && value.Version == version {
			return value, nil
		}
	}
	return AutomationPolicy{}, ErrAutomationPolicyNotFound
}
