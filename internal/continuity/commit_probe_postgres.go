//go:build postgres

package continuity

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) CurrentProgramVersion(ctx context.Context, tenant, id string) (int64, error) {
	var version int64
	err := r.pool.QueryRow(ctx, `SELECT p.version FROM programs p JOIN tenants t ON t.id=p.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND p.id=$2::uuid`, tenant, id).Scan(&version)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	return version, err
}

func (r *PostgresRepository) CurrentMatterVersion(ctx context.Context, tenant, id string) (int64, error) {
	var version int64
	err := r.pool.QueryRow(ctx, `SELECT m.version FROM matters m JOIN tenants t ON t.id=m.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND m.id=$2::uuid`, tenant, id).Scan(&version)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	return version, err
}
