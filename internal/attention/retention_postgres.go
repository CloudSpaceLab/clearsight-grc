//go:build postgres

package attention

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	RetentionWorkClass = "notification-retention"
	NotificationMetadataRetention = 365 * 24 * time.Hour
)

type RetentionMaintainer struct {
	pool *pgxpool.Pool
}

func NewRetentionMaintainer(pool *pgxpool.Pool) *RetentionMaintainer {
	return &RetentionMaintainer{pool: pool}
}

func (m *RetentionMaintainer) Maintain(ctx context.Context, now time.Time, limit int) (int, error) {
	if m == nil || m.pool == nil {
		return 0, fmt.Errorf("notification retention is unavailable")
	}
	if limit <= 0 {
		limit = 500
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	cutoff := now.Add(-NotificationMetadataRetention)

	emailDeleted, err := m.deleteEmailDeliveries(ctx, cutoff, limit)
	if err != nil {
		return 0, err
	}
	remaining := limit - emailDeleted
	if remaining <= 0 {
		return emailDeleted, nil
	}
	inAppDeleted, err := m.deleteInAppNotifications(ctx, cutoff, remaining)
	if err != nil {
		return emailDeleted, err
	}
	return emailDeleted + inAppDeleted, nil
}

func (m *RetentionMaintainer) deleteEmailDeliveries(ctx context.Context, cutoff time.Time, limit int) (int, error) {
	tag, err := m.pool.Exec(ctx, `
		WITH doomed AS (
			SELECT id
			FROM notification_email_deliveries
			WHERE last_attempted_at < $1
			ORDER BY last_attempted_at,id
			LIMIT $2
		)
		DELETE FROM notification_email_deliveries delivery
		USING doomed
		WHERE delivery.id=doomed.id`, cutoff, limit)
	if err != nil {
		return 0, fmt.Errorf("prune notification email delivery metadata: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

func (m *RetentionMaintainer) deleteInAppNotifications(ctx context.Context, cutoff time.Time, limit int) (int, error) {
	tag, err := m.pool.Exec(ctx, `
		WITH doomed AS (
			SELECT id
			FROM in_app_notifications
			WHERE occurred_at < $1
			ORDER BY occurred_at,id
			LIMIT $2
		)
		DELETE FROM in_app_notifications notification
		USING doomed
		WHERE notification.id=doomed.id`, cutoff, limit)
	if err != nil {
		return 0, fmt.Errorf("prune in-app notification metadata: %w", err)
	}
	return int(tag.RowsAffected()), nil
}
