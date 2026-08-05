//go:build postgres

package runtime

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct{ pool *pgxpool.Pool }

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}
func (r *PostgresRepository) ScheduleTimer(ctx context.Context, t Timer) (Timer, error) {
	_, err := r.pool.Exec(ctx, `INSERT INTO workflow_timers(id,tenant_id,workflow_id,task_id,timer_type,due_at,state,dedupe_key,payload) VALUES($1,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,NULLIF($4,'')::uuid,$5,$6,'READY',$7,$8) ON CONFLICT(tenant_id,dedupe_key) DO NOTHING`, t.ID, t.TenantID, t.WorkflowID, t.TaskID, t.Type, t.DueAt, t.DedupeKey, t.Payload)
	if err != nil {
		return Timer{}, err
	}
	return r.timerByDedupe(ctx, t.TenantID, t.DedupeKey)
}
func (r *PostgresRepository) timerByDedupe(ctx context.Context, tenant, dedupe string) (Timer, error) {
	var t Timer
	err := r.pool.QueryRow(ctx, `SELECT wt.id::text,tn.slug,wt.workflow_id::text,COALESCE(wt.task_id::text,''),wt.timer_type,wt.due_at,wt.state,wt.dedupe_key,wt.payload,wt.attempts,wt.lease_until,COALESCE(wt.locked_by,'') FROM workflow_timers wt JOIN tenants tn ON tn.id=wt.tenant_id WHERE (tn.id::text=$1 OR tn.slug=$1) AND wt.dedupe_key=$2`, tenant, dedupe).Scan(&t.ID, &t.TenantID, &t.WorkflowID, &t.TaskID, &t.Type, &t.DueAt, &t.State, &t.DedupeKey, &t.Payload, &t.Attempts, &t.LeaseUntil, &t.LockedBy)
	return t, err
}
func (r *PostgresRepository) ClaimDueTimers(ctx context.Context, worker string, now time.Time, lease time.Duration, limit int) ([]Timer, error) {
	rows, err := r.pool.Query(ctx, `WITH due AS (SELECT id FROM workflow_timers WHERE (state='READY' OR (state='CLAIMED' AND lease_until<$1)) AND due_at<=$1 ORDER BY due_at,id LIMIT $2 FOR UPDATE SKIP LOCKED), claimed AS (UPDATE workflow_timers wt SET state='CLAIMED',locked_by=$3,lease_until=$1+$4::interval,attempts=attempts+1 FROM due WHERE wt.id=due.id RETURNING wt.id,wt.tenant_id,wt.workflow_id,wt.task_id,wt.timer_type,wt.due_at,wt.state,wt.dedupe_key,wt.payload,wt.attempts,wt.lease_until,wt.locked_by) SELECT c.id::text,t.slug,c.workflow_id::text,COALESCE(c.task_id::text,''),c.timer_type,c.due_at,c.state,c.dedupe_key,c.payload,c.attempts,c.lease_until,c.locked_by FROM claimed c JOIN tenants t ON t.id=c.tenant_id`, now, limit, worker, lease.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Timer{}
	for rows.Next() {
		var t Timer
		if err := rows.Scan(&t.ID, &t.TenantID, &t.WorkflowID, &t.TaskID, &t.Type, &t.DueAt, &t.State, &t.DedupeKey, &t.Payload, &t.Attempts, &t.LeaseUntil, &t.LockedBy); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
func (r *PostgresRepository) CompleteTimer(ctx context.Context, t Timer, e OutboxEvent, now time.Time) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `UPDATE workflow_timers SET state='FIRED',fired_at=$2,lease_until=NULL,locked_by=NULL WHERE id=$1::uuid AND state='CLAIMED' AND locked_by=$3`, t.ID, now, t.LockedBy)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errors.New("timer claim lost")
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(id,tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at) VALUES($1,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3,$4::uuid,$5,$6,$7,$7,$7)`, e.ID, e.TenantID, e.AggregateType, e.AggregateID, e.EventType, e.Payload, e.OccurredAt)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *PostgresRepository) FailTimer(ctx context.Context, t Timer, message string, next time.Time) error {
	tag, err := r.pool.Exec(ctx, `UPDATE workflow_timers SET state='READY',due_at=$2,last_error=$3,lease_until=NULL,locked_by=NULL WHERE id=$1::uuid AND state='CLAIMED' AND locked_by=$4`, t.ID, next, message, t.LockedBy)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errors.New("timer claim lost")
	}
	return nil
}
func (r *PostgresRepository) ClaimOutbox(ctx context.Context, worker string, now time.Time, lease time.Duration, limit int) ([]OutboxEvent, error) {
	rows, err := r.pool.Query(ctx, `WITH due AS (SELECT id FROM outbox_events WHERE published_at IS NULL AND COALESCE(next_attempt_at,available_at)<=$1 AND (lease_until IS NULL OR lease_until<$1) ORDER BY COALESCE(next_attempt_at,available_at),id LIMIT $2 FOR UPDATE SKIP LOCKED), claimed AS (UPDATE outbox_events oe SET locked_by=$3,lease_until=$1+$4::interval,attempts=attempts+1 FROM due WHERE oe.id=due.id RETURNING oe.id,oe.tenant_id,oe.aggregate_type,oe.aggregate_id,oe.event_type,oe.payload,oe.occurred_at,oe.attempts,oe.locked_by,oe.next_attempt_at) SELECT c.id::text,t.slug,c.aggregate_type,c.aggregate_id::text,c.event_type,c.payload,c.occurred_at,c.attempts,c.locked_by,c.next_attempt_at FROM claimed c JOIN tenants t ON t.id=c.tenant_id`, now, limit, worker, lease.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []OutboxEvent{}
	for rows.Next() {
		var e OutboxEvent
		if err := rows.Scan(&e.ID, &e.TenantID, &e.AggregateType, &e.AggregateID, &e.EventType, &e.Payload, &e.OccurredAt, &e.Attempts, &e.LockedBy, &e.NextAttemptAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
func (r *PostgresRepository) MarkPublished(ctx context.Context, e OutboxEvent, at time.Time) error {
	tag, err := r.pool.Exec(ctx, `UPDATE outbox_events SET published_at=$2,locked_by=NULL,lease_until=NULL,last_error=NULL WHERE id=$1::uuid AND published_at IS NULL AND locked_by=$3`, e.ID, at, e.LockedBy)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errors.New("outbox claim lost")
	}
	return nil
}
func (r *PostgresRepository) MarkFailed(ctx context.Context, e OutboxEvent, message string, next time.Time) error {
	tag, err := r.pool.Exec(ctx, `UPDATE outbox_events SET next_attempt_at=$2,last_error=$3,locked_by=NULL,lease_until=NULL WHERE id=$1::uuid AND published_at IS NULL AND locked_by=$4`, e.ID, next, message, e.LockedBy)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errors.New("outbox claim lost")
	}
	return nil
}
func (r *PostgresRepository) RecordInbox(ctx context.Context, tenant, consumer, eventID string, at time.Time) (bool, error) {
	tag, err := r.pool.Exec(ctx, `INSERT INTO inbox_receipts(tenant_id,consumer,event_id,processed_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2,$3,$4) ON CONFLICT DO NOTHING`, tenant, consumer, eventID, at)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}
