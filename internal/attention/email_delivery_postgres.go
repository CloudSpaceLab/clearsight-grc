//go:build postgres

package attention

import (
	"context"
	"fmt"

	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CriticalEmailPostgresRepository struct {
	pool *pgxpool.Pool
}

func NewCriticalEmailPostgresRepository(pool *pgxpool.Pool) *CriticalEmailPostgresRepository {
	return &CriticalEmailPostgresRepository{pool: pool}
}

func (r *CriticalEmailPostgresRepository) LoadCriticalEmailContext(ctx context.Context, event workflowruntime.OutboxEvent, intent Intent) (CriticalEmailContext, error) {
	if r == nil || r.pool == nil {
		return CriticalEmailContext{}, fmt.Errorf("critical attention email repository is unavailable")
	}
	var value CriticalEmailContext
	err := r.pool.QueryRow(ctx, `
		SELECT le.id::text,le.name,p.display_name,
		       COALESCE((
		         SELECT su.user_name
		         FROM scim_users su
		         JOIN scim_sources ss ON ss.tenant_id=su.tenant_id AND ss.id=su.source_id
		         WHERE su.tenant_id=episode.tenant_id
		           AND su.principal_id=$4::uuid
		           AND su.active AND su.deleted_at IS NULL
		           AND ss.status='ACTIVE'
		         ORDER BY su.updated_at DESC,su.id
		         LIMIT 1
		       ),''),
		       episode.notice_sequence,
		       EXISTS (
		         SELECT 1
		         FROM (
		           SELECT risk.owner_principal_id AS principal_id
		           FROM risks risk
		           WHERE episode.subject_type='RISK'
		             AND risk.tenant_id=episode.tenant_id
		             AND risk.legal_entity_id=episode.legal_entity_id
		             AND risk.id=episode.subject_id
		           UNION
		           SELECT check_config.owner_principal_id
		           FROM risk_indicator_links link
		           JOIN monitoring_checks check_config
		             ON check_config.tenant_id=link.tenant_id
		            AND check_config.id=link.monitoring_check_id
		            AND check_config.version=link.monitoring_check_version
		            AND check_config.program_id=link.program_id
		           WHERE episode.condition_key='indicator_breaches'
		             AND link.tenant_id=episode.tenant_id
		             AND link.legal_entity_id=episode.legal_entity_id
		             AND link.id=episode.member_id
		           UNION
		           SELECT check_config.reviewer_principal_id
		           FROM risk_indicator_links link
		           JOIN monitoring_checks check_config
		             ON check_config.tenant_id=link.tenant_id
		            AND check_config.id=link.monitoring_check_id
		            AND check_config.version=link.monitoring_check_version
		            AND check_config.program_id=link.program_id
		           WHERE episode.condition_key='indicator_breaches'
		             AND link.tenant_id=episode.tenant_id
		             AND link.legal_entity_id=episode.legal_entity_id
		             AND link.id=episode.member_id
		           UNION
		           SELECT loss.owner_principal_id
		           FROM operational_losses loss
		           WHERE episode.subject_type='LOSS'
		             AND loss.tenant_id=episode.tenant_id
		             AND loss.legal_entity_id=episode.legal_entity_id
		             AND loss.id=episode.subject_id
		         ) current_recipient
		         WHERE current_recipient.principal_id=$4::uuid
		       )
		FROM attention_episodes episode
		JOIN tenants tenant ON tenant.id=episode.tenant_id
		JOIN legal_entities le ON le.tenant_id=episode.tenant_id AND le.id=episode.legal_entity_id
		JOIN principals p ON p.tenant_id=episode.tenant_id AND p.id=$4::uuid
		WHERE (tenant.id::text=$1 OR tenant.slug=$1)
		  AND episode.id=$2::uuid
		  AND episode.legal_entity_id=$3::uuid
		  AND p.status='ACTIVE'
		  AND p.valid_from<=clock_timestamp()
		  AND (p.valid_until IS NULL OR clock_timestamp()<p.valid_until)`,
		event.TenantID, intent.EpisodeID, intent.LegalEntityID, intent.PrincipalID,
	).Scan(&value.LegalEntityID, &value.BrandName, &value.RecipientName, &value.RecipientAddress, &value.CurrentNoticeSequence, &value.StillEligible)
	if err != nil {
		return CriticalEmailContext{}, fmt.Errorf("load critical attention email context: %w", err)
	}
	return value, nil
}

func (r *CriticalEmailPostgresRepository) ClaimCriticalEmail(ctx context.Context, event workflowruntime.OutboxEvent, intent Intent, record EmailDeliveryRecord) (bool, error) {
	if r == nil || r.pool == nil {
		return false, fmt.Errorf("critical attention email repository is unavailable")
	}
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO notification_email_deliveries(
			tenant_id,legal_entity_id,episode_id,notice_sequence,source_event_id,principal_id,delivery_class,
			recipient_fingerprint,status,first_attempted_at,last_attempted_at,created_at,updated_at
		)
		SELECT episode.tenant_id,episode.legal_entity_id,episode.id,$4,$5::uuid,$6::uuid,$7,$8,$9,$10,$10,$10,$10
		FROM attention_episodes episode
		JOIN tenants tenant ON tenant.id=episode.tenant_id
		WHERE (tenant.id::text=$1 OR tenant.slug=$1)
		  AND episode.id=$2::uuid AND episode.legal_entity_id=$3::uuid
		ON CONFLICT(tenant_id,episode_id,notice_sequence,principal_id,delivery_class) DO UPDATE
		SET source_event_id=EXCLUDED.source_event_id,
		    recipient_fingerprint=EXCLUDED.recipient_fingerprint,
		    status=EXCLUDED.status,
		    failure_code='',
		    provider_message_id='',
		    attempt_count=notification_email_deliveries.attempt_count+1,
		    last_attempted_at=EXCLUDED.last_attempted_at,
		    delivered_at=NULL,
		    updated_at=clock_timestamp()
		WHERE notification_email_deliveries.status='TEMPORARY_FAILURE'`,
		event.TenantID, intent.EpisodeID, intent.LegalEntityID, intent.NoticeSequence, event.ID,
		intent.PrincipalID, criticalEmailDeliveryClass, record.RecipientFingerprint, record.Status, record.AttemptedAt,
	)
	if err != nil {
		return false, fmt.Errorf("claim critical attention email: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *CriticalEmailPostgresRepository) RecordCriticalEmail(ctx context.Context, event workflowruntime.OutboxEvent, intent Intent, record EmailDeliveryRecord) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("critical attention email repository is unavailable")
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE notification_email_deliveries delivery
		SET recipient_fingerprint=$6,
		    status=$7,
		    failure_code=$8,
		    provider_message_id=$9,
		    last_attempted_at=$10,
		    delivered_at=$11,
		    updated_at=clock_timestamp()
		FROM tenants tenant
		WHERE tenant.id=delivery.tenant_id
		  AND (tenant.id::text=$1 OR tenant.slug=$1)
		  AND delivery.episode_id=$2::uuid
		  AND delivery.notice_sequence=$3
		  AND delivery.principal_id=$4::uuid
		  AND delivery.delivery_class=$5`,
		event.TenantID, intent.EpisodeID, intent.NoticeSequence, intent.PrincipalID, criticalEmailDeliveryClass,
		record.RecipientFingerprint, record.Status, record.FailureCode, record.ProviderMessageID,
		record.AttemptedAt, record.DeliveredAt,
	)
	if err != nil {
		return fmt.Errorf("record critical attention email: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("record critical attention email: delivery claim is unavailable")
	}
	return nil
}

var _ CriticalEmailRepository = (*CriticalEmailPostgresRepository)(nil)
