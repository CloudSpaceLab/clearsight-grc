package workflow

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrNotificationInvalidCursor = errors.New("notification cursor is invalid")
	ErrNotificationNotFound      = errors.New("notification is unavailable")
	ErrNotificationUnavailable   = errors.New("notification persistence is unavailable")
)

func (s *Service) ListNotifications(ctx context.Context, filter NotificationFilter) (NotificationPage, error) {
	repository, ok := s.repo.(notificationRepository)
	if !ok || repository == nil {
		return NotificationPage{}, ErrNotificationUnavailable
	}
	filter.TenantID = strings.TrimSpace(filter.TenantID)
	filter.LegalEntityID = strings.TrimSpace(filter.LegalEntityID)
	filter.PrincipalID = strings.TrimSpace(filter.PrincipalID)
	if filter.TenantID == "" || filter.LegalEntityID == "" || filter.PrincipalID == "" {
		return NotificationPage{}, fmt.Errorf("notification scope is required")
	}
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 25
	}
	var cursor *notificationCursor
	if strings.TrimSpace(filter.Cursor) != "" {
		value, err := decodeNotificationCursor(filter.Cursor)
		if err != nil {
			return NotificationPage{}, err
		}
		cursor = &value
	}
	page, err := repository.ListInAppNotifications(ctx, filter, cursor)
	if err != nil {
		return NotificationPage{}, err
	}
	page.AsOf = time.Now().UTC()
	return page, nil
}

func (s *Service) MarkNotificationRead(ctx context.Context, filter NotificationFilter, notificationID string, at time.Time) (InAppNotification, error) {
	repository, ok := s.repo.(notificationRepository)
	if !ok || repository == nil {
		return InAppNotification{}, ErrNotificationUnavailable
	}
	filter.TenantID = strings.TrimSpace(filter.TenantID)
	filter.LegalEntityID = strings.TrimSpace(filter.LegalEntityID)
	filter.PrincipalID = strings.TrimSpace(filter.PrincipalID)
	notificationID = strings.TrimSpace(notificationID)
	if filter.TenantID == "" || filter.LegalEntityID == "" || filter.PrincipalID == "" || notificationID == "" {
		return InAppNotification{}, ErrNotificationNotFound
	}
	if at.IsZero() {
		at = time.Now().UTC()
	} else {
		at = at.UTC()
	}
	return repository.MarkInAppNotificationRead(ctx, filter, notificationID, at)
}

func encodeNotificationCursor(cursor notificationCursor) string {
	payload, err := json.Marshal(cursor)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(payload)
}

func decodeNotificationCursor(value string) (notificationCursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(value))
	if err != nil {
		return notificationCursor{}, ErrNotificationInvalidCursor
	}
	var cursor notificationCursor
	if err := json.Unmarshal(raw, &cursor); err != nil || cursor.OccurredAt.IsZero() || !validNotificationUUID(cursor.ID) {
		return notificationCursor{}, ErrNotificationInvalidCursor
	}
	cursor.OccurredAt = cursor.OccurredAt.UTC()
	return cursor, nil
}


func validNotificationUUID(value string) bool {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) != 36 {
		return false
	}
	for index, character := range trimmed {
		switch index {
		case 8, 13, 18, 23:
			if character != '-' {
				return false
			}
		default:
			if !((character >= '0' && character <= '9') || (character >= 'a' && character <= 'f') || (character >= 'A' && character <= 'F')) {
				return false
			}
		}
	}
	return true
}
