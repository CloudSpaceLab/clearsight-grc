//go:build postgres

package continuity

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) VisibleOpenMatterCounts(ctx context.Context, tenant string, programIDs []string, principalID string, at *time.Time) (map[string]int, error) {
	counts := make(map[string]int, len(programIDs))
	if len(programIDs) == 0 {
		return counts, nil
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
			SELECT ml.program_id::text, count(DISTINCT m.id)
			FROM matter_links ml
			JOIN target_programs target ON target.program_id=ml.program_id
			JOIN matters m ON m.tenant_id=ml.tenant_id AND m.id=ml.matter_id
			WHERE ml.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
			  AND m.status NOT IN ('CLOSED','CANCELLED')
			  AND CASE
					WHEN NOT (m.scope ? 'access') THEN true
					WHEN jsonb_typeof(m.scope->'access')<>'string' THEN false
					WHEN upper(btrim(m.scope->>'access')) IN ('PUBLIC','INTERNAL') THEN true
					WHEN upper(btrim(m.scope->>'access'))='RESTRICTED' THEN
						CASE
							WHEN jsonb_typeof(m.scope->'allowed_principal_ids')<>'array' THEN false
							ELSE
								NOT EXISTS (
									SELECT 1
									FROM jsonb_array_elements(m.scope->'allowed_principal_ids') entry(value)
									WHERE jsonb_typeof(entry.value)<>'string'
								)
								AND EXISTS (
									SELECT 1
									FROM jsonb_array_elements_text(m.scope->'allowed_principal_ids') nonblank(value)
									WHERE btrim(nonblank.value)<>''
								)
								AND EXISTS (
									SELECT 1
									FROM jsonb_array_elements_text(m.scope->'allowed_principal_ids') allowed(value)
									WHERE btrim(allowed.value)=$3
								)
						END
					ELSE false
				END
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
			)
			SELECT ml.program_id::text, count(DISTINCT ml.matter_id)
			FROM matter_links ml
			JOIN target_programs target ON target.program_id=ml.program_id
			JOIN historical_matter historical ON historical.matter_id=ml.matter_id
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
		var count int
		if err := rows.Scan(&programID, &count); err != nil {
			return nil, err
		}
		counts[programID] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return counts, nil
}

var _ programMatterVisibilityRepository = (*PostgresRepository)(nil)
