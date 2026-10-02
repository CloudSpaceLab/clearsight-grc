package workflow

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNotificationServiceScopesPagesAndMarksRead(t *testing.T) {
	repo := NewMemoryRepository(nil)
	base := time.Date(2026, 10, 2, 6, 0, 0, 0, time.UTC)
	records := []inAppNotificationRecord{
		{TenantID: "bank", LegalEntityID: "entity-ng", PrincipalID: "person-a", OutboxEventID: "10000000-0000-4000-8000-000000000003", Kind: matterOwnerNotificationKind, Title: "Third", SubjectType: "MATTER", SubjectID: "20000000-0000-4000-8000-000000000003", ActionPath: "#work/matters/20000000-0000-4000-8000-000000000003", OccurredAt: base.Add(3 * time.Minute)},
		{TenantID: "bank", LegalEntityID: "entity-ng", PrincipalID: "person-a", OutboxEventID: "10000000-0000-4000-8000-000000000002", Kind: matterOwnerNotificationKind, Title: "Second", SubjectType: "MATTER", SubjectID: "20000000-0000-4000-8000-000000000002", ActionPath: "#work/matters/20000000-0000-4000-8000-000000000002", OccurredAt: base.Add(2 * time.Minute)},
		{TenantID: "bank", LegalEntityID: "entity-ng", PrincipalID: "person-a", OutboxEventID: "10000000-0000-4000-8000-000000000001", Kind: matterOwnerNotificationKind, Title: "First", SubjectType: "MATTER", SubjectID: "20000000-0000-4000-8000-000000000001", ActionPath: "#work/matters/20000000-0000-4000-8000-000000000001", OccurredAt: base.Add(time.Minute)},
		{TenantID: "bank", LegalEntityID: "entity-ng", PrincipalID: "person-b", OutboxEventID: "10000000-0000-4000-8000-000000000004", Kind: matterOwnerNotificationKind, Title: "Other actor", SubjectType: "MATTER", SubjectID: "20000000-0000-4000-8000-000000000004", ActionPath: "#work/matters/20000000-0000-4000-8000-000000000004", OccurredAt: base.Add(4 * time.Minute)},
		{TenantID: "bank", LegalEntityID: "entity-gh", PrincipalID: "person-a", OutboxEventID: "10000000-0000-4000-8000-000000000005", Kind: matterOwnerNotificationKind, Title: "Sibling entity", SubjectType: "MATTER", SubjectID: "20000000-0000-4000-8000-000000000005", ActionPath: "#work/matters/20000000-0000-4000-8000-000000000005", OccurredAt: base.Add(5 * time.Minute)},
	}
	for _, record := range records {
		if err := repo.StoreInAppNotification(context.Background(), record); err != nil {
			t.Fatal(err)
		}
	}
	service := NewService(repo)
	scope := NotificationFilter{TenantID: "bank", LegalEntityID: "entity-ng", PrincipalID: "person-a", Limit: 2}

	first, err := service.ListNotifications(context.Background(), scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 2 || first.Items[0].Title != "Third" || first.Items[1].Title != "Second" {
		t.Fatalf("unexpected first page: %#v", first.Items)
	}
	if first.UnreadCount != 3 || first.NextCursor == "" {
		t.Fatalf("first page metadata = %#v", first)
	}

	second, err := service.ListNotifications(context.Background(), NotificationFilter{
		TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID, PrincipalID: scope.PrincipalID,
		Limit: 2, Cursor: first.NextCursor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Items) != 1 || second.Items[0].Title != "First" || second.NextCursor != "" {
		t.Fatalf("unexpected second page: %#v", second)
	}

	readAt := base.Add(10 * time.Minute)
	marked, err := service.MarkNotificationRead(context.Background(), scope, first.Items[0].ID, readAt)
	if err != nil {
		t.Fatal(err)
	}
	if marked.ReadAt == nil || !marked.ReadAt.Equal(readAt) {
		t.Fatalf("read time = %v", marked.ReadAt)
	}
	markedAgain, err := service.MarkNotificationRead(context.Background(), scope, first.Items[0].ID, readAt.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if markedAgain.ReadAt == nil || !markedAgain.ReadAt.Equal(readAt) {
		t.Fatalf("idempotent read time = %v", markedAgain.ReadAt)
	}

	unread, err := service.ListNotifications(context.Background(), NotificationFilter{
		TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID, PrincipalID: scope.PrincipalID,
		UnreadOnly: true, Limit: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if unread.UnreadCount != 2 || len(unread.Items) != 2 {
		t.Fatalf("unread page = %#v", unread)
	}
	if _, err := service.MarkNotificationRead(context.Background(), NotificationFilter{
		TenantID: "bank", LegalEntityID: "entity-ng", PrincipalID: "person-b",
	}, first.Items[0].ID, readAt); !errors.Is(err, ErrNotificationNotFound) {
		t.Fatalf("cross-actor mark error = %v", err)
	}
}

func TestNotificationServiceRejectsInvalidCursor(t *testing.T) {
	service := NewService(NewMemoryRepository(nil))
	_, err := service.ListNotifications(context.Background(), NotificationFilter{
		TenantID: "bank", LegalEntityID: "entity-ng", PrincipalID: "person-a", Cursor: "not-a-cursor",
	})
	if !errors.Is(err, ErrNotificationInvalidCursor) {
		t.Fatalf("cursor error = %v", err)
	}
}
