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

func (r *PostgresDeliveryReader) Health(ctx context.Context, tenantID string, asOf time.Time, window time.Duration) (DeliveryHealth, error) {
	if r == nil || r.pool == nil || strings.TrimSpace(tenantID) == "" {
		return DeliveryHealth{}, ErrDeliveryReadUnavailable
	}
	asOf = asOf.UTC()
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	if window <= 0 || window > 30*24*time.Hour {
		window = 24 * time.Hour
	}
	start := asOf.Add(-window)
	result := DeliveryHealth{AsOf: asOf, WindowStart: start, Classes: []DeliveryClassHealth{}, Failures: []DeliveryFailureSummary{}}

	rows, err := r.pool.Query(ctx, `
		SELECT d.delivery_class,
		       count(*) FILTER (WHERE d.status='DELIVERED')::int,
		       count(*) FILTER (WHERE d.status='TEMPORARY_FAILURE')::int,
		       count(*) FILTER (WHERE d.status IN ('RECIPIENT_REJECTED','PERMANENT_FAILURE'))::int,
		       count(*) FILTER (WHERE d.status IN ('DELIVERY_OUTCOME_UNKNOWN','DELIVERY_STARTED'))::int,
		       count(*) FILTER (WHERE d.status='CONTACT_UNAVAILABLE')::int,
		       max(d.last_attempted_at),
		       max(d.delivered_at)
		FROM notification_email_deliveries d
		JOIN tenants t ON t.id=d.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND d.last_attempted_at >= $2
		  AND d.last_attempted_at <= $3
		GROUP BY d.delivery_class
		ORDER BY d.delivery_class`, tenantID, start, asOf)
	if err != nil {
		return DeliveryHealth{}, fmt.Errorf("read notification delivery health: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item DeliveryClassHealth
		if err := rows.Scan(
			&item.DeliveryClass, &item.Delivered, &item.Retrying, &item.Failed,
			&item.OutcomeUnknown, &item.ContactUnavailable, &item.LastAttemptAt, &item.LastDeliveredAt,
		); err != nil {
			return DeliveryHealth{}, fmt.Errorf("scan notification delivery health: %w", err)
		}
		result.Delivered += item.Delivered
		result.Retrying += item.Retrying
		result.Failed += item.Failed
		result.OutcomeUnknown += item.OutcomeUnknown
		result.ContactUnavailable += item.ContactUnavailable
		result.Classes = append(result.Classes, item)
	}
	if err := rows.Err(); err != nil {
		return DeliveryHealth{}, fmt.Errorf("iterate notification delivery health: %w", err)
	}

	failureRows, err := r.pool.Query(ctx, `
		SELECT d.delivery_class,d.status,d.failure_code,count(*)::int,max(d.last_attempted_at)
		FROM notification_email_deliveries d
		JOIN tenants t ON t.id=d.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND d.last_attempted_at >= $2
		  AND d.last_attempted_at <= $3
		  AND d.status IN ('TEMPORARY_FAILURE','DELIVERY_STARTED','DELIVERY_OUTCOME_UNKNOWN','CONTACT_UNAVAILABLE','RECIPIENT_REJECTED','PERMANENT_FAILURE')
		GROUP BY d.delivery_class,d.status,d.failure_code
		ORDER BY max(d.last_attempted_at) DESC,d.delivery_class,d.status
		LIMIT 20`, tenantID, start, asOf)
	if err != nil {
		return DeliveryHealth{}, fmt.Errorf("read notification delivery failures: %w", err)
	}
	defer failureRows.Close()
	for failureRows.Next() {
		var item DeliveryFailureSummary
		if err := failureRows.Scan(&item.DeliveryClass, &item.Status, &item.FailureCode, &item.Count, &item.LastAttemptAt); err != nil {
			return DeliveryHealth{}, fmt.Errorf("scan notification delivery failure: %w", err)
		}
		result.Failures = append(result.Failures, item)
	}
	if err := failureRows.Err(); err != nil {
		return DeliveryHealth{}, fmt.Errorf("iterate notification delivery failures: %w", err)
	}
	return result, nil
}

func (r *PostgresDeliveryReader) RecordHistory(ctx context.Context, tenantID, legalEntityID, subjectType, subjectID string, limit int) (RecordNotificationHistory, error) {
	if r == nil || r.pool == nil || strings.TrimSpace(tenantID) == "" || strings.TrimSpace(legalEntityID) == "" || strings.TrimSpace(subjectID) == "" {
		return RecordNotificationHistory{}, ErrDeliveryReadUnavailable
	}
	subjectType = NormalizeRecordSubject(subjectType)
	if subjectType != "RISK" && subjectType != "LOSS" {
		return RecordNotificationHistory{}, ErrDeliveryReadUnavailable
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	asOf := time.Now().UTC()
	rows, err := r.pool.Query(ctx, `
		SELECT n.outbox_event_id::text,n.notification_kind,n.occurred_at,count(*)::int,
		       COALESCE(email.status,''),COALESCE(email.attempt_count,0)
		FROM in_app_notifications n
		JOIN tenants t ON t.id=n.tenant_id
		JOIN legal_entities le ON le.tenant_id=n.tenant_id AND le.id=n.legal_entity_id
		LEFT JOIN LATERAL (
			SELECT d.status,d.attempt_count
			FROM notification_email_deliveries d
			WHERE d.tenant_id=n.tenant_id
			  AND d.source_event_id=n.outbox_event_id
			  AND d.delivery_class='ATTENTION_CRITICAL'
			ORDER BY d.last_attempted_at DESC,d.id DESC
			LIMIT 1
		) email ON true
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND (le.id::text=$2 OR le.code=$2)
		  AND n.subject_type=$3
		  AND n.subject_id=$4::uuid
		GROUP BY n.outbox_event_id,n.notification_kind,n.occurred_at,email.status,email.attempt_count
		ORDER BY n.occurred_at DESC,n.outbox_event_id DESC
		LIMIT $5`, tenantID, legalEntityID, subjectType, subjectID, limit)
	if err != nil {
		return RecordNotificationHistory{}, fmt.Errorf("read record notification history: %w", err)
	}
	defer rows.Close()
	result := RecordNotificationHistory{Items: []RecordNotificationEvent{}, AsOf: asOf}
	for rows.Next() {
		var item RecordNotificationEvent
		if err := rows.Scan(&item.EventID, &item.Kind, &item.OccurredAt, &item.InAppDeliveries, &item.EmailStatus, &item.EmailAttempts); err != nil {
			return RecordNotificationHistory{}, fmt.Errorf("scan record notification history: %w", err)
		}
		result.Items = append(result.Items, item)
	}
	if err := rows.Err(); err != nil {
		return RecordNotificationHistory{}, fmt.Errorf("iterate record notification history: %w", err)
	}
	return result, nil
}

var _ DeliveryReader = (*PostgresDeliveryReader)(nil)
