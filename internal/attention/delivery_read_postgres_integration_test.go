//go:build postgres && postgresintegration

package attention

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestDeliveryReaderReturnsRedactedHealthAndRecordHistory(t *testing.T) {
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
		tenantID    = "95555555-5555-4555-8555-555555555501"
		entityID    = "95555555-5555-4555-8555-555555555502"
		principalID = "95555555-5555-4555-8555-555555555503"
		riskID      = "95555555-5555-4555-8555-555555555504"
		eventID     = "95555555-5555-4555-8555-555555555505"
		episodeID   = "95555555-5555-4555-8555-555555555506"
		sourceID    = "95555555-5555-4555-8555-555555555507"
		memberID    = "95555555-5555-4555-8555-555555555508"
	)
	now := time.Date(2026, 10, 5, 14, 0, 0, 0, time.UTC)
	cleanup := func(ctx context.Context) {
		_, _ = pool.Exec(ctx, `DELETE FROM notification_email_deliveries WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(ctx, `DELETE FROM in_app_notifications WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(ctx, `DELETE FROM attention_episodes WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(ctx, `DELETE FROM principals WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(ctx, `DELETE FROM legal_entities WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	}
	cleanup(ctx)
	t.Cleanup(func() { cleanup(context.Background()) })

	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'notification-read','Notification read')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from)
		VALUES($1::uuid,$2::uuid,'READ-NG','Read Nigeria','NG',$3)`, entityID, tenantID, now.Add(-24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from)
		VALUES($1::uuid,$2::uuid,'PERSON','Risk owner','ACTIVE',$3)`, principalID, tenantID, now.Add(-24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO attention_episodes(
			id,tenant_id,legal_entity_id,condition_key,member_id,subject_type,subject_id,state,
			last_condition_state,opened_source_id,last_source_id,opened_at,last_observed_at,notice_sequence,created_at,updated_at
		) VALUES($1::uuid,$2::uuid,$3::uuid,'indicator_breaches',$4::uuid,'RISK',$5::uuid,'OPEN',
		         'CRITICAL',$6::uuid,$6::uuid,$7,$7,1,$7,$7)`,
		episodeID, tenantID, entityID, memberID, riskID, sourceID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO in_app_notifications(
			tenant_id,legal_entity_id,outbox_event_id,principal_id,notification_kind,
			subject_type,subject_id,title,summary,action_path,occurred_at,created_at
		) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,'ATTENTION_INDICATOR_BREACHES_OPENED',
		         'RISK',$5::uuid,'Risk indicator breached','Open the record to review current state.','#risks/test',$6,$6)`,
		tenantID, entityID, eventID, principalID, riskID, now.Add(-30*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO notification_email_deliveries(
			tenant_id,legal_entity_id,episode_id,notice_sequence,source_event_id,principal_id,delivery_class,
			status,failure_code,first_attempted_at,last_attempted_at,created_at,updated_at
		) VALUES($1::uuid,$2::uuid,$3::uuid,1,$4::uuid,$5::uuid,'ATTENTION_CRITICAL',
		         'PERMANENT_FAILURE','SMTP_REJECTED',$6,$6,$6,$6)`,
		tenantID, entityID, episodeID, eventID, principalID, now.Add(-25*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO notification_email_deliveries(
			tenant_id,principal_id,delivery_class,digest_date,status,first_attempted_at,last_attempted_at,delivered_at,created_at,updated_at
		) VALUES($1::uuid,$2::uuid,'DAILY_DIGEST',$3::date,'DELIVERED',$3,$3,$3,$3,$3)`,
		tenantID, principalID, now.Add(-20*time.Minute)); err != nil {
		t.Fatal(err)
	}

	reader := NewPostgresDeliveryReader(pool)
	health, err := reader.Health(ctx, tenantID, now, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if health.Delivered != 1 || health.Failed != 1 || health.Retrying != 0 || len(health.Failures) != 1 {
		t.Fatalf("health=%#v", health)
	}
	if health.Failures[0].FailureCode != "SMTP_REJECTED" {
		t.Fatalf("failure=%#v", health.Failures[0])
	}

	history, err := reader.RecordHistory(ctx, tenantID, entityID, "RISK", riskID, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(history.Items) != 1 {
		t.Fatalf("history=%#v", history)
	}
	item := history.Items[0]
	if item.EventID != eventID || item.InAppDeliveries != 1 || item.EmailStatus != "PERMANENT_FAILURE" || item.EmailAttempts != 1 {
		t.Fatalf("history item=%#v", item)
	}
}
