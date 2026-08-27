//go:build postgres && postgresintegration

package continuity

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func seedPostgresTestLegalEntity(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, code string) string {
	t.Helper()
	var entityID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO legal_entities(tenant_id,code,name,jurisdiction)
		VALUES($1::uuid,$2,$2,'NG')
		RETURNING id::text`, tenantID, code).Scan(&entityID); err != nil {
		t.Fatal(err)
	}
	return entityID
}
