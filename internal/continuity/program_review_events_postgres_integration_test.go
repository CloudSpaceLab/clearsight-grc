//go:build postgres && postgresintegration

package continuity

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestProgramReviewEventReadIsStrictlyBoundedAndNewestFirst(t *testing.T) {
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
		tenantID  = "93333333-3333-7333-8333-333333333331"
		programID = "93333333-3333-7333-8333-333333333332"
	)
	_, _ = pool.Exec(ctx, `DELETE FROM continuity_events WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM continuity_events WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	})
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'review-event-limit','Review Event Limit')`, tenantID); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	for version := int64(1); version <= 70; version++ {
		if _, err := pool.Exec(ctx, `
			INSERT INTO continuity_events(tenant_id,aggregate_type,aggregate_id,aggregate_version,event_type,payload,actor_type,occurred_at)
			VALUES($1::uuid,'PROGRAM',$2::uuid,$3,'REVIEW_TEST_CHANGE','{}'::jsonb,'SYSTEM',$4)`,
			tenantID, programID, version, now.Add(time.Duration(version)*time.Second)); err != nil {
			t.Fatal(err)
		}
	}

	repo := NewPostgresRepository(pool)
	events, truncated, err := repo.ProgramEventsAfterVersion(ctx, "review-event-limit", programID, 0, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 8 || !truncated {
		t.Fatalf("expected a strictly bounded truncated read, got len=%d truncated=%v", len(events), truncated)
	}
	if events[0].AggregateVersion != 70 || events[7].AggregateVersion != 63 {
		t.Fatalf("expected newest-first versions 70..63, got first=%d last=%d", events[0].AggregateVersion, events[7].AggregateVersion)
	}

	events, truncated, err = repo.ProgramEventsAfterVersion(ctx, "review-event-limit", programID, 68, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || truncated || events[0].AggregateVersion != 70 || events[1].AggregateVersion != 69 {
		t.Fatalf("expected complete newest-first tail, got events=%#v truncated=%v", events, truncated)
	}
}
