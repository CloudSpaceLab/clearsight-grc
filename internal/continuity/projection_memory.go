package continuity

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

type memoryProjectionData struct {
	mu     sync.Mutex
	states map[string]map[string][]ProgramStateSnapshot
	jobs   map[string]ProjectionJob
}

var memoryProjectionStores sync.Map

func projectionData(repo *MemoryRepository) *memoryProjectionData {
	value, _ := memoryProjectionStores.LoadOrStore(repo, &memoryProjectionData{states: map[string]map[string][]ProgramStateSnapshot{}, jobs: map[string]ProjectionJob{}})
	return value.(*memoryProjectionData)
}

func (r *MemoryRepository) SaveProgramState(_ context.Context, tenant, programID string, expectedProgramVersion int64, state ProgramStateSnapshot) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	aggregate, ok := r.programs[tenant][programID]
	if !ok {
		return 0, ErrNotFound
	}
	if aggregate.Program.Version != expectedProgramVersion {
		return 0, ErrVersionConflict
	}
	data := projectionData(r)
	data.mu.Lock()
	defer data.mu.Unlock()
	if data.states[tenant] == nil {
		data.states[tenant] = map[string][]ProgramStateSnapshot{}
	}
	version := int64(len(data.states[tenant][programID]) + 1)
	state.ProgramVersion = expectedProgramVersion
	state.ProjectionVersion = version
	data.states[tenant][programID] = append(data.states[tenant][programID], state)
	aggregate.CurrentState = &state
	r.programs[tenant][programID] = aggregate
	return version, nil
}

func (r *MemoryRepository) ProgramStateAt(_ context.Context, tenant, programID string, at *time.Time) (*ProgramStateSnapshot, error) {
	data := projectionData(r)
	data.mu.Lock()
	defer data.mu.Unlock()
	states := data.states[tenant][programID]
	for index := len(states) - 1; index >= 0; index-- {
		if at == nil || !states[index].GeneratedAt.After(*at) {
			value := states[index]
			return &value, nil
		}
	}
	return nil, nil
}

func (r *MemoryRepository) QueueProgramState(_ context.Context, tenant, programID string, sourceVersion int64, reason, triggerID, requestedBy string) (ProjectionJob, error) {
	data := projectionData(r)
	data.mu.Lock()
	defer data.mu.Unlock()
	for _, existing := range data.jobs {
		if existing.TenantID == tenant && existing.AggregateID == programID && (existing.Status == ProjectionJobReady || existing.Status == ProjectionJobClaimed) {
			if sourceVersion > existing.SourceAggregateVersion {
				existing.SourceAggregateVersion = sourceVersion
				existing.Reason = reason
				existing.TriggerID = triggerID
				existing.UpdatedAt = time.Now().UTC()
				data.jobs[existing.ID] = existing
			}
			return existing, nil
		}
	}
	jobID, err := id.NewUUIDv7()
	if err != nil {
		return ProjectionJob{}, err
	}
	now := time.Now().UTC()
	job := ProjectionJob{ID: jobID, TenantID: tenant, Projection: ProjectionProgramState, AggregateType: "PROGRAM", AggregateID: programID, SourceAggregateVersion: sourceVersion, Reason: reason, TriggerID: triggerID, RequestedBy: requestedBy, Status: ProjectionJobReady, AvailableAt: now, CreatedAt: now, UpdatedAt: now}
	data.jobs[job.ID] = job
	return job, nil
}

func (r *MemoryRepository) ClaimProgramState(_ context.Context, workerID string, now time.Time, lease time.Duration, limit int) ([]ProjectionJob, error) {
	data := projectionData(r)
	data.mu.Lock()
	defer data.mu.Unlock()
	values := []ProjectionJob{}
	for idValue, job := range data.jobs {
		stale := job.Status == ProjectionJobClaimed && job.ClaimedAt != nil && job.ClaimedAt.Add(lease).Before(now)
		if (job.Status != ProjectionJobReady && !stale) || job.AvailableAt.After(now) {
			continue
		}
		claimed := now
		job.Status = ProjectionJobClaimed
		job.ClaimedAt = &claimed
		job.ClaimedBy = workerID
		job.Attempts++
		job.UpdatedAt = now
		data.jobs[idValue] = job
		values = append(values, job)
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].AvailableAt.Equal(values[j].AvailableAt) {
			return values[i].ID < values[j].ID
		}
		return values[i].AvailableAt.Before(values[j].AvailableAt)
	})
	if len(values) > limit {
		values = values[:limit]
	}
	return values, nil
}

func (r *MemoryRepository) CompleteProgramState(_ context.Context, job ProjectionJob, now time.Time) error {
	data := projectionData(r)
	data.mu.Lock()
	defer data.mu.Unlock()
	current, ok := data.jobs[job.ID]
	if !ok {
		return ErrNotFound
	}
	if current.Status != ProjectionJobClaimed || current.ClaimedBy != job.ClaimedBy {
		return ErrVersionConflict
	}
	completed := now
	current.Status = ProjectionJobCompleted
	current.CompletedAt = &completed
	current.UpdatedAt = now
	data.jobs[job.ID] = current
	return nil
}

func (r *MemoryRepository) FailProgramState(_ context.Context, job ProjectionJob, failure string, retryAt time.Time, maxAttempts int) error {
	data := projectionData(r)
	data.mu.Lock()
	defer data.mu.Unlock()
	current, ok := data.jobs[job.ID]
	if !ok {
		return ErrNotFound
	}
	if current.Status != ProjectionJobClaimed || current.ClaimedBy != job.ClaimedBy {
		return ErrVersionConflict
	}
	current.LastError = failure
	current.ClaimedAt = nil
	current.ClaimedBy = ""
	current.UpdatedAt = time.Now().UTC()
	if current.Attempts >= maxAttempts {
		current.Status = ProjectionJobFailed
	} else {
		current.Status = ProjectionJobReady
		current.AvailableAt = retryAt
	}
	data.jobs[job.ID] = current
	return nil
}

func (r *MemoryRepository) ListProgramStateJobs(_ context.Context, tenant string, status ProjectionJobStatus, limit int) ([]ProjectionJob, error) {
	data := projectionData(r)
	data.mu.Lock()
	defer data.mu.Unlock()
	values := []ProjectionJob{}
	for _, job := range data.jobs {
		if job.TenantID != tenant || (status != "" && job.Status != status) {
			continue
		}
		values = append(values, job)
	}
	sort.Slice(values, func(i, j int) bool { return values[i].UpdatedAt.After(values[j].UpdatedAt) })
	if len(values) > limit {
		values = values[:limit]
	}
	return values, nil
}
