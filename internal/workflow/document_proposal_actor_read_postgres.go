//go:build postgres

package workflow

import (
	"context"
	"fmt"
)

func (r *PostgresRepository) ListDocumentProposalActorWork(ctx context.Context, filter ListFilter) ([]Task, error) {
	hasCursor, afterDue, afterUpdated, afterID := postgresWorkCursorArgs(filter)
	rows, err := r.pool.Query(ctx, postgresDocumentProposalRowsSQL()+`
		ORDER BY wt.due_at IS NULL,wt.due_at ASC NULLS LAST,wt.updated_at DESC,wt.id ASC
		LIMIT $7`,
		filter.TenantID, filter.PrincipalID, filter.LegalEntityID, string(filter.Status),
		DocumentProposalWorkflowKind, filter.ActiveOnly, filter.Limit,
		filter.Search, filter.DueFrom, filter.DueBefore,
		filter.UpdatedThrough, hasCursor, afterDue, afterUpdated, afterID,
	)
	if err != nil {
		return nil, fmt.Errorf("list actor document proposal work: %w", err)
	}
	defer rows.Close()
	values := make([]Task, 0, filter.Limit)
	for rows.Next() {
		task, scanErr := scanTask(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		task.DocumentProposalVisible = true
		values = append(values, task)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

var _ documentProposalActorWorkRepository = (*PostgresRepository)(nil)
