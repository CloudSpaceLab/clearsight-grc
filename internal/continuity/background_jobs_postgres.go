//go:build postgres

package continuity

import (
	"context"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/operations"
)

func (r *PostgresRepository) BackgroundJobs(ctx context.Context, tenant string, limit int) (operations.Snapshot, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	summary := operations.QueueSummary{Queue: "program-projection"}
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE j.status IN ('READY','CLAIMED')),count(*) FILTER (WHERE j.status='CLAIMED'),count(*) FILTER (WHERE j.status='FAILED'),COALESCE(max(j.attempts) FILTER (WHERE j.status IN ('READY','CLAIMED','FAILED')),0),min(j.available_at) FILTER (WHERE j.status IN ('READY','CLAIMED')) FROM continuity_projection_jobs j JOIN tenants t ON t.id=j.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND j.projection_name='PROGRAM_STATE'`, tenant).Scan(&summary.Pending, &summary.Running, &summary.Terminal, &summary.HighestAttempts, &summary.OldestPending); err != nil {
		return operations.Snapshot{}, err
	}
	rows, err := r.pool.Query(ctx, `SELECT j.id::text,j.reason,j.status,j.attempts,j.available_at,COALESCE(j.claimed_by,''),COALESCE(j.last_error,''),j.created_at,j.updated_at FROM continuity_projection_jobs j JOIN tenants t ON t.id=j.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND j.projection_name='PROGRAM_STATE' AND j.status IN ('READY','CLAIMED','FAILED') ORDER BY j.updated_at DESC,j.id DESC LIMIT $2`, tenant, limit)
	if err != nil {
		return operations.Snapshot{}, err
	}
	defer rows.Close()
	jobs := []operations.Job{}
	for rows.Next() {
		var job operations.Job
		var available, created, updated time.Time
		if err := rows.Scan(&job.ID, &job.Kind, &job.State, &job.Attempts, &available, &job.LockedBy, &job.LastError, &created, &updated); err != nil {
			return operations.Snapshot{}, err
		}
		job.Queue = "program-projection"
		job.AvailableAt, job.CreatedAt, job.UpdatedAt = &available, &created, &updated
		if job.State == string(ProjectionJobFailed) {
			job.TerminalAt = &updated
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return operations.Snapshot{}, err
	}
	return operations.Snapshot{Queues: []operations.QueueSummary{summary}, Jobs: jobs}, nil
}
