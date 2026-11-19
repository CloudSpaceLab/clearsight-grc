//go:build postgres && postgresintegration

package frameworkcatalog

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPublicationTransactionRollsBackEditionAndClauses(t *testing.T) {
	connection := os.Getenv("TEST_DATABASE_URL")
	if connection == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connection)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	publication := validTestPublication()
	publication.FrameworkCode = "TRANSACTION-ROLLBACK-TEST"
	publication.Jurisdiction = "TEST-INTERNAL"
	digest, err := ValidatePublication(publication)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	edition, err := publishInTransaction(ctx, tx, publication, "test-only", digest)
	if err != nil {
		t.Fatal(err)
	}
	var clauses int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM framework_catalog_clauses WHERE edition_id=$1::uuid`, edition.ID).Scan(&clauses); err != nil || clauses != len(publication.Clauses) {
		t.Fatalf("clauses in transaction: %d, %v", clauses, err)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	var editions int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM framework_catalog_editions WHERE id=$1::uuid`, edition.ID).Scan(&editions); err != nil || editions != 0 {
		t.Fatalf("edition after rollback: %d, %v", editions, err)
	}
}
