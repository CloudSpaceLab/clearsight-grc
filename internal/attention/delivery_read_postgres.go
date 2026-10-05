//go:build postgres

package attention

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresDeliveryReader struct {
	pool *pgxpool.Pool
}

func NewPostgresDeliveryReader(pool *pgxpool.Pool) *PostgresDeliveryReader {
	return &PostgresDeliveryReader{pool: pool}
}

func (r *PostgresDeliveryReader) Health(ctx context.Context, tenant string, since time.Time, limit int) (DeliveryHealth, error) {
	if r == nil || r.pool == nil || strings.TrimSpace(tenant) == "" {
		return DeliveryHealth{}, fmt.Errorf("notification delivery health is unavailable")
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	now := time.Now().UTC()
	since = since.UTC()
	if since.IsZero() || !since.Before(now) {
		since = now.Add(-24 * time.Hour)
	}
	rows, err := r.pool.Query(ctx, `
		SELECT d.delivery_class,d.status,count(*)::int
		FROM notification_email_deliveries d
		JOIN tenants t ON t.id=d.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND d.last_attempted_at>=$2
		GROUP BY d.delivery_class,d.status
		ORDER BY d.delivery_class,d.status`, tenant, since)
	if err != nil {
		return DeliveryHealth{}, fmt.Errorf("load notification delivery counts: %w", err)
	}
	defer rows.Close()
	result := DeliveryHealth{AsOf: now, WindowStart: since, Counts: []DeliveryStatusCount{}, RecentFailures: []DeliveryFailure{}}
	for rows.Next() {
		var item DeliveryStatusCount
		if err := rows.Scan(&item.DeliveryClass, &item.Status, &item.Count); err != nil {
			return DeliveryHealth{}, err
		}
		result.Counts = append(result.Counts, item)
	}
	if err := rows.Err(); err != nil {
		return DeliveryHealth{}, err
	}
	failures, err := r.pool.Query(ctx, `
		SELECT d.delivery_class,d.status,COALESCE(d.failure_code,''),d.attempt_count,d.last_attempted_at
		FROM notification_email_deliveries d
		JOIN tenants t ON t.id=d.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND d.last_attempted_at>=$2
		  AND d.status IN ('TEMPORARY_FAILURE','PERMANENT_FAILURE','DELIVERY_OUTCOME_UNKNOWN','RECIPIENT_REJECTED','CONTACT_UNAVAILABLE')
		ORDER BY d.last_attempted_at DESC,d.id DESC
		LIMIT $3`, tenant, since, limit)
	if err != nil {
		return DeliveryHealth{}, fmt.Errorf("load notification delivery failures: %w", err)
	}
	defer failures.Close()
	for failures.Next() {
		var item DeliveryFailure
		if err := failures.Scan(&item.DeliveryClass, &item.Status, &item.FailureCode, &item.AttemptCount, &item.AttemptedAt); err != nil {
			return DeliveryHealth{}, err
		}
		result.RecentFailures = append(result.RecentFailures, item)
	}
	return result, failures.Err()
}

func (r *PostgresDeliveryReader) RecordHistory(ctx context.Context, tenant, legalEntityID, subjectType, subjectID string, limit int) ([]NotificationHistoryItem, error) {
	if r == nil || r.pool == nil || strings.TrimSpace(tenant) == "" || strings.TrimSpace(legalEntityID) == "" ||
		strings.TrimSpace(subjectID) == "" {
		return nil, fmt.Errorf("notification history is unavailable")
	}
	subjectType = strings.ToUpper(strings.TrimSpace(subjectType))
	if subjectType != "RISK" && subjectType != "LOSS" {
		return nil, fmt.Errorf("unsupported notification history subject")
	}
	if limit < 1 || limit > 100 {
		limit = 30
	}
	rows, err := r.pool.Query(ctx, `
		WITH scope AS (
			SELECT t.id AS tenant_id,le.id AS legal_entity_id
			FROM tenants t
			JOIN legal_entities le ON le.tenant_id=t.id
			WHERE (t.id::text=$1 OR t.slug=$1)
			  AND (le.id::text=$2 OR le.code=$2)
			  AND le.valid_from<=clock_timestamp()
			  AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until)
			LIMIT 1
		), history AS (
			SELECT DISTINCT ON (n.outbox_event_id,n.notification_kind)
			       'IN_APP'::text AS channel,n.notification_kind AS kind,'DELIVERED'::text AS status,
			       0::int AS notice_sequence,n.occurred_at
			FROM in_app_notifications n
			JOIN scope s ON s.tenant_id=n.tenant_id AND s.legal_entity_id=n.legal_entity_id
			WHERE n.subject_type=$3
			  AND n.subject_id=$4::uuid
			ORDER BY n.outbox_event_id,n.notification_kind,n.occurred_at DESC

			UNION ALL

			SELECT d.delivery_class AS channel,'ATTENTION_CRITICAL'::text AS kind,d.status,
			       d.notice_sequence,d.last_attempted_at AS occurred_at
			FROM notification_email_deliveries d
			JOIN attention_episodes episode
			  ON episode.tenant_id=d.tenant_id
			 AND episode.legal_entity_id=d.legal_entity_id
			 AND episode.id=d.episode_id
			JOIN scope s ON s.tenant_id=d.tenant_id AND s.legal_entity_id=d.legal_entity_id
			WHERE d.delivery_class='ATTENTION_CRITICAL'
			  AND episode.subject_type=$3
			  AND episode.subject_id=$4::uuid
		)
		SELECT channel,kind,status,notice_sequence,occurred_at
		FROM history
		ORDER BY occurred_at DESC,channel,kind
		LIMIT $5`, tenant, legalEntityID, subjectType, subjectID, limit)
	if err != nil {
		return nil, fmt.Errorf("load notification history: %w", err)
	}
	defer rows.Close()
	items := make([]NotificationHistoryItem, 0)
	for rows.Next() {
		var item NotificationHistoryItem
		if err := rows.Scan(&item.Channel, &item.Kind, &item.Status, &item.NoticeSequence, &item.OccurredAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
