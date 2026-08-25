//go:build postgres

package evidence

import (
	"context"
	"encoding/json"
)

func (r *PostgresRepository) ListVisibleRequests(ctx context.Context, tenant, principal string, limit int) ([]Request, error) {
	return r.listActorRequests(ctx, tenant, principal, limit, false)
}

func (r *PostgresRepository) ListManageableRequests(ctx context.Context, tenant, principal string, limit int) ([]Request, error) {
	return r.listActorRequests(ctx, tenant, principal, limit, true)
}

func (r *PostgresRepository) listActorRequests(ctx context.Context, tenant, principal string, limit int, includeCreated bool) ([]Request, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+requestProjection+`,
		       COALESCE(er.recipient_type,''),COALESCE(er.recipient_principal_id::text,''),COALESCE(er.recipient_audience_hash,''::bytea),er.recipient_hint,
		       er.recipient_state,er.recipient_revision,er.recipient_issue_reason
		FROM capture_requests er
		JOIN tenants t ON t.id=er.tenant_id
		LEFT JOIN matters m ON er.subject_type='MATTER' AND m.tenant_id=er.tenant_id AND m.id::text=er.subject_id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND (
			(er.recipient_type='INTERNAL_PRINCIPAL' AND er.recipient_state='ASSIGNED' AND er.recipient_principal_id=$2::uuid)
			OR ($4::boolean AND er.created_by=$2::uuid)
		  )
		  AND (
			er.subject_type<>'MATTER' OR (
				m.id IS NOT NULL AND
				CASE
					WHEN NOT (m.scope ? 'access') THEN true
					WHEN upper(btrim(COALESCE(m.scope->>'access',''))) IN ('PUBLIC','INTERNAL') THEN true
					WHEN upper(btrim(COALESCE(m.scope->>'access','')))='RESTRICTED' THEN
						CASE
							WHEN jsonb_typeof(m.scope->'allowed_principal_ids')='array'
							 AND NOT EXISTS (
								SELECT 1
								FROM jsonb_array_elements(m.scope->'allowed_principal_ids') AS entry(value)
								WHERE jsonb_typeof(entry.value)<>'string'
							 )
							 AND EXISTS (
								SELECT 1
								FROM jsonb_array_elements_text(m.scope->'allowed_principal_ids') AS nonblank(value)
								WHERE btrim(nonblank.value)<>''
							 )
							THEN EXISTS (
								SELECT 1
								FROM jsonb_array_elements_text(m.scope->'allowed_principal_ids') AS allowed(value)
								WHERE btrim(allowed.value)=($2::uuid)::text
							)
							ELSE false
						END
					ELSE false
				END
			)
		  )
		ORDER BY CASE er.status WHEN 'READY' THEN 0 WHEN 'IN_PROGRESS' THEN 1 ELSE 2 END,er.deadline,er.id
		LIMIT $3`, tenant, principal, limit, includeCreated)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]Request, 0, limit)
	for rows.Next() {
		value, scanErr := scanRequestWithRecipient(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func scanRequestWithRecipient(row scanner) (Request, error) {
	var value Request
	var facts, presentation, sections, fields, sourceBindings []byte
	var recipientType, principalID, hint, state, issueReason string
	var audienceHash []byte
	if err := row.Scan(
		&value.ID, &value.TenantID, &value.SubjectType, &value.SubjectID, &value.Title, &value.Purpose, &value.WhyYou, &value.Sensitivity, &value.AudienceType,
		&value.EstimatedMinutes, &value.Deadline, &facts, &presentation, &sections, &fields, &sourceBindings,
		&value.FormTemplateID, &value.FormTemplateVersion, &value.CollectionPeriodStart, &value.CollectionPeriodEnd,
		&value.Origin.Type, &value.Origin.ID, &value.Origin.Version, &value.Status, &value.CreatedBy, &value.Version, &value.CreatedAt, &value.UpdatedAt,
		&recipientType, &principalID, &audienceHash, &hint, &state, &value.Recipient.Revision, &issueReason,
	); err != nil {
		return Request{}, err
	}
	if err := json.Unmarshal(facts, &value.KnownFacts); err != nil {
		return Request{}, err
	}
	if err := json.Unmarshal(presentation, &value.Presentation); err != nil {
		return Request{}, err
	}
	if err := json.Unmarshal(sections, &value.Sections); err != nil {
		return Request{}, err
	}
	if err := json.Unmarshal(fields, &value.Fields); err != nil {
		return Request{}, err
	}
	if err := json.Unmarshal(sourceBindings, &value.SourceBindings); err != nil {
		return Request{}, err
	}
	value.Recipient = Recipient{
		Type:         RecipientType(recipientType),
		PrincipalID:  principalID,
		AudienceHash: append([]byte(nil), audienceHash...),
		AudienceHint: hint,
		State:        RecipientState(state),
		Revision:     value.Recipient.Revision,
		IssueReason:  issueReason,
	}
	return value, nil
}

var _ visibleRequestRepository = (*PostgresRepository)(nil)
var _ manageableRequestRepository = (*PostgresRepository)(nil)
