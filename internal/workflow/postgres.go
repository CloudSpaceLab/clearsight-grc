//go:build postgres

package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct{ pool *pgxpool.Pool }

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, input CreateInput) (Task, error) {
	payload, err := json.Marshal(input.Context)
	if err != nil {
		return Task{}, fmt.Errorf("encode workflow context: %w", err)
	}
	var task Task
	var contextJSON []byte
	err = r.pool.QueryRow(ctx, `
		INSERT INTO workflow_tasks(tenant_id,workflow_id,step_key,responsibility,principal_id,title,status,due_at,context)
		VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2::uuid,$3,$4,NULLIF($5,'')::uuid,$6,'READY',$7,$8::jsonb)
		RETURNING id::text,(SELECT slug FROM tenants WHERE id=tenant_id),workflow_id::text,step_key,responsibility,
		          COALESCE(principal_id::text,''),title,status,due_at,claimed_at,completed_at,context,version,created_at,updated_at`,
		input.TenantID, input.WorkflowID, input.StepKey, input.Responsibility, input.PrincipalID, input.Title, input.DueAt, string(payload),
	).Scan(&task.ID, &task.TenantID, &task.WorkflowID, &task.StepKey, &task.Responsibility, &task.PrincipalID, &task.Title, &task.Status, &task.DueAt, &task.ClaimedAt, &task.CompletedAt, &contextJSON, &task.Version, &task.CreatedAt, &task.UpdatedAt)
	if err != nil {
		return Task{}, fmt.Errorf("create workflow task: %w", err)
	}
	if err := json.Unmarshal(contextJSON, &task.Context); err != nil {
		return Task{}, fmt.Errorf("decode workflow context: %w", err)
	}
	return task, nil
}

func (r *PostgresRepository) Get(ctx context.Context, tenantID, id string) (Task, error) {
	return r.scanOne(ctx, `
		SELECT wt.id::text,t.slug,wt.workflow_id::text,wt.step_key,wt.responsibility,
		       COALESCE(wt.principal_id::text,''),wt.title,wt.status,wt.due_at,wt.claimed_at,wt.completed_at,
		       wt.context,wt.version,wt.created_at,wt.updated_at
		FROM workflow_tasks wt JOIN tenants t ON t.id=wt.tenant_id
		WHERE wt.id=$1::uuid AND (t.id::text=$2 OR t.slug=$2)`, id, tenantID)
}

func (r *PostgresRepository) List(ctx context.Context, filter ListFilter) ([]Task, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT wt.id::text,t.slug,wt.workflow_id::text,wt.step_key,wt.responsibility,
		       COALESCE(wt.principal_id::text,''),wt.title,wt.status,wt.due_at,wt.claimed_at,wt.completed_at,
		       wt.context,wt.version,wt.created_at,wt.updated_at
		FROM workflow_tasks wt JOIN tenants t ON t.id=wt.tenant_id
		WHERE (t.slug=$1 OR t.id::text=$1)
		  AND ($2='' OR wt.principal_id::text=$2)
		  AND ($3='' OR wt.status=$3)
		ORDER BY wt.updated_at DESC LIMIT $4`, filter.TenantID, filter.PrincipalID, string(filter.Status), filter.Limit)
	if err != nil {
		return nil, fmt.Errorf("list workflow tasks: %w", err)
	}
	defer rows.Close()
	values := []Task{}
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		values = append(values, task)
	}
	return values, rows.Err()
}

func (r *PostgresRepository) Transition(ctx context.Context, id string, input TransitionInput) (Task, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Task{}, fmt.Errorf("begin workflow transition: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var task Task
	var contextJSON []byte
	err = tx.QueryRow(ctx, `
		UPDATE workflow_tasks wt
		SET status=$3,
		    claimed_at=CASE WHEN $3='IN_PROGRESS' AND claimed_at IS NULL THEN clock_timestamp() ELSE claimed_at END,
		    completed_at=CASE WHEN $3='COMPLETED' THEN clock_timestamp() ELSE completed_at END,
		    version=version+1,
		    updated_at=clock_timestamp()
		WHERE wt.id=$1::uuid
		  AND wt.tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2)
		  AND wt.version=$4
		RETURNING wt.id::text,(SELECT slug FROM tenants WHERE id=wt.tenant_id),wt.workflow_id::text,wt.step_key,wt.responsibility,
		          COALESCE(wt.principal_id::text,''),wt.title,wt.status,wt.due_at,wt.claimed_at,wt.completed_at,
		          wt.context,wt.version,wt.created_at,wt.updated_at`, id, input.TenantID, string(input.Status), input.ExpectedVersion,
	).Scan(&task.ID, &task.TenantID, &task.WorkflowID, &task.StepKey, &task.Responsibility, &task.PrincipalID, &task.Title, &task.Status, &task.DueAt, &task.ClaimedAt, &task.CompletedAt, &contextJSON, &task.Version, &task.CreatedAt, &task.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		if _, getErr := r.Get(ctx, input.TenantID, id); getErr == nil {
			return Task{}, ErrVersionConflict
		}
		return Task{}, ErrTaskNotFound
	}
	if err != nil {
		return Task{}, fmt.Errorf("transition workflow task: %w", err)
	}
	if err := json.Unmarshal(contextJSON, &task.Context); err != nil {
		return Task{}, fmt.Errorf("decode workflow context: %w", err)
	}
	metadata, err := json.Marshal(map[string]string{"status": string(input.Status), "reason": input.Reason})
	if err != nil {
		return Task{}, fmt.Errorf("encode workflow event: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO workflow_events(tenant_id,workflow_id,task_id,event_type,actor_id,safe_metadata)
		VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2::uuid,$3::uuid,$4,NULLIF($5,'')::uuid,$6::jsonb)`,
		input.TenantID, task.WorkflowID, task.ID, "TASK_"+string(input.Status), input.ActorID, string(metadata),
	); err != nil {
		return Task{}, fmt.Errorf("record workflow event: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Task{}, fmt.Errorf("commit workflow transition: %w", err)
	}
	return task, nil
}

func (r *PostgresRepository) scanOne(ctx context.Context, query string, args ...any) (Task, error) {
	task, err := scanTask(r.pool.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return Task{}, ErrTaskNotFound
	}
	return task, err
}

type scanner interface{ Scan(...any) error }

func scanTask(row scanner) (Task, error) {
	var task Task
	var contextJSON []byte
	if err := row.Scan(&task.ID, &task.TenantID, &task.WorkflowID, &task.StepKey, &task.Responsibility, &task.PrincipalID, &task.Title, &task.Status, &task.DueAt, &task.ClaimedAt, &task.CompletedAt, &contextJSON, &task.Version, &task.CreatedAt, &task.UpdatedAt); err != nil {
		return Task{}, err
	}
	if err := json.Unmarshal(contextJSON, &task.Context); err != nil {
		return Task{}, fmt.Errorf("decode workflow context: %w", err)
	}
	return task, nil
}
