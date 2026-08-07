//go:build postgres

package workflow

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct{ pool *pgxpool.Pool }

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
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
