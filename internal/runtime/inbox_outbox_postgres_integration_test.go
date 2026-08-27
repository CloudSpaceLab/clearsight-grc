//go:build postgres && postgresintegration

package runtime

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const webhookRuntimeTenantID = "7f111111-1111-7111-8111-111111111111"

func TestRecordInboxWithOutboxIsAtomicAndReplaySafe(t *testing.T) {
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
	cleanupWebhookRuntimeFixture(ctx, pool)
	t.Cleanup(func() { cleanupWebhookRuntimeFixture(context.Background(), pool) })
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'webhook-runtime-test','Webhook runtime test')`, webhookRuntimeTenantID); err != nil {
		t.Fatal(err)
	}

	repository := NewPostgresRepository(pool)
	now := time.Date(2026, 8, 15, 13, 30, 0, 0, time.UTC)
	first := []InboxReceipt{
		{TenantID: webhookRuntimeTenantID, Consumer: "source-webhook-provider-v1", EventID: "provider-1"},
		{TenantID: webhookRuntimeTenantID, Consumer: "source-webhook-checkpoint-v1", EventID: "checkpoint-1"},
	}
	event := OutboxEvent{
		ID: "7f222222-2222-7222-8222-222222222222", TenantID: webhookRuntimeTenantID,
		AggregateType: "SOURCE_BINDING", AggregateID: "7f333333-3333-7333-8333-333333333333",
		EventType: "SourceBindingChanged", Payload: []byte(`{"binding_id":"7f333333-3333-7333-8333-333333333333"}`), OccurredAt: now,
	}
	created, err := repository.RecordInboxWithOutbox(ctx, first, event, now)
	if err != nil || !created {
		t.Fatalf("first acceptance created=%v err=%v", created, err)
	}
	assertWebhookRuntimeCounts(t, ctx, pool, 2, 1)

	replay := event
	replay.ID = "7f444444-4444-7444-8444-444444444444"
	created, err = repository.RecordInboxWithOutbox(ctx, first, replay, now.Add(time.Second))
	if err != nil || created {
		t.Fatalf("provider replay created=%v err=%v", created, err)
	}
	assertWebhookRuntimeCounts(t, ctx, pool, 2, 1)

	conflictReceipts := []InboxReceipt{
		{TenantID: webhookRuntimeTenantID, Consumer: "source-webhook-provider-v1", EventID: "provider-2"},
		{TenantID: webhookRuntimeTenantID, Consumer: "source-webhook-checkpoint-v1", EventID: "checkpoint-1"},
	}
	conflict := event
	conflict.ID = "7f555555-5555-7555-8555-555555555555"
	created, err = repository.RecordInboxWithOutbox(ctx, conflictReceipts, conflict, now.Add(2*time.Second))
	if created || !errors.Is(err, ErrInboxReceiptConflict) {
		t.Fatalf("secondary conflict created=%v err=%v", created, err)
	}
	assertWebhookRuntimeCounts(t, ctx, pool, 2, 1)
	var inserted bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM inbox_receipts WHERE tenant_id=$1::uuid AND consumer='source-webhook-provider-v1' AND event_id='provider-2')`, webhookRuntimeTenantID).Scan(&inserted); err != nil {
		t.Fatal(err)
	}
	if inserted {
		t.Fatal("provider receipt survived a rolled-back secondary conflict")
	}
}

func assertWebhookRuntimeCounts(t *testing.T, ctx context.Context, pool *pgxpool.Pool, inbox, outbox int) {
	t.Helper()
	var inboxCount, outboxCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM inbox_receipts WHERE tenant_id=$1::uuid`, webhookRuntimeTenantID).Scan(&inboxCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid`, webhookRuntimeTenantID).Scan(&outboxCount); err != nil {
		t.Fatal(err)
	}
	if inboxCount != inbox || outboxCount != outbox {
		t.Fatalf("durable counts inbox=%d/%d outbox=%d/%d", inboxCount, inbox, outboxCount, outbox)
	}
}

func cleanupWebhookRuntimeFixture(ctx context.Context, pool *pgxpool.Pool) {
	_, _ = pool.Exec(ctx, `DELETE FROM outbox_events WHERE tenant_id=$1::uuid`, webhookRuntimeTenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM inbox_receipts WHERE tenant_id=$1::uuid`, webhookRuntimeTenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, webhookRuntimeTenantID)
}
