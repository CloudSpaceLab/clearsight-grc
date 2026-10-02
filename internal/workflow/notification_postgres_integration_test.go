//go:build postgres && postgresintegration

package workflow

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresInAppNotificationsStayActorAndEntityScoped(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	const (
		tenantID    = "98000000-0000-4000-8000-000000000001"
		entityNG    = "98000000-0000-4000-8000-000000000002"
		entityGH    = "98000000-0000-4000-8000-000000000003"
		principalA  = "98000000-0000-4000-8000-000000000004"
		principalB  = "98000000-0000-4000-8000-000000000005"
		eventA1     = "98000000-0000-4000-8000-000000000006"
		eventA2     = "98000000-0000-4000-8000-000000000007"
		eventB      = "98000000-0000-4000-8000-000000000008"
		eventSibling = "98000000-0000-4000-8000-000000000009"
		matterA1    = "98000000-0000-4000-8000-000000000011"
		matterA2    = "98000000-0000-4000-8000-000000000012"
		matterB     = "98000000-0000-4000-8000-000000000013"
		matterGH    = "98000000-0000-4000-8000-000000000014"
	)

	cleanup := func(cleanCtx context.Context) {
		_, _ = pool.Exec(cleanCtx, `DELETE FROM in_app_notifications WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM outbox_events WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM principals WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM legal_entities WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	}
	cleanup(ctx)
	t.Cleanup(func() { cleanup(context.Background()) })

	base := time.Date(2026, 10, 2, 6, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'notification-scope-test','Notification Scope Test')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES
		($1::uuid,$3::uuid,'NOTIFY-NG','Notification Nigeria','NG',$4),
		($2::uuid,$3::uuid,'NOTIFY-GH','Notification Ghana','GH',$4)`,
		entityNG, entityGH, tenantID, base.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES
		($1::uuid,$3::uuid,'PERSON','Person A','ACTIVE',$4),
		($2::uuid,$3::uuid,'PERSON','Person B','ACTIVE',$4)`,
		principalA, principalB, tenantID, base.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO outbox_events(id,tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at) VALUES
		($1::uuid,$5::uuid,'MATTER',$6::uuid,'MATTER_OWNER_CHANGED','{}'::jsonb,$10,$10),
		($2::uuid,$5::uuid,'MATTER',$7::uuid,'MATTER_OWNER_CHANGED','{}'::jsonb,$11,$11),
		($3::uuid,$5::uuid,'MATTER',$8::uuid,'MATTER_OWNER_CHANGED','{}'::jsonb,$12,$12),
		($4::uuid,$5::uuid,'MATTER',$9::uuid,'MATTER_OWNER_CHANGED','{}'::jsonb,$13,$13)`,
		eventA1, eventA2, eventB, eventSibling, tenantID, matterA1, matterA2, matterB, matterGH,
		base.Add(time.Minute), base.Add(2*time.Minute), base.Add(3*time.Minute), base.Add(4*time.Minute)); err != nil {
		t.Fatal(err)
	}

	repository := NewPostgresRepository(pool)
	store := func(eventID, entityID, principalID, matterID, title string, occurred time.Time) {
		t.Helper()
		if err := repository.StoreInAppNotification(ctx, inAppNotificationRecord{
			TenantID: "notification-scope-test", LegalEntityID: entityID, PrincipalID: principalID,
			OutboxEventID: eventID, Kind: matterOwnerNotificationKind, Title: title,
			SubjectType: "MATTER", SubjectID: matterID, ActionPath: "#work/matters/" + matterID, OccurredAt: occurred,
		}); err != nil {
			t.Fatal(err)
		}
	}
	store(eventA1, entityNG, principalA, matterA1, "First", base.Add(time.Minute))
	store(eventA2, entityNG, principalA, matterA2, "Second", base.Add(2*time.Minute))
	store(eventB, entityNG, principalB, matterB, "Other actor", base.Add(3*time.Minute))
	store(eventSibling, entityGH, principalA, matterGH, "Sibling entity", base.Add(4*time.Minute))

	service := NewService(repository)
	scopeA := NotificationFilter{TenantID: "notification-scope-test", LegalEntityID: entityNG, PrincipalID: principalA, Limit: 1}
	first, err := service.ListNotifications(ctx, scopeA)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 1 || first.Items[0].Title != "Second" || first.UnreadCount != 2 || first.NextCursor == "" {
		t.Fatalf("first page = %#v", first)
	}
	second, err := service.ListNotifications(ctx, NotificationFilter{
		TenantID: scopeA.TenantID, LegalEntityID: scopeA.LegalEntityID, PrincipalID: scopeA.PrincipalID,
		Limit: 1, Cursor: first.NextCursor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Items) != 1 || second.Items[0].Title != "First" || second.NextCursor != "" {
		t.Fatalf("second page = %#v", second)
	}

	scopeB := NotificationFilter{TenantID: "notification-scope-test", LegalEntityID: entityNG, PrincipalID: principalB, Limit: 10}
	pageB, err := service.ListNotifications(ctx, scopeB)
	if err != nil {
		t.Fatal(err)
	}
	if len(pageB.Items) != 1 {
		t.Fatalf("actor B page = %#v", pageB)
	}
	if _, err := service.MarkNotificationRead(ctx, scopeA, pageB.Items[0].ID, base.Add(time.Hour)); !errors.Is(err, ErrNotificationNotFound) {
		t.Fatalf("cross-actor mark error = %v", err)
	}

	if _, err := service.MarkNotificationRead(ctx, scopeA, first.Items[0].ID, base.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	unread, err := service.ListNotifications(ctx, NotificationFilter{
		TenantID: scopeA.TenantID, LegalEntityID: scopeA.LegalEntityID, PrincipalID: scopeA.PrincipalID,
		UnreadOnly: true, Limit: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if unread.UnreadCount != 1 || len(unread.Items) != 1 || unread.Items[0].Title != "First" {
		t.Fatalf("unread page = %#v", unread)
	}
}
