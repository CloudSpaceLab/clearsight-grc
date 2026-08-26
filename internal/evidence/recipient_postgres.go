//go:build postgres

package evidence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) InternalRecipientEligible(ctx context.Context, tenant, principalID string) (bool, error) {
	var eligible bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM principals p
			JOIN tenants t ON t.id=p.tenant_id
			WHERE (t.id::text=$1 OR t.slug=$1)
			  AND p.id=$2::uuid
			  AND p.kind='PERSON'
			  AND p.status='ACTIVE'
			  AND p.valid_from<=now()
			  AND (p.valid_until IS NULL OR now()<p.valid_until)
		)`, tenant, principalID).Scan(&eligible)
	return eligible, err
}

func (r *PostgresRepository) CreateRequestWithRecipient(ctx context.Context, value Request) (Request, error) {
	if !value.Deadline.After(value.CreatedAt) {
		return Request{}, ErrRequestClosed
	}
	facts, err := json.Marshal(value.KnownFacts)
	if err != nil {
		return Request{}, err
	}
	presentation, err := json.Marshal(value.Presentation)
	if err != nil {
		return Request{}, err
	}
	sections, err := json.Marshal(value.Sections)
	if err != nil {
		return Request{}, err
	}
	fields, err := json.Marshal(value.Fields)
	if err != nil {
		return Request{}, err
	}
	sourceBindingValues := value.SourceBindings
	if sourceBindingValues == nil {
		sourceBindingValues = []RequestBindingReference{}
	}
	sourceBindings, err := json.Marshal(sourceBindingValues)
	if err != nil {
		return Request{}, err
	}
	state := value.Recipient.State
	if state == "" {
		state = RecipientStateAssigned
	}
	revision := value.Recipient.Revision
	if revision <= 0 {
		revision = 1
	}
	row := r.pool.QueryRow(ctx, `
		INSERT INTO capture_requests(
			id,tenant_id,subject_type,subject_id,title,purpose,why_you,sensitivity,audience_type,
			recipient_type,recipient_principal_id,recipient_audience_hash,recipient_hint,recipient_state,recipient_revision,recipient_issue_reason,
			estimated_minutes,deadline,known_facts,fields,source_bindings,form_template_id,form_template_version,collection_period_start,collection_period_end,status,created_by,version,created_at,updated_at,legal_entity_id
		) VALUES(
			$1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3,$4,$5,$6,$7,$8,$9,
			$10,NULLIF($11,'')::uuid,$12,$13,$14,$15,'',$16,$17,$18::jsonb,$19::jsonb,$20::jsonb,NULLIF($21,'')::uuid,NULLIF($22,0),$23,$24,$25,NULLIF($26,'')::uuid,$27,$28,$28,$29::uuid
		)
		RETURNING id::text,(SELECT slug FROM tenants WHERE id=tenant_id),legal_entity_id::text,subject_type,subject_id,title,purpose,why_you,sensitivity,audience_type,estimated_minutes,deadline,known_facts,fields,source_bindings,COALESCE(form_template_id::text,''),COALESCE(form_template_version,0),collection_period_start,collection_period_end,status,COALESCE(created_by::text,''),version,created_at,updated_at`,
		value.ID, value.TenantID, value.SubjectType, value.SubjectID, value.Title, value.Purpose, value.WhyYou, value.Sensitivity, value.AudienceType,
		value.Recipient.Type, value.Recipient.PrincipalID, nullableAudienceHash(value.Recipient), value.Recipient.AudienceHint, state, revision,
		value.EstimatedMinutes, value.Deadline, string(facts), string(fields), string(sourceBindings), value.FormTemplateID, value.FormTemplateVersion, value.CollectionPeriodStart, value.CollectionPeriodEnd, value.Status, value.CreatedBy, value.Version, value.CreatedAt, value.LegalEntityID)
	created, err := scanRequest(row)
	if err != nil {
		return Request{}, fmt.Errorf("create evidence request with recipient: %w", err)
	}
	created.Recipient = cloneRecipient(value.Recipient)
	created.Recipient.State = state
	created.Recipient.Revision = revision
	return created, nil
}

func (r *PostgresRepository) GetRequestRecipient(ctx context.Context, tenant, requestID string) (Recipient, error) {
	var recipientType, principalID, displayName, hint, state, issueReason string
	var audienceHash []byte
	var revision int64
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(er.recipient_type,''),COALESCE(er.recipient_principal_id::text,''),COALESCE(rp.display_name,''),COALESCE(er.recipient_audience_hash,''::bytea),er.recipient_hint,
		       er.recipient_state,er.recipient_revision,er.recipient_issue_reason
		FROM capture_requests er
		JOIN tenants t ON t.id=er.tenant_id
		LEFT JOIN principals rp ON rp.tenant_id=er.tenant_id AND rp.id=er.recipient_principal_id
		WHERE er.id=$1::uuid AND (t.id::text=$2 OR t.slug=$2)`, requestID, tenant).Scan(&recipientType, &principalID, &displayName, &audienceHash, &hint, &state, &revision, &issueReason)
	if errors.Is(err, pgx.ErrNoRows) {
		return Recipient{}, ErrNotFound
	}
	if err != nil {
		return Recipient{}, err
	}
	return Recipient{
		Type: RecipientType(recipientType), PrincipalID: principalID, DisplayName: displayName,
		AudienceHash: append([]byte(nil), audienceHash...), AudienceHint: hint,
		State: RecipientState(state), Revision: revision, IssueReason: issueReason,
	}, nil
}

func (r *PostgresRepository) ListRecipientRequests(ctx context.Context, tenant, principalID string, limit int) ([]Request, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT er.id::text,t.id::text,er.legal_entity_id::text,er.subject_type,er.subject_id,er.title,er.purpose,er.why_you,er.sensitivity,er.audience_type,
		       er.estimated_minutes,er.deadline,er.known_facts,er.fields,er.source_bindings,COALESCE(er.form_template_id::text,''),COALESCE(er.form_template_version,0),er.collection_period_start,er.collection_period_end,er.status,COALESCE(er.created_by::text,''),er.version,er.created_at,er.updated_at,
		       er.recipient_type,COALESCE(er.recipient_principal_id::text,''),COALESCE(rp.display_name,''),COALESCE(er.recipient_audience_hash,''::bytea),er.recipient_hint,
		       er.recipient_state,er.recipient_revision,er.recipient_issue_reason
		FROM capture_requests er
		JOIN tenants t ON t.id=er.tenant_id
		LEFT JOIN principals rp ON rp.tenant_id=er.tenant_id AND rp.id=er.recipient_principal_id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND er.recipient_type='INTERNAL_PRINCIPAL'
		  AND er.recipient_state='ASSIGNED'
		  AND er.recipient_principal_id=$2::uuid
		ORDER BY CASE er.status WHEN 'READY' THEN 0 WHEN 'IN_PROGRESS' THEN 1 ELSE 2 END,er.deadline,er.id
		LIMIT $3`, tenant, principalID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]Request, 0, limit)
	for rows.Next() {
		value, err := scanRequestWithRecipient(rows)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func nullableAudienceHash(recipient Recipient) any {
	if recipient.Type != RecipientExternalAudience || len(recipient.AudienceHash) == 0 {
		return nil
	}
	return recipient.AudienceHash
}

var _ recipientStore = (*PostgresRepository)(nil)
var _ internalRecipientDirectory = (*PostgresRepository)(nil)
