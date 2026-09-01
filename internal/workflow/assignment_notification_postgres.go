//go:build postgres

package workflow

import (
	"context"
	"fmt"
	"time"

	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) LoadAssignmentNotification(ctx context.Context, event workflowruntime.OutboxEvent, assignment assignmentNotificationEvent) (assignmentNotificationContext, error) {
	if r == nil || r.pool == nil {
		return assignmentNotificationContext{}, fmt.Errorf("staff assignment notification repository is unavailable")
	}
	var value assignmentNotificationContext
	var dueAt *time.Time
	if assignment.NotificationKind == matterOwnerNotificationKind {
		err := r.pool.QueryRow(ctx, `
			SELECT le.id::text,le.name,p.display_name,
			       COALESCE((SELECT su.user_name FROM scim_users su JOIN scim_sources ss ON ss.tenant_id=su.tenant_id AND ss.id=su.source_id
			                 WHERE su.tenant_id=m.tenant_id AND su.principal_id=$3::uuid AND su.active AND su.deleted_at IS NULL AND ss.status='ACTIVE' LIMIT 1),''),
			       COALESCE(m.owner_principal_id::text,''),m.id::text,m.title,'Confirm scope and owner',m.due_at
			FROM matters m
			JOIN tenants t ON t.id=m.tenant_id
			JOIN legal_entities le ON le.tenant_id=m.tenant_id AND le.id=m.legal_entity_id
			JOIN principals p ON p.tenant_id=m.tenant_id AND p.id=$3::uuid
			WHERE (t.id::text=$1 OR t.slug=$1) AND m.id=$2::uuid`, event.TenantID, event.AggregateID, assignment.PrincipalID).
			Scan(&value.LegalEntityID, &value.BankName, &value.RecipientName, &value.RecipientAddress,
				&value.CurrentPrincipalID, &value.MatterID, &value.MatterTitle, &value.WorkTitle, &dueAt)
		if err != nil {
			return assignmentNotificationContext{}, normalizeAssignmentNotificationLoadError(err)
		}
	} else {
		err := r.pool.QueryRow(ctx, `
			SELECT le.id::text,le.name,p.display_name,
			       COALESCE((SELECT su.user_name FROM scim_users su JOIN scim_sources ss ON ss.tenant_id=su.tenant_id AND ss.id=su.source_id
			                 WHERE su.tenant_id=m.tenant_id AND su.principal_id=$4::uuid AND su.active AND su.deleted_at IS NULL AND ss.status='ACTIVE' LIMIT 1),''),
			       COALESCE(a.owner_principal_id::text,''),m.id::text,m.title,a.title,COALESCE(a.due_at,m.due_at)
			FROM matter_actions a
			JOIN matters m ON m.tenant_id=a.tenant_id AND m.id=a.matter_id
			JOIN tenants t ON t.id=m.tenant_id
			JOIN legal_entities le ON le.tenant_id=m.tenant_id AND le.id=m.legal_entity_id
			JOIN principals p ON p.tenant_id=m.tenant_id AND p.id=$4::uuid
			WHERE (t.id::text=$1 OR t.slug=$1) AND m.id=$2::uuid AND a.id=$3::uuid`, event.TenantID, event.AggregateID, assignment.ActionID, assignment.PrincipalID).
			Scan(&value.LegalEntityID, &value.BankName, &value.RecipientName, &value.RecipientAddress,
				&value.CurrentPrincipalID, &value.MatterID, &value.MatterTitle, &value.WorkTitle, &dueAt)
		if err != nil {
			return assignmentNotificationContext{}, normalizeAssignmentNotificationLoadError(err)
		}
	}
	if dueAt != nil {
		value.DueAt = dueAt.UTC()
	}
	return value, nil
}

func (r *PostgresRepository) GetAssignmentNotification(ctx context.Context, event workflowruntime.OutboxEvent, assignment assignmentNotificationEvent) (assignmentNotificationRecord, bool, error) {
	if r == nil || r.pool == nil {
		return assignmentNotificationRecord{}, false, fmt.Errorf("staff assignment notification repository is unavailable")
	}
	var value assignmentNotificationRecord
	err := r.pool.QueryRow(ctx, `
		SELECT d.status,d.failure_code,d.provider_message_id,COALESCE(d.recipient_fingerprint,'\x'::bytea),d.last_attempted_at,d.delivered_at
		FROM staff_assignment_notification_deliveries d
		JOIN tenants t ON t.id=d.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND d.outbox_event_id=$2::uuid AND d.principal_id=$3::uuid AND d.notification_kind=$4`,
		event.TenantID, event.ID, assignment.PrincipalID, assignment.NotificationKind).
		Scan(&value.Status, &value.FailureCode, &value.ProviderMessageID, &value.RecipientFingerprint, &value.AttemptedAt, &value.DeliveredAt)
	if err == pgx.ErrNoRows {
		return assignmentNotificationRecord{}, false, nil
	}
	if err != nil {
		return assignmentNotificationRecord{}, false, fmt.Errorf("load staff assignment notification receipt: %w", err)
	}
	return value, true, nil
}

func (r *PostgresRepository) ClaimAssignmentNotification(ctx context.Context, event workflowruntime.OutboxEvent, assignment assignmentNotificationEvent, record assignmentNotificationRecord) (bool, error) {
	if r == nil || r.pool == nil {
		return false, fmt.Errorf("staff assignment notification repository is unavailable")
	}
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO staff_assignment_notification_deliveries(
			tenant_id,legal_entity_id,outbox_event_id,principal_id,notification_kind,recipient_fingerprint,status,
			last_attempted_at,created_at,updated_at)
		SELECT m.tenant_id,m.legal_entity_id,$3::uuid,$4::uuid,$5,$6,$7,$8,$8,$8
		FROM matters m JOIN tenants t ON t.id=m.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND m.id=$2::uuid
		ON CONFLICT(tenant_id,outbox_event_id,principal_id,notification_kind) DO UPDATE SET
			recipient_fingerprint=EXCLUDED.recipient_fingerprint,status=EXCLUDED.status,failure_code='',provider_message_id='',
			attempt_count=staff_assignment_notification_deliveries.attempt_count+1,last_attempted_at=EXCLUDED.last_attempted_at,
			delivered_at=NULL,updated_at=EXCLUDED.updated_at
		WHERE staff_assignment_notification_deliveries.status='TEMPORARY_FAILURE'`,
		event.TenantID, event.AggregateID, event.ID, assignment.PrincipalID, assignment.NotificationKind,
		record.RecipientFingerprint, record.Status, record.AttemptedAt)
	if err != nil {
		return false, fmt.Errorf("claim staff assignment notification delivery: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *PostgresRepository) RecordAssignmentNotification(ctx context.Context, event workflowruntime.OutboxEvent, assignment assignmentNotificationEvent, record assignmentNotificationRecord) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("staff assignment notification repository is unavailable")
	}
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO staff_assignment_notification_deliveries(
			tenant_id,legal_entity_id,outbox_event_id,principal_id,notification_kind,recipient_fingerprint,status,
			failure_code,provider_message_id,last_attempted_at,delivered_at,created_at,updated_at)
		SELECT m.tenant_id,m.legal_entity_id,$3::uuid,$4::uuid,$5,$6,$7,$8,$9,$10,$11,$10,$10
		FROM matters m JOIN tenants t ON t.id=m.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND m.id=$2::uuid
		ON CONFLICT(tenant_id,outbox_event_id,principal_id,notification_kind) DO UPDATE SET
			recipient_fingerprint=EXCLUDED.recipient_fingerprint,status=EXCLUDED.status,failure_code=EXCLUDED.failure_code,
			provider_message_id=EXCLUDED.provider_message_id,
			last_attempted_at=EXCLUDED.last_attempted_at,delivered_at=EXCLUDED.delivered_at,updated_at=EXCLUDED.updated_at`,
		event.TenantID, event.AggregateID, event.ID, assignment.PrincipalID, assignment.NotificationKind,
		record.RecipientFingerprint, record.Status, record.FailureCode, record.ProviderMessageID, record.AttemptedAt, record.DeliveredAt)
	if err != nil {
		return fmt.Errorf("record staff assignment notification receipt: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("record staff assignment notification receipt: Matter is unavailable")
	}
	return nil
}

func normalizeAssignmentNotificationLoadError(err error) error {
	if err == pgx.ErrNoRows {
		return fmt.Errorf("staff assignment notification context is unavailable")
	}
	return fmt.Errorf("load staff assignment notification context: %w", err)
}
