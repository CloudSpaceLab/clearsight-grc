package documentcoverage

import (
	"context"
	"encoding/json"
	"sync"
)

type MemoryRepository struct {
	mu     sync.RWMutex
	items  map[string]Assessment
	queued map[string]bool
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{items: map[string]Assessment{}, queued: map[string]bool{}}
}

func (r *MemoryRepository) BeginVersion(_ context.Context, value Assessment) (Assessment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := assessmentKey(value.TenantID, value.DocumentID)
	if current, ok := r.items[key]; ok && sameAssessmentTuple(current, value) {
		return cloneAssessment(current), nil
	}
	if value.Version < 1 {
		value.Version = 1
	}
	r.items[key] = cloneAssessment(value)
	return cloneAssessment(value), nil
}

func (r *MemoryRepository) CompleteVersion(_ context.Context, value Assessment, expected int64) (Assessment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := assessmentKey(value.TenantID, value.DocumentID)
	current, ok := r.items[key]
	if !ok {
		return Assessment{}, ErrNotFound
	}
	if current.Version != expected || current.ID != value.ID {
		return Assessment{}, ErrVersionConflict
	}
	value.Version = current.Version
	r.items[key] = cloneAssessment(value)
	return cloneAssessment(value), nil
}

func (r *MemoryRepository) Current(_ context.Context, tenant, documentID string) (Assessment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.items[assessmentKey(tenant, documentID)]
	if !ok {
		return Assessment{}, ErrNotFound
	}
	return cloneAssessment(value), nil
}

func (r *MemoryRepository) Review(_ context.Context, value Assessment, expected int64) (Assessment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := assessmentKey(value.TenantID, value.DocumentID)
	current, ok := r.items[key]
	if !ok {
		return Assessment{}, ErrNotFound
	}
	if current.Version != expected || current.ID != value.ID {
		return Assessment{}, ErrVersionConflict
	}
	value.Version = expected + 1
	r.items[key] = cloneAssessment(value)
	return cloneAssessment(value), nil
}

func (r *MemoryRepository) MarkFailed(ctx context.Context, value Assessment, expected int64) (Assessment, error) {
	return r.CompleteVersion(ctx, value, expected)
}

func (r *MemoryRepository) QueueRecompare(_ context.Context, tenant, documentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.queued[assessmentKey(tenant, documentID)] = true
	return nil
}

func assessmentKey(tenant, documentID string) string { return tenant + "\x00" + documentID }

func sameAssessmentTuple(left, right Assessment) bool {
	return left.DocumentSHA256 == right.DocumentSHA256 &&
		left.AnalyzerVersion == right.AnalyzerVersion &&
		left.MatcherVersion == right.MatcherVersion &&
		left.ProgramSnapshotHash == right.ProgramSnapshotHash
}

func cloneAssessment(value Assessment) Assessment {
	raw, _ := json.Marshal(value)
	var clone Assessment
	_ = json.Unmarshal(raw, &clone)
	return clone
}

var _ Repository = (*MemoryRepository)(nil)
