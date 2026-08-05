//go:build postgres

package continuity

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) MatterAggregateByTriggerKey(ctx context.Context, tenant, triggerKey string) (MatterAggregate, error) {
	var id string
	err := r.pool.QueryRow(ctx, `SELECT m.id::text FROM matters m JOIN tenants t ON t.id=m.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND m.trigger_key=$2 ORDER BY m.updated_at DESC,m.id DESC LIMIT 1`, tenant, triggerKey).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return MatterAggregate{}, ErrNotFound
	}
	if err != nil {
		return MatterAggregate{}, err
	}
	return r.GetMatter(ctx, tenant, id)
}

func (r *PostgresRepository) ProgramByCode(ctx context.Context, tenant, code string) (ProgramAggregate, error) {
	var id string
	err := r.pool.QueryRow(ctx, `SELECT p.id::text FROM programs p JOIN tenants t ON t.id=p.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND p.code=$2 ORDER BY CASE WHEN p.status='RETIRED' THEN 1 ELSE 0 END,p.updated_at DESC LIMIT 1`, tenant, code).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProgramAggregate{}, ErrNotFound
	}
	if err != nil {
		return ProgramAggregate{}, err
	}
	return r.GetProgram(ctx, tenant, id)
}

var _ programCodeRepository = (*PostgresRepository)(nil)
var _ matterTriggerLookupRepository = (*PostgresRepository)(nil)
