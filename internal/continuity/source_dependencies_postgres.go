//go:build postgres

package continuity

import "context"

func (r *PostgresRepository) ProgramIDsForEvidenceSource(ctx context.Context, tenant, sourceID string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT ecs.program_id::text
		FROM evidence_contract_sources ecs
		JOIN evidence_contracts ec
		  ON ec.tenant_id=ecs.tenant_id AND ec.program_id=ecs.program_id AND ec.id=ecs.contract_id
		JOIN programs p
		  ON p.tenant_id=ecs.tenant_id AND p.id=ecs.program_id
		JOIN tenants t ON t.id=ecs.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND ecs.source_id=$2::uuid
		  AND ec.status='ACTIVE'
		  AND p.status IN ('ACTIVE','PAUSED')
		ORDER BY ecs.program_id::text`, tenant, sourceID)
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
		SELECT NOT EXISTS (
			SELECT 1
			FROM evidence_contract_sources ecs
			JOIN evidence_contracts ec
			  ON ec.tenant_id=ecs.tenant_id AND ec.program_id=ecs.program_id AND ec.id=ecs.contract_id
			JOIN evidence_sources es
			  ON es.tenant_id=ecs.tenant_id AND es.id=ecs.source_id
			JOIN tenants t ON t.id=ecs.tenant_id
			WHERE (t.id::text=$1 OR t.slug=$1)
			  AND ecs.program_id=$2::uuid
			  AND ec.status='ACTIVE'
			  AND es.health<>'CURRENT'
		)`, tenant, programID).Scan(&current)
	return current, err
}
