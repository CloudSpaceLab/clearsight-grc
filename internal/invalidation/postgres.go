//go:build postgres

package invalidation

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const PostgresChannel = "clearsight_actor_invalidation"

func StartPostgresListener(ctx context.Context, pool *pgxpool.Pool, hub *Hub, logger *slog.Logger) {
	if ctx == nil || pool == nil || hub == nil {
		return
	}
	go listenPostgres(ctx, pool, hub, logger)
}

func listenPostgres(ctx context.Context, pool *pgxpool.Pool, hub *Hub, logger *slog.Logger) {
	backoff := time.Second
	for ctx.Err() == nil {
		connection, err := pool.Acquire(ctx)
		if err != nil {
			sleepInvalidationRetry(ctx, backoff)
			backoff = nextInvalidationBackoff(backoff)
			continue
		}
		_, err = connection.Exec(ctx, "LISTEN "+PostgresChannel)
		if err != nil {
			connection.Release()
			sleepInvalidationRetry(ctx, backoff)
			backoff = nextInvalidationBackoff(backoff)
			continue
		}
		backoff = time.Second
		for ctx.Err() == nil {
			notification, waitErr := connection.Conn().WaitForNotification(ctx)
			if waitErr != nil {
				break
			}
			var event Event
			if json.Unmarshal([]byte(notification.Payload), &event) == nil {
				hub.Publish(event)
			}
		}
		connection.Release()
		if ctx.Err() == nil && logger != nil {
			logger.Warn("realtime invalidation listener reconnecting")
		}
		sleepInvalidationRetry(ctx, backoff)
		backoff = nextInvalidationBackoff(backoff)
	}
}

func sleepInvalidationRetry(ctx context.Context, duration time.Duration) {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

func nextInvalidationBackoff(current time.Duration) time.Duration {
	current *= 2
	if current > 30*time.Second {
		return 30 * time.Second
	}
	return current
}
