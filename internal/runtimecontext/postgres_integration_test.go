//go:build postgres && postgresintegration

package runtimecontext

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresResolverUsesExactVerifiedScope(t *testing.T) {
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
		tenantID    = "8f100000-0000-4000-8000-000000000001"
		entityID    = "8f100000-0000-4000-8000-000000000002"
		principalID = "8f100000-0000-4000-8000-000000000003"
		otherTenant = "8f200000-0000-4000-8000-000000000001"
	)
	cleanup := func(cleanCtx context.Context) {
		_, _ = pool.Exec(cleanCtx, `DELETE FROM tenants WHERE id IN ($1::uuid,$2::uuid)`, tenantID, otherTenant)
	}
	cleanup(ctx)
	t.Cleanup(func() { cleanup(context.Background()) })

	now := time.Now().UTC().Truncate(time.Second)
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES
		($1::uuid,'runtime-context-test','Reference Bank'),
		($2::uuid,'runtime-context-other','Other Bank')`, tenantID, otherTenant); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from)
		VALUES($1::uuid,$2::uuid,'REFERENCE-NG','Reference Bank Nigeria','NG',$3)`, entityID, tenantID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from)
		VALUES($1::uuid,$2::uuid,'PERSON','Compliance Officer','ACTIVE',$3)`, principalID, tenantID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}

	resolver := NewPostgresResolver(pool)
	value, err := resolver.Resolve(ctx, Scope{TenantID: "runtime-context-test", LegalEntityID: "REFERENCE-NG", PrincipalID: principalID})
	if err != nil {
		t.Fatal(err)
	}
	if value.TenantName != "Reference Bank" || value.LegalEntityName != "Reference Bank Nigeria" || value.PrincipalName != "Compliance Officer" {
		t.Fatalf("resolved context = %#v", value)
	}

	_, err = resolver.Resolve(ctx, Scope{TenantID: "runtime-context-other", LegalEntityID: "REFERENCE-NG", PrincipalID: principalID})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-tenant resolve error = %v", err)
	}
}

func TestPostgresResolverRejectsInactiveOrIncompleteScope(t *testing.T) {
	resolver := NewPostgresResolver(nil)
	for _, scope := range []Scope{
		{},
		{TenantID: "tenant", LegalEntityID: "entity"},
		{TenantID: "tenant", PrincipalID: "principal"},
	} {
		if _, err := resolver.Resolve(context.Background(), scope); !errors.Is(err, ErrInvalid) {
			t.Fatalf("scope %#v error = %v", scope, err)
		}
	}
}
