//go:build postgres

package evidence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

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
	fields, err := json.Marshal(value.Fields)
	if err != nil {
		return Request{}, err
	}
	row := r.pool.QueryRow(ctx, `
		INSERT INTO capture_requests(
			id,tenant_id,subject_type,subject_id,title,purpose,why_you,sensitivity,audience_type,
			recipient_type,recipient_principal_id,recipient_audience_hash,recipient_hint,
			estimated_minutes,deadline,known_facts,fields,status,created_by,version,created_at,updated_at
		) VALUES(
			$1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3,$4,$5,$6,$7,$8,$9,
			$10,NULLIF($11,'')::uuid,$12,$13,$14,$15,$16::jsonb,$17::jsonb,$18,NULLIF($19,'')::uuid,$20,$21,$21
		)
		RETURNING id::text,(SELECT slug FROM tenants WHERE id=tenant_id),subject_type,subject_id,title,purpose,why_you,sensitivity,audience_type,estimated_minutes,deadline,known_facts,fields,status,COALESCE(created_by::text,''),version,created_at,updated_at`,
		value.ID, value.TenantID, value.SubjectType, value.SubjectID, value.Title, value.Purpose, value.WhyYou, value.Sensitivity, value.AudienceType,
		value.Recipient.Type, value.Recipient.PrincipalID, nullableAudienceHash(value.Recipient), value.Recipient.AudienceHint,
		value.EstimatedMinutes, value.Deadline, string(facts), string(fields), value.Status, value.CreatedBy, value.Version, value.CreatedAt)
	created, err := scanRequest(row)
	if err != nil {
		return Request{}, fmt.Errorf("create evidence request with recipient: %w", err)
	}
	created.Recipient = cloneRecipient(value.Recipient)
	return created, nil
}

func (r *PostgresRepository) GetRequestRecipient(ctx context.Context, tenant, requestID string) (Recipient, error) {
	var recipientType, principalID, hint string
	var audienceHash []byte
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(er.recipient_type,''),COALESCE(er.recipient_principal_id::text,''),COALESCE(er.recipient_audience_hash,''::bytea),er.recipient_hint
		FROM capture_requests er
		JOIN tenants t ON t.id=er.tenant_id
		WHERE er.id=$1::uuid AND (t.id::text=$2 OR t.slug=$2)`, requestID, tenant).Scan(&recipientType, &principalID, &audienceHash, &hint)
	if errors.Is(err, pgx.ErrNoRows) {
		return Recipient{}, ErrNotFound
	}
	if err != nil {
		return Recipient{}, err
	}
	return Recipient{Type: RecipientType(recipientType), PrincipalID: principalID, AudienceHash: append([]byte(nil), audienceHash...), AudienceHint: hint}, nil
}

func (r *PostgresRepository) ListRecipientRequests(ctx context.Context, tenant, principalID string, limit int) ([]Request, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT er.id::text,t.slug,er.subject_type,er.subject_id,er.title,er.purpose,er.why_you,er.sensitivity,er.audience_type,er.estimated_minutes,er.deadline,er.known_facts,er.fields,er.status,COALESCE(er.created_by::text,''),er.version,er.created_at,er.updated_at
		FROM capture_requests er
		JOIN tenants t ON t.id=er.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND er.recipient_type='INTERNAL_PRINCIPAL'
		  AND er.recipient_principal_id=$2::uuid
		ORDER BY CASE er.status WHEN 'READY' THEN 0 WHEN 'IN_PROGRESS' THEN 1 ELSE 2 END,er.deadline,er.id
		LIMIT $3`, tenant, principalID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]Request, 0, limit)
	for rows.Next() {
		value, err := scanRequest(rows)
		if err != nil {
			return nil, err
		}
		value.Recipient = Recipient{Type: RecipientInternalPrincipal, PrincipalID: principalID}
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

func cloneRecipient(value Recipient) Recipient {
	value.AudienceHash = append([]byte(nil), value.AudienceHash...)
	return value
}

var _ recipientStore = (*PostgresRepository)(nil)
var _ internalRecipientDirectory = (*PostgresRepository)(nil)
var _ = time.Time{}
