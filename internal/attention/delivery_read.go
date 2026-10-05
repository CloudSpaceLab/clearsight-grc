package attention

import (
	"context"
	"time"
)

type DeliveryStatusCount struct {
	DeliveryClass string `json:"delivery_class"`
	Status        string `json:"status"`
	Count         int    `json:"count"`
}

type DeliveryFailure struct {
	DeliveryClass string    `json:"delivery_class"`
	Status        string    `json:"status"`
	FailureCode   string    `json:"failure_code,omitempty"`
	AttemptCount  int       `json:"attempt_count"`
	AttemptedAt   time.Time `json:"attempted_at"`
}

type DeliveryHealth struct {
	AsOf           time.Time             `json:"as_of"`
	WindowStart    time.Time             `json:"window_start"`
	Counts         []DeliveryStatusCount `json:"counts"`
	RecentFailures []DeliveryFailure     `json:"recent_failures"`
}

type NotificationHistoryItem struct {
	Channel        string    `json:"channel"`
	Kind           string    `json:"kind"`
	Status         string    `json:"status"`
	NoticeSequence int       `json:"notice_sequence,omitempty"`
	OccurredAt     time.Time `json:"occurred_at"`
}

type DeliveryReader interface {
	Health(context.Context, string, time.Time, int) (DeliveryHealth, error)
	RecordHistory(context.Context, string, string, string, string, int) ([]NotificationHistoryItem, error)
}
