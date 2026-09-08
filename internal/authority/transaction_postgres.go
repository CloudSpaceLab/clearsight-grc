//go:build postgres

package authority

import (
	"context"
	"github.com/jackc/pgx/v5"
)

type transactionContextKey struct{}

// WithPostgresTransaction lets a server-owned material command resolve current
// authority on its existing transaction. Callers must keep this context within
// the transaction lifetime; request data must never select a transaction.
func WithPostgresTransaction(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, transactionContextKey{}, tx)
}

type postgresQueryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (s *postgresService) queryer(ctx context.Context) postgresQueryer {
	if tx, ok := ctx.Value(transactionContextKey{}).(pgx.Tx); ok && tx != nil {
		return tx
	}
	return s.pool
}
