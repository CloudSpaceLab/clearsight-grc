//go:build postgres

package continuity

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) MatterAggregateByTriggerKey(ctx context.Context, tenant, triggerKey string) (MatterAggregate, error) {
	var id string
	enforce, actorTenant, actorEntity := postgresActorScope(ctx)
	err := r.pool.QueryRow(ctx, `SELECT m.id::text FROM matters m JOIN tenants t ON t.id=m.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND m.trigger_key=$2 AND (NOT $3 OR ((t.id::text=$4 OR t.slug=$4) AND m.legal_entity_id IS NOT NULL AND ($5='*' OR m.legal_entity_id=(SELECT le.id FROM legal_entities le WHERE le.tenant_id=m.tenant_id AND (le.id::text=$5 OR le.code=$5) AND le.valid_from<=clock_timestamp() AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until) ORDER BY le.valid_from DESC,le.id LIMIT 1)))) ORDER BY m.updated_at DESC,m.id DESC LIMIT 1`, tenant, triggerKey, enforce, actorTenant, actorEntity).Scan(&id)
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
	enforce, actorTenant, actorEntity := postgresActorScope(ctx)
	err := r.pool.QueryRow(ctx, `SELECT p.id::text FROM programs p JOIN tenants t ON t.id=p.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND p.code=$2 AND (NOT $3 OR ((t.id::text=$4 OR t.slug=$4) AND p.legal_entity_id IS NOT NULL AND ($5='*' OR p.legal_entity_id=(SELECT le.id FROM legal_entities le WHERE le.tenant_id=p.tenant_id AND (le.id::text=$5 OR le.code=$5) AND le.valid_from<=clock_timestamp() AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until) ORDER BY le.valid_from DESC,le.id LIMIT 1)))) ORDER BY CASE WHEN p.status='RETIRED' THEN 1 ELSE 0 END,p.updated_at DESC,p.id LIMIT 1`, tenant, code, enforce, actorTenant, actorEntity).Scan(&id)
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
