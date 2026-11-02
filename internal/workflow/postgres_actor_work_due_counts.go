//go:build postgres

package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// CountActorWorkDue counts authorized workflow and document-proposal rows
// under one PostgreSQL MVCC snapshot. Both subqueries use the exact source
// selection as ListActorWorkPage; there is no page LIMIT or offset.
func (r *PostgresRepository) CountActorWorkDue(ctx context.Context, filter ListFilter, asOf time.Time) (ActorWorkDueCounts, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly,
	})
	if err != nil {
		return ActorWorkDueCounts{}, fmt.Errorf("start authorized work counts: %w", err)
	}
	defer tx.Rollback(ctx)

	var counts ActorWorkDueCounts
	soon := asOf.Add(7 * 24 * time.Hour)
	accumulate := func(sql string, args ...any) error {
		var total, overdue, dueSoon, undated int64
		if err := tx.QueryRow(ctx, sql, args...).Scan(&total, &overdue, &dueSoon, &undated); err != nil {
			return err
		}
		counts.Total += total
		counts.Overdue += overdue
		counts.DueSoon += dueSoon
		counts.Undated += undated
		return nil
	}
	// Placeholder $9 is used in the page LIMIT but not the shared source
	// selection. Keep it typed in the outer query to preserve binding parity.
	workflowSQL := `
		SELECT count(*)::bigint,
		       count(*) FILTER (WHERE due_at IS NOT NULL AND due_at < $18::timestamptz)::bigint,
		       count(*) FILTER (WHERE due_at >= $18::timestamptz AND due_at <= $19::timestamptz)::bigint,
		       count(*) FILTER (WHERE due_at IS NULL)::bigint
		FROM (` + postgresActorWorkRowsSQL() + `) authorized_work
		WHERE $9::integer > 0`
	if err := accumulate(workflowSQL,
		filter.TenantID, filter.PrincipalID, "", "", true, false, true,
		filter.LegalEntityID, 1, "", nil, nil, filter.UpdatedThrough,
		false, nil, nil, "", asOf, soon,
	); err != nil {
		return ActorWorkDueCounts{}, fmt.Errorf("count authorized workflow assignments: %w", err)
	}

	// The separately governed document-proposal path applies its own current
	// accepted handoff/step and legal-entity validation. It is disjoint from
	// generic actor-visible workflow kinds and remains subject to the same tx.
	proposalSQL := `
		SELECT count(*)::bigint,
		       count(*) FILTER (WHERE due_at IS NOT NULL AND due_at < $16::timestamptz)::bigint,
		       count(*) FILTER (WHERE due_at >= $16::timestamptz AND due_at <= $17::timestamptz)::bigint,
		       count(*) FILTER (WHERE due_at IS NULL)::bigint
		FROM (` + postgresDocumentProposalRowsSQL() + `) authorized_work
		WHERE $7::integer > 0`
	if err := accumulate(proposalSQL,
		filter.TenantID, filter.PrincipalID, filter.LegalEntityID, "", DocumentProposalWorkflowKind,
		true, 1, "", nil, nil, filter.UpdatedThrough,
		false, nil, nil, "", asOf, soon,
	); err != nil {
		return ActorWorkDueCounts{}, fmt.Errorf("count authorized document proposal assignments: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return ActorWorkDueCounts{}, fmt.Errorf("complete authorized work counts: %w", err)
	}
	return counts, nil
}
