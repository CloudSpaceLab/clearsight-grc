package autonomy

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"sync"
	"time"
)

func (r *MemoryRepository) ListFormPolicyChoices(tenant, form string, version int64, limit int) []AutomationPolicy {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := []AutomationPolicy{}
	for _, value := range r.policies {
		if value.TenantID != tenant || value.ActionClass != "FORM_RESPONSE_CREATE_MATTER" {
			continue
		}
		newer := false
		for _, candidate := range r.policies {
			if candidate.TenantID == tenant && candidate.Code == value.Code && candidate.Version > value.Version {
				newer = true
				break
			}
		}
		if newer {
			continue
		}
		var scope struct {
			FormID  string `json:"form_template_id"`
			Version int64  `json:"form_template_version"`
		}
		if json.Unmarshal(value.Eligibility, &scope) != nil || scope.FormID != form || scope.Version != version {
			continue
		}
		values = append(values, value)
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].Code == values[j].Code {
			return values[i].Version > values[j].Version
		}
		return values[i].Code < values[j].Code
	})
	latest := []AutomationPolicy{}
	for _, value := range values {
		if len(latest) > 0 && latest[len(latest)-1].Code == value.Code {
			continue
		}
		latest = append(latest, value)
		if len(latest) == limit {
			break
		}
	}
	return latest
}

// CommitFormPolicy holds the canonical automation record lock while the caller
// commits its in-memory typed extension. PostgreSQL uses one database transaction.
func (r *MemoryRepository) CommitFormPolicy(value AutomationPolicy, expected int64, commit func()) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if value.ActionClass != "FORM_RESPONSE_CREATE_MATTER" || value.ID == "" || value.TenantID == "" || commit == nil {
		return errors.New("invalid form automation policy")
	}
	index := -1
	for i, existing := range r.policies {
		if existing.TenantID == value.TenantID && existing.ID == value.ID {
			index = i
			break
		}
	}
	if index < 0 && expected != 0 || index >= 0 && r.policies[index].RecordVersion != expected {
		return errors.New("form automation policy version conflict")
	}
	commit()
	if index < 0 {
		r.policies = append(r.policies, value)
	} else {
		r.policies[index] = value
	}
	return nil
}

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
