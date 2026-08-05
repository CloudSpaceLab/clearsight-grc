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
		if len(values) >= limit {
			break
		}
	}
	sort.Slice(values, func(i, j int) bool { return values[i].CreatedAt.Before(values[j].CreatedAt) })
	return values, nil
}

func (r *MemoryRepository) CompleteProgramState(_ context.Context, job ProjectionJob, now time.Time) error {
	data := projectionData(r)
	data.mu.Lock()
	defer data.mu.Unlock()
	current, ok := data.jobs[job.ID]
	if !ok || current.ClaimedBy != job.ClaimedBy {
		return ErrVersionConflict
	}
	current.Status = ProjectionJobCompleted
	current.CompletedAt = &now
	current.UpdatedAt = now
	data.jobs[job.ID] = current
	return nil
}

func (r *MemoryRepository) FailProgramState(_ context.Context, job ProjectionJob, message string, retryAt time.Time) error {
	data := projectionData(r)
	data.mu.Lock()
	defer data.mu.Unlock()
	current, ok := data.jobs[job.ID]
	if !ok || current.ClaimedBy != job.ClaimedBy {
		return ErrVersionConflict
	}
	current.Status = ProjectionJobReady
	if current.Attempts >= 5 {
		current.Status = ProjectionJobFailed
	}
	current.LastError = message
	current.AvailableAt = retryAt
	current.ClaimedAt = nil
	current.ClaimedBy = ""
	current.UpdatedAt = time.Now().UTC()
	data.jobs[job.ID] = current
	return nil
}

func (r *MemoryRepository) ProjectionHealth(_ context.Context, tenant string) ([]ProjectionHealth, error) {
	data := projectionData(r)
	data.mu.Lock()
	defer data.mu.Unlock()
	health := ProjectionHealth{TenantID: tenant, Projection: ProjectionProgramState, DisplayName: "Program status updates", State: "CURRENT", UpdatedAt: time.Now().UTC()}
	for _, job := range data.jobs {
		if job.TenantID != tenant {
			continue
		}
		switch job.Status {
		case ProjectionJobReady, ProjectionJobClaimed:
			health.Pending++
			if health.OldestPending == nil || job.CreatedAt.Before(*health.OldestPending) {
				value := job.CreatedAt
				health.OldestPending = &value
			}
		case ProjectionJobFailed:
			health.Failed++
			health.LastError = job.LastError
		case ProjectionJobCompleted:
			if job.CompletedAt != nil && (health.LastCompleted == nil || job.CompletedAt.After(*health.LastCompleted)) {
				value := *job.CompletedAt
				health.LastCompleted = &value
			}
		}
	}
	if health.Failed > 0 {
		health.State = "NEEDS_ATTENTION"
	} else if health.Pending > 0 {
		health.State = "UPDATE_PENDING"
	}
	if health.OldestPending != nil {
		health.LagSeconds = int64(time.Since(*health.OldestPending).Seconds())
	}
	return []ProjectionHealth{health}, nil
}

func (r *MemoryRepository) ReconcileProgramState(ctx context.Context, tenant string, limit int) (ReconcileResult, error) {
	r.mu.RLock()
	programs := make([]Program, 0, len(r.programs[tenant]))
	for _, aggregate := range r.programs[tenant] {
		programs = append(programs, aggregate.Program)
	}
	r.mu.RUnlock()
	data := projectionData(r)
	data.mu.Lock()
	active := map[string]bool{}
	for _, job := range data.jobs {
		if job.TenantID == tenant && (job.Status == ProjectionJobReady || job.Status == ProjectionJobClaimed) {
			active[job.AggregateID] = true
		}
	}
	data.mu.Unlock()
	type candidate struct {
		program   Program
		projected int64
		queued    bool
	}
	candidates := make([]candidate, 0, len(programs))
	for _, program := range programs {
		state, _ := r.ProgramStateAt(ctx, tenant, program.ID, nil)
		projected := int64(0)
		if state != nil {
			projected = state.ProgramVersion
		}
		candidates = append(candidates, candidate{program: program, projected: projected, queued: active[program.ID]})
	}
	sort.Slice(candidates, func(i, j int) bool {
		left := candidates[i].projected < candidates[i].program.Version && !candidates[i].queued
		right := candidates[j].projected < candidates[j].program.Version && !candidates[j].queued
		if left != right {
			return left
		}
		return candidates[i].program.UpdatedAt.Before(candidates[j].program.UpdatedAt)
	})
	result := ReconcileResult{TenantID: tenant}
	for _, item := range candidates {
		if result.Checked >= limit {
			break
		}
		result.Checked++
		if item.projected >= item.program.Version {
			result.Current++
			continue
		}
		if item.queued {
			result.AlreadyQueued++
			continue
		}
		if _, err := r.QueueProgramState(ctx, tenant, item.program.ID, item.program.Version, "RECONCILE", "", "system"); err != nil {
			return result, err
		}
		result.Queued++
	}
	return result, nil
}

func (r *MemoryRepository) CreateMatterWithLink(_ context.Context, bundle MatterLinkBundle) (Matter, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	matter := bundle.Matter
	if r.programs[matter.TenantID][bundle.Link.ProgramID].Program.ID == "" {
		return Matter{}, ErrNotFound
	}
	if r.matters[matter.TenantID] == nil {
		r.matters[matter.TenantID] = map[string]MatterAggregate{}
		r.matterEvents[matter.TenantID] = map[string][]Event{}
	}
	if matter.TriggerKey != "" {
		for _, existing := range r.matters[matter.TenantID] {
			if existing.Matter.TriggerKey == matter.TriggerKey && existing.Matter.Status != MatterClosed && existing.Matter.Status != MatterCancelled {
				return Matter{}, ErrDuplicate
			}
		}
	}
	aggregate := MatterAggregate{Matter: matter, Closure: ClosureAssessment{Ready: false}}
	if err := applyMatterEventToAggregate(&aggregate, bundle.LinkEvent); err != nil {
		return Matter{}, err
	}
	aggregate.Matter.Version = bundle.LinkEvent.AggregateVersion
	aggregate.Matter.UpdatedAt = bundle.LinkEvent.OccurredAt
	r.matters[matter.TenantID][matter.ID] = aggregate
	r.matterEvents[matter.TenantID][matter.ID] = []Event{bundle.MatterEvent, bundle.LinkEvent}
	return aggregate.Matter, nil
}

func (r *MemoryRepository) ApplyTriggerBundle(ctx context.Context, bundle TriggerBundle) (TriggerBundleResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	trigger := bundle.Trigger
	program, ok := r.programs[trigger.TenantID][trigger.ProgramID]
	if !ok {
		return TriggerBundleResult{}, ErrNotFound
	}
	if r.triggers[trigger.TenantID] == nil {
		r.triggers[trigger.TenantID] = map[string]Trigger{}
	}
	if _, exists := r.triggers[trigger.TenantID][trigger.DedupeKey]; exists {
		for _, aggregate := range r.matters[trigger.TenantID] {
			if aggregate.Matter.TriggerKey == trigger.DedupeKey && aggregate.Matter.Status != MatterClosed && aggregate.Matter.Status != MatterCancelled {
				matter := aggregate.Matter
				return TriggerBundleResult{Inserted: false, Matter: &matter}, nil
			}
		}
		return TriggerBundleResult{Inserted: false}, nil
	}
	if bundle.ProgramEvent.AggregateVersion != program.Program.Version+1 {
		return TriggerBundleResult{}, ErrVersionConflict
	}
	if err := applyProgramEventToAggregate(&program, bundle.ProgramEvent); err != nil {
		return TriggerBundleResult{}, err
	}
	program.Program.Version = bundle.ProgramEvent.AggregateVersion
	program.Program.UpdatedAt = bundle.ProgramEvent.OccurredAt
	r.programs[trigger.TenantID][trigger.ProgramID] = program
	r.programEvents[trigger.TenantID][trigger.ProgramID] = append(r.programEvents[trigger.TenantID][trigger.ProgramID], bundle.ProgramEvent)
	r.triggers[trigger.TenantID][trigger.DedupeKey] = trigger
	result := TriggerBundleResult{Inserted: true}
	if bundle.Matter != nil && bundle.MatterEvent != nil && bundle.Link != nil && bundle.LinkEvent != nil {
		matter := *bundle.Matter
		if r.matters[matter.TenantID] == nil {
			r.matters[matter.TenantID] = map[string]MatterAggregate{}
			r.matterEvents[matter.TenantID] = map[string][]Event{}
		}
		aggregate := MatterAggregate{Matter: matter, Closure: ClosureAssessment{Ready: false}}
		if err := applyMatterEventToAggregate(&aggregate, *bundle.LinkEvent); err != nil {
			return TriggerBundleResult{}, err
		}
		aggregate.Matter.Version = bundle.LinkEvent.AggregateVersion
		aggregate.Matter.UpdatedAt = bundle.LinkEvent.OccurredAt
		r.matters[matter.TenantID][matter.ID] = aggregate
		r.matterEvents[matter.TenantID][matter.ID] = []Event{*bundle.MatterEvent, *bundle.LinkEvent}
		created := aggregate.Matter
		result.Matter = &created
	}
	_, _ = r.QueueProgramState(ctx, trigger.TenantID, trigger.ProgramID, program.Program.Version, trigger.Type, trigger.ID, "system")
	return result, nil
}

var _ ProgramStateRepository = (*MemoryRepository)(nil)
var _ ProjectionRepository = (*MemoryRepository)(nil)
var _ CompoundRepository = (*MemoryRepository)(nil)
var _ TriggerBundleRepository = (*MemoryRepository)(nil)
