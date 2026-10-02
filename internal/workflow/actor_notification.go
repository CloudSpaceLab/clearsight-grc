package workflow

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

var (
	ErrNotificationInvalid  = errors.New("actor notification input is invalid")
	ErrNotificationNotFound = errors.New("actor notification is unavailable")
)

type ActorNotification struct {
	ID               string     `json:"id"`
	NotificationKind string     `json:"notification_kind"`
	TargetType       string     `json:"target_type"`
	TargetID         string     `json:"target_id"`
	ActionID         string     `json:"action_id,omitempty"`
	TargetPath       string     `json:"target_path"`
	OccurredAt       time.Time  `json:"occurred_at"`
	ReadAt           *time.Time `json:"read_at,omitempty"`
}

type ActorNotificationScope struct {
	TenantID      string
	LegalEntityID string
	PrincipalID   string
}

type ActorNotificationPage struct {
	Items      []ActorNotification `json:"items"`
	Unread     int                 `json:"unread"`
	NextCursor string              `json:"next_cursor,omitempty"`
}

type ActorNotificationFilter struct {
	Limit  int
	Cursor string
}

type actorNotificationCursor struct {
	OccurredAt time.Time `json:"t"`
	ID         string    `json:"i"`
}

type actorNotificationRepository interface {
	ListActorNotifications(context.Context, ActorNotificationScope, ActorNotificationFilter, *actorNotificationCursor) (ActorNotificationPage, error)
	MarkActorNotificationRead(context.Context, ActorNotificationScope, string, time.Time) (ActorNotification, error)
}

type actorNotificationWriter interface {
	RecordActorNotification(context.Context, workflowruntime.OutboxEvent, assignmentNotificationEvent) error
}

type ActorNotificationService struct {
	repo actorNotificationRepository
	now  func() time.Time
}

func NewActorNotificationService(repo actorNotificationRepository) *ActorNotificationService {
	return &ActorNotificationService{repo: repo, now: time.Now}
}

func (s *ActorNotificationService) List(ctx context.Context, scope ActorNotificationScope, filter ActorNotificationFilter) (ActorNotificationPage, error) {
	if s == nil || s.repo == nil || !validActorNotificationScope(scope) {
		return ActorNotificationPage{}, ErrNotificationInvalid
	}
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 20
	}
	var cursor *actorNotificationCursor
	if strings.TrimSpace(filter.Cursor) != "" {
		value, err := decodeActorNotificationCursor(filter.Cursor)
		if err != nil {
			return ActorNotificationPage{}, ErrNotificationInvalid
		}
		cursor = &value
	}
	page, err := s.repo.ListActorNotifications(ctx, scope, filter, cursor)
	if err != nil {
		return ActorNotificationPage{}, err
	}
	hasMore := len(page.Items) > filter.Limit
	if hasMore {
		page.Items = page.Items[:filter.Limit]
		last := page.Items[len(page.Items)-1]
		page.NextCursor = encodeActorNotificationCursor(actorNotificationCursor{OccurredAt: last.OccurredAt, ID: last.ID})
	} else {
		page.NextCursor = ""
	}
	return page, nil
}

func (s *ActorNotificationService) MarkRead(ctx context.Context, scope ActorNotificationScope, id string) (ActorNotification, error) {
	id = strings.TrimSpace(id)
	if s == nil || s.repo == nil || !validActorNotificationScope(scope) || id == "" {
		return ActorNotification{}, ErrNotificationInvalid
	}
	now := time.Now().UTC()
	if s.now != nil {
		now = s.now().UTC()
	}
	return s.repo.MarkActorNotificationRead(ctx, scope, id, now)
}

type InAppNotificationConsumer struct {
	repo actorNotificationWriter
}

func NewInAppNotificationConsumer(repo actorNotificationWriter) *InAppNotificationConsumer {
	return &InAppNotificationConsumer{repo: repo}
}

func (c *InAppNotificationConsumer) Publish(ctx context.Context, event workflowruntime.OutboxEvent) error {
	if c == nil || c.repo == nil {
		return ErrNotificationInvalid
	}
	assignments, relevant, err := decodeAssignmentNotificationEvent(event)
	if err != nil {
		return err
	}
	if !relevant {
		return nil
	}
	for _, assignment := range assignments {
		if err := c.repo.RecordActorNotification(ctx, event, assignment); err != nil {
			return err
		}
	}
	return nil
}

func actorNotificationFromParts(id, kind, matterID, actionID string, occurredAt time.Time, readAt *time.Time) ActorNotification {
	return ActorNotification{
		ID: id, NotificationKind: kind, TargetType: "MATTER", TargetID: matterID, ActionID: actionID,
		TargetPath: "#work/matters/" + matterID, OccurredAt: occurredAt.UTC(), ReadAt: readAt,
	}
}

func validActorNotificationScope(scope ActorNotificationScope) bool {
	return strings.TrimSpace(scope.TenantID) != "" && strings.TrimSpace(scope.LegalEntityID) != "" && strings.TrimSpace(scope.PrincipalID) != ""
}

func encodeActorNotificationCursor(value actorNotificationCursor) string {
	if value.OccurredAt.IsZero() || strings.TrimSpace(value.ID) == "" {
		return ""
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

func decodeActorNotificationCursor(value string) (actorNotificationCursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(value))
	if err != nil {
		return actorNotificationCursor{}, err
	}
	var cursor actorNotificationCursor
	if err := json.Unmarshal(raw, &cursor); err != nil {
		return actorNotificationCursor{}, err
	}
	cursor.ID = strings.TrimSpace(cursor.ID)
	if cursor.OccurredAt.IsZero() || cursor.ID == "" {
		return actorNotificationCursor{}, fmt.Errorf("invalid notification cursor")
	}
	return cursor, nil
}
