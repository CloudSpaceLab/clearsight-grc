//go:build postgres && postgresintegration

package workflow

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// An empty source is sufficient to exercise both fully authorized SQL
// selections, parameter bindings, repeatable-read aggregation and shape.
func TestActorWorkDueCountsPostgresQueriesUseAuthorizedSelections(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	service := NewService(NewPostgresRepository(pool))
	value, err := service.ReadActorWorkDueCounts(ctx,
		"missing-work-count-tenant-20261026",
		"missing-entity",
		"00000000-0000-4000-8000-000000000123",
	)
	if err != nil {
		t.Fatalf("authorized workflow/proposal COUNT SQL failed: %v", err)
	}
	if value.Total != 0 || value.Overdue != 0 || value.DueSoon != 0 || value.Undated != 0 {
		t.Fatalf("nonexistent source must be empty: %#v", value)
	}
	if value.CountBasis != "SOURCE_SNAPSHOT" || value.Coverage != "WORKFLOW_ASSIGNMENTS_ONLY" {
		t.Fatalf("count receipt lost source qualification: %#v", value)
	}
}

// The update commits after the service receives the request but before its
// repository establishes the snapshot. It must be included as current work.
func TestActorWorkCountsIncludeCommitBeforeSnapshot(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	var tenant, entity, principal, matter, flow, task string
	if err = pool.QueryRow(ctx, `SELECT gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text`).Scan(&tenant, &entity, &principal, &matter, &flow, &task); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, query := range []string{`DELETE FROM workflow_tasks WHERE tenant_id=$1::uuid`, `DELETE FROM workflow_instances WHERE tenant_id=$1::uuid`, `DELETE FROM matters WHERE tenant_id=$1::uuid`, `DELETE FROM principals WHERE tenant_id=$1::uuid`, `DELETE FROM legal_entities WHERE tenant_id=$1::uuid`, `DELETE FROM tenants WHERE id=$1::uuid`} {
			if _, e := pool.Exec(context.Background(), query, tenant); e != nil {
				t.Errorf("fixture cleanup: %v", e)
			}
		}
	})
	must := func(query string, args ...any) {
		t.Helper()
		if _, e := pool.Exec(ctx, query, args...); e != nil {
			t.Fatal(e)
		}
	}
	must(`INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$1,'Count race')`, tenant)
	must(`INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2::uuid,'COUNTS','Counts','NG',clock_timestamp()-interval '1 day')`, entity, tenant)
	must(`INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($1::uuid,$2::uuid,'PERSON','Reviewer','ACTIVE',clock_timestamp()-interval '1 day')`, principal, tenant)
	must(`INSERT INTO matters(id,tenant_id,legal_entity_id,reference,matter_type,status,priority,title,summary,scope) VALUES($1::uuid,$2::uuid,$3::uuid,'COUNT-RACE','CONTROL_GAP','TRIAGE',3,'Count race','Count race','{"access":"INTERNAL"}'::jsonb)`, matter, tenant, entity)
	must(`INSERT INTO workflow_instances(id,tenant_id,kind,subject_type,subject_id,state,policy_version) VALUES($1::uuid,$2::uuid,'MATTER_LIFECYCLE','MATTER',$3::uuid,'ACTIVE','v1')`, flow, tenant, matter)
	must(`INSERT INTO workflow_tasks(id,tenant_id,workflow_id,step_key,responsibility,principal_id,title,status,created_at,updated_at) VALUES($1::uuid,$2::uuid,$3::uuid,'review','REVIEWER',$4::uuid,'Review','READY',clock_timestamp()-interval '1 hour',clock_timestamp()-interval '1 hour')`, task, tenant, flow, principal)
	repo := &waitingCountRepository{PostgresRepository: NewPostgresRepository(pool), entered: make(chan struct{}), resume: make(chan struct{})}
	type result struct {
		counts ActorWorkDueCounts
		err    error
	}
	done := make(chan result, 1)
	go func() {
		counts, e := NewService(repo).ReadActorWorkDueCounts(ctx, tenant, entity, principal)
		done <- result{counts, e}
	}()
	select {
	case <-repo.entered:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	var changedAt time.Time
	if err := pool.QueryRow(ctx, `UPDATE workflow_tasks SET updated_at=clock_timestamp(),title='Updated' WHERE id=$1::uuid RETURNING updated_at`, task).Scan(&changedAt); err != nil {
		t.Fatal(err)
	}
	close(repo.resume)
	select {
	case r := <-done:
		if r.err != nil {
			t.Fatal(r.err)
		}
		if r.counts.Total != 1 || r.counts.AsOf.Before(changedAt) {
			t.Fatalf("visible update lost from snapshot: update=%v result=%+v", changedAt, r.counts)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}

	// A completion committed after the snapshot must not alter that snapshot;
	// the next independent count must see it.
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	var snapshot string
	var asOf time.Time
	if err := tx.QueryRow(ctx, `SELECT statement_timestamp(), pg_current_snapshot()::text`).Scan(&asOf, &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot == "" {
		t.Fatal("missing PostgreSQL snapshot")
	}
	must(`UPDATE workflow_tasks SET status='COMPLETED',completed_at=clock_timestamp(),updated_at=clock_timestamp() WHERE id=$1::uuid`, task)
	filter := ListFilter{TenantID: tenant, LegalEntityID: entity, PrincipalID: principal, VisibleActorWorkOnly: true, ActiveOnly: true}
	old, err := countActorWorkDueAtSnapshot(ctx, tx, filter, asOf)
	if err != nil || old.Total != 1 {
		t.Fatalf("previous snapshot: %+v, %v", old, err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	newer, err := NewService(NewPostgresRepository(pool)).ReadActorWorkDueCounts(ctx, tenant, entity, principal)
	if err != nil || newer.Total != 0 {
		t.Fatalf("new snapshot: %+v, %v", newer, err)
	}
}

type waitingCountRepository struct {
	*PostgresRepository
	entered, resume chan struct{}
}

func (r *waitingCountRepository) CountActorWorkDue(ctx context.Context, filter ListFilter) (ActorWorkDueCounts, error) {
	close(r.entered)
	select {
	case <-r.resume:
		return r.PostgresRepository.CountActorWorkDue(ctx, filter)
	case <-ctx.Done():
		return ActorWorkDueCounts{}, ctx.Err()
	}
}
