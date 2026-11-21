//go:build postgres

package sourceaccess

import (
	"context"
	"github.com/jackc/pgx/v5"
)

type catalogProofTransactionKey struct{}
type catalogProofTransaction struct{ tx pgx.Tx }

// WithCatalogProofTransaction marks a server-owned material transaction for
// bounded catalog proof; callers must not accept a transaction from clients.
func WithCatalogProofTransaction(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, catalogProofTransactionKey{}, catalogProofTransaction{tx: tx})
}

func CatalogProofTransaction(ctx context.Context) (pgx.Tx, bool) {
	value, ok := ctx.Value(catalogProofTransactionKey{}).(catalogProofTransaction)
	return value.tx, ok
}
