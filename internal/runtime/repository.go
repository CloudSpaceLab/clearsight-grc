package runtime

import (
	"context"
	"time"
)

type Repository interface {
	ScheduleTimer(context.Context, Timer) (Timer, error)
	ClaimDueTimers(context.Context, string, time.Time, time.Duration, int) ([]Timer, error)
	CompleteTimer(context.Context, Timer, OutboxEvent, time.Time) error
	FailTimer(context.Context, Timer, int, string, time.Time, time.Time) (bool, error)
	CancelPendingTaskTimers(context.Context, string, string, string) (int, error)
	ClaimOutbox(context.Context, string, time.Time, time.Duration, int) ([]OutboxEvent, error)
	MarkPublished(context.Context, OutboxEvent, time.Time) error
	MarkFailed(context.Context, OutboxEvent, int, string, time.Time, time.Time) (bool, error)
	InboxProcessed(context.Context, string, string, string) (bool, error)
	RecordInbox(context.Context, string, string, string, time.Time) (bool, error)
}

type DelegationLifecycle interface {
	ActivateDueDelegations(context.Context, time.Time, int) (int, error)
	ExpireDueDelegations(context.Context, time.Time, int) (int, error)
}

type Publisher interface {
	Publish(context.Context, OutboxEvent) error
}
