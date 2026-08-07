//go:build postgres

package continuity

import "context"

func (r *PostgresRepository) ProgramIDsForEvidenceSource(ctx context.Context, tenant, sourceID string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		WITH dependent_programs AS (
			SELECT ecs.program_id
			FROM evidence_contract_sources ecs
			JOIN evidence_contracts ec
			  ON ec.tenant_id=ecs.tenant_id AND ec.program_id=ecs.program_id AND ec.id=ecs.contract_id
			WHERE ecs.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
			  AND ecs.source_id=$2::uuid
			  AND ec.status='ACTIVE'
			UNION
			SELECT pr.program_id
			FROM program_requirements pr
			WHERE pr.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
			  AND pr.source_id=$2::uuid
			  AND pr.status='APPROVED'
			  AND pr.effective_from<=clock_timestamp()
			  AND (pr.effective_until IS NULL OR pr.effective_until>clock_timestamp())
		)
		SELECT DISTINCT dp.program_id::text
		FROM dependent_programs dp
		JOIN programs p
		  ON p.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND p.id=dp.program_id
		WHERE p.status IN ('ACTIVE','PAUSED')
		ORDER BY dp.program_id::text`, tenant, sourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	values := make([]string, 0)
	for rows.Next() {
		var programID string
		if err := rows.Scan(&programID); err != nil {
			return nil, err
		}
		values = append(values, programID)
	}
	return values, rows.Err()
}

func (r *PostgresRepository) EvidenceSourcesCurrentForProgram(ctx context.Context, tenant, programID string) (bool, error) {
	var current bool
	err := r.pool.QueryRow(ctx, `
		WITH dependent_sources AS (
			SELECT ecs.source_id
			FROM evidence_contract_sources ecs
			JOIN evidence_contracts ec
			  ON ec.tenant_id=ecs.tenant_id AND ec.program_id=ecs.program_id AND ec.id=ecs.contract_id
			WHERE ecs.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
			  AND ecs.program_id=$2::uuid
			  AND ec.status='ACTIVE'
			UNION
			SELECT pr.source_id
			FROM program_requirements pr
			WHERE pr.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
			  AND pr.program_id=$2::uuid
			  AND pr.source_id IS NOT NULL
			  AND pr.status='APPROVED'
			  AND pr.effective_from<=clock_timestamp()
			  AND (pr.effective_until IS NULL OR pr.effective_until>clock_timestamp())
		)
		SELECT NOT EXISTS (
			SELECT 1
			FROM dependent_sources ds
			JOIN evidence_sources es
			  ON es.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND es.id=ds.source_id
			WHERE es.health<>'CURRENT'
		)`, tenant, programID).Scan(&current)
	return current, err
}
