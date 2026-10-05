package attention

import (
	"context"
	"errors"
	"strings"
	"time"
)

var ErrDeliveryReadUnavailable = errors.New("notification delivery read is unavailable")

type DeliveryHealth struct {
	AsOf               time.Time                 `json:"as_of"`
	WindowStart        time.Time                 `json:"window_start"`
	Delivered          int                       `json:"delivered"`
	Retrying           int                       `json:"retrying"`
	Failed             int                       `json:"failed"`
	OutcomeUnknown     int                       `json:"outcome_unknown"`
	ContactUnavailable int                       `json:"contact_unavailable"`
	Classes            []DeliveryClassHealth     `json:"classes"`
	Failures           []DeliveryFailureSummary `json:"failures"`
}

type DeliveryClassHealth struct {
	DeliveryClass      string     `json:"delivery_class"`
	Delivered          int        `json:"delivered"`
	Retrying           int        `json:"retrying"`
	Failed             int        `json:"failed"`
	OutcomeUnknown     int        `json:"outcome_unknown"`
	ContactUnavailable int        `json:"contact_unavailable"`
	LastAttemptAt      *time.Time `json:"last_attempt_at,omitempty"`
	LastDeliveredAt    *time.Time `json:"last_delivered_at,omitempty"`
}

type DeliveryFailureSummary struct {
	DeliveryClass string    `json:"delivery_class"`
	Status        string    `json:"status"`
	FailureCode   string    `json:"failure_code,omitempty"`
	Count         int       `json:"count"`
	LastAttemptAt time.Time `json:"last_attempt_at"`
}

type RecordNotificationHistory struct {
	Items []RecordNotificationEvent `json:"items"`
	AsOf  time.Time                 `json:"as_of"`
}

type RecordNotificationEvent struct {
	EventID         string    `json:"event_id"`
	Kind            string    `json:"kind"`
	OccurredAt      time.Time `json:"occurred_at"`
	InAppDeliveries int       `json:"in_app_deliveries"`
	EmailStatus     string    `json:"email_status,omitempty"`
	EmailAttempts   int       `json:"email_attempts,omitempty"`
}

type DeliveryReader interface {
	Health(context.Context, string, time.Time, time.Duration) (DeliveryHealth, error)
	RecordHistory(context.Context, string, string, string, string, int) (RecordNotificationHistory, error)
}

func NormalizeRecordSubject(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}
