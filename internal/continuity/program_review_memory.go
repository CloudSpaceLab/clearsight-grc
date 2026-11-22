package continuity

import (
	"context"
	"sync"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type memoryProgramReviewData struct {
	mu          sync.RWMutex
	checkpoints map[string]map[string]map[string][]ProgramReviewCheckpoint
	events      map[string]Event
}

var memoryProgramReviewStores sync.Map

func programReviewData(repo *MemoryRepository) *memoryProgramReviewData {
	value, _ := memoryProgramReviewStores.LoadOrStore(repo, &memoryProgramReviewData{checkpoints: map[string]map[string]map[string][]ProgramReviewCheckpoint{}, events: map[string]Event{}})
	return value.(*memoryProgramReviewData)
}

func (r *MemoryRepository) LatestProgramReview(ctx context.Context, tenant, programID, principalID string) (*ProgramReviewCheckpoint, error) {
	if !hasTrustedSystemScope(ctx) {
		actor, ok := identity.FromContext(ctx)
		if !ok || actor.TenantID != tenant || actor.PrincipalID != principalID {
			return nil, ErrNotFound
		}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	program, exists := r.programs[tenant][programID]
	if !exists || !r.visibleLegalEntity(ctx, program.Program.TenantID, program.Program.LegalEntityID) || !visibleProgramScope(ctx, program.Program) {
		return nil, ErrNotFound
	}
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

func (r *MemoryRepository) RecordProgramReview(_ context.Context, checkpoint ProgramReviewCheckpoint, event Event) (ProgramReviewCheckpoint, error) {
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
		if existing.ProgramVersion == checkpoint.ProgramVersion && existing.ProjectionVersion == checkpoint.ProjectionVersion && existing.visibleReceiptPresent && checkpoint.visibleReceiptPresent && existing.visibleProjectionVersion == checkpoint.visibleProjectionVersion {
			return existing, nil
		}
	}
	reviews.checkpoints[checkpoint.TenantID][checkpoint.ProgramID][checkpoint.PrincipalID] = append(values, checkpoint)
	reviews.events[checkpoint.ID] = event
	return checkpoint, nil
}

func (r *MemoryRepository) ProgramStateVersion(ctx context.Context, tenant, programID string, projectionVersion int64) (*ProgramStateSnapshot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	program, exists := r.programs[tenant][programID]
	if !exists || !r.visibleLegalEntity(ctx, program.Program.TenantID, program.Program.LegalEntityID) || !visibleProgramScope(ctx, program.Program) {
		return nil, ErrNotFound
	}
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
