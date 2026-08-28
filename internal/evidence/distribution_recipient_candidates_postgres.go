//go:build postgres

package evidence

import (
	"context"
	"strings"
)

func (r *PostgresRepository) SearchDistributionRecipientCandidates(ctx context.Context, tenantID, legalEntityID, search string, limit int) ([]RecipientCandidate, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.id::text,p.display_name,COALESCE(recipient_context.label,'')
		FROM principals p
		JOIN tenants t ON t.id=p.tenant_id
		LEFT JOIN LATERAL (
			SELECT available_context.label
			FROM (
				SELECT position.title AS label,0 AS priority
				FROM org_positions position
				WHERE position.tenant_id=p.tenant_id
				  AND position.occupant_principal_id=p.id
				  AND (position.legal_entity_id IS NULL OR position.legal_entity_id=$2::uuid)
				  AND position.valid_from<=clock_timestamp()
				  AND (position.valid_until IS NULL OR clock_timestamp()<position.valid_until)
				UNION ALL
				SELECT role.name AS label,1 AS priority
				FROM scim_users directory_user
				JOIN scim_sources source ON source.tenant_id=directory_user.tenant_id AND source.id=directory_user.source_id AND source.status='ACTIVE'
				JOIN directory_group_members membership ON membership.tenant_id=directory_user.tenant_id AND membership.scim_user_id=directory_user.id
				JOIN directory_groups directory_group ON directory_group.tenant_id=membership.tenant_id AND directory_group.id=membership.group_id AND directory_group.source_id=directory_user.source_id AND directory_group.deleted_at IS NULL
				JOIN directory_group_role_bindings binding ON binding.tenant_id=directory_group.tenant_id AND binding.group_id=directory_group.id
				JOIN role_templates role ON role.tenant_id=binding.tenant_id AND role.id=binding.role_template_id
				WHERE directory_user.tenant_id=p.tenant_id
				  AND directory_user.principal_id=p.id
				  AND directory_user.active
				  AND directory_user.deleted_at IS NULL
				  AND binding.legal_entity_id=$2::uuid
				  AND binding.valid_from<=clock_timestamp()
				  AND (binding.valid_until IS NULL OR clock_timestamp()<binding.valid_until)
				  AND role.valid_from<=clock_timestamp()
				  AND (role.valid_until IS NULL OR clock_timestamp()<role.valid_until)
			) available_context
			ORDER BY available_context.priority,lower(available_context.label),available_context.label
			LIMIT 1
		) recipient_context ON true
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND p.kind='PERSON'
		  AND p.status='ACTIVE'
		  AND p.valid_from<=clock_timestamp()
		  AND (p.valid_until IS NULL OR clock_timestamp()<p.valid_until)
		  AND (
			EXISTS (
				SELECT 1 FROM org_positions position
				WHERE position.tenant_id=p.tenant_id
				  AND position.occupant_principal_id=p.id
				  AND (position.legal_entity_id IS NULL OR position.legal_entity_id=$2::uuid)
				  AND position.valid_from<=clock_timestamp()
				  AND (position.valid_until IS NULL OR clock_timestamp()<position.valid_until)
			)
			OR EXISTS (
				SELECT 1
				FROM scim_users directory_user
				JOIN scim_sources source ON source.tenant_id=directory_user.tenant_id AND source.id=directory_user.source_id AND source.status='ACTIVE'
				JOIN directory_group_members membership ON membership.tenant_id=directory_user.tenant_id AND membership.scim_user_id=directory_user.id
				JOIN directory_groups directory_group ON directory_group.tenant_id=membership.tenant_id AND directory_group.id=membership.group_id AND directory_group.source_id=directory_user.source_id AND directory_group.deleted_at IS NULL
				JOIN directory_group_role_bindings binding ON binding.tenant_id=directory_group.tenant_id AND binding.group_id=directory_group.id
				JOIN role_templates role ON role.tenant_id=binding.tenant_id AND role.id=binding.role_template_id
				WHERE directory_user.tenant_id=p.tenant_id
				  AND directory_user.principal_id=p.id
				  AND directory_user.active
				  AND directory_user.deleted_at IS NULL
				  AND binding.legal_entity_id=$2::uuid
				  AND binding.valid_from<=clock_timestamp()
				  AND (binding.valid_until IS NULL OR clock_timestamp()<binding.valid_until)
				  AND role.valid_from<=clock_timestamp()
				  AND (role.valid_until IS NULL OR clock_timestamp()<role.valid_until)
			)
		  )
		  AND ($3='' OR strpos(lower(p.display_name),lower($3))>0 OR strpos(lower(COALESCE(recipient_context.label,'')),lower($3))>0)
		ORDER BY lower(p.display_name),lower(COALESCE(recipient_context.label,'')),p.id
		LIMIT $4`, tenantID, legalEntityID, strings.TrimSpace(search), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]RecipientCandidate, 0, limit)
	for rows.Next() {
		var value RecipientCandidate
		if err := rows.Scan(&value.PrincipalID, &value.DisplayName, &value.ContextLabel); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

var _ distributionRecipientCandidateDirectory = (*PostgresRepository)(nil)
