//go:build postgres

package runtime

import (
	"context"
	"errors"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/operations"
	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) RetryTerminalJob(ctx context.Context, input operations.RetryInput) (operations.RecoveryReceipt, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return operations.RecoveryReceipt{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	now := time.Now().UTC()
	var attempts int
	var terminalAt time.Time
	switch input.Queue {
	case operations.QueueOutbox:
		err = tx.QueryRow(ctx, `SELECT oe.attempts,oe.dead_lettered_at FROM outbox_events oe JOIN tenants t ON t.id=oe.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND oe.id=$2::uuid AND oe.published_at IS NULL AND oe.dead_lettered_at IS NOT NULL FOR UPDATE`, input.TenantID, input.JobID).Scan(&attempts, &terminalAt)
	case operations.QueueTimers:
		err = tx.QueryRow(ctx, `SELECT wt.attempts,wt.failed_at FROM workflow_timers wt JOIN tenants t ON t.id=wt.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND wt.id=$2::uuid AND wt.state='FAILED' AND wt.failed_at IS NOT NULL FOR UPDATE`, input.TenantID, input.JobID).Scan(&attempts, &terminalAt)
	default:
		return operations.RecoveryReceipt{}, operations.ErrRecoveryInvalid
	}
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && attempts != input.ExpectedAttempts) {
		return operations.RecoveryReceipt{}, operations.ErrRecoveryConflict
	}
	if err != nil {
		return operations.RecoveryReceipt{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO operational_recovery_events(tenant_id,queue,job_id,decision,previous_attempts,terminal_at,actor_principal_id,rationale,recovered_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2,$3::uuid,'RETRY',$4,$5,$6::uuid,$7,$8)`, input.TenantID, input.Queue, input.JobID, attempts, terminalAt, input.ActorPrincipalID, input.Rationale, now); err != nil {
		return operations.RecoveryReceipt{}, err
	}
	if input.Queue == operations.QueueOutbox {
		_, err = tx.Exec(ctx, `UPDATE outbox_events SET attempts=0,dead_lettered_at=NULL,next_attempt_at=$3,locked_by=NULL,lease_until=NULL WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND id=$2::uuid`, input.TenantID, input.JobID, now)
	} else {
		_, err = tx.Exec(ctx, `UPDATE workflow_timers SET attempts=0,state='READY',failed_at=NULL,due_at=$3,locked_by=NULL,lease_until=NULL WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND id=$2::uuid`, input.TenantID, input.JobID, now)
	}
	if err != nil {
		return operations.RecoveryReceipt{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return operations.RecoveryReceipt{}, err
	}
	return operations.RecoveryReceipt{JobID: input.JobID, Queue: input.Queue, PreviousAttempts: attempts, State: "READY", RetriedAt: now}, nil
}
