//go:build postgres

package governance

import (
	"context"
	"strings"
)

func (r *PostgresRepository) SearchDelegationCandidates(ctx context.Context, tenantID, legalEntityID, responsibility, query string, limit int) ([]DelegationCandidate, error) {
	rows, err := r.pool.Query(ctx, `
		WITH tenant_scope AS (
			SELECT id FROM tenants WHERE id::text=$1 OR slug=$1
		), candidate_values AS (
			SELECT p.id::text AS principal_id,p.display_name,COALESCE(position_context.title,'') AS context_label,
				EXISTS (
					SELECT 1 FROM responsibility_assignments ra
					WHERE ra.tenant_id=p.tenant_id AND ra.legal_entity_id=$2::uuid
					  AND ra.principal_id=p.id AND ra.responsibility=$3
					  AND ra.valid_from<=clock_timestamp() AND (ra.valid_until IS NULL OR clock_timestamp()<ra.valid_until)
					  AND ((ra.object_type='LEGAL_ENTITY' AND (ra.object_id IS NULL OR ra.object_id=$2::uuid)) OR (ra.object_type='*' AND ra.object_id IS NULL))
				) OR EXISTS (
					SELECT 1 FROM delegations d
					WHERE d.tenant_id=p.tenant_id AND d.legal_entity_id=$2::uuid AND d.to_principal_id=p.id
					  AND d.responsibility=$3 AND d.status='ACTIVE' AND d.starts_at<=clock_timestamp() AND clock_timestamp()<d.ends_at
					  AND COALESCE(d.scope->>'object_type','')='' AND COALESCE(d.scope->>'object_id','')=''
					  AND COALESCE(d.scope->>'decision_type','')='' AND NOT (d.scope ? 'min_materiality') AND NOT (d.scope ? 'max_materiality')
				) AS can_give,
				NOT EXISTS (
					SELECT 1 FROM segregation_rules sr
					JOIN role_templates rt ON rt.tenant_id=sr.tenant_id AND rt.code=sr.prohibited_role_code
					  AND rt.valid_from<=clock_timestamp() AND (rt.valid_until IS NULL OR clock_timestamp()<rt.valid_until)
					JOIN position_role_bindings prb ON prb.tenant_id=rt.tenant_id AND prb.role_template_id=rt.id
					  AND prb.valid_from<=clock_timestamp() AND (prb.valid_until IS NULL OR clock_timestamp()<prb.valid_until)
					JOIN org_positions blocked_position ON blocked_position.tenant_id=prb.tenant_id AND blocked_position.id=prb.position_id
					  AND blocked_position.legal_entity_id=$2::uuid AND blocked_position.occupant_principal_id=p.id
					  AND blocked_position.valid_from<=clock_timestamp() AND (blocked_position.valid_until IS NULL OR clock_timestamp()<blocked_position.valid_until)
					WHERE sr.tenant_id=p.tenant_id AND sr.status='ACTIVE' AND sr.responsibility=$3
				) AS can_receive
			FROM principals p
			JOIN tenant_scope ts ON ts.id=p.tenant_id
			JOIN LATERAL (
				SELECT op.title FROM org_positions op
				WHERE op.tenant_id=p.tenant_id AND op.legal_entity_id=$2::uuid AND op.occupant_principal_id=p.id
				  AND op.valid_from<=clock_timestamp() AND (op.valid_until IS NULL OR clock_timestamp()<op.valid_until)
				ORDER BY lower(op.title),op.id LIMIT 1
			) position_context ON true
			WHERE p.kind='PERSON' AND p.status='ACTIVE' AND p.valid_from<=clock_timestamp()
			  AND (p.valid_until IS NULL OR clock_timestamp()<p.valid_until)
		), filtered AS (
			SELECT * FROM candidate_values
			WHERE (can_give OR can_receive)
			  AND ($4='' OR strpos(lower(display_name),lower($4))>0 OR strpos(lower(context_label),lower($4))>0)
			ORDER BY lower(display_name),lower(context_label),principal_id LIMIT $5
		)
		SELECT principal_id,display_name,context_label,can_give,can_receive FROM filtered
		ORDER BY lower(display_name),lower(context_label),principal_id`, tenantID, legalEntityID, responsibility, strings.TrimSpace(query), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]DelegationCandidate, 0, limit)
	for rows.Next() {
		var value DelegationCandidate
		if err := rows.Scan(&value.PrincipalID, &value.DisplayName, &value.ContextLabel, &value.CanGive, &value.CanReceive); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

var _ delegationCandidateRepository = (*PostgresRepository)(nil)
