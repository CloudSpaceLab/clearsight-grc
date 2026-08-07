//go:build postgres

package continuity

import "context"

func (r *PostgresRepository) CurrentProgramSourceState(ctx context.Context, tenant, programID string) (ProgramSourceState, error) {
	var required, current int
	err := r.pool.QueryRow(ctx, `
		WITH dependent_sources AS (
			SELECT ecs.source_id
			FROM evidence_contract_sources ecs
			JOIN evidence_contracts ec
			  ON ec.tenant_id=ecs.tenant_id AND ec.program_id=ecs.program_id AND ec.id=ecs.contract_id
			LEFT JOIN program_requirements target_requirement
			  ON target_requirement.tenant_id=ec.tenant_id AND target_requirement.program_id=ec.program_id AND target_requirement.id=ec.requirement_id
			LEFT JOIN control_implementations target_implementation
			  ON target_implementation.tenant_id=ec.tenant_id AND target_implementation.program_id=ec.program_id AND target_implementation.id=ec.control_implementation_id
			WHERE ecs.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
			  AND ecs.program_id=$2::uuid
			  AND ec.status='ACTIVE'
			  AND (
				(ec.requirement_id IS NOT NULL
				 AND target_requirement.status='APPROVED'
				 AND target_requirement.effective_from<=clock_timestamp()
				 AND (target_requirement.effective_until IS NULL OR clock_timestamp()<target_requirement.effective_until))
				OR
				(ec.control_implementation_id IS NOT NULL
				 AND target_implementation.status NOT IN ('INACTIVE','RETIRED')
				 AND target_implementation.effective_from<=clock_timestamp()
				 AND (target_implementation.effective_until IS NULL OR clock_timestamp()<target_implementation.effective_until))
			  )
			UNION
			SELECT pr.source_id
			FROM program_requirements pr
			WHERE pr.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
			  AND pr.program_id=$2::uuid
			  AND pr.source_id IS NOT NULL
			  AND pr.status='APPROVED'
			  AND pr.effective_from<=clock_timestamp()
			  AND (pr.effective_until IS NULL OR clock_timestamp()<pr.effective_until)
		)
		SELECT count(*)::integer,
		       count(*) FILTER (WHERE es.status='ACTIVE' AND es.health='CURRENT')::integer
		FROM dependent_sources ds
		LEFT JOIN evidence_sources es
		  ON es.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND es.id=ds.source_id`, tenant, programID).Scan(&required, &current)
	if err != nil {
		return ProgramSourceState{}, err
	}
	return ProgramSourceState{Required: required, Current: required > 0 && current == required, Known: true}, nil
}
