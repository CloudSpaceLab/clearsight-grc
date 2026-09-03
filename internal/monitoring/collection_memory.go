package monitoring

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

func (r *MemoryRepository) UpsertCollectionCycle(_ context.Context, value CollectionCycle) (CollectionCycle, error) {
	validated, err := validateCollectionCycle(value)
	if err != nil {
		return CollectionCycle{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, existing := range r.collectionCycles {
		if existing.TenantID == validated.TenantID && existing.MonitoringCheckID == validated.MonitoringCheckID && existing.Sequence == validated.Sequence {
			if existing.State == CycleClaimed {
				return CollectionCycle{}, ErrConflict
			}
			validated.ID = existing.ID
			validated.CreatedAt = existing.CreatedAt
			r.collectionCycles[key] = cloneValue(validated)
			return cloneValue(validated), nil
		}
	}
	key := collectionCycleKey(validated.TenantID, validated.ID)
	if _, exists := r.collectionCycles[key]; exists {
		return CollectionCycle{}, ErrConflict
	}
	r.collectionCycles[key] = cloneValue(validated)
	return cloneValue(validated), nil
}

func (r *MemoryRepository) CollectionCycle(_ context.Context, tenant, cycleID string) (CollectionCycle, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.collectionCycles[collectionCycleKey(tenant, cycleID)]
	if !ok {
		return CollectionCycle{}, ErrNotFound
	}
	return cloneValue(value), nil
}

func (r *MemoryRepository) ClaimDueCollectionCycles(_ context.Context, worker string, now time.Time, lease time.Duration, limit int) ([]CollectionCycle, error) {
	if strings.TrimSpace(worker) == "" || now.IsZero() || lease <= 0 {
		return nil, ErrInvalid
	}
	limit = boundedCollectionLimit(limit)
	r.mu.Lock()
	defer r.mu.Unlock()
	due := make([]CollectionCycle, 0)
	for _, value := range r.collectionCycles {
		ready := (value.State == CycleScheduled || value.State == CycleAwaitingResponse) && value.NextActionAt != nil && !value.NextActionAt.After(now)
		reclaimable := value.State == CycleClaimed && value.NextActionAt != nil && !value.NextActionAt.After(now) && value.LeaseUntil != nil && !value.LeaseUntil.After(now)
		if ready || reclaimable {
			due = append(due, cloneValue(value))
		}
	}
	sort.Slice(due, func(i, j int) bool {
		if due[i].NextActionAt.Equal(*due[j].NextActionAt) {
			return due[i].ID < due[j].ID
		}
		return due[i].NextActionAt.Before(*due[j].NextActionAt)
	})
	if len(due) > limit {
		due = due[:limit]
	}
	for index := range due {
		token, err := id.NewUUIDv7()
		if err != nil {
			return nil, err
		}
		leaseUntil := now.UTC().Add(lease)
		due[index].State = CycleClaimed
		due[index].LeaseOwner = strings.TrimSpace(worker)
		due[index].LeaseToken = token
		due[index].LeaseUntil = &leaseUntil
		due[index].UpdatedAt = now.UTC()
		r.collectionCycles[collectionCycleKey(due[index].TenantID, due[index].ID)] = cloneValue(due[index])
	}
	return due, nil
}

func (r *MemoryRepository) CompleteCollectionAction(_ context.Context, claim CollectionCycle, completion CollectionActionCompletion) (CollectionCycle, error) {
	if completion.At.IsZero() {
		return CollectionCycle{}, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current, err := r.currentCollectionClaim(claim, completion.At)
	if err != nil {
		return CollectionCycle{}, err
	}
	switch completion.State {
	case CycleScheduled, CycleAwaitingResponse, CycleComplete, CycleCancelled, CycleBlocked:
	default:
		return CollectionCycle{}, ErrInvalid
	}
	current.State = completion.State
	current.NextActionAt = cloneTime(completion.NextActionAt)
	if completion.CurrentRequestID != "" {
		current.CurrentRequestID = strings.TrimSpace(completion.CurrentRequestID)
	}
	if completion.DeliveryState != "" {
		current.DeliveryState = completion.DeliveryState
	}
	if completion.DeliveryReference != "" {
		current.DeliveryReference = strings.TrimSpace(completion.DeliveryReference)
	}
	if completion.RemindersSent != nil {
		current.RemindersSent = *completion.RemindersSent
	}
	current.LeaseOwner = ""
	current.LeaseToken = ""
	current.LeaseUntil = nil
	current.UpdatedAt = completion.At.UTC()
	validated, err := validateCollectionCycle(current)
	if err != nil {
		return CollectionCycle{}, err
	}
	r.collectionCycles[collectionCycleKey(current.TenantID, current.ID)] = cloneValue(validated)
	return cloneValue(validated), nil
}

func (r *MemoryRepository) FailCollectionAction(_ context.Context, claim CollectionCycle, safeError string, retryAt *time.Time, maxAttempts int, at time.Time) (CollectionCycle, error) {
	if maxAttempts < 1 || maxAttempts > 20 || strings.TrimSpace(safeError) == "" || len(strings.TrimSpace(safeError)) > 1000 || at.IsZero() {
		return CollectionCycle{}, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current, err := r.currentCollectionClaim(claim, at)
	if err != nil {
		return CollectionCycle{}, err
	}
	current.Attempts++
	current.SafeError = strings.TrimSpace(safeError)
	current.LeaseOwner = ""
	current.LeaseToken = ""
	current.LeaseUntil = nil
	current.UpdatedAt = at.UTC()
	if current.Attempts >= maxAttempts || retryAt == nil {
		current.State = CycleFailed
		current.NextActionAt = nil
	} else {
		retry := retryAt.UTC()
		current.State = CycleScheduled
		current.NextActionAt = &retry
	}
	r.collectionCycles[collectionCycleKey(current.TenantID, current.ID)] = cloneValue(current)
	return cloneValue(current), nil
}

func (r *MemoryRepository) CancelCollectionCyclesByCheck(_ context.Context, tenant, checkID string, at time.Time) (int, error) {
	if strings.TrimSpace(tenant) == "" || strings.TrimSpace(checkID) == "" || at.IsZero() {
		return 0, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for key, current := range r.collectionCycles {
		if current.TenantID != tenant || current.MonitoringCheckID != checkID || current.State == CycleComplete || current.State == CycleCancelled || current.State == CycleFailed {
			continue
		}
		current.State = CycleCancelled
		current.NextActionAt = nil
		current.LeaseOwner = ""
		current.LeaseToken = ""
		current.LeaseUntil = nil
		current.UpdatedAt = at.UTC()
		r.collectionCycles[key] = cloneValue(current)
		count++
	}
	return count, nil
}

func (r *MemoryRepository) ListCollectionSummaries(_ context.Context, tenant, programID string, limit int) ([]CollectionSummary, error) {
	if strings.TrimSpace(tenant) == "" || strings.TrimSpace(programID) == "" {
		return nil, ErrInvalid
	}
	limit = boundedCollectionLimit(limit)
	r.mu.RLock()
	defer r.mu.RUnlock()
	latest := make(map[string]CollectionCycle)
	generatedAt := time.Time{}
	for _, value := range r.collectionCycles {
		if value.TenantID != tenant || value.ProgramID != programID {
			continue
		}
		if current, ok := latest[value.MonitoringCheckID]; !ok || value.Sequence > current.Sequence {
			latest[value.MonitoringCheckID] = cloneValue(value)
		}
		if value.UpdatedAt.After(generatedAt) {
			generatedAt = value.UpdatedAt
		}
	}
	values := make([]CollectionCycle, 0, len(latest))
	for _, value := range latest {
		values = append(values, value)
	}
	sort.Slice(values, func(i, j int) bool { return values[i].MonitoringCheckID < values[j].MonitoringCheckID })
	if len(values) > limit {
		values = values[:limit]
	}
	summaries := make([]CollectionSummary, len(values))
	for index, value := range values {
		summaries[index] = collectionSummary(value, generatedAt)
	}
	return summaries, nil
}

func (r *MemoryRepository) currentCollectionClaim(claim CollectionCycle, at time.Time) (CollectionCycle, error) {
	current, ok := r.collectionCycles[collectionCycleKey(claim.TenantID, claim.ID)]
	if !ok {
		return CollectionCycle{}, ErrNotFound
	}
	if current.State != CycleClaimed || claim.LeaseToken == "" || current.LeaseToken != claim.LeaseToken || current.LeaseOwner != claim.LeaseOwner || current.LeaseUntil == nil || current.LeaseUntil.Before(at) {
		return CollectionCycle{}, ErrConflict
	}
	return current, nil
}

func collectionCycleKey(tenant, id string) string { return tenant + "\x00" + id }

func boundedCollectionLimit(limit int) int {
	if limit < 1 || limit > 100 {
		return 50
	}
	return limit
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := value.UTC()
	return &copy
}

var _ CollectionCycleRepository = (*MemoryRepository)(nil)
