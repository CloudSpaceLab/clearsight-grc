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

func TestPostgresLeaseGenerationFencesStaleSameWorker(t *testing.T) {
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
		tenantID   = "67666666-6666-7666-8666-666666666661"
		workflowID = "67666666-6666-7666-8666-666666666662"
		subjectID  = "67666666-6666-7666-8666-666666666663"
		timerID    = "67666666-6666-7666-8666-666666666664"
		outboxID   = "67666666-6666-7666-8666-666666666665"
	)

	now := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	// Runtime claims are global; isolate this assertion from unfinished work
	// retained by earlier serialized integration packages.
	if _, err := pool.Exec(ctx, `UPDATE workflow_timers SET state='CANCELLED',locked_by=NULL,lease_until=NULL WHERE state IN ('READY','CLAIMED')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE outbox_events SET published_at=$1,locked_by=NULL,lease_until=NULL WHERE published_at IS NULL AND dead_lettered_at IS NULL`, now); err != nil {
		t.Fatal(err)
	}
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID) })
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'runtime-lease-fencing-test','Runtime Lease Fencing Test')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO workflow_instances(id,tenant_id,kind,subject_type,subject_id,state,policy_version) VALUES($1::uuid,$2::uuid,'TEST','MATTER',$3::uuid,'ACTIVE','test:v1')`, workflowID, tenantID, subjectID); err != nil {
		t.Fatal(err)
	}

	repo := NewPostgresRepository(pool)
	if _, err := repo.ScheduleTimer(ctx, Timer{
		ID: timerID, TenantID: "runtime-lease-fencing-test", WorkflowID: workflowID,
		Type: "SOURCE_PULL", DueAt: now, DedupeKey: "lease-fenced-timer", Payload: json.RawMessage(`{}`),
	}); err != nil {
		t.Fatal(err)
	}
	firstTimers, err := repo.ClaimDueTimers(ctx, "same-worker", now, time.Minute, 1)
	if err != nil || len(firstTimers) != 1 || firstTimers[0].ID != timerID {
		t.Fatalf("first timer claim: %v %#v", err, firstTimers)
	}
	secondTimers, err := repo.ClaimDueTimers(ctx, "same-worker", now.Add(2*time.Minute), time.Minute, 1)
	if err != nil || len(secondTimers) != 1 || secondTimers[0].ID != timerID {
		t.Fatalf("second timer claim: %v %#v", err, secondTimers)
	}
	if firstTimers[0].LeaseUntil == nil || secondTimers[0].LeaseUntil == nil || firstTimers[0].LeaseUntil.Equal(*secondTimers[0].LeaseUntil) {
		t.Fatalf("timer lease generation did not change: first=%#v second=%#v", firstTimers[0], secondTimers[0])
	}
	staleEvent := OutboxEvent{ID: outboxID, TenantID: "runtime-lease-fencing-test", AggregateType: "WORKFLOW", AggregateID: workflowID, EventType: "SourcePull", Payload: json.RawMessage(`{}`), OccurredAt: now.Add(2 * time.Minute)}
	if err := repo.CompleteTimer(ctx, firstTimers[0], staleEvent, now.Add(2*time.Minute+10*time.Second)); err == nil {
		t.Fatal("stale same-worker timer claim completed the newer PostgreSQL lease")
	}
	if _, err := repo.FailTimer(ctx, firstTimers[0], 3, "STALE", now.Add(2*time.Minute+10*time.Second), now.Add(4*time.Minute)); err == nil {
		t.Fatal("stale same-worker timer claim failed the newer PostgreSQL lease")
	}
	if err := repo.CompleteTimer(ctx, secondTimers[0], staleEvent, now.Add(2*time.Minute+10*time.Second)); err != nil {
		t.Fatalf("current timer claim could not complete: %v", err)
	}

	firstEvents, err := repo.ClaimOutbox(ctx, "same-worker", now.Add(2*time.Minute+20*time.Second), time.Minute, 1)
	if err != nil || len(firstEvents) != 1 || firstEvents[0].ID != outboxID {
		t.Fatalf("first outbox claim: %v %#v", err, firstEvents)
	}
	secondEvents, err := repo.ClaimOutbox(ctx, "same-worker", now.Add(4*time.Minute), time.Minute, 1)
	if err != nil || len(secondEvents) != 1 || secondEvents[0].ID != outboxID {
		t.Fatalf("second outbox claim: %v %#v", err, secondEvents)
	}
	if firstEvents[0].LeaseUntil == nil || secondEvents[0].LeaseUntil == nil || firstEvents[0].LeaseUntil.Equal(*secondEvents[0].LeaseUntil) {
		t.Fatalf("outbox lease generation did not change: first=%#v second=%#v", firstEvents[0], secondEvents[0])
	}
	at := now.Add(4*time.Minute + 10*time.Second)
	if err := repo.MarkPublished(ctx, firstEvents[0], at); err == nil {
		t.Fatal("stale same-worker outbox claim published the newer PostgreSQL lease")
	}
	if _, err := repo.MarkFailed(ctx, firstEvents[0], 3, "STALE", at, now.Add(6*time.Minute)); err == nil {
		t.Fatal("stale same-worker outbox claim failed the newer PostgreSQL lease")
	}
	if err := repo.MarkPublished(ctx, secondEvents[0], at); err != nil {
		t.Fatalf("current outbox claim could not publish: %v", err)
	}
}
