//go:build postgres

package attention

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DigestPostgresRepository struct {
	pool *pgxpool.Pool
}

func NewDigestPostgresRepository(pool *pgxpool.Pool) *DigestPostgresRepository {
	return &DigestPostgresRepository{pool: pool}
}

func (r *DigestPostgresRepository) DueDigests(ctx context.Context, now time.Time, limit int) ([]DigestCandidate, error) {
	if r == nil || r.pool == nil {
		return nil, fmt.Errorf("daily digest repository is unavailable")
	}
	rows, err := r.pool.Query(ctx, `
		WITH candidates AS (
			SELECT t.id AS tenant_id,p.id AS principal_id,p.display_name,
			       COALESCE(pref.digest_minute,420) AS digest_minute,
			       COALESCE(pref.time_zone,'UTC') AS time_zone,
			       COALESCE(pref.quiet_hours_enabled,false) AS quiet_enabled,
			       COALESCE(pref.quiet_start_minute,1320) AS quiet_start,
			       COALESCE(pref.quiet_end_minute,420) AS quiet_end
			FROM tenants t
			JOIN principals p ON p.tenant_id=t.id
			LEFT JOIN user_notification_preferences pref
			  ON pref.tenant_id=t.id AND pref.principal_id=p.id
			WHERE p.status='ACTIVE'
			  AND p.valid_from<=$1
			  AND (p.valid_until IS NULL OR $1<p.valid_until)
			  AND COALESCE(pref.daily_digest_enabled,true)
		),
		scheduled AS (
			SELECT c.*,
			       timezone(c.time_zone,$1) AS local_now,
			       CASE
			         WHEN c.quiet_enabled AND (
			           (c.quiet_start<c.quiet_end AND c.digest_minute>=c.quiet_start AND c.digest_minute<c.quiet_end)
			           OR
			           (c.quiet_start>c.quiet_end AND (c.digest_minute>=c.quiet_start OR c.digest_minute<c.quiet_end))
			         ) THEN c.quiet_end
			         ELSE c.digest_minute
			       END AS effective_minute
			FROM candidates c
		),
		eligible AS (
			SELECT s.*
			FROM scheduled s
			WHERE (extract(hour FROM s.local_now)::int*60 + extract(minute FROM s.local_now)::int)>=s.effective_minute
			  AND NOT EXISTS (
			    SELECT 1
			    FROM notification_email_deliveries d
			    WHERE d.tenant_id=s.tenant_id
			      AND d.principal_id=s.principal_id
			      AND d.delivery_class='DAILY_DIGEST'
			      AND d.digest_date=s.local_now::date
			      AND (
			        d.status<>'TEMPORARY_FAILURE'
			        OR d.last_attempted_at > $1 - CASE
			          WHEN d.attempt_count<=1 THEN interval '1 minute'
			          WHEN d.attempt_count=2 THEN interval '5 minutes'
			          WHEN d.attempt_count=3 THEN interval '15 minutes'
			          ELSE interval '1 hour'
			        END
			      )
			  )
		),
		summaries AS (
			SELECT e.tenant_id,e.principal_id,e.local_now,e.display_name,
			       COALESCE((
			         SELECT su.user_name
			         FROM scim_users su
			         JOIN scim_sources ss ON ss.tenant_id=su.tenant_id AND ss.id=su.source_id
			         WHERE su.tenant_id=e.tenant_id
			           AND su.principal_id=e.principal_id
			           AND su.active AND su.deleted_at IS NULL AND ss.status='ACTIVE'
			         ORDER BY su.updated_at DESC,su.id
			         LIMIT 1
			       ),'') AS recipient_address,
			       (SELECT count(*) FROM in_app_notifications n
			        WHERE n.tenant_id=e.tenant_id AND n.principal_id=e.principal_id
			          AND n.occurred_at>$1-interval '24 hours') AS material_changes,
			       (SELECT count(*) FROM workflow_tasks wt
			        WHERE wt.tenant_id=e.tenant_id AND wt.principal_id=e.principal_id
			          AND wt.status IN ('READY','IN_PROGRESS','BLOCKED','ESCALATED')) AS assigned_work,
			       (SELECT count(*) FROM workflow_tasks wt
			        WHERE wt.tenant_id=e.tenant_id AND wt.principal_id=e.principal_id
			          AND wt.status IN ('READY','IN_PROGRESS','BLOCKED','ESCALATED')
			          AND wt.due_at>=$1 AND wt.due_at<$1+interval '7 days') AS due_soon,
			       (SELECT count(*) FROM in_app_notifications n
			        WHERE n.tenant_id=e.tenant_id AND n.principal_id=e.principal_id
			          AND n.occurred_at>$1-interval '24 hours'
			          AND n.notification_kind LIKE 'ATTENTION\_%\_WORSENED' ESCAPE '\') AS worsened,
			       (SELECT count(*) FROM in_app_notifications n
			        WHERE n.tenant_id=e.tenant_id AND n.principal_id=e.principal_id
			          AND n.occurred_at>$1-interval '24 hours'
			          AND n.notification_kind LIKE 'ATTENTION\_%\_CLEARED' ESCAPE '\') AS cleared
			FROM eligible e
		)
		SELECT s.tenant_id::text,s.principal_id::text,s.local_now::date,s.display_name,s.recipient_address,
		       s.material_changes,s.assigned_work,s.due_soon,s.worsened,s.cleared
		FROM summaries s
		WHERE s.material_changes>0 OR s.assigned_work>0 OR s.due_soon>0 OR s.worsened>0 OR s.cleared>0
		ORDER BY s.tenant_id,s.principal_id
		LIMIT $2`, now.UTC(), limit)
	if err != nil {
		return nil, fmt.Errorf("select due daily digests: %w", err)
	}
	defer rows.Close()

	result := make([]DigestCandidate, 0, limit)
	for rows.Next() {
		var value DigestCandidate
		if err := rows.Scan(
			&value.TenantID, &value.PrincipalID, &value.LocalDate, &value.RecipientName, &value.RecipientAddress,
			&value.MaterialChanges, &value.AssignedWork, &value.DueSoon, &value.Worsened, &value.Cleared,
		); err != nil {
			return nil, fmt.Errorf("scan due daily digest: %w", err)
		}
		value.BrandName = "ClearSight"
		result = append(result, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate due daily digests: %w", err)
	}
	return result, nil
}

func (r *DigestPostgresRepository) ClaimDigest(ctx context.Context, candidate DigestCandidate, record EmailDeliveryRecord) (bool, error) {
	if r == nil || r.pool == nil {
		return false, fmt.Errorf("daily digest repository is unavailable")
	}
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO notification_email_deliveries(
			tenant_id,principal_id,delivery_class,digest_date,recipient_fingerprint,status,
			first_attempted_at,last_attempted_at,created_at,updated_at
		)
		SELECT t.id,$2::uuid,'DAILY_DIGEST',$3::date,$4,$5,$6,$6,$6,$6
		FROM tenants t
		WHERE (t.id::text=$1 OR t.slug=$1)
		ON CONFLICT(tenant_id,digest_date,principal_id)
		    WHERE delivery_class='DAILY_DIGEST'
		DO UPDATE SET
			recipient_fingerprint=EXCLUDED.recipient_fingerprint,
			status=EXCLUDED.status,
			failure_code='',
			provider_message_id='',
			attempt_count=notification_email_deliveries.attempt_count+1,
			last_attempted_at=EXCLUDED.last_attempted_at,
			delivered_at=NULL,
			updated_at=clock_timestamp()
		WHERE notification_email_deliveries.status='TEMPORARY_FAILURE'`,
		candidate.TenantID, candidate.PrincipalID, candidate.LocalDate,
		record.RecipientFingerprint, record.Status, record.AttemptedAt,
	)
	if err != nil {
		return false, fmt.Errorf("claim daily digest: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *DigestPostgresRepository) RecordDigest(ctx context.Context, candidate DigestCandidate, record EmailDeliveryRecord) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("daily digest repository is unavailable")
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE notification_email_deliveries d
		SET recipient_fingerprint=$4,status=$5,failure_code=$6,provider_message_id=$7,
		    last_attempted_at=$8,delivered_at=$9,updated_at=clock_timestamp()
		FROM tenants t
		WHERE t.id=d.tenant_id
		  AND (t.id::text=$1 OR t.slug=$1)
		  AND d.principal_id=$2::uuid
		  AND d.delivery_class='DAILY_DIGEST'
		  AND d.digest_date=$3::date`,
		candidate.TenantID, candidate.PrincipalID, candidate.LocalDate,
		record.RecipientFingerprint, record.Status, record.FailureCode, record.ProviderMessageID,
		record.AttemptedAt, record.DeliveredAt,
	)
	if err != nil {
		return fmt.Errorf("record daily digest: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("record daily digest: delivery claim is unavailable")
	}
	return nil
}

var _ DigestRepository = (*DigestPostgresRepository)(nil)
