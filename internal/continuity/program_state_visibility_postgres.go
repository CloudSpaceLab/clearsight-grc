//go:build postgres

package continuity

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// visibleMatterScopeSQL is shared by current reads and checkpoint writes.
// The arguments are SQL expressions supplied only by repository code.
func visibleMatterScopeSQL(scope, principal string) string {
	return `CASE
 WHEN jsonb_typeof(` + scope + `) IS DISTINCT FROM 'object' THEN false
 WHEN NOT (` + scope + ` ? 'access') THEN true
 WHEN jsonb_typeof(` + scope + `->'access')<>'string' THEN false
 WHEN upper(btrim(` + scope + `->>'access')) IN ('PUBLIC','INTERNAL') THEN true
 WHEN upper(btrim(` + scope + `->>'access'))='RESTRICTED' THEN CASE
   WHEN jsonb_typeof(` + scope + `->'allowed_principal_ids')<>'array' THEN false
   ELSE NOT EXISTS (SELECT 1 FROM jsonb_array_elements(` + scope + `->'allowed_principal_ids') entry(value) WHERE jsonb_typeof(entry.value)<>'string')
    AND EXISTS (SELECT 1 FROM jsonb_array_elements_text(` + scope + `->'allowed_principal_ids') nonblank(value) WHERE btrim(nonblank.value)<>'')
    AND EXISTS (SELECT 1 FROM jsonb_array_elements_text(` + scope + `->'allowed_principal_ids') allowed(value) WHERE btrim(allowed.value)=` + principal + `)
 END ELSE false END`
}

const visibleMatterNotArchivedSQL = `NOT EXISTS (SELECT 1 FROM demo_record_archives archive WHERE archive.tenant_id=m.tenant_id AND archive.legal_entity_id=m.legal_entity_id AND archive.record_type='MATTER' AND archive.record_id=m.id AND archive.restored_at IS NULL)`

// program and principal are repository-owned SQL expressions, never input text.
func visibleProgramMatterCountSQL(program, principal string) string {
	return `(SELECT count(DISTINCT m.id) FROM matter_links ml JOIN matters m ON m.tenant_id=ml.tenant_id AND m.id=ml.matter_id
		WHERE ml.tenant_id=` + program + `.tenant_id AND ml.program_id=` + program + `.id AND ml.retired_at IS NULL AND m.legal_entity_id=` + program + `.legal_entity_id AND m.status NOT IN ('CLOSED','CANCELLED') AND ` + visibleMatterNotArchivedSQL + ` AND ` + visibleMatterScopeSQL("m.scope", principal) + `)`
}

func (r *PostgresRepository) VisibleProgramMatterVisibility(ctx context.Context, tenant string, programIDs []string, principalID string, at *time.Time) (map[string]programMatterVisibility, error) {
	visibility := make(map[string]programMatterVisibility, len(programIDs))
	if len(programIDs) == 0 {
		return visibility, nil
	}

	var (
		rows pgx.Rows
		err  error
	)
	if at == nil {
		rows, err = r.pool.Query(ctx, `
			WITH target_programs AS (
				SELECT unnest($2::uuid[]) AS program_id
			)
			SELECT ml.program_id::text, count(DISTINCT m.id)::int, max(m.updated_at)
			FROM matter_links ml
			JOIN target_programs target ON target.program_id=ml.program_id
			JOIN matters m ON m.tenant_id=ml.tenant_id AND m.id=ml.matter_id
			JOIN programs p ON p.tenant_id=ml.tenant_id AND p.id=ml.program_id AND p.legal_entity_id=m.legal_entity_id
			WHERE ml.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
			  AND ml.retired_at IS NULL
			  AND `+visibleMatterNotArchivedSQL+`
			  AND m.status NOT IN ('CLOSED','CANCELLED')
			  AND `+visibleMatterScopeSQL("m.scope", "$3")+`
			GROUP BY ml.program_id`, tenant, programIDs, principalID)
	} else {
		rows, err = r.pool.Query(ctx, `
			WITH target_programs AS (
				SELECT unnest($2::uuid[]) AS program_id
			), historical_matter AS (
				SELECT DISTINCT ON (ce.aggregate_id)
					ce.aggregate_id AS matter_id,
					ce.payload AS matter
				FROM continuity_events ce
				WHERE ce.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
				  AND ce.aggregate_type='MATTER'
				  AND ce.event_type IN ('MATTER_CREATED','MATTER_STATE_CHANGED')
				  AND ce.occurred_at<=$4
				ORDER BY ce.aggregate_id,ce.aggregate_version DESC
			), historical_activity AS (
				SELECT ce.aggregate_id AS matter_id, max(ce.occurred_at) AS latest_activity_at
				FROM continuity_events ce
				WHERE ce.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
				  AND ce.aggregate_type='MATTER'
				  AND ce.occurred_at<=$4
				GROUP BY ce.aggregate_id
			)
			SELECT ml.program_id::text, count(DISTINCT ml.matter_id)::int, max(activity.latest_activity_at)
			FROM matter_links ml
			JOIN target_programs target ON target.program_id=ml.program_id
			JOIN historical_matter historical ON historical.matter_id=ml.matter_id
			JOIN historical_activity activity ON activity.matter_id=ml.matter_id
			WHERE ml.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
			  AND ml.created_at<=$4
			  AND COALESCE(historical.matter->>'status','') NOT IN ('CLOSED','CANCELLED')
			  AND CASE
					WHEN NOT ((historical.matter->'scope') ? 'access') THEN true
					WHEN jsonb_typeof((historical.matter->'scope')->'access')<>'string' THEN false
					WHEN upper(btrim((historical.matter->'scope')->>'access')) IN ('PUBLIC','INTERNAL') THEN true
					WHEN upper(btrim((historical.matter->'scope')->>'access'))='RESTRICTED' THEN
						CASE
							WHEN jsonb_typeof((historical.matter->'scope')->'allowed_principal_ids')<>'array' THEN false
							ELSE
								NOT EXISTS (
									SELECT 1
									FROM jsonb_array_elements((historical.matter->'scope')->'allowed_principal_ids') entry(value)
									WHERE jsonb_typeof(entry.value)<>'string'
								)
								AND EXISTS (
									SELECT 1
									FROM jsonb_array_elements_text((historical.matter->'scope')->'allowed_principal_ids') nonblank(value)
									WHERE btrim(nonblank.value)<>''
								)
								AND EXISTS (
									SELECT 1
									FROM jsonb_array_elements_text((historical.matter->'scope')->'allowed_principal_ids') allowed(value)
									WHERE btrim(allowed.value)=$3
								)
						END
					ELSE false
				END
			GROUP BY ml.program_id`, tenant, programIDs, principalID, at.UTC())
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var programID string
		var value programMatterVisibility
		if err := rows.Scan(&programID, &value.OpenCount, &value.LatestAt); err != nil {
			return nil, err
		}
		value.LatestAt = value.LatestAt.UTC()
		visibility[programID] = value
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return visibility, nil
}

var _ programMatterVisibilityRepository = (*PostgresRepository)(nil)
