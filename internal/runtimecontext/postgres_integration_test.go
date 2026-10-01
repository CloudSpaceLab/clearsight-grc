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
		tenantID         = "8f100000-0000-4000-8000-000000000001"
		entityID         = "8f100000-0000-4000-8000-000000000002"
		principalID      = "8f100000-0000-4000-8000-000000000003"
		eligibleEntityID = "8f100000-0000-4000-8000-000000000004"
		hiddenEntityID   = "8f100000-0000-4000-8000-000000000005"
		positionID       = "8f100000-0000-4000-8000-000000000006"
		otherTenant      = "8f200000-0000-4000-8000-000000000001"
	)
	cleanup := func(cleanCtx context.Context) {
		_, _ = pool.Exec(cleanCtx, `DELETE FROM org_positions WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM principals WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM legal_entities WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM tenants WHERE id IN ($1::uuid,$2::uuid)`, tenantID, otherTenant)
	}
	cleanup(ctx)
	t.Cleanup(func() { cleanup(context.Background()) })

	now := time.Now().UTC().Truncate(time.Second)
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES
		($1::uuid,'runtime-context-test','Reference Group'),
		($2::uuid,'runtime-context-other','Other Group')`, tenantID, otherTenant); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES
		($1::uuid,$2::uuid,'REFERENCE-NG','Reference Nigeria','NG',$5),
		($3::uuid,$2::uuid,'REFERENCE-GH','Reference Ghana','GH',$5),
		($4::uuid,$2::uuid,'REFERENCE-SA','Reference South Africa','ZA',$5)`,
		entityID, tenantID, eligibleEntityID, hiddenEntityID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from)
		VALUES($1::uuid,$2::uuid,'PERSON','Compliance Officer','ACTIVE',$3)`, principalID, tenantID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO org_positions(id,tenant_id,legal_entity_id,code,title,occupant_principal_id,valid_from)
		VALUES($1::uuid,$2::uuid,$3::uuid,'GROUP-RISK-GH','Ghana risk oversight',$4::uuid,$5)`,
		positionID, tenantID, eligibleEntityID, principalID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}

	resolver := NewPostgresResolver(pool)
	scope := Scope{TenantID: "runtime-context-test", LegalEntityID: "REFERENCE-NG", PrincipalID: principalID}
	value, err := resolver.Resolve(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	if value.TenantName != "Reference Group" || value.LegalEntityName != "Reference Nigeria" || value.PrincipalName != "Compliance Officer" {
		t.Fatalf("resolved context = %#v", value)
	}

	hierarchy, err := resolver.ResolveHierarchy(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	if hierarchy.State != HierarchyComplete {
		t.Fatalf("hierarchy state = %q", hierarchy.State)
	}
	if hierarchy.Root.ID != tenantID || hierarchy.Root.Code != "runtime-context-test" || hierarchy.Root.Name != "Reference Group" {
		t.Fatalf("hierarchy root = %#v", hierarchy.Root)
	}
	if hierarchy.Current.ID != entityID || hierarchy.Current.Code != "REFERENCE-NG" || !hierarchy.Current.Current {
		t.Fatalf("current scope = %#v", hierarchy.Current)
	}
	if len(hierarchy.LegalEntities) != 2 {
		t.Fatalf("legal entity count = %d, values = %#v", len(hierarchy.LegalEntities), hierarchy.LegalEntities)
	}
	if hierarchy.LegalEntities[0].ID != entityID || hierarchy.LegalEntities[1].ID != eligibleEntityID {
		t.Fatalf("eligible legal entities = %#v", hierarchy.LegalEntities)
	}
	for _, node := range hierarchy.LegalEntities {
		if node.ID == hiddenEntityID {
			t.Fatalf("unauthorized sibling leaked into hierarchy: %#v", node)
		}
	}

	_, err = resolver.Resolve(ctx, Scope{TenantID: "runtime-context-other", LegalEntityID: "REFERENCE-NG", PrincipalID: principalID})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-tenant resolve error = %v", err)
	}
	_, err = resolver.ResolveHierarchy(ctx, Scope{TenantID: "runtime-context-other", LegalEntityID: "REFERENCE-NG", PrincipalID: principalID})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-tenant hierarchy error = %v", err)
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
		if _, err := resolver.ResolveHierarchy(context.Background(), scope); !errors.Is(err, ErrInvalid) {
			t.Fatalf("hierarchy scope %#v error = %v", scope, err)
		}
	}
}
