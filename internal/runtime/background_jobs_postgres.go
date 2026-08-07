//go:build postgres

package runtime

import (
	"context"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/operations"
)

func (r *PostgresRepository) BackgroundJobs(ctx context.Context, tenant string, limit int) (operations.Snapshot, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	timerSummary := operations.QueueSummary{Queue: "workflow-timers"}
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE wt.state IN ('READY','CLAIMED')),count(*) FILTER (WHERE wt.state='CLAIMED'),count(*) FILTER (WHERE wt.state='FAILED'),COALESCE(max(wt.attempts) FILTER (WHERE wt.state IN ('READY','CLAIMED','FAILED')),0),min(wt.due_at) FILTER (WHERE wt.state IN ('READY','CLAIMED')) FROM workflow_timers wt JOIN tenants t ON t.id=wt.tenant_id WHERE t.id::text=$1 OR t.slug=$1`, tenant).Scan(&timerSummary.Pending, &timerSummary.Running, &timerSummary.Terminal, &timerSummary.HighestAttempts, &timerSummary.OldestPending); err != nil {
		return operations.Snapshot{}, err
	}
	outboxSummary := operations.QueueSummary{Queue: "outbox-delivery"}
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE oe.published_at IS NULL AND oe.dead_lettered_at IS NULL),count(*) FILTER (WHERE oe.published_at IS NULL AND oe.dead_lettered_at IS NULL AND oe.locked_by IS NOT NULL AND oe.locked_by<>'' AND oe.lease_until IS NOT NULL),count(*) FILTER (WHERE oe.dead_lettered_at IS NOT NULL),COALESCE(max(oe.attempts) FILTER (WHERE oe.published_at IS NULL),0),min(COALESCE(oe.next_attempt_at,oe.available_at)) FILTER (WHERE oe.published_at IS NULL AND oe.dead_lettered_at IS NULL) FROM outbox_events oe JOIN tenants t ON t.id=oe.tenant_id WHERE t.id::text=$1 OR t.slug=$1`, tenant).Scan(&outboxSummary.Pending, &outboxSummary.Running, &outboxSummary.Terminal, &outboxSummary.HighestAttempts, &outboxSummary.OldestPending); err != nil {
		return operations.Snapshot{}, err
	}

	jobs := make([]operations.Job, 0, limit*2)
	timerRows, err := r.pool.Query(ctx, `SELECT wt.id::text,wt.timer_type,wt.state,wt.attempts,wt.due_at,wt.lease_until,COALESCE(wt.locked_by,''),COALESCE(wt.last_error,''),wt.failed_at FROM workflow_timers wt JOIN tenants t ON t.id=wt.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND wt.state IN ('READY','CLAIMED','FAILED') ORDER BY COALESCE(wt.failed_at,wt.due_at) DESC,wt.id DESC LIMIT $2`, tenant, limit)
	if err != nil {
		return operations.Snapshot{}, err
	}
	for timerRows.Next() {
		var job operations.Job
		var due time.Time
		if err := timerRows.Scan(&job.ID, &job.Kind, &job.State, &job.Attempts, &due, &job.LeaseUntil, &job.LockedBy, &job.LastError, &job.TerminalAt); err != nil {
			timerRows.Close()
			return operations.Snapshot{}, err
		}
		job.Queue = "workflow-timers"
		job.AvailableAt = &due
		jobs = append(jobs, job)
	}
	if err := timerRows.Err(); err != nil {
		timerRows.Close()
		return operations.Snapshot{}, err
	}
	timerRows.Close()

	outboxRows, err := r.pool.Query(ctx, `SELECT oe.id::text,oe.event_type,CASE WHEN oe.dead_lettered_at IS NOT NULL THEN 'DEAD_LETTERED' WHEN oe.locked_by IS NOT NULL AND oe.locked_by<>'' AND oe.lease_until IS NOT NULL THEN 'CLAIMED' ELSE 'READY' END,oe.attempts,COALESCE(oe.next_attempt_at,oe.available_at),oe.lease_until,COALESCE(oe.locked_by,''),COALESCE(oe.last_error,''),oe.dead_lettered_at,oe.occurred_at FROM outbox_events oe JOIN tenants t ON t.id=oe.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND oe.published_at IS NULL ORDER BY COALESCE(oe.dead_lettered_at,oe.next_attempt_at,oe.available_at) DESC,oe.id DESC LIMIT $2`, tenant, limit)
	if err != nil {
		return operations.Snapshot{}, err
	}
	for outboxRows.Next() {
		var job operations.Job
		var available, created time.Time
		if err := outboxRows.Scan(&job.ID, &job.Kind, &job.State, &job.Attempts, &available, &job.LeaseUntil, &job.LockedBy, &job.LastError, &job.TerminalAt, &created); err != nil {
			outboxRows.Close()
			return operations.Snapshot{}, err
		}
		job.Queue = "outbox-delivery"
		job.AvailableAt = &available
		job.CreatedAt = &created
		jobs = append(jobs, job)
	}
	if err := outboxRows.Err(); err != nil {
		outboxRows.Close()
		return operations.Snapshot{}, err
	}
	outboxRows.Close()
	return operations.Snapshot{Queues: []operations.QueueSummary{timerSummary, outboxSummary}, Jobs: jobs}, nil
}
