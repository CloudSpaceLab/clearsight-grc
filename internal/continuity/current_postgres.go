//go:build postgres

package continuity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CurrentPostgresRepository is the production current-state reader. It embeds
// the event-capable repository for writes and point-in-time audit, but ordinary
// current reads come from normalized tables and therefore do not grow with
// continuity-event history depth.
type CurrentPostgresRepository struct{ *PostgresRepository }

func NewCurrentPostgresRepository(pool *pgxpool.Pool) *CurrentPostgresRepository {
	return &CurrentPostgresRepository{PostgresRepository: NewPostgresRepository(pool)}
}

func (r *CurrentPostgresRepository) GetProgram(ctx context.Context, tenant, id string) (ProgramAggregate, error) {
	var raw []byte
	enforce, actorTenant, actorEntity := postgresActorScope(ctx)
	if err := r.pool.QueryRow(ctx, currentProgramSQL, tenant, id, enforce, actorTenant, actorEntity).Scan(&raw); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ProgramAggregate{}, ErrNotFound
		}
		return ProgramAggregate{}, fmt.Errorf("read current program: %w", err)
	}
	var aggregate ProgramAggregate
	if err := json.Unmarshal(raw, &aggregate); err != nil {
		return ProgramAggregate{}, fmt.Errorf("decode current program: %w", err)
	}
	return decorateProgram(aggregate), nil
}

func (r *CurrentPostgresRepository) ListPrograms(ctx context.Context, tenant string, limit int) ([]ProgramAggregate, error) {
	enforce, actorTenant, actorEntity := postgresActorScope(ctx)
	rows, err := r.pool.Query(ctx, `SELECT p.id::text FROM programs p JOIN tenants t ON t.id=p.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND (NOT $2 OR ((t.id::text=$3 OR t.slug=$3) AND p.legal_entity_id IS NOT NULL AND ($4='*' OR p.legal_entity_id=(SELECT le.id FROM legal_entities le WHERE le.tenant_id=p.tenant_id AND (le.id::text=$4 OR le.code=$4) AND le.valid_from<=clock_timestamp() AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until) ORDER BY le.valid_from DESC,le.id LIMIT 1)))) ORDER BY CASE p.status WHEN 'ACTIVE' THEN 0 WHEN 'PAUSED' THEN 1 WHEN 'DRAFT' THEN 2 ELSE 3 END,p.updated_at DESC,p.id LIMIT $5`, tenant, enforce, actorTenant, actorEntity, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]string, 0, limit)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	values := make([]ProgramAggregate, 0, len(ids))
	for _, id := range ids {
		value, err := r.GetProgram(ctx, tenant, id)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func (r *CurrentPostgresRepository) GetMatter(ctx context.Context, tenant, id string) (MatterAggregate, error) {
	var raw []byte
	enforce, actorTenant, actorEntity := postgresActorScope(ctx)
	if err := r.pool.QueryRow(ctx, currentMatterSQL, tenant, id, enforce, actorTenant, actorEntity).Scan(&raw); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MatterAggregate{}, ErrNotFound
		}
		return MatterAggregate{}, fmt.Errorf("read current matter: %w", err)
	}
	var aggregate MatterAggregate
	if err := json.Unmarshal(raw, &aggregate); err != nil {
		return MatterAggregate{}, fmt.Errorf("decode current matter: %w", err)
	}
	aggregate.Closure = assessClosure(aggregate)
	return decorateMatter(aggregate), nil
}

func (r *CurrentPostgresRepository) ListMatters(ctx context.Context, tenant, status string, limit int) ([]MatterAggregate, error) {
	actor, enforceVisibility := identity.FromContext(ctx)
	principalID := ""
	actorTenant := ""
	if enforceVisibility {
		principalID = actor.PrincipalID
		actorTenant = actor.TenantID
	}
	enforceEntity, actorEntityTenant, actorEntity := postgresActorScope(ctx)
	rows, err := r.pool.Query(ctx, `
		SELECT m.id::text
		FROM matters m
		JOIN tenants t ON t.id=m.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND ($2='' OR ($2='OPEN' AND m.status NOT IN ('CLOSED','CANCELLED')) OR m.status=$2)
		  AND (NOT $3 OR t.id::text=$5 OR t.slug=$5)
		  AND (NOT $7 OR ((t.id::text=$8 OR t.slug=$8) AND m.legal_entity_id IS NOT NULL AND ($9='*' OR m.legal_entity_id=(SELECT le.id FROM legal_entities le WHERE le.tenant_id=m.tenant_id AND (le.id::text=$9 OR le.code=$9) AND le.valid_from<=clock_timestamp() AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until) ORDER BY le.valid_from DESC,le.id LIMIT 1))))
		  AND (
			NOT $3 OR
			CASE
				WHEN NOT (m.scope ? 'access') THEN true
				WHEN jsonb_typeof(m.scope->'access')<>'string' THEN false
				WHEN upper(btrim(m.scope->>'access')) IN ('PUBLIC','INTERNAL') THEN true
				WHEN upper(btrim(m.scope->>'access'))='RESTRICTED' THEN
					CASE
						WHEN jsonb_typeof(m.scope->'allowed_principal_ids')<>'array' THEN false
						ELSE
							NOT EXISTS (
								SELECT 1 FROM jsonb_array_elements(m.scope->'allowed_principal_ids') entry(value)
								WHERE jsonb_typeof(entry.value)<>'string'
							)
							AND EXISTS (
								SELECT 1 FROM jsonb_array_elements_text(m.scope->'allowed_principal_ids') nonblank(value)
								WHERE btrim(nonblank.value)<>''
							)
							AND EXISTS (
								SELECT 1 FROM jsonb_array_elements_text(m.scope->'allowed_principal_ids') allowed(value)
								WHERE btrim(allowed.value)=$4
							)
					END
				ELSE false
			END
		  )
		ORDER BY m.priority DESC,m.due_at NULLS LAST,m.updated_at DESC,m.id
		LIMIT $6`, tenant, status, enforceVisibility, principalID, actorTenant, limit, enforceEntity, actorEntityTenant, actorEntity)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]string, 0, limit)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	values := make([]MatterAggregate, 0, len(ids))
	for _, id := range ids {
		value, err := r.GetMatter(ctx, tenant, id)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

const currentProgramSQL = `
SELECT jsonb_build_object(
  'program', (to_jsonb(p)-'tenant_id'-'program_type') || jsonb_build_object('tenant_id',t.id::text,'type',p.program_type),
  'requirements', COALESCE((SELECT jsonb_agg((to_jsonb(v)-'tenant_id'-'object_name') || jsonb_build_object('tenant_id',t.id::text,'object',v.object_name) ORDER BY v.created_at,v.id) FROM program_requirements v WHERE v.tenant_id=p.tenant_id AND v.program_id=p.id),'[]'::jsonb),
  'applicability', COALESCE((SELECT jsonb_agg((to_jsonb(v)-'tenant_id') || jsonb_build_object('tenant_id',t.id::text) ORDER BY v.created_at,v.id) FROM program_applicability v WHERE v.tenant_id=p.tenant_id AND v.program_id=p.id),'[]'::jsonb),
  'control_objectives', COALESCE((SELECT jsonb_agg((to_jsonb(v)-'tenant_id') || jsonb_build_object('tenant_id',t.id::text) ORDER BY v.created_at,v.id) FROM control_objectives v WHERE v.tenant_id=p.tenant_id AND v.program_id=p.id),'[]'::jsonb),
  'control_implementations', COALESCE((SELECT jsonb_agg((to_jsonb(v)-'tenant_id') || jsonb_build_object('tenant_id',t.id::text) ORDER BY v.created_at,v.id) FROM control_implementations v WHERE v.tenant_id=p.tenant_id AND v.program_id=p.id),'[]'::jsonb),
  'requirement_control_links', COALESCE((SELECT jsonb_agg((to_jsonb(v)-'tenant_id') || jsonb_build_object('tenant_id',t.id::text) ORDER BY v.created_at,v.id) FROM requirement_control_links v WHERE v.tenant_id=p.tenant_id AND v.program_id=p.id),'[]'::jsonb),
  'evidence_contracts', COALESCE((SELECT jsonb_agg((to_jsonb(v)-'tenant_id') || jsonb_build_object('tenant_id',t.id::text) ORDER BY v.created_at,v.id) FROM evidence_contracts v WHERE v.tenant_id=p.tenant_id AND v.program_id=p.id),'[]'::jsonb),
  'evidence_assessments', COALESCE((SELECT jsonb_agg((to_jsonb(v)-'tenant_id') || jsonb_build_object('tenant_id',t.id::text) ORDER BY v.assessed_at,v.id) FROM evidence_assessments v WHERE v.tenant_id=p.tenant_id AND v.program_id=p.id),'[]'::jsonb),
  'triggers', COALESCE((SELECT jsonb_agg((to_jsonb(v)-'tenant_id'-'trigger_type'-'created_at') || jsonb_build_object('tenant_id',t.id::text,'type',v.trigger_type) ORDER BY v.observed_at,v.id) FROM program_trigger_events v WHERE v.tenant_id=p.tenant_id AND v.program_id=p.id),'[]'::jsonb),
  'current_state', (SELECT (to_jsonb(v)-'tenant_id'-'overall_state') || jsonb_build_object('tenant_id',t.id::text,'overall',v.overall_state) FROM program_state_snapshots v WHERE v.tenant_id=p.tenant_id AND v.program_id=p.id ORDER BY v.generated_at DESC,v.projection_version DESC LIMIT 1)
)
FROM programs p
JOIN tenants t ON t.id=p.tenant_id
WHERE (t.id::text=$1 OR t.slug=$1) AND p.id=$2::uuid
  AND (NOT $3 OR ((t.id::text=$4 OR t.slug=$4) AND p.legal_entity_id IS NOT NULL AND ($5='*' OR p.legal_entity_id=(SELECT le.id FROM legal_entities le WHERE le.tenant_id=p.tenant_id AND (le.id::text=$5 OR le.code=$5) AND le.valid_from<=clock_timestamp() AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until) ORDER BY le.valid_from DESC,le.id LIMIT 1))))`

const currentMatterSQL = `
SELECT jsonb_build_object(
  'matter', (to_jsonb(m)-'tenant_id'-'matter_type') || jsonb_build_object('tenant_id',t.id::text,'type',m.matter_type),
  'links', COALESCE((SELECT jsonb_agg((to_jsonb(v)-'tenant_id') || jsonb_build_object('tenant_id',t.id::text) ORDER BY v.created_at,v.id) FROM matter_links v WHERE v.tenant_id=m.tenant_id AND v.matter_id=m.id),'[]'::jsonb),
  'decisions', COALESCE((SELECT jsonb_agg((to_jsonb(v)-'tenant_id'-'decision_type'-'matter_version') || jsonb_build_object('tenant_id',t.id::text,'type',v.decision_type) ORDER BY v.matter_version,v.id) FROM matter_decisions v WHERE v.tenant_id=m.tenant_id AND v.matter_id=m.id),'[]'::jsonb),
  'actions', COALESCE((SELECT jsonb_agg((to_jsonb(v)-'tenant_id') || jsonb_build_object('tenant_id',t.id::text) ORDER BY v.created_at,v.id) FROM matter_actions v WHERE v.tenant_id=m.tenant_id AND v.matter_id=m.id),'[]'::jsonb),
  'verification_contracts', COALESCE((SELECT jsonb_agg((to_jsonb(v)-'tenant_id') || jsonb_build_object('tenant_id',t.id::text) ORDER BY v.created_at,v.id) FROM verification_contracts v WHERE v.tenant_id=m.tenant_id AND v.matter_id=m.id),'[]'::jsonb),
  'verification_results', COALESCE((SELECT jsonb_agg((to_jsonb(v)-'tenant_id') || jsonb_build_object('tenant_id',t.id::text) ORDER BY v.observed_at,v.id) FROM verification_results v WHERE v.tenant_id=m.tenant_id AND v.matter_id=m.id),'[]'::jsonb),
  'response_packages', COALESCE((SELECT jsonb_agg((to_jsonb(v)-'tenant_id'-'matter_version') || jsonb_build_object('tenant_id',t.id::text) ORDER BY v.matter_version,v.id) FROM response_packages v WHERE v.tenant_id=m.tenant_id AND v.matter_id=m.id),'[]'::jsonb)
)
FROM matters m
JOIN tenants t ON t.id=m.tenant_id
WHERE (t.id::text=$1 OR t.slug=$1) AND m.id=$2::uuid
  AND (NOT $3 OR ((t.id::text=$4 OR t.slug=$4) AND m.legal_entity_id IS NOT NULL AND ($5='*' OR m.legal_entity_id=(SELECT le.id FROM legal_entities le WHERE le.tenant_id=m.tenant_id AND (le.id::text=$5 OR le.code=$5) AND le.valid_from<=clock_timestamp() AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until) ORDER BY le.valid_from DESC,le.id LIMIT 1))))`

var _ Repository = (*CurrentPostgresRepository)(nil)
