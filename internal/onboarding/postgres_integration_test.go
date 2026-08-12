//go:build postgres && postgresintegration

package onboarding

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresStateReturnsCanonicalTenantIdentity(t *testing.T) {
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
		tenantID    = "93333333-3333-7333-8333-333333333331"
		principalID = "93333333-3333-7333-8333-333333333332"
		tenantSlug  = "onboarding-canonical-test"
	)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID) })
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,'Onboarding Canonical Test')`, tenantID, tenantSlug); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name) VALUES($1::uuid,$2::uuid,'PERSON','Onboarding Actor')`, principalID, tenantID); err != nil {
		t.Fatal(err)
	}

	repository := NewPostgresRepository(pool)
	created, err := repository.Upsert(ctx, State{
		TenantID: tenantSlug, PrincipalID: principalID, GuideCode: "general-first-run", GuideVersion: 1,
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if created.TenantID != tenantID {
		t.Fatalf("created onboarding tenant identity = %q, want %q", created.TenantID, tenantID)
	}
	loaded, err := repository.Get(ctx, tenantSlug, principalID, "general-first-run")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.TenantID != tenantID {
		t.Fatalf("loaded onboarding tenant identity = %q, want %q", loaded.TenantID, tenantID)
	}
}
