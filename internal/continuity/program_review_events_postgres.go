//go:build postgres

package continuity

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) ProgramEventsAfterVersion(ctx context.Context, tenant, programID string, afterVersion int64, limit int) ([]Event, bool, error) {
	if limit <= 0 {
		return []Event{}, false, nil
	}
	enforce, actorTenant, actorEntity := postgresActorScope(ctx)
	principal := ""
	if actor, ok := identity.FromContext(ctx); ok {
		principal = actor.PrincipalID
	}
	var raw []byte
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE((SELECT jsonb_agg(jsonb_build_object(
		 'id', e.id::text, 'tenant_id',t.slug, 'aggregate_type',e.aggregate_type,
		 'aggregate_id',e.aggregate_id::text, 'aggregate_version',e.aggregate_version,
		 'type',e.event_type,'payload',e.payload,'actor_type',e.actor_type,
		 'actor_id',COALESCE(e.actor_id::text,''),'occurred_at',e.occurred_at)
		 ORDER BY e.aggregate_version DESC) FROM (
		 SELECT ce.* FROM continuity_events ce WHERE ce.tenant_id=p.tenant_id AND ce.aggregate_type='PROGRAM'
		 AND ce.aggregate_id=p.id AND ce.aggregate_version>$3 ORDER BY ce.aggregate_version DESC LIMIT $4
		) e),'[]'::jsonb)
		FROM programs p JOIN tenants t ON t.id=p.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND p.id=$2::uuid
		AND (NOT $5 OR ((t.id::text=$6 OR t.slug=$6) AND p.legal_entity_id IS NOT NULL AND ($7='*' OR p.legal_entity_id=(SELECT le.id FROM legal_entities le WHERE le.tenant_id=p.tenant_id AND (le.id::text=$7 OR le.code=$7) AND le.valid_from<=clock_timestamp() AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until) ORDER BY le.valid_from DESC,le.id LIMIT 1))))
		AND (NOT $8::boolean OR (btrim($9::text)<>'' AND `+visibleMatterScopeSQL("p.scope", "btrim($9::text)")+`))`, tenant, programID, afterVersion, limit+1, enforce, actorTenant, actorEntity, !hasTrustedSystemScope(ctx), principal).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, ErrNotFound
	}
	if err != nil {
		return nil, false, err
	}
	var values []Event
	if err = json.Unmarshal(raw, &values); err != nil {
		return nil, false, err
	}
	if len(values) > limit {
		return values[:limit], true, nil
	}
	return values, false, nil
}
