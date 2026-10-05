//go:build postgres && postgresintegration

package attention

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestNotificationRetentionPrunesOnlyExpiredDeliveryMetadata(t *testing.T) {
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
		tenantID    = "96666666-6666-4666-8666-666666666601"
		entityID    = "96666666-6666-4666-8666-666666666602"
		principalID = "96666666-6666-4666-8666-666666666603"
	)
	now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	old := now.Add(-120 * 24 * time.Hour)
	current := now.Add(-24 * time.Hour)

	_, _ = pool.Exec(ctx, `DELETE FROM notification_email_deliveries WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM in_app_notifications WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM principals WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM legal_entities WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM notification_email_deliveries WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM in_app_notifications WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM principals WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM legal_entities WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	})

	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'notification-retention','Notification retention')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from)
		VALUES($1::uuid,$2::uuid,'RET-NG','Retention Nigeria','NG',$3)`, entityID, tenantID, old.Add(-24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from)
		VALUES($1::uuid,$2::uuid,'PERSON','Retention owner','ACTIVE',$3)`, principalID, tenantID, old.Add(-24*time.Hour)); err != nil {
		t.Fatal(err)
	}

	for index, occurred := range []time.Time{old, current} {
		eventID := []string{
			"96666666-6666-4666-8666-666666666611",
			"96666666-6666-4666-8666-666666666612",
		}[index]
		subjectID := []string{
			"96666666-6666-4666-8666-666666666621",
			"96666666-6666-4666-8666-666666666622",
		}[index]
		if _, err := pool.Exec(ctx, `
			INSERT INTO in_app_notifications(
				tenant_id,legal_entity_id,outbox_event_id,principal_id,notification_kind,
				subject_type,subject_id,title,summary,action_path,occurred_at,created_at
			) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,'ATTENTION_TEST','RISK',$5::uuid,
			         'Test notice','Current state only','#risks/test',$6,$6)`,
			tenantID, entityID, eventID, principalID, subjectID, occurred); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO notification_email_deliveries(
				tenant_id,principal_id,delivery_class,digest_date,status,
				first_attempted_at,last_attempted_at,created_at,updated_at
			) VALUES($1::uuid,$2::uuid,'DAILY_DIGEST',$3::date,'DELIVERED',$4,$4,$4,$4)`,
			tenantID, principalID, occurred, occurred); err != nil {
			t.Fatal(err)
		}
	}

	maintainer := NewNotificationRetentionMaintainer(pool, 90*24*time.Hour)
	processed, err := maintainer.Maintain(ctx, now, 20)
	if err != nil {
		t.Fatal(err)
	}
	if processed != 2 {
		t.Fatalf("processed=%d want 2", processed)
	}

	var inAppCount, emailCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM in_app_notifications WHERE tenant_id=$1::uuid`, tenantID).Scan(&inAppCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM notification_email_deliveries WHERE tenant_id=$1::uuid`, tenantID).Scan(&emailCount); err != nil {
		t.Fatal(err)
	}
	if inAppCount != 1 || emailCount != 1 {
		t.Fatalf("retained metadata in_app=%d email=%d want 1/1", inAppCount, emailCount)
	}
}
