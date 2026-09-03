//go:build postgres

package evidence

import (
	"context"
	"errors"
	"time"

	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/jackc/pgx/v5"
)

type PostgresCommunicationDeliveryRepository struct {
	repo *PostgresRepository
}

func NewPostgresCommunicationDeliveryRepository(repo *PostgresRepository) *PostgresCommunicationDeliveryRepository {
	return &PostgresCommunicationDeliveryRepository{repo: repo}
}

func (repository *PostgresCommunicationDeliveryRepository) LoadCommunicationDelivery(ctx context.Context, tenantID, distributionID string) (communicationDeliveryBundle, error) {
	if repository == nil || repository.repo == nil || repository.repo.pool == nil {
		return communicationDeliveryBundle{}, ErrCommunicationUnavailable
	}
	var bundle communicationDeliveryBundle
	err := repository.repo.pool.QueryRow(ctx, `
		SELECT d.id::text,d.tenant_id::text,d.legal_entity_id::text,d.form_template_id::text,d.form_template_version,
		       d.subject_type,d.subject_id::text,d.title,d.purpose,d.access_policy,d.status,d.deadline,d.route_expires_at,
		       d.reminder_policy,d.created_by::text,d.version,d.created_at,d.updated_at,
		       COALESCE((
		           SELECT q.origin_type
		           FROM capture_distribution_recipients r
		           JOIN capture_requests q ON q.tenant_id=r.tenant_id AND q.legal_entity_id=r.legal_entity_id AND q.id=r.request_id
		           WHERE r.tenant_id=d.tenant_id AND r.legal_entity_id=d.legal_entity_id AND r.distribution_id=d.id AND r.role='TO'
		           ORDER BY r.created_at,r.id
		           LIMIT 1
		       ),'')
		FROM capture_form_distributions d
		WHERE d.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND d.id=$2::uuid`, tenantID, distributionID).Scan(
		&bundle.Distribution.ID, &bundle.Distribution.TenantID, &bundle.Distribution.LegalEntityID,
		&bundle.Distribution.FormTemplateID, &bundle.Distribution.FormTemplateVersion,
		&bundle.Distribution.SubjectType, &bundle.Distribution.SubjectID, &bundle.Distribution.Title,
		&bundle.Distribution.Purpose, &bundle.Distribution.AccessPolicy, &bundle.Distribution.Status,
		&bundle.Distribution.Deadline, &bundle.Distribution.RouteExpiresAt, &bundle.Distribution.ReminderPolicy,
		&bundle.Distribution.CreatedBy, &bundle.Distribution.Version, &bundle.Distribution.CreatedAt, &bundle.Distribution.UpdatedAt,
		&bundle.OriginType,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return communicationDeliveryBundle{}, ErrCommunicationNotFound
	}
	if err != nil {
		return communicationDeliveryBundle{}, err
	}

	rows, err := repository.repo.pool.Query(ctx, `
		SELECT id::text,distribution_id::text,tenant_id::text,legal_entity_id::text,role,recipient_type,
		       COALESCE(principal_id::text,''),COALESCE(request_id::text,''),audience_hint,contact_label,state,version,created_at,updated_at,
		       address_hash,address_ciphertext,address_key_id
		FROM capture_distribution_recipients
		WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND distribution_id=$2::uuid AND recipient_type='EXTERNAL_AUDIENCE'
		ORDER BY created_at,id`, tenantID, distributionID)
	if err != nil {
		return communicationDeliveryBundle{}, err
	}
	defer rows.Close()
	bundle.Recipients = make([]communicationDeliveryRecipient, 0)
	for rows.Next() {
		var value communicationDeliveryRecipient
		if err := rows.Scan(
			&value.ID, &value.DistributionID, &value.TenantID, &value.LegalEntityID, &value.Role, &value.Type,
			&value.PrincipalID, &value.RequestID, &value.AudienceHint, &value.ContactLabel, &value.State, &value.Version,
			&value.CreatedAt, &value.UpdatedAt, &value.ProtectedAddress.Hash, &value.ProtectedAddress.Ciphertext, &value.ProtectedAddress.KeyID,
		); err != nil {
			return communicationDeliveryBundle{}, err
		}
		value.ProtectedAddress.Hash = append([]byte(nil), value.ProtectedAddress.Hash...)
		value.ProtectedAddress.Ciphertext = append([]byte(nil), value.ProtectedAddress.Ciphertext...)
		bundle.Recipients = append(bundle.Recipients, value)
	}
	if err := rows.Err(); err != nil {
		return communicationDeliveryBundle{}, err
	}
	return bundle, nil
}

func (repository *PostgresCommunicationDeliveryRepository) GetCommunicationDeliveryAttempt(ctx context.Context, outboxEventID, recipientID string, action CommunicationAction) (CommunicationDeliveryAttempt, bool, error) {
	if repository == nil || repository.repo == nil || repository.repo.pool == nil {
		return CommunicationDeliveryAttempt{}, false, ErrCommunicationUnavailable
	}
	var attempt CommunicationDeliveryAttempt
	err := repository.repo.pool.QueryRow(ctx, `
		SELECT status,failure_code,provider_message_id,attempted_at
		FROM form_delivery_attempts
		WHERE outbox_event_id=$1::uuid AND recipient_id=$2::uuid AND action=$3`, outboxEventID, recipientID, action).Scan(
		&attempt.Status, &attempt.FailureCode, &attempt.ProviderMessage, &attempt.AttemptedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return CommunicationDeliveryAttempt{}, false, nil
	}
	if err != nil {
		return CommunicationDeliveryAttempt{}, false, err
	}
	attempt.AttemptedAt = attempt.AttemptedAt.UTC()
	return attempt, true, nil
}

func (repository *PostgresCommunicationDeliveryRepository) RecordCommunicationDeliveryAttempt(ctx context.Context, event workflowruntime.OutboxEvent, bundle communicationDeliveryBundle, recipient communicationDeliveryRecipient, template CommunicationTemplate, receipt InvitationDeliveryReceipt, failureCode string, attemptedAt time.Time) error {
	if repository == nil || repository.repo == nil || repository.repo.pool == nil {
		return ErrCommunicationUnavailable
	}
	status := communicationDeliveryAttemptStatus(receipt, failureCode)
	if status != "SKIPPED" && (template.ID == "" || template.Version < 1) {
		return ErrCommunicationUnavailable
	}
	if status == "SKIPPED" {
		template.ID = ""
		template.Version = 0
	}
	if failureCode == "" {
		failureCode = string(receipt.FailureCode)
	}

	tx, err := repository.repo.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		INSERT INTO form_delivery_attempts(
			tenant_id,legal_entity_id,distribution_id,recipient_id,action,template_id,template_version,outbox_event_id,
			status,provider_message_id,recipient_hint,failure_code,attempted_at
		) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,NULLIF($6,'')::uuid,NULLIF($7,0),$8::uuid,$9,$10,$11,$12,$13)
		ON CONFLICT (tenant_id,outbox_event_id,recipient_id,action) DO UPDATE
		SET template_id=EXCLUDED.template_id,template_version=EXCLUDED.template_version,status=EXCLUDED.status,
		    provider_message_id=EXCLUDED.provider_message_id,recipient_hint=EXCLUDED.recipient_hint,
		    failure_code=EXCLUDED.failure_code,attempted_at=EXCLUDED.attempted_at
		WHERE form_delivery_attempts.status<>'DELIVERED'`,
		bundle.Distribution.TenantID, bundle.Distribution.LegalEntityID, bundle.Distribution.ID, recipient.ID,
		template.Action, template.ID, template.Version, event.ID, status, receipt.ProviderMessageID,
		recipient.AudienceHint, failureCode, attemptedAt.UTC())
	if err != nil {
		return err
	}
	if status == "DELIVERED" && template.Action == CommunicationInvitation {
		if _, err := tx.Exec(ctx, `
			UPDATE capture_distribution_recipients
			SET state=CASE WHEN state='PENDING' THEN 'DELIVERED' ELSE state END,
			    version=CASE WHEN state='PENDING' THEN version+1 ELSE version END,
			    updated_at=CASE WHEN state='PENDING' THEN $4 ELSE updated_at END
			WHERE tenant_id=$1::uuid AND distribution_id=$2::uuid AND id=$3::uuid AND state<>'REVOKED'`,
			bundle.Distribution.TenantID, bundle.Distribution.ID, recipient.ID, attemptedAt.UTC()); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

var _ communicationDeliveryRepository = (*PostgresCommunicationDeliveryRepository)(nil)
