//go:build postgres && postgresintegration

package formpolicy

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresMaintenanceLeaseRejectsStaleWorkerAndCanBeReclaimed(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	cleanupPolicyFixture(t, pool)
	t.Cleanup(func() { cleanupPolicyFixture(t, pool) })
	now := time.Now().UTC()
	seedPolicyFixture(t, ctx, pool, now)
	if _, err := pool.Exec(ctx, `INSERT INTO form_response_policy_maintenance_jobs(tenant_id,legal_entity_id,job_type,response_revision_id,due_at,state,created_at,updated_at) VALUES($1::uuid,$2::uuid,'RECONCILE',md5('form-policy:response:1')::uuid,$3,'READY',$3,$3)`, policyTenantID, policyEntityID, now); err != nil {
		t.Fatal(err)
	}
	repo := NewPostgresRepository(pool)
	claimed, err := repo.ClaimReconciliation(ctx, "worker-a", now, time.Minute, 10)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claimed=%#v err=%v", claimed, err)
	}
	if err := repo.CompleteReconciliation(ctx, claimed[0].ID, "worker-b", now); !errors.Is(err, ErrConflict) {
		t.Fatalf("wrong-worker completion err=%v", err)
	}
	beforeExpiry, err := repo.ClaimReconciliation(ctx, "worker-b", now.Add(30*time.Second), time.Minute, 10)
	if err != nil || len(beforeExpiry) != 0 {
		t.Fatalf("before expiry=%#v err=%v", beforeExpiry, err)
	}
	afterExpiry, err := repo.ClaimReconciliation(ctx, "worker-b", now.Add(2*time.Minute), time.Minute, 10)
	if err != nil || len(afterExpiry) != 1 || afterExpiry[0].ID != claimed[0].ID {
		t.Fatalf("after expiry=%#v err=%v", afterExpiry, err)
	}
	if err := repo.CompleteReconciliation(ctx, afterExpiry[0].ID, "worker-b", now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
}
