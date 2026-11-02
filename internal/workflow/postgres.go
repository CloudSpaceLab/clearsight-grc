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
	hasCursor, afterDue, afterUpdated, afterID := postgresWorkCursorArgs(filter)
	rows, err := r.pool.Query(ctx, postgresActorWorkRowsSQL()+`
		ORDER BY
		  CASE WHEN $5::boolean THEN wt.due_at IS NULL ELSE false END ASC,
		  CASE WHEN $5::boolean THEN wt.due_at END ASC NULLS LAST,
		  wt.updated_at DESC,
		  wt.id ASC
		LIMIT $9`,
		filter.TenantID, filter.PrincipalID, string(filter.Status), filter.WorkflowKind,
		filter.ActiveOnly, filter.VisibleMatterWorkOnly, filter.VisibleActorWorkOnly, filter.LegalEntityID, filter.Limit,
		filter.Search, filter.DueFrom, filter.DueBefore,
		filter.UpdatedThrough, hasCursor, afterDue, afterUpdated, afterID,
	)
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
	var sourceBindingsJSON []byte
	var matterScope []byte
	if err := row.Scan(
		&task.ID, &task.TenantID, &task.WorkflowID, &task.StepKey, &task.Responsibility,
		&task.PrincipalID, &task.Title, &task.Status, &task.DueAt, &task.ClaimedAt, &task.CompletedAt,
		&contextJSON, &sourceBindingsJSON, &task.Version, &task.CreatedAt, &task.UpdatedAt,
		&task.WorkflowKind, &task.MatterID, &task.MatterPriority, &matterScope,
		&task.EvidenceRequestID, &task.EvidenceRecipientID, &task.EvidenceSubjectVisible,
	); err != nil {
		return Task{}, err
	}
	if err := json.Unmarshal(contextJSON, &task.Context); err != nil {
		return Task{}, fmt.Errorf("decode workflow context: %w", err)
	}
	if err := json.Unmarshal(sourceBindingsJSON, &task.SourceBindings); err != nil {
		return Task{}, fmt.Errorf("decode workflow source bindings: %w", err)
	}
	task.MatterScope = append(task.MatterScope[:0], matterScope...)
	return task, nil
}
