package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
)

type MemoryRepository struct {
	mu      sync.RWMutex
	forms   map[string]FormTemplate
	checks  map[string]MonitoringCheck
	results map[string]MonitoringResult
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{forms: map[string]FormTemplate{}, checks: map[string]MonitoringCheck{}, results: map[string]MonitoringResult{}}
}

func (r *MemoryRepository) CreateFormRevision(_ context.Context, value FormTemplate) (FormTemplate, error) {
	if value.TenantID == "" || value.ID == "" || value.Version < 1 {
		return FormTemplate{}, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := revisionKey(value.TenantID, value.ID, value.Version)
	if _, exists := r.forms[key]; exists {
		return FormTemplate{}, ErrConflict
	}
	stored := cloneValue(value)
	r.forms[key] = stored
	return cloneValue(stored), nil
}

func (r *MemoryRepository) FormRevision(_ context.Context, tenant, id string, version int64) (FormTemplate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.forms[revisionKey(tenant, id, version)]
	if !ok {
		return FormTemplate{}, ErrNotFound
	}
	return cloneValue(value), nil
}

func (r *MemoryRepository) ListFormRevisions(_ context.Context, tenant string, limit int) ([]FormTemplate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := make([]FormTemplate, 0)
	for _, value := range r.forms {
		if value.TenantID == tenant {
			values = append(values, cloneValue(value))
		}
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].Code == values[j].Code {
			return values[i].Version > values[j].Version
		}
		return values[i].Code < values[j].Code
	})
	return boundedForms(values, limit), nil
}

func (r *MemoryRepository) TransitionForm(_ context.Context, input LifecycleTransition) (FormTemplate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := revisionKey(input.TenantID, input.ID, input.ExpectedVersion)
	current, ok := r.forms[key]
	if !ok {
		return FormTemplate{}, ErrNotFound
	}
	nextLifecycle, err := transitionLifecycle(current.Lifecycle, input)
	if err != nil {
		return FormTemplate{}, err
	}
	next := cloneValue(current)
	next.Lifecycle = nextLifecycle
	nextKey := revisionKey(input.TenantID, input.ID, next.Version)
	if _, exists := r.forms[nextKey]; exists {
		return FormTemplate{}, ErrConflict
	}
	if next.IsCurrent || current.IsCurrent {
		for storedKey, stored := range r.forms {
			if stored.TenantID == input.TenantID && stored.ID == input.ID && stored.IsCurrent {
				stored.IsCurrent = false
				stored.Status = LifecycleRetired
				until := input.At.UTC()
				stored.EffectiveUntil = &until
				stored.UpdatedAt = until
				r.forms[storedKey] = stored
			}
		}
	}
	r.forms[nextKey] = cloneValue(next)
	return cloneValue(next), nil
}

func (r *MemoryRepository) CreateCheckRevision(_ context.Context, value MonitoringCheck) (MonitoringCheck, error) {
	if value.TenantID == "" || value.ID == "" || value.ProgramID == "" || value.Version < 1 {
		return MonitoringCheck{}, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := revisionKey(value.TenantID, value.ID, value.Version)
	if _, exists := r.checks[key]; exists {
		return MonitoringCheck{}, ErrConflict
	}
	stored := cloneValue(value)
	r.checks[key] = stored
	return cloneValue(stored), nil
}

func (r *MemoryRepository) ReviseCheck(_ context.Context, input CheckRevisionUpdate) (MonitoringCheck, error) {
	if input.TenantID == "" || input.ID == "" || input.ExpectedVersion < 1 || input.ActorID == "" || input.At.IsZero() {
		return MonitoringCheck{}, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.checks[revisionKey(input.TenantID, input.ID, input.ExpectedVersion)]
	if !ok {
		return MonitoringCheck{}, ErrNotFound
	}
	if current.InputKind != InputForm || current.Status != LifecycleActive || !current.IsCurrent {
		return MonitoringCheck{}, ErrConflict
	}
	next := cloneValue(current)
	next.CollectionPolicy = &input.Policy
	now := input.At.UTC()
	next.Lifecycle = Lifecycle{
		Status: LifecycleDraft, Version: current.Version + 1, CreatedBy: input.ActorID,
		CreatedAt: now, UpdatedAt: now,
	}
	key := revisionKey(input.TenantID, input.ID, next.Version)
	if _, exists := r.checks[key]; exists {
		return MonitoringCheck{}, ErrConflict
	}
	r.checks[key] = cloneValue(next)
	return cloneValue(next), nil
}

func (r *MemoryRepository) CheckRevision(_ context.Context, tenant, id string, version int64) (MonitoringCheck, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.checks[revisionKey(tenant, id, version)]
	if !ok {
		return MonitoringCheck{}, ErrNotFound
	}
	return cloneValue(value), nil
}

func (r *MemoryRepository) ListCheckRevisions(_ context.Context, tenant, programID string, limit int) ([]MonitoringCheck, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := make([]MonitoringCheck, 0)
	for _, value := range r.checks {
		if value.TenantID == tenant && value.ProgramID == programID {
			values = append(values, cloneValue(value))
		}
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].Code == values[j].Code {
			return values[i].Version > values[j].Version
		}
		return values[i].Code < values[j].Code
	})
	return boundedChecks(values, limit), nil
}

func (r *MemoryRepository) TransitionCheck(_ context.Context, input LifecycleTransition) (MonitoringCheck, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := revisionKey(input.TenantID, input.ID, input.ExpectedVersion)
	current, ok := r.checks[key]
	if !ok {
		return MonitoringCheck{}, ErrNotFound
	}
	nextLifecycle, err := transitionLifecycle(current.Lifecycle, input)
	if err != nil {
		return MonitoringCheck{}, err
	}
	next := cloneValue(current)
	next.Lifecycle = nextLifecycle
	nextKey := revisionKey(input.TenantID, input.ID, next.Version)
	if _, exists := r.checks[nextKey]; exists {
		return MonitoringCheck{}, ErrConflict
	}
	if next.IsCurrent || current.IsCurrent {
		for storedKey, stored := range r.checks {
			if stored.TenantID == input.TenantID && stored.ID == input.ID && stored.IsCurrent {
				stored.IsCurrent = false
				stored.Status = LifecycleRetired
				until := input.At.UTC()
				stored.EffectiveUntil = &until
				stored.UpdatedAt = until
				r.checks[storedKey] = stored
			}
		}
	}
	r.checks[nextKey] = cloneValue(next)
	return cloneValue(next), nil
}

func transitionLifecycle(current Lifecycle, input LifecycleTransition) (Lifecycle, error) {
	if input.TenantID == "" || input.ID == "" || input.ActorID == "" || input.ExpectedVersion < 1 || input.At.IsZero() {
		return Lifecycle{}, ErrInvalid
	}
	allowed := false
	switch current.Status {
	case LifecycleDraft:
		allowed = input.To == LifecyclePendingApproval
	case LifecyclePendingApproval:
		allowed = input.To == LifecycleActive || input.To == LifecycleRejected
	case LifecycleActive:
		allowed = input.To == LifecyclePaused || input.To == LifecycleRetired
	case LifecyclePaused:
		allowed = input.To == LifecycleActive || input.To == LifecycleRetired
	}
	if !allowed {
		return Lifecycle{}, ErrInvalid
	}
	now := input.At.UTC()
	next := current
	next.Status = input.To
	next.Version++
	next.UpdatedAt = now
	next.IsCurrent = input.To == LifecycleActive || input.To == LifecyclePaused
	switch input.To {
	case LifecyclePendingApproval:
		next.SubmittedBy = input.ActorID
		next.EffectiveFrom = nil
		next.EffectiveUntil = nil
	case LifecycleActive:
		next.ApprovedBy = input.ActorID
		if next.EffectiveFrom == nil {
			next.EffectiveFrom = &now
		}
		next.EffectiveUntil = nil
	case LifecycleRejected:
		next.RejectedBy = input.ActorID
		next.EffectiveFrom = nil
		next.EffectiveUntil = nil
	case LifecyclePaused:
		next.EffectiveUntil = nil
	case LifecycleRetired:
		next.IsCurrent = false
		if next.EffectiveFrom == nil {
			next.EffectiveFrom = &now
		}
		next.EffectiveUntil = &now
	}
	return next, nil
}

func (r *MemoryRepository) AppendResult(_ context.Context, value MonitoringResult) (MonitoringResult, error) {
	if value.TenantID == "" || value.ID == "" || value.MonitoringCheckID == "" || value.MonitoringCheckVersion < 1 || value.InputReferenceID == "" || value.EvaluatorVersion == "" {
		return MonitoringResult{}, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.results {
		if existing.TenantID == value.TenantID && existing.MonitoringCheckID == value.MonitoringCheckID && existing.InputReferenceID == value.InputReferenceID && existing.EvaluatorVersion == value.EvaluatorVersion {
			return MonitoringResult{}, ErrConflict
		}
	}
	if _, exists := r.results[value.ID]; exists {
		return MonitoringResult{}, ErrConflict
	}
	stored := cloneValue(value)
	r.results[value.ID] = stored
	return cloneValue(stored), nil
}

func (r *MemoryRepository) ListResults(_ context.Context, tenant, checkID string, limit int) ([]MonitoringResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := make([]MonitoringResult, 0)
	for _, value := range r.results {
		if value.TenantID == tenant && value.MonitoringCheckID == checkID {
			values = append(values, cloneValue(value))
		}
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].EvaluatedAt.Equal(values[j].EvaluatedAt) {
			return values[i].ID < values[j].ID
		}
		return values[i].EvaluatedAt.After(values[j].EvaluatedAt)
	})
	if limit < 1 || limit > 500 {
		limit = 100
	}
	if len(values) > limit {
		values = values[:limit]
	}
	return values, nil
}

func revisionKey(tenant, id string, version int64) string {
	return fmt.Sprintf("%s\x00%s\x00%d", tenant, id, version)
}

func cloneValue[T any](value T) T {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	var cloned T
	if err := json.Unmarshal(encoded, &cloned); err != nil {
		panic(err)
	}
	return cloned
}

func boundedForms(values []FormTemplate, limit int) []FormTemplate {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	if len(values) > limit {
		return values[:limit]
	}
	return values
}

func boundedChecks(values []MonitoringCheck, limit int) []MonitoringCheck {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	if len(values) > limit {
		return values[:limit]
	}
	return values
}

var _ Repository = (*MemoryRepository)(nil)
