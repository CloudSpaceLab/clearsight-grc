//go:build postgres

package evidence

import "context"

func (r *PostgresRepository) ListVisibleRequests(ctx context.Context, tenant, principal string, limit int) ([]Request, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT er.id::text,t.slug,er.subject_type,er.subject_id,er.title,er.purpose,er.why_you,er.sensitivity,er.audience_type,er.estimated_minutes,er.deadline,er.known_facts,er.fields,er.status,COALESCE(er.created_by::text,''),er.version,er.created_at,er.updated_at
		FROM capture_requests er
		JOIN tenants t ON t.id=er.tenant_id
		LEFT JOIN matters m ON er.subject_type='MATTER' AND m.tenant_id=er.tenant_id AND m.id::text=er.subject_id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND er.recipient_type='INTERNAL_PRINCIPAL'
		  AND er.recipient_principal_id=$2::uuid
		  AND (
			er.subject_type<>'MATTER' OR (
				m.id IS NOT NULL AND (
					NOT (m.scope ? 'access') OR
					upper(COALESCE(m.scope->>'access','')) IN ('PUBLIC','INTERNAL') OR
					(
						upper(COALESCE(m.scope->>'access',''))='RESTRICTED' AND
						jsonb_typeof(m.scope->'allowed_principal_ids')='array' AND
						EXISTS (
							SELECT 1
							FROM jsonb_array_elements_text(m.scope->'allowed_principal_ids') allowed(principal_id)
							WHERE allowed.principal_id=$2
						)
					)
				)
			)
		  )
		ORDER BY CASE er.status WHEN 'READY' THEN 0 WHEN 'IN_PROGRESS' THEN 1 ELSE 2 END,er.deadline,er.id
		LIMIT $3`, tenant, principal, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := []Request{}
	for rows.Next() {
		value, scanErr := scanRequest(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		value.Recipient = Recipient{Type: RecipientInternalPrincipal, PrincipalID: principal}
		values = append(values, value)
	}
	return values, rows.Err()
}

var _ visibleRequestRepository = (*PostgresRepository)(nil)
