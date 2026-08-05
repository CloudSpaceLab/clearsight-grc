package continuity

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const ProjectionProgramState = "PROGRAM_STATE"

type ProjectionJobStatus string

const (
	ProjectionJobReady     ProjectionJobStatus = "READY"
	ProjectionJobClaimed   ProjectionJobStatus = "CLAIMED"
	ProjectionJobCompleted ProjectionJobStatus = "COMPLETED"
	ProjectionJobFailed    ProjectionJobStatus = "FAILED"
)

type ProjectionJob struct {
	ID                     string              `json:"id"`
	TenantID               string              `json:"tenant_id"`
	Projection             string              `json:"projection"`
	AggregateType          string              `json:"aggregate_type"`
	AggregateID            string              `json:"aggregate_id"`
	SourceAggregateVersion int64               `json:"source_aggregate_version"`
	Reason                 string              `json:"reason"`
	TriggerID              string              `json:"trigger_id,omitempty"`
	RequestedBy            string              `json:"requested_by,omitempty"`
	Status                 ProjectionJobStatus `json:"status"`
	Attempts               int                 `json:"attempts"`
	AvailableAt            time.Time           `json:"available_at"`
	ClaimedAt              *time.Time          `json:"claimed_at,omitempty"`
	ClaimedBy              string              `json:"claimed_by,omitempty"`
	CompletedAt            *time.Time          `json:"completed_at,omitempty"`
	LastError              string              `json:"last_error,omitempty"`
	CreatedAt              time.Time           `json:"created_at"`
	UpdatedAt              time.Time           `json:"updated_at"`
}

type ProjectionHealth struct {
	TenantID      string     `json:"tenant_id"`
	Projection    string     `json:"projection"`
	DisplayName   string     `json:"display_name"`
	State         string     `json:"state"`
	Pending       int        `json:"pending"`
	Failed        int        `json:"failed"`
	OldestPending *time.Time `json:"oldest_pending,omitempty"`
	LastCompleted *time.Time `json:"last_completed,omitempty"`
	LastError     string     `json:"last_error,omitempty"`
	LagSeconds    int64      `json:"lag_seconds"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type ReconcileResult struct {
	TenantID      string `json:"tenant_id"`
	Checked       int    `json:"checked"`
	Queued        int    `json:"queued"`
	AlreadyQueued int    `json:"already_queued"`
	Current       int    `json:"current"`
}

type ProgramStateRepository interface {
	SaveProgramState(context.Context, string, string, int64, ProgramStateSnapshot) (int64, error)
	ProgramStateAt(context.Context, string, string, *time.Time) (*ProgramStateSnapshot, error)
}

type ProjectionRepository interface {
	QueueProgramState(context.Context, string, string, int64, string, string, string) (ProjectionJob, error)
	ClaimProgramState(context.Context, string, time.Time, time.Duration, int) ([]ProjectionJob, error)
	CompleteProgramState(context.Context, ProjectionJob, time.Time) error
	FailProgramState(context.Context, ProjectionJob, string, time.Time) error
	ProjectionHealth(context.Context, string) ([]ProjectionHealth, error)
	ReconcileProgramState(context.Context, string, int) (ReconcileResult, error)
}

// TransactionalProjectionRepository marks repositories that create a
// deduplicated status-update job in the same transaction as every material
// Program/Matter command. The service must not issue a second queue write for
// these repositories after the command has committed.
type TransactionalProjectionRepository interface {
	ProjectionQueuedWithCommands() bool
}

type MatterLinkBundle struct {
	Matter      Matter
	MatterEvent Event
	Link        MatterLink
	LinkEvent   Event
}

type CompoundRepository interface {
	CreateMatterWithLink(context.Context, MatterLinkBundle) (Matter, error)
}

type TriggerBundle struct {
	Trigger      Trigger
	ProgramEvent Event
	Matter       *Matter
	MatterEvent  *Event
	Link         *MatterLink
	LinkEvent    *Event
}

type TriggerBundleResult struct {
	Inserted bool
	Matter   *Matter
}

type TriggerBundleRepository interface {
	ApplyTriggerBundle(context.Context, TriggerBundle) (TriggerBundleResult, error)
}

func (s *Service) ProjectionHealth(ctx context.Context, tenant string) ([]ProjectionHealth, error) {
	if strings.TrimSpace(tenant) == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}
	repo, ok := s.repo.(ProjectionRepository)
	if !ok {
		return []ProjectionHealth{{TenantID: tenant, Projection: ProjectionProgramState, DisplayName: "Program status updates", State: "NOT_CONFIGURED", UpdatedAt: s.now().UTC()}}, nil
	}
	return repo.ProjectionHealth(ctx, tenant)
}

func (s *Service) ReconcileProgramState(ctx context.Context, tenant string, limit int) (ReconcileResult, error) {
	if strings.TrimSpace(tenant) == "" {
		return ReconcileResult{}, fmt.Errorf("tenant_id is required")
	}
	repo, ok := s.repo.(ProjectionRepository)
	if !ok {
		return ReconcileResult{}, fmt.Errorf("Program status maintenance is unavailable")
	}
	if limit <= 0 || limit > 1000 {
		limit = 250
	}
	return repo.ReconcileProgramState(ctx, tenant, limit)
}

func (s *Service) QueueProgramStateRebuild(ctx context.Context, tenant, programID, requestedBy, reason string) (ProjectionJob, error) {
	if strings.TrimSpace(tenant) == "" || strings.TrimSpace(programID) == "" || strings.TrimSpace(reason) == "" {
		return ProjectionJob{}, fmt.Errorf("tenant_id, program_id and reason are required")
	}
	aggregate, err := s.repo.GetProgram(ctx, tenant, programID)
	if err != nil {
		return ProjectionJob{}, err
	}
	repo, ok := s.repo.(ProjectionRepository)
	if !ok {
		return ProjectionJob{}, fmt.Errorf("Program status maintenance is unavailable")
	}
	return repo.QueueProgramState(ctx, tenant, programID, aggregate.Program.Version, "MANUAL_REBUILD", reason, requestedBy)
}

type ProjectionMaintainer struct {
	Service  *Service
	Repo     ProjectionRepository
	WorkerID string
	Lease    time.Duration
	Now      func() time.Time
}

func (m *ProjectionMaintainer) Maintain(ctx context.Context, now time.Time, limit int) (int, error) {
	if m == nil || m.Service == nil || m.Repo == nil {
		return 0, nil
	}
	if m.WorkerID == "" {
		m.WorkerID = "projection-worker"
	}
	if m.Lease <= 0 {
		m.Lease = 30 * time.Second
	}
	if m.Now != nil {
		now = m.Now().UTC()
	}
	jobs, err := m.Repo.ClaimProgramState(ctx, m.WorkerID, now, m.Lease, limit)
	if err != nil {
		return 0, err
	}
	completed := 0
	for _, job := range jobs {
		if err := m.Service.refreshProgram(ctx, job.TenantID, job.AggregateID, job.Reason, job.TriggerID); err != nil {
			retryAt := now.Add(time.Duration(min(job.Attempts+1, 10)) * time.Second)
			_ = m.Repo.FailProgramState(ctx, job, err.Error(), retryAt)
			continue
		}
		if err := m.Repo.CompleteProgramState(ctx, job, now); err != nil {
			return completed, err
		}
		completed++
	}
	return completed, nil
}
