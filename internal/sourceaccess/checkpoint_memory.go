package sourceaccess

import (
	"context"
	"sort"
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
		return cloneBindingCheckpoint(existing), nil
	}
	value := BindingCheckpoint{
		TenantID: tenantID, SourceID: sourceID, BindingID: bindingID, BindingVersion: bindingVersion,
		NextAttemptAt: now, CreatedAt: now, UpdatedAt: now,
	}
	r.checkpoints[key] = value
	return cloneBindingCheckpoint(value), nil
}

func (r *MemoryCheckpointRepository) BindingCheckpoint(_ context.Context, tenantID, bindingID string, bindingVersion int64) (BindingCheckpoint, error) {
	if r == nil {
		return BindingCheckpoint{}, ErrCatalogStorage
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.checkpoints[checkpointKey(tenantID, bindingID, bindingVersion)]
	if !ok {
		return BindingCheckpoint{}, ErrCatalogNotFound
	}
	return cloneBindingCheckpoint(value), nil
}

func (r *MemoryCheckpointRepository) ClaimBindingCheckpoints(ctx context.Context, worker string, now time.Time, lease time.Duration, limit int) ([]BindingCheckpoint, error) {
	if r == nil || r.catalog == nil {
		return nil, ErrCatalogStorage
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	keys := make([]string, 0, len(r.checkpoints))
	for key := range r.checkpoints {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		left, right := r.checkpoints[keys[i]], r.checkpoints[keys[j]]
		if left.NextAttemptAt.Equal(right.NextAttemptAt) {
			return keys[i] < keys[j]
		}
		return left.NextAttemptAt.Before(right.NextAttemptAt)
	})
	result := make([]BindingCheckpoint, 0, min(limit, len(keys)))
	for _, key := range keys {
		if len(result) >= limit {
			break
		}
		value := r.checkpoints[key]
		if value.FailedAt != nil || value.NextAttemptAt.After(now) || (value.LeaseUntil != nil && !value.LeaseUntil.Before(now)) {
			continue
		}
		binding, err := r.catalog.BindingRevision(ctx, value.TenantID, value.BindingID, value.BindingVersion)
		if err != nil {
			return nil, err
		}
		if !statefulBindingRevision(binding) || binding.SourceID != value.SourceID {
			continue
		}
		leaseUntil := now.Add(lease)
		value.Attempts++
		value.LockedBy = worker
		value.LeaseUntil = &leaseUntil
		value.UpdatedAt = now
		r.checkpoints[key] = value
		result = append(result, cloneBindingCheckpoint(value))
	}
	return result, nil
}

func (r *MemoryCheckpointRepository) AdvanceBindingCheckpoint(_ context.Context, claimed BindingCheckpoint, position CheckpointPosition, at, next time.Time) (BindingCheckpoint, error) {
	if err := validateCheckpointPosition(position); err != nil {
		return BindingCheckpoint{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := checkpointKey(claimed.TenantID, claimed.BindingID, claimed.BindingVersion)
	value, ok := r.checkpoints[key]
	if !ok {
		return BindingCheckpoint{}, ErrCatalogNotFound
	}
	if value.LockedBy == "" || value.LockedBy != claimed.LockedBy || claimed.LockedBy == "" || value.LeaseUntil == nil || claimed.LeaseUntil == nil || !value.LeaseUntil.Equal(*claimed.LeaseUntil) || value.LeaseUntil.Before(at) {
		return BindingCheckpoint{}, ErrCheckpointClaimLost
	}
	value.Position = position
	value.Attempts = 0
	value.LockedBy = ""
	value.LeaseUntil = nil
	value.LastErrorCode = ""
	value.FailedAt = nil
	value.NextAttemptAt = next
	value.UpdatedAt = at
	r.checkpoints[key] = value
	return cloneBindingCheckpoint(value), nil
}

func (r *MemoryCheckpointRepository) FailBindingCheckpoint(_ context.Context, claimed BindingCheckpoint, maxAttempts int, errorCode string, at, next time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := checkpointKey(claimed.TenantID, claimed.BindingID, claimed.BindingVersion)
	value, ok := r.checkpoints[key]
	if !ok {
		return false, ErrCatalogNotFound
	}
	if value.LockedBy == "" || value.LockedBy != claimed.LockedBy || claimed.LockedBy == "" || value.LeaseUntil == nil || claimed.LeaseUntil == nil || !value.LeaseUntil.Equal(*claimed.LeaseUntil) || value.LeaseUntil.Before(at) {
		return false, ErrCheckpointClaimLost
	}
	terminal := value.Attempts >= maxAttempts
	value.LockedBy = ""
	value.LeaseUntil = nil
	value.LastErrorCode = errorCode
	value.UpdatedAt = at
	if terminal {
		failedAt := at
		value.FailedAt = &failedAt
	} else {
		value.NextAttemptAt = next
	}
	r.checkpoints[key] = value
	return terminal, nil
}

func checkpointKey(tenantID, bindingID string, bindingVersion int64) string {
	return catalogRevisionKey(tenantID, bindingID, bindingVersion)
}

func cloneBindingCheckpoint(value BindingCheckpoint) BindingCheckpoint {
	if value.LeaseUntil != nil {
		copy := *value.LeaseUntil
		value.LeaseUntil = &copy
	}
	if value.FailedAt != nil {
		copy := *value.FailedAt
		value.FailedAt = &copy
	}
	return value
}
