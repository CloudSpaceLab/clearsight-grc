//go:build postgres && postgresintegration

package runtime

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresPoisonWorkBecomesTerminalAndIsNotReclaimed(t *testing.T) {
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
		tenantID   = "66666666-6666-7666-8666-666666666661"
		workflowID = "66666666-6666-7666-8666-666666666662"
		subjectID  = "66666666-6666-7666-8666-666666666663"
		timerID    = "66666666-6666-7666-8666-666666666664"
		outboxID   = "66666666-6666-7666-8666-666666666665"
	)

	now := time.Now().UTC()
	// The postgresintegration stage intentionally shares one database across
	// packages. Runtime claims are global by design, so quarantine unfinished
	// work left by earlier packages before asserting this test's claim order.
	if _, err := pool.Exec(ctx, `UPDATE workflow_timers SET state='CANCELLED',locked_by=NULL,lease_until=NULL WHERE state IN ('READY','CLAIMED')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE outbox_events SET published_at=$1,locked_by=NULL,lease_until=NULL WHERE published_at IS NULL AND dead_lettered_at IS NULL`, now); err != nil {
		t.Fatal(err)
	}
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID) })
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'runtime-failure-isolation-test','Runtime Failure Isolation Test')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO workflow_instances(id,tenant_id,kind,subject_type,subject_id,state,policy_version) VALUES($1::uuid,$2::uuid,'TEST','MATTER',$3::uuid,'ACTIVE','test:v1')`, workflowID, tenantID, subjectID); err != nil {
		t.Fatal(err)
	}

	repo := NewPostgresRepository(pool)
	if _, err := repo.ScheduleTimer(ctx, Timer{
		ID: timerID, TenantID: "runtime-failure-isolation-test", WorkflowID: workflowID,
		Type: "REMINDER", DueAt: now, DedupeKey: "terminal-timer", Payload: json.RawMessage(`{}`),
	}); err != nil {
		t.Fatal(err)
	}
	timers, err := repo.ClaimDueTimers(ctx, "timer-worker", now, time.Minute, 1)
	if err != nil || len(timers) != 1 || timers[0].ID != timerID {
		t.Fatalf("claim timer: %v %#v", err, timers)
	}
	terminal, err := repo.FailTimer(ctx, timers[0], 1, "poison timer", now, now.Add(time.Minute))
	if err != nil || !terminal {
		t.Fatalf("terminal timer failure: terminal=%v err=%v", terminal, err)
	}
	storedTimer, err := repo.timerByDedupe(ctx, "runtime-failure-isolation-test", "terminal-timer")
	if err != nil {
		t.Fatal(err)
	}
	if storedTimer.State != TimerFailed || storedTimer.FailedAt == nil || storedTimer.LastError != "poison timer" {
		t.Fatalf("unexpected failed timer: %#v", storedTimer)
	}
	timers, err = repo.ClaimDueTimers(ctx, "timer-worker-2", now.Add(2*time.Minute), time.Minute, 1)
	if err != nil || len(timers) != 0 {
		t.Fatalf("terminal timer was reclaimed: %v %#v", err, timers)
	}

	if _, err := pool.Exec(ctx, `INSERT INTO outbox_events(id,tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at) VALUES($1::uuid,$2::uuid,'WORKFLOW',$3::uuid,'PoisonEvent','{}'::jsonb,$4,$4,$4)`, outboxID, tenantID, workflowID, now); err != nil {
		t.Fatal(err)
	}
	events, err := repo.ClaimOutbox(ctx, "outbox-worker", now, time.Minute, 1)
	if err != nil || len(events) != 1 || events[0].ID != outboxID {
		t.Fatalf("claim outbox: %v %#v", err, events)
	}
	terminal, err = repo.MarkFailed(ctx, events[0], 1, "poison event", now, now.Add(time.Minute))
	if err != nil || !terminal {
		t.Fatalf("dead-letter outbox: terminal=%v err=%v", terminal, err)
	}
	events, err = repo.ClaimOutbox(ctx, "outbox-worker-2", now.Add(2*time.Minute), time.Minute, 1)
	if err != nil || len(events) != 0 {
		t.Fatalf("dead-letter event was reclaimed: %v %#v", err, events)
	}

	timerHealth, err := repo.TimerQueueHealth(ctx)
	if err != nil || timerHealth.Terminal < 1 || timerHealth.HighestAttempts < 1 {
		t.Fatalf("unexpected timer health: %#v err=%v", timerHealth, err)
	}
	outboxHealth, err := repo.OutboxQueueHealth(ctx)
	if err != nil || outboxHealth.Terminal < 1 || outboxHealth.HighestAttempts < 1 {
		t.Fatalf("unexpected outbox health: %#v err=%v", outboxHealth, err)
	}
}
