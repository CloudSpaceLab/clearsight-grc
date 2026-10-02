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

func (r *PostgresRepository) RecordActorNotification(ctx context.Context, event workflowruntime.OutboxEvent, assignment assignmentNotificationEvent) error {
	if r == nil || r.pool == nil || strings.TrimSpace(event.ID) == "" || strings.TrimSpace(event.TenantID) == "" ||
		strings.TrimSpace(event.AggregateID) == "" || strings.TrimSpace(assignment.PrincipalID) == "" || strings.TrimSpace(assignment.NotificationKind) == "" {
		return ErrNotificationInvalid
	}
	_, err := r.pool.Exec(ctx, \`
		INSERT INTO actor_notifications(
			tenant_id,legal_entity_id,outbox_event_id,principal_id,notification_kind,matter_id,action_id,occurred_at)
		SELECT m.tenant_id,m.legal_entity_id,$3::uuid,$4::uuid,$5,m.id,NULLIF($6,'')::uuid,$7
		FROM matters m
		JOIN tenants t ON t.id=m.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND m.id=$2::uuid
		  AND (
		    ($5='MATTER_COMMENT_MENTIONED')
		    OR ($5='MATTER_OWNER_ASSIGNED' AND m.owner_principal_id=$4::uuid)
		    OR ($5 IN ('ACTION_PERFORMER_ASSIGNED','ACTION_UPDATE_REQUESTED') AND EXISTS (
		      SELECT 1 FROM matter_actions a
		      WHERE a.tenant_id=m.tenant_id AND a.matter_id=m.id AND a.id=NULLIF($6,'')::uuid
		        AND a.owner_principal_id=$4::uuid
		    ))
		  )
		ON CONFLICT(tenant_id,outbox_event_id,principal_id,notification_kind) DO NOTHING\`,
		event.TenantID, event.AggregateID, event.ID, assignment.PrincipalID, assignment.NotificationKind, assignment.ActionID, event.OccurredAt.UTC())
	if err != nil {
		return fmt.Errorf("record actor notification: %w", err)
	}
	// Zero rows is an intentional no-op for an assignment/update superseded
	// before this outbox event was consumed.
	return nil
}

func (r *PostgresRepository) ListActorNotifications(ctx context.Context, scope ActorNotificationScope, filter ActorNotificationFilter, cursor *actorNotificationCursor) (ActorNotificationPage, error) {
	if r == nil || r.pool == nil || !validActorNotificationScope(scope) {
		return ActorNotificationPage{}, ErrNotificationInvalid
	}
	var unread int
	if err := r.pool.QueryRow(ctx, \`
		SELECT count(*)
		FROM actor_notifications n
		JOIN tenants t ON t.id=n.tenant_id
		JOIN legal_entities le ON le.id=n.legal_entity_id AND le.tenant_id=n.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND (le.id::text=$2 OR le.code=$2)
		  AND n.principal_id=$3::uuid
		  AND n.read_at IS NULL\`,
		scope.TenantID, scope.LegalEntityID, scope.PrincipalID).Scan(&unread); err != nil {
		return ActorNotificationPage{}, fmt.Errorf("count unread actor notifications: %w", err)
	}

	var cursorAt time.Time
	var cursorID string
	hasCursor := cursor != nil
	if cursor != nil {
		cursorAt = cursor.OccurredAt.UTC()
		cursorID = cursor.ID
	}
	rows, err := r.pool.Query(ctx, \`
		SELECT n.id::text,n.notification_kind,n.matter_id::text,COALESCE(n.action_id::text,''),n.occurred_at,n.read_at
		FROM actor_notifications n
		JOIN tenants t ON t.id=n.tenant_id
		JOIN legal_entities le ON le.id=n.legal_entity_id AND le.tenant_id=n.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND (le.id::text=$2 OR le.code=$2)
		  AND n.principal_id=$3::uuid
		  AND (NOT $4 OR (n.occurred_at,n.id)<($5,$6::uuid))
		ORDER BY n.occurred_at DESC,n.id DESC
		LIMIT $7\`,
		scope.TenantID, scope.LegalEntityID, scope.PrincipalID, hasCursor, cursorAt, nullUUIDCursor(cursorID), filter.Limit+1)
	if err != nil {
		return ActorNotificationPage{}, fmt.Errorf("list actor notifications: %w", err)
	}
	defer rows.Close()

	items := make([]ActorNotification, 0, filter.Limit+1)
	for rows.Next() {
		var id, kind, matterID, actionID string
		var occurredAt time.Time
		var readAt *time.Time
		if err := rows.Scan(&id, &kind, &matterID, &actionID, &occurredAt, &readAt); err != nil {
			return ActorNotificationPage{}, fmt.Errorf("scan actor notification: %w", err)
		}
		items = append(items, actorNotificationFromParts(id, kind, matterID, actionID, occurredAt, readAt))
	}
	if err := rows.Err(); err != nil {
		return ActorNotificationPage{}, fmt.Errorf("iterate actor notifications: %w", err)
	}
	return ActorNotificationPage{Items: items, Unread: unread}, nil
}

func (r *PostgresRepository) MarkActorNotificationRead(ctx context.Context, scope ActorNotificationScope, id string, at time.Time) (ActorNotification, error) {
	if r == nil || r.pool == nil || !validActorNotificationScope(scope) || strings.TrimSpace(id) == "" {
		return ActorNotification{}, ErrNotificationInvalid
	}
	var notificationID, kind, matterID, actionID string
	var occurredAt time.Time
	var readAt *time.Time
	err := r.pool.QueryRow(ctx, \`
		UPDATE actor_notifications n
		SET read_at=COALESCE(n.read_at,$5)
		FROM tenants t,legal_entities le
		WHERE n.tenant_id=t.id
		  AND n.legal_entity_id=le.id AND le.tenant_id=n.tenant_id
		  AND (t.id::text=$1 OR t.slug=$1)
		  AND (le.id::text=$2 OR le.code=$2)
		  AND n.principal_id=$3::uuid
		  AND n.id=$4::uuid
		RETURNING n.id::text,n.notification_kind,n.matter_id::text,COALESCE(n.action_id::text,''),n.occurred_at,n.read_at\`,
		scope.TenantID, scope.LegalEntityID, scope.PrincipalID, id, at.UTC()).
		Scan(&notificationID, &kind, &matterID, &actionID, &occurredAt, &readAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ActorNotification{}, ErrNotificationNotFound
	}
	if err != nil {
		return ActorNotification{}, fmt.Errorf("mark actor notification read: %w", err)
	}
	return actorNotificationFromParts(notificationID, kind, matterID, actionID, occurredAt, readAt), nil
}

func nullUUIDCursor(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

var _ actorNotificationRepository = (*PostgresRepository)(nil)
var _ actorNotificationWriter = (*PostgresRepository)(nil)
