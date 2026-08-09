package continuity

import (
	"context"
	"sync"
)

type memoryProgramReviewData struct {
	mu          sync.RWMutex
	checkpoints map[string]map[string]map[string][]ProgramReviewCheckpoint
}

var memoryProgramReviewStores sync.Map

func programReviewData(repo *MemoryRepository) *memoryProgramReviewData {
	value, _ := memoryProgramReviewStores.LoadOrStore(repo, &memoryProgramReviewData{checkpoints: map[string]map[string]map[string][]ProgramReviewCheckpoint{}})
	return value.(*memoryProgramReviewData)
}

func (r *MemoryRepository) LatestProgramReview(_ context.Context, tenant, programID, principalID string) (*ProgramReviewCheckpoint, error) {
	data := programReviewData(r)
	data.mu.RLock()
	defer data.mu.RUnlock()
	values := data.checkpoints[tenant][programID][principalID]
	if len(values) == 0 {
		return nil, nil
	}
	value := values[len(values)-1]
	return &value, nil
}

func (r *MemoryRepository) RecordProgramReview(_ context.Context, checkpoint ProgramReviewCheckpoint) (ProgramReviewCheckpoint, error) {
	r.mu.RLock()
	aggregate, exists := r.programs[checkpoint.TenantID][checkpoint.ProgramID]
	r.mu.RUnlock()
	if !exists {
		return ProgramReviewCheckpoint{}, ErrNotFound
	}
	data := projectionData(r)
	data.mu.Lock()
	states := data.states[checkpoint.TenantID][checkpoint.ProgramID]
	if len(states) == 0 {
		data.mu.Unlock()
		return ProgramReviewCheckpoint{}, ErrInvalidState
	}
	current := states[len(states)-1]
	data.mu.Unlock()
	if aggregate.Program.Version != checkpoint.ProgramVersion || current.ProgramVersion != checkpoint.ProgramVersion || current.ProjectionVersion != checkpoint.ProjectionVersion {
		return ProgramReviewCheckpoint{}, ErrVersionConflict
	}

	reviews := programReviewData(r)
	reviews.mu.Lock()
	defer reviews.mu.Unlock()
	if reviews.checkpoints[checkpoint.TenantID] == nil {
		reviews.checkpoints[checkpoint.TenantID] = map[string]map[string][]ProgramReviewCheckpoint{}
	}
	if reviews.checkpoints[checkpoint.TenantID][checkpoint.ProgramID] == nil {
		reviews.checkpoints[checkpoint.TenantID][checkpoint.ProgramID] = map[string][]ProgramReviewCheckpoint{}
	}
	values := reviews.checkpoints[checkpoint.TenantID][checkpoint.ProgramID][checkpoint.PrincipalID]
	for _, existing := range values {
		if existing.ProgramVersion == checkpoint.ProgramVersion && existing.ProjectionVersion == checkpoint.ProjectionVersion {
			return existing, nil
		}
	}
	reviews.checkpoints[checkpoint.TenantID][checkpoint.ProgramID][checkpoint.PrincipalID] = append(values, checkpoint)
	return checkpoint, nil
}

func (r *MemoryRepository) ProgramStateVersion(_ context.Context, tenant, programID string, projectionVersion int64) (*ProgramStateSnapshot, error) {
	data := projectionData(r)
	data.mu.Lock()
	defer data.mu.Unlock()
	for _, state := range data.states[tenant][programID] {
		if state.ProjectionVersion == projectionVersion {
			value := state
			return &value, nil
		}
	}
	return nil, nil
}
