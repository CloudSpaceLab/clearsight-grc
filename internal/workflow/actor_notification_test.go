package workflow

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

func TestActorNotificationServiceScopesUnreadAndReadState(t *testing.T) {
	now := time.Date(2026, 10, 2, 7, 0, 0, 0, time.UTC)
	scope := ActorNotificationScope{TenantID: "bank", LegalEntityID: "bank-ng", PrincipalID: "person-a"}
	other := ActorNotificationScope{TenantID: "bank", LegalEntityID: "bank-ng", PrincipalID: "person-b"}
	first := actorNotificationFromParts("00000000-0000-4000-8000-000000000001", matterOwnerNotificationKind, "matter-1", "", now.Add(-time.Minute), nil)
	second := actorNotificationFromParts("00000000-0000-4000-8000-000000000002", actionPerformerNotificationKind, "matter-2", "action-2", now, nil)
	repo := NewMemoryActorNotificationRepository(
		struct {
			Scope ActorNotificationScope
			Value ActorNotification
		}{scope, first},
		struct {
			Scope ActorNotificationScope
			Value ActorNotification
		}{scope, second},
		struct {
			Scope ActorNotificationScope
			Value ActorNotification
		}{other, actorNotificationFromParts("00000000-0000-4000-8000-000000000003", matterOwnerNotificationKind, "matter-other", "", now, nil)},
	)
	service := NewActorNotificationService(repo)
	service.now = func() time.Time { return now.Add(time.Hour) }

	page, err := service.List(context.Background(), scope, ActorNotificationFilter{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if page.Unread != 2 || len(page.Items) != 1 || page.Items[0].ID != second.ID || page.NextCursor == "" {
		t.Fatalf("first page = %#v", page)
	}
	next, err := service.List(context.Background(), scope, ActorNotificationFilter{Limit: 1, Cursor: page.NextCursor})
	if err != nil {
		t.Fatal(err)
	}
	if next.Unread != 2 || len(next.Items) != 1 || next.Items[0].ID != first.ID || next.NextCursor != "" {
		t.Fatalf("second page = %#v", next)
	}

	read, err := service.MarkRead(context.Background(), scope, second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if read.ReadAt == nil || !read.ReadAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("read notification = %#v", read)
	}
	readAgain, err := service.MarkRead(context.Background(), scope, second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if readAgain.ReadAt == nil || !readAgain.ReadAt.Equal(*read.ReadAt) {
		t.Fatalf("read timestamp changed: first=%v second=%v", read.ReadAt, readAgain.ReadAt)
	}
	after, err := service.List(context.Background(), scope, ActorNotificationFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if after.Unread != 1 {
		t.Fatalf("unread after mark = %d", after.Unread)
	}
}

func TestActorNotificationServiceRejectsInvalidCursorAndForeignID(t *testing.T) {
	scope := ActorNotificationScope{TenantID: "bank", LegalEntityID: "bank-ng", PrincipalID: "person-a"}
	service := NewActorNotificationService(NewMemoryActorNotificationRepository())
	if _, err := service.List(context.Background(), scope, ActorNotificationFilter{Cursor: "not-a-cursor"}); err != ErrNotificationInvalid {
		t.Fatalf("invalid cursor error = %v", err)
	}
	if _, err := service.MarkRead(context.Background(), scope, "missing"); err != ErrNotificationNotFound {
		t.Fatalf("missing mark-read error = %v", err)
	}
}

type actorNotificationWriterStub struct {
	events []assignmentNotificationEvent
}

func (s *actorNotificationWriterStub) RecordActorNotification(_ context.Context, _ workflowruntime.OutboxEvent, event assignmentNotificationEvent) error {
	s.events = append(s.events, event)
	return nil
}

func TestInAppNotificationConsumerUsesCanonicalAssignmentEventDecoder(t *testing.T) {
	writer := &actorNotificationWriterStub{}
	consumer := NewInAppNotificationConsumer(writer)
	payload, err := json.Marshal(map[string]any{
		"matter": map[string]any{"id": "matter-1"},
		"previous_owner_principal_id": "00000000-0000-4000-8000-000000000001",
		"owner_principal_id": "00000000-0000-4000-8000-000000000002",
	})
	if err != nil {
		t.Fatal(err)
	}
	event := workflowruntime.OutboxEvent{
		AggregateType: "MATTER", AggregateID: "matter-1", EventType: "MATTER_OWNER_CHANGED", Payload: payload,
	}
	if err := consumer.Publish(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if len(writer.events) != 1 || writer.events[0].NotificationKind != matterOwnerNotificationKind ||
		writer.events[0].PrincipalID != "00000000-0000-4000-8000-000000000002" {
		t.Fatalf("projected notifications = %#v", writer.events)
	}

	if err := consumer.Publish(context.Background(), workflowruntime.OutboxEvent{EventType: "UNRELATED"}); err != nil {
		t.Fatal(err)
	}
	if len(writer.events) != 1 {
		t.Fatalf("unrelated event produced notification: %#v", writer.events)
	}
}
