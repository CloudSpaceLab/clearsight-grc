//go:build postgres

package workflow

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) StoreInAppNotification(ctx context.Context, record inAppNotificationRecord) error {
	if r == nil || r.pool == nil {
		return ErrNotificationUnavailable
	}
	record.TenantID = strings.TrimSpace(record.TenantID)
	record.LegalEntityID = strings.TrimSpace(record.LegalEntityID)
	record.PrincipalID = strings.TrimSpace(record.PrincipalID)
	record.OutboxEventID = strings.TrimSpace(record.OutboxEventID)
	record.Kind = strings.TrimSpace(record.Kind)
	record.SubjectType = strings.TrimSpace(record.SubjectType)
	record.SubjectID = strings.TrimSpace(record.SubjectID)
	record.Title = strings.TrimSpace(record.Title)
	record.Summary = strings.TrimSpace(record.Summary)
	record.ActionPath = strings.TrimSpace(record.ActionPath)
	if record.TenantID == "" || record.LegalEntityID == "" || !validNotificationUUID(record.PrincipalID) || !validNotificationUUID(record.OutboxEventID) ||
		record.Kind == "" || record.SubjectType == "" || !validNotificationUUID(record.SubjectID) || record.Title == "" ||
		!strings.HasPrefix(record.ActionPath, "#") || record.OccurredAt.IsZero() {
		return fmt.Errorf("invalid in-app notification record")
	}
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO in_app_notifications(
			tenant_id,legal_entity_id,outbox_event_id,principal_id,notification_kind,
			subject_type,subject_id,title,summary,action_path,occurred_at)
		SELECT t.id,le.id,oe.id,p.id,$5,$6,$7::uuid,$8,$9,$10,$11
		FROM tenants t
		JOIN legal_entities le ON le.tenant_id=t.id AND (le.id::text=$2 OR le.code=$2)
		JOIN principals p ON p.tenant_id=t.id AND p.id::text=$3
		JOIN outbox_events oe ON oe.tenant_id=t.id AND oe.id=$4::uuid
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND le.valid_from<=$11
		  AND (le.valid_until IS NULL OR $11<le.valid_until)
		  AND p.valid_from<=$11
		  AND (p.valid_until IS NULL OR $11<p.valid_until)
		ON CONFLICT(tenant_id,outbox_event_id,principal_id,notification_kind) DO NOTHING`,
		record.TenantID, record.LegalEntityID, record.PrincipalID, record.OutboxEventID,
		record.Kind, record.SubjectType, record.SubjectID, record.Title, record.Summary,
		record.ActionPath, record.OccurredAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("store in-app notification: %w", err)
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		err := r.pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM in_app_notifications n
				JOIN tenants t ON t.id=n.tenant_id
				WHERE (t.id::text=$1 OR t.slug=$1)
				  AND n.outbox_event_id=$2::uuid
				  AND n.principal_id=$3::uuid
				  AND n.notification_kind=$4
				  AND n.legal_entity_id=(
				    SELECT le.id FROM legal_entities le
				    WHERE le.tenant_id=n.tenant_id AND (le.id::text=$5 OR le.code=$5)
				    ORDER BY le.valid_from DESC,le.id LIMIT 1
				  )
			)`, record.TenantID, record.OutboxEventID, record.PrincipalID, record.Kind, record.LegalEntityID).Scan(&exists)
		if err != nil {
			return fmt.Errorf("verify in-app notification insert: %w", err)
		}
		if !exists {
			return ErrNotificationNotFound
		}
	}
	return nil
}

func (r *PostgresRepository) ListInAppNotifications(ctx context.Context, filter NotificationFilter, cursor *notificationCursor) (NotificationPage, error) {
	if r == nil || r.pool == nil {
		return NotificationPage{}, ErrNotificationUnavailable
	}
	var cursorAt time.Time
	var cursorID string
	hasCursor := cursor != nil
	if cursor != nil {
		cursorAt = cursor.OccurredAt.UTC()
		cursorID = cursor.ID
	}
	var unread int
	err := r.pool.QueryRow(ctx, `
		WITH selected_scope AS (
			SELECT t.id AS tenant_id,le.id AS legal_entity_id,p.id AS principal_id
			FROM tenants t
			JOIN legal_entities le ON le.tenant_id=t.id
			JOIN principals p ON p.tenant_id=t.id
			WHERE (t.id::text=$1 OR t.slug=$1)
			  AND (le.id::text=$2 OR le.code=$2)
			  AND p.id::text=$3
			ORDER BY le.valid_from DESC,le.id
			LIMIT 1
		)
		SELECT count(*)
		FROM in_app_notifications n
		JOIN selected_scope scope
		  ON scope.tenant_id=n.tenant_id
		 AND scope.legal_entity_id=n.legal_entity_id
		 AND scope.principal_id=n.principal_id
		WHERE n.read_at IS NULL`,
		filter.TenantID, filter.LegalEntityID, filter.PrincipalID,
	).Scan(&unread)
	if err != nil {
		return NotificationPage{}, fmt.Errorf("count unread in-app notifications: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		WITH selected_scope AS (
			SELECT t.id AS tenant_id,le.id AS legal_entity_id,p.id AS principal_id
			FROM tenants t
			JOIN legal_entities le ON le.tenant_id=t.id
			JOIN principals p ON p.tenant_id=t.id
			WHERE (t.id::text=$1 OR t.slug=$1)
			  AND (le.id::text=$2 OR le.code=$2)
			  AND p.id::text=$3
			ORDER BY le.valid_from DESC,le.id
			LIMIT 1
		)
		SELECT n.id::text,n.notification_kind,n.title,n.summary,n.subject_type,n.subject_id::text,
		       n.action_path,n.occurred_at,n.read_at,n.legal_entity_id::text,n.principal_id::text,n.outbox_event_id::text
		FROM in_app_notifications n
		JOIN selected_scope scope
		  ON scope.tenant_id=n.tenant_id
		 AND scope.legal_entity_id=n.legal_entity_id
		 AND scope.principal_id=n.principal_id
		WHERE (NOT $4::boolean OR n.read_at IS NULL)
		  AND (NOT $5::boolean OR n.occurred_at<$6 OR (n.occurred_at=$6 AND n.id<$7::uuid))
		ORDER BY n.occurred_at DESC,n.id DESC
		LIMIT $8`,
		filter.TenantID, filter.LegalEntityID, filter.PrincipalID, filter.UnreadOnly,
		hasCursor, cursorAt, nullableNotificationUUID(cursorID), filter.Limit+1,
	)
	if err != nil {
		return NotificationPage{}, fmt.Errorf("list in-app notifications: %w", err)
	}
	defer rows.Close()

	items := make([]InAppNotification, 0, filter.Limit+1)
	for rows.Next() {
		var item InAppNotification
		if err := rows.Scan(
			&item.ID, &item.Kind, &item.Title, &item.Summary, &item.SubjectType, &item.SubjectID,
			&item.ActionPath, &item.OccurredAt, &item.ReadAt, &item.LegalEntityID, &item.PrincipalID, &item.OutboxEventID,
		); err != nil {
			return NotificationPage{}, fmt.Errorf("scan in-app notification: %w", err)
		}
		item.TenantID = filter.TenantID
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return NotificationPage{}, fmt.Errorf("iterate in-app notifications: %w", err)
	}
	page := NotificationPage{Items: items, UnreadCount: unread}
	if len(items) > filter.Limit {
		page.Items = items[:filter.Limit]
		last := page.Items[len(page.Items)-1]
		page.NextCursor = encodeNotificationCursor(notificationCursor{OccurredAt: last.OccurredAt, ID: last.ID})
	}
	return page, nil
}

func (r *PostgresRepository) MarkInAppNotificationRead(ctx context.Context, filter NotificationFilter, notificationID string, at time.Time) (InAppNotification, error) {
	if r == nil || r.pool == nil {
		return InAppNotification{}, ErrNotificationUnavailable
	}
	if !validNotificationUUID(notificationID) {
		return InAppNotification{}, ErrNotificationNotFound
	}
	var item InAppNotification
	err := r.pool.QueryRow(ctx, `
		WITH selected_scope AS (
			SELECT t.id AS tenant_id,le.id AS legal_entity_id,p.id AS principal_id
			FROM tenants t
			JOIN legal_entities le ON le.tenant_id=t.id
			JOIN principals p ON p.tenant_id=t.id
			WHERE (t.id::text=$1 OR t.slug=$1)
			  AND (le.id::text=$2 OR le.code=$2)
			  AND p.id::text=$3
			ORDER BY le.valid_from DESC,le.id
			LIMIT 1
		)
		UPDATE in_app_notifications n
		SET read_at=COALESCE(n.read_at,$5)
		FROM selected_scope scope
		WHERE n.tenant_id=scope.tenant_id
		  AND n.legal_entity_id=scope.legal_entity_id
		  AND n.principal_id=scope.principal_id
		  AND n.id=$4::uuid
		RETURNING n.id::text,n.notification_kind,n.title,n.summary,n.subject_type,n.subject_id::text,
		          n.action_path,n.occurred_at,n.read_at,n.legal_entity_id::text,n.principal_id::text,n.outbox_event_id::text`,
		filter.TenantID, filter.LegalEntityID, filter.PrincipalID, notificationID, at.UTC(),
	).Scan(
		&item.ID, &item.Kind, &item.Title, &item.Summary, &item.SubjectType, &item.SubjectID,
		&item.ActionPath, &item.OccurredAt, &item.ReadAt, &item.LegalEntityID, &item.PrincipalID, &item.OutboxEventID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return InAppNotification{}, ErrNotificationNotFound
	}
	if err != nil {
		return InAppNotification{}, fmt.Errorf("mark in-app notification read: %w", err)
	}
	item.TenantID = filter.TenantID
	return item, nil
}

func nullableNotificationUUID(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

var _ notificationRepository = (*PostgresRepository)(nil)
var _ inAppNotificationWriter = (*PostgresRepository)(nil)


func (r *PostgresRepository) LoadCurrentEscalationPrincipal(ctx context.Context, event workflowruntime.OutboxEvent, taskID string) (string, error) {
	if r == nil || r.pool == nil {
		return "", ErrNotificationUnavailable
	}
	var principalID string
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(task.principal_id::text,'')
		FROM workflow_tasks task
		JOIN tenants tenant ON tenant.id=task.tenant_id
		WHERE (tenant.id::text=$1 OR tenant.slug=$1)
		  AND task.id=$2::uuid
		  AND task.workflow_id=$3::uuid
		  AND task.status='ESCALATED'`,
		event.TenantID, taskID, event.AggregateID,
	).Scan(&principalID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("load current escalation principal: %w", err)
	}
	return strings.TrimSpace(principalID), nil
}
