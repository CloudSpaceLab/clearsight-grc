package workflow

import (
	"context"
	"time"
)

type InAppNotification struct {
	ID            string     `json:"id"`
	Kind          string     `json:"kind"`
	Title         string     `json:"title"`
	Summary       string     `json:"summary"`
	SubjectType   string     `json:"subject_type"`
	SubjectID     string     `json:"subject_id"`
	ActionPath    string     `json:"action_path"`
	OccurredAt    time.Time  `json:"occurred_at"`
	ReadAt        *time.Time `json:"read_at,omitempty"`
	TenantID      string     `json:"-"`
	LegalEntityID string     `json:"-"`
	PrincipalID   string     `json:"-"`
	OutboxEventID string     `json:"-"`
}

type NotificationPage struct {
	Items       []InAppNotification `json:"items"`
	NextCursor  string              `json:"next_cursor,omitempty"`
	UnreadCount int                 `json:"unread_count"`
	AsOf        time.Time           `json:"as_of"`
}

type NotificationFilter struct {
	TenantID      string
	LegalEntityID string
	PrincipalID   string
	Cursor        string
	UnreadOnly    bool
	Limit         int
}

type notificationCursor struct {
	OccurredAt time.Time `json:"occurred_at"`
	ID         string    `json:"id"`
}

type inAppNotificationRecord struct {
	TenantID      string
	LegalEntityID string
	PrincipalID   string
	OutboxEventID string
	Kind          string
	Title         string
	Summary       string
	SubjectType   string
	SubjectID     string
	ActionPath    string
	OccurredAt    time.Time
}

type notificationRepository interface {
	ListInAppNotifications(context.Context, NotificationFilter, *notificationCursor) (NotificationPage, error)
	MarkInAppNotificationRead(context.Context, NotificationFilter, string, time.Time) (InAppNotification, error)
}

type inAppNotificationWriter interface {
	StoreInAppNotification(context.Context, inAppNotificationRecord) error
}
