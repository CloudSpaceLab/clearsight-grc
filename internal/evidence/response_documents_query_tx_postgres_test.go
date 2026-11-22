//go:build postgres

package evidence

import (
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"testing"
)

// Both execution contexts must use the same exact-occurrence query path.
func TestDocumentRowsQuerierAcceptsTransactionAndPool(t *testing.T) {
	var tx pgx.Tx
	var pool *pgxpool.Pool
	var transaction documentRowsQuerier = tx
	var ordinary documentRowsQuerier = pool
	_ = transaction
	_ = ordinary
}
