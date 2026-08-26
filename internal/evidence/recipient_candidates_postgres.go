//go:build postgres

package evidence

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) ListRecipientCandidates(ctx context.Context, tenant, legalEntityID, requestID, actorPrincipalID string, limit int) ([]RecipientCandidate, error) {
	return r.SearchRecipientCandidates(ctx, tenant, legalEntityID, requestID, actorPrincipalID, "", boundedRecipientCandidateLimit(limit))
}

func (r *PostgresRepository) SearchRecipientCandidates(ctx context.Context, tenant, legalEntityID, requestID, actorPrincipalID, search string, limit int) ([]RecipientCandidate, error) {
	rows, err := r.pool.Query(ctx, `
		WITH request_scope AS (
			SELECT er.tenant_id,er.legal_entity_id,er.subject_type,er.subject_id
			FROM capture_requests er
			JOIN tenants t ON t.id=er.tenant_id
			WHERE (t.id::text=$1 OR t.slug=$1)
			  AND er.legal_entity_id=$2::uuid
			  AND er.id=$3::uuid
			  AND er.created_by=$4::uuid
			  AND er.audience_type='INTERNAL'
			  AND er.status IN ('READY','IN_PROGRESS')
			  AND clock_timestamp()<er.deadline
			  AND EXISTS (
				SELECT 1 FROM principals requester
				WHERE requester.tenant_id=er.tenant_id
				  AND requester.id=er.created_by
				  AND `+internalRecipientEligibilityPredicate("requester", "er.tenant_id", "er.legal_entity_id", "er.subject_type", "er.subject_id")+`
			  )
		), candidate_values AS (
		SELECT p.id::text AS principal_id,p.display_name,COALESCE(recipient_context.label,'') AS context_label
		FROM request_scope rs
		JOIN principals p ON p.tenant_id=rs.tenant_id
		LEFT JOIN LATERAL (
			SELECT available_context.label
			FROM (
				SELECT eligible_position.title AS label,0 AS priority
				FROM org_positions eligible_position
				WHERE eligible_position.tenant_id=rs.tenant_id
				  AND eligible_position.occupant_principal_id=p.id
				  AND (eligible_position.legal_entity_id IS NULL OR eligible_position.legal_entity_id=rs.legal_entity_id)
				  AND eligible_position.valid_from<=clock_timestamp()
				  AND (eligible_position.valid_until IS NULL OR clock_timestamp()<eligible_position.valid_until)
				UNION ALL
				SELECT eligible_role.name AS label,1 AS priority
				FROM scim_users eligible_user
				JOIN scim_sources eligible_source ON eligible_source.tenant_id=eligible_user.tenant_id AND eligible_source.id=eligible_user.source_id AND eligible_source.status='ACTIVE'
				JOIN directory_group_members eligible_membership ON eligible_membership.tenant_id=eligible_user.tenant_id AND eligible_membership.scim_user_id=eligible_user.id
				JOIN directory_groups eligible_group ON eligible_group.tenant_id=eligible_membership.tenant_id AND eligible_group.id=eligible_membership.group_id AND eligible_group.source_id=eligible_user.source_id AND eligible_group.deleted_at IS NULL
				JOIN directory_group_role_bindings eligible_binding ON eligible_binding.tenant_id=eligible_group.tenant_id AND eligible_binding.group_id=eligible_group.id
				JOIN role_templates eligible_role ON eligible_role.tenant_id=eligible_binding.tenant_id AND eligible_role.id=eligible_binding.role_template_id
				WHERE eligible_user.tenant_id=rs.tenant_id
				  AND eligible_user.principal_id=p.id
				  AND eligible_user.active
				  AND eligible_user.deleted_at IS NULL
				  AND eligible_binding.legal_entity_id=rs.legal_entity_id
				  AND eligible_binding.valid_from<=clock_timestamp()
				  AND (eligible_binding.valid_until IS NULL OR clock_timestamp()<eligible_binding.valid_until)
				  AND eligible_role.valid_from<=clock_timestamp()
				  AND (eligible_role.valid_until IS NULL OR clock_timestamp()<eligible_role.valid_until)
			) available_context
			ORDER BY available_context.priority,lower(available_context.label),available_context.label
			LIMIT 1
		) recipient_context ON true
		WHERE `+internalRecipientEligibilityPredicate("p", "rs.tenant_id", "rs.legal_entity_id", "rs.subject_type", "rs.subject_id")+`
		  AND ($6='' OR strpos(lower(p.display_name),lower($6))>0 OR strpos(lower(COALESCE(recipient_context.label,'')),lower($6))>0)
		ORDER BY lower(p.display_name),lower(COALESCE(recipient_context.label,'')),p.id
		LIMIT $5
		)
		SELECT COALESCE(candidate_values.principal_id,''),COALESCE(candidate_values.display_name,''),COALESCE(candidate_values.context_label,'')
		FROM request_scope
		LEFT JOIN candidate_values ON true
		ORDER BY lower(candidate_values.display_name),lower(candidate_values.context_label),candidate_values.principal_id`, tenant, legalEntityID, requestID, actorPrincipalID, limit, strings.TrimSpace(search))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]RecipientCandidate, 0, limit)
	foundAuthorizedScope := false
	for rows.Next() {
		foundAuthorizedScope = true
		var value RecipientCandidate
		if err := rows.Scan(&value.PrincipalID, &value.DisplayName, &value.ContextLabel); err != nil {
			return nil, err
		}
		if value.PrincipalID == "" {
			continue
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if !foundAuthorizedScope {
		return nil, ErrNotFound
	}
	return values, nil
}

func internalRecipientEligibilityPredicate(principalAlias, tenantExpression, legalEntityExpression, subjectTypeExpression, subjectIDExpression string) string {
	return fmt.Sprintf(`
		%[1]s.kind='PERSON'
		AND %[1]s.status='ACTIVE'
		AND %[1]s.valid_from<=clock_timestamp()
		AND (%[1]s.valid_until IS NULL OR clock_timestamp()<%[1]s.valid_until)
		AND (
			EXISTS (
				SELECT 1 FROM org_positions eligible_position
				WHERE eligible_position.tenant_id=%[2]s
				  AND eligible_position.occupant_principal_id=%[1]s.id
				  AND (eligible_position.legal_entity_id IS NULL OR eligible_position.legal_entity_id=%[3]s)
				  AND eligible_position.valid_from<=clock_timestamp()
				  AND (eligible_position.valid_until IS NULL OR clock_timestamp()<eligible_position.valid_until)
			)
			OR EXISTS (
				SELECT 1
				FROM scim_users eligible_user
				JOIN scim_sources eligible_source ON eligible_source.tenant_id=eligible_user.tenant_id AND eligible_source.id=eligible_user.source_id AND eligible_source.status='ACTIVE'
				JOIN directory_group_members eligible_membership ON eligible_membership.tenant_id=eligible_user.tenant_id AND eligible_membership.scim_user_id=eligible_user.id
				JOIN directory_groups eligible_group ON eligible_group.tenant_id=eligible_membership.tenant_id AND eligible_group.id=eligible_membership.group_id AND eligible_group.source_id=eligible_user.source_id AND eligible_group.deleted_at IS NULL
				JOIN directory_group_role_bindings eligible_binding ON eligible_binding.tenant_id=eligible_group.tenant_id AND eligible_binding.group_id=eligible_group.id
				JOIN role_templates eligible_role ON eligible_role.tenant_id=eligible_binding.tenant_id AND eligible_role.id=eligible_binding.role_template_id
				WHERE eligible_user.tenant_id=%[2]s
				  AND eligible_user.principal_id=%[1]s.id
				  AND eligible_user.active
				  AND eligible_user.deleted_at IS NULL
				  AND eligible_binding.legal_entity_id=%[3]s
				  AND eligible_binding.valid_from<=clock_timestamp()
				  AND (eligible_binding.valid_until IS NULL OR clock_timestamp()<eligible_binding.valid_until)
				  AND eligible_role.valid_from<=clock_timestamp()
				  AND (eligible_role.valid_until IS NULL OR clock_timestamp()<eligible_role.valid_until)
			)
		)
		AND CASE %[4]s
			WHEN 'PROGRAM' THEN EXISTS (
				SELECT 1 FROM programs eligible_subject
				WHERE eligible_subject.tenant_id=%[2]s
				  AND eligible_subject.legal_entity_id=%[3]s
				  AND eligible_subject.id::text=%[5]s
				  AND %s
			)
			WHEN 'MATTER' THEN EXISTS (
				SELECT 1 FROM matters eligible_subject
				WHERE eligible_subject.tenant_id=%[2]s
				  AND eligible_subject.legal_entity_id=%[3]s
				  AND eligible_subject.id::text=%[5]s
				  AND %s
			)
			ELSE false
		END`, principalAlias, tenantExpression, legalEntityExpression, subjectTypeExpression, subjectIDExpression,
		recipientSubjectVisibilityPredicate("eligible_subject", principalAlias+".id::text"),
		recipientSubjectVisibilityPredicate("eligible_subject", principalAlias+".id::text"))
}

func recipientSubjectVisibilityPredicate(subjectAlias, principalExpression string) string {
	return fmt.Sprintf(`CASE
		WHEN NOT (%[1]s.scope ? 'access') THEN true
		WHEN upper(btrim(COALESCE(%[1]s.scope->>'access',''))) IN ('PUBLIC','INTERNAL') THEN true
		WHEN upper(btrim(COALESCE(%[1]s.scope->>'access','')))='RESTRICTED' THEN
			jsonb_typeof(%[1]s.scope->'allowed_principal_ids')='array'
			AND NOT EXISTS (
				SELECT 1 FROM jsonb_array_elements(%[1]s.scope->'allowed_principal_ids') entry(value)
				WHERE jsonb_typeof(entry.value)<>'string'
			)
			AND EXISTS (
				SELECT 1 FROM jsonb_array_elements_text(%[1]s.scope->'allowed_principal_ids') allowed(value)
				WHERE btrim(allowed.value)=%[2]s
			)
		ELSE false
	END`, subjectAlias, principalExpression)
}

func (r *PostgresRepository) InternalRecipientDisplayName(ctx context.Context, tenant, principalID string) (string, error) {
	var displayName string
	err := r.pool.QueryRow(ctx, `
		SELECT p.display_name
		FROM principals p
		JOIN tenants t ON t.id=p.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND p.id=$2::uuid`, tenant, principalID).Scan(&displayName)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return displayName, err
}

var _ recipientCandidateRepository = (*PostgresRepository)(nil)
var _ internalRecipientLabelDirectory = (*PostgresRepository)(nil)
