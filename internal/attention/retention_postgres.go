//go:build postgres

package attention

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const NotificationRetentionWorkClass = "notification-retention"

type NotificationRetentionMaintainer struct {
	pool      *pgxpool.Pool
	retention time.Duration
}

func NewNotificationRetentionMaintainer(pool *pgxpool.Pool, retention time.Duration) *NotificationRetentionMaintainer {
	return &NotificationRetentionMaintainer{pool: pool, retention: retention}
}

func (m *NotificationRetentionMaintainer) Maintain(ctx context.Context, now time.Time, limit int) (int, error) {
	if m == nil || m.pool == nil || m.retention <= 0 {
		return 0, fmt.Errorf("notification retention is unavailable")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	if limit <= 0 || limit > 5000 {
		limit = 500
	}
	cutoff := now.Add(-m.retention)
	total := 0

	tag, err := m.pool.Exec(ctx, `
		WITH doomed AS (
			SELECT id
			FROM in_app_notifications
			WHERE occurred_at < $1
			ORDER BY occurred_at,id
			LIMIT $2
		)
		DELETE FROM in_app_notifications n
		USING doomed
		WHERE n.id=doomed.id`, cutoff, limit)
	if err != nil {
		return total, fmt.Errorf("prune in-app notification metadata: %w", err)
	}
	total += int(tag.RowsAffected())

	tag, err = m.pool.Exec(ctx, `
		WITH doomed AS (
			SELECT id
			FROM notification_email_deliveries
			WHERE last_attempted_at < $1
			ORDER BY last_attempted_at,id
			LIMIT $2
		)
		DELETE FROM notification_email_deliveries d
		USING doomed
		WHERE d.id=doomed.id`, cutoff, limit)
	if err != nil {
		return total, fmt.Errorf("prune notification email metadata: %w", err)
	}
	total += int(tag.RowsAffected())
	return total, nil
}
