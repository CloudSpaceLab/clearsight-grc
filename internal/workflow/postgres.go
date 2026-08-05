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
	payload, _ := json.Marshal(input.Context)
	var task Task
	var contextJSON []byte
	err := r.pool.QueryRow(ctx, `INSERT INTO workflow_tasks(tenant_id,workflow_id,step_key,responsibility,principal_id,title,status,due_at,context) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2::uuid,$3,$4,NULLIF($5,'')::uuid,$6,'READY',$7,$8::jsonb) RETURNING id::text,(SELECT slug FROM tenants WHERE id=tenant_id),workflow_id::text,step_key,responsibility,COALESCE(principal_id::text,''),title,status,due_at,context,version,created_at,updated_at`, input.TenantID, input.WorkflowID, input.StepKey, input.Responsibility, input.PrincipalID, input.Title, input.DueAt, string(payload)).Scan(&task.ID, &task.TenantID, &task.WorkflowID, &task.StepKey, &task.Responsibility, &task.PrincipalID, &task.Title, &task.Status, &task.DueAt, &contextJSON, &task.Version, &task.CreatedAt, &task.UpdatedAt)
	if err != nil { return Task{}, fmt.Errorf("create workflow task: %w", err) }
	_ = json.Unmarshal(contextJSON, &task.Context)
	return task, nil
}
func (r *PostgresRepository) Get(ctx context.Context, id string) (Task, error) {
	return r.scanOne(ctx, `SELECT wt.id::text,t.slug,wt.workflow_id::text,wt.step_key,wt.responsibility,COALESCE(wt.principal_id::text,''),wt.title,wt.status,wt.due_at,wt.context,wt.version,wt.created_at,wt.updated_at FROM workflow_tasks wt JOIN tenants t ON t.id=wt.tenant_id WHERE wt.id=$1::uuid`, id)
}
func (r *PostgresRepository) List(ctx context.Context, filter ListFilter) ([]Task, error) {
	rows, err := r.pool.Query(ctx, `SELECT wt.id::text,t.slug,wt.workflow_id::text,wt.step_key,wt.responsibility,COALESCE(wt.principal_id::text,''),wt.title,wt.status,wt.due_at,wt.context,wt.version,wt.created_at,wt.updated_at FROM workflow_tasks wt JOIN tenants t ON t.id=wt.tenant_id WHERE ($1='' OR t.slug=$1 OR t.id::text=$1) AND ($2='' OR wt.principal_id::text=$2) AND ($3='' OR wt.status=$3) ORDER BY wt.updated_at DESC LIMIT $4`, filter.TenantID, filter.PrincipalID, string(filter.Status), filter.Limit)
	if err != nil { return nil, fmt.Errorf("list workflow tasks: %w", err) }
	defer rows.Close()
	values := []Task{}
	for rows.Next() { task, err := scanTask(rows); if err != nil { return nil, err }; values = append(values, task) }
	return values, rows.Err()
}
func (r *PostgresRepository) Transition(ctx context.Context, id string, input TransitionInput) (Task, error) {
	task, err := r.scanOne(ctx, `UPDATE workflow_tasks SET status=$2,version=version+1,updated_at=clock_timestamp() WHERE id=$1::uuid AND version=$3 RETURNING id::text,(SELECT slug FROM tenants WHERE id=tenant_id),workflow_id::text,step_key,responsibility,COALESCE(principal_id::text,''),title,status,due_at,context,version,created_at,updated_at`, id, string(input.Status), input.ExpectedVersion)
	if errors.Is(err, ErrTaskNotFound) { if _, getErr := r.Get(ctx, id); getErr == nil { return Task{}, ErrVersionConflict } }
	return task, err
}
func (r *PostgresRepository) scanOne(ctx context.Context, query string, args ...any) (Task, error) {
	row := r.pool.QueryRow(ctx, query, args...)
	task, err := scanTask(row)
	if errors.Is(err, pgx.ErrNoRows) { return Task{}, ErrTaskNotFound }
	return task, err
}
type scanner interface{ Scan(...any) error }
func scanTask(row scanner) (Task, error) {
	var task Task
	var contextJSON []byte
	if err := row.Scan(&task.ID, &task.TenantID, &task.WorkflowID, &task.StepKey, &task.Responsibility, &task.PrincipalID, &task.Title, &task.Status, &task.DueAt, &contextJSON, &task.Version, &task.CreatedAt, &task.UpdatedAt); err != nil { return Task{}, err }
	_ = json.Unmarshal(contextJSON, &task.Context)
	return task, nil
}
