package sourceaccess

import (
	"context"
	"sync"
	"time"
)

type MemoryCheckpointRepository struct {
	mu          sync.Mutex
	catalog     CatalogRepository
	checkpoints map[string]BindingCheckpoint
}

func NewMemoryCheckpointRepository(catalog CatalogRepository) *MemoryCheckpointRepository {
	return &MemoryCheckpointRepository{catalog: catalog, checkpoints: map[string]BindingCheckpoint{}}
}

func (r *MemoryCheckpointRepository) EnsureBindingCheckpoint(ctx context.Context, tenantID, sourceID, bindingID string, bindingVersion int64, now time.Time) (BindingCheckpoint, error) {
	if r == nil || r.catalog == nil || bindingVersion < 1 {
		return BindingCheckpoint{}, ErrCatalogInvalid
	}
	binding, err := r.catalog.BindingRevision(ctx, tenantID, bindingID, bindingVersion)
	if err != nil {
		return BindingCheckpoint{}, err
	}
	if binding.SourceID != sourceID || !statefulBindingRevision(binding) {
		return BindingCheckpoint{}, ErrCatalogInvalid
	}
	key := checkpointKey(tenantID, bindingID, bindingVersion)
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.checkpoints[key]; ok {
		return existing, nil
	}
	value := BindingCheckpoint{
		TenantID: tenantID, SourceID: sourceID, BindingID: bindingID, BindingVersion: bindingVersion,
		Generation: 0, CreatedAt: now, UpdatedAt: now,
	}
	r.checkpoints[key] = value
	return value, nil
}

func (r *MemoryCheckpointRepository) BindingCheckpoint(_ context.Context, tenantID, bindingID string, bindingVersion int64) (BindingCheckpoint, error) {
	if r == nil || bindingVersion < 1 {
		return BindingCheckpoint{}, ErrCatalogInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.checkpoints[checkpointKey(tenantID, bindingID, bindingVersion)]
	if !ok {
		return BindingCheckpoint{}, ErrCatalogNotFound
	}
	return value, nil
}

func (r *MemoryCheckpointRepository) AdvanceBindingCheckpoint(_ context.Context, expected BindingCheckpoint, position CheckpointPosition, at time.Time) (BindingCheckpoint, error) {
	if r == nil {
		return BindingCheckpoint{}, ErrCatalogStorage
	}
	if err := validateCheckpointPosition(position); err != nil {
		return BindingCheckpoint{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := checkpointKey(expected.TenantID, expected.BindingID, expected.BindingVersion)
	value, ok := r.checkpoints[key]
	if !ok {
		return BindingCheckpoint{}, ErrCatalogNotFound
	}
	if value.SourceID != expected.SourceID || value.Generation != expected.Generation || !checkpointPositionEqual(value.Position, expected.Position) {
		return BindingCheckpoint{}, ErrCheckpointConflict
	}
	value.Position = position
	value.Generation++
	value.UpdatedAt = at
	r.checkpoints[key] = value
	return value, nil
}

func checkpointKey(tenantID, bindingID string, bindingVersion int64) string {
	return catalogRevisionKey(tenantID, bindingID, bindingVersion)
}
