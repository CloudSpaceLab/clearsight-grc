//go:build postgres && postgresintegration

package activity

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresActivityFederatesOperationalRecoveryReceipts(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	const (
		tenantID        = "8c000000-0000-7000-8000-000000000001"
		actorID         = "8c000000-0000-7000-8000-000000000002"
		outboxReceiptID = "8c000000-0000-7000-8000-000000000003"
		outboxJobID     = "8c000000-0000-7000-8000-000000000004"
		timerReceiptID  = "8c000000-0000-7000-8000-000000000005"
		timerJobID      = "8c000000-0000-7000-8000-000000000006"
	)
	cleanup := func(cleanCtx context.Context) {
		_, _ = pool.Exec(cleanCtx, `DELETE FROM operational_recovery_events WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM principals WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	}
	cleanup(ctx)
	t.Cleanup(func() { cleanup(context.Background()) })

	now := time.Now().UTC().Truncate(time.Second)
	mustExecActivity(t, ctx, pool, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'activity-recovery','Activity Recovery')`, tenantID)
	mustExecActivity(t, ctx, pool, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($1::uuid,$2::uuid,'PERSON','Operations administrator','ACTIVE',$3)`, actorID, tenantID, now.Add(-time.Hour))
	mustExecActivity(t, ctx, pool, `INSERT INTO operational_recovery_events(id,tenant_id,queue,job_id,decision,previous_attempts,terminal_at,actor_principal_id,rationale,recovered_at) VALUES
		($1::uuid,$2::uuid,'outbox-delivery',$3::uuid,'RETRY',5,$4,$5::uuid,'Retry approved after the delivery dependency recovered.',$6),
		($7::uuid,$2::uuid,'workflow-timers',$8::uuid,'RETRY',3,$9,$5::uuid,'Retry approved after the workflow dependency recovered.',$10)`,
		outboxReceiptID, tenantID, outboxJobID, now.Add(-10*time.Minute), actorID, now,
		timerReceiptID, timerJobID, now.Add(-20*time.Minute), now.Add(-time.Minute))

	service := NewService(NewPostgresRepository(pool))
	page, err := service.List(ctx, Query{TenantID: tenantID, Category: CategorySystem, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 {
		t.Fatalf("expected two recovery events, got %#v", page)
	}
	outboxEvent := page.Items[0]
	if outboxEvent.ID != "recovery:"+outboxReceiptID || outboxEvent.EventType != "OUTBOX_DELIVERY_RETRIED" || outboxEvent.ObjectType != "RUNTIME_RECOVERY" || outboxEvent.ObjectID != outboxJobID || outboxEvent.Source != "OPERATIONAL_RECOVERY_EVENT" || outboxEvent.Outcome != OutcomeSucceeded || outboxEvent.ActorID != actorID || outboxEvent.ActorDisplayName != "Operations administrator" || outboxEvent.ActorKind != ActorInternalUser {
		t.Fatalf("unexpected outbox recovery event: %#v", outboxEvent)
	}
	timerEvent := page.Items[1]
	if timerEvent.ID != "recovery:"+timerReceiptID || timerEvent.EventType != "WORKFLOW_TIMER_RETRIED" || timerEvent.ObjectID != timerJobID {
		t.Fatalf("unexpected workflow timer recovery event: %#v", timerEvent)
	}

	got, err := service.Get(ctx, tenantID, timerEvent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != timerEvent.ID || got.Source != "OPERATIONAL_RECOVERY_EVENT" || got.Action != "Workflow timer retried" {
		t.Fatalf("unexpected recovery detail: %#v", got)
	}
}
