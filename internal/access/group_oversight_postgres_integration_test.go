//go:build postgres && postgresintegration

package access

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresResolverGroupOversightRequiresLegalEntityWidePermission(t *testing.T) {
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
		tenantID    = "8c111111-1111-7111-8111-111111111111"
		entityA     = "8c111111-1111-7111-8111-111111111112"
		entityB     = "8c111111-1111-7111-8111-111111111113"
		entityC     = "8c111111-1111-7111-8111-111111111114"
		principalID = "8c111111-1111-7111-8111-111111111115"
		roleID      = "8c111111-1111-7111-8111-111111111116"
		positionA   = "8c111111-1111-7111-8111-111111111117"
		positionB   = "8c111111-1111-7111-8111-111111111118"
		positionC   = "8c111111-1111-7111-8111-111111111119"
		bindingA    = "8c111111-1111-7111-8111-111111111120"
		bindingB    = "8c111111-1111-7111-8111-111111111121"
		bindingC    = "8c111111-1111-7111-8111-111111111122"
	)
	cleanup := func(cleanCtx context.Context) {
		_, _ = pool.Exec(cleanCtx, `DELETE FROM position_role_bindings WHERE id IN ($1::uuid,$2::uuid,$3::uuid)`, bindingA, bindingB, bindingC)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM org_positions WHERE id IN ($1::uuid,$2::uuid,$3::uuid)`, positionA, positionB, positionC)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM role_templates WHERE id=$1::uuid`, roleID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM principals WHERE id=$1::uuid`, principalID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM legal_entities WHERE id IN ($1::uuid,$2::uuid,$3::uuid)`, entityA, entityB, entityC)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	}
	cleanup(ctx)
	t.Cleanup(func() { cleanup(context.Background()) })

	now := time.Now().UTC().Truncate(time.Second)
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'group-oversight-test','Group Oversight Test')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES
		($1::uuid,$4::uuid,'A','Alpha','NG',$5),
		($2::uuid,$4::uuid,'B','Beta','GH',$5),
		($3::uuid,$4::uuid,'C','Gamma','KE',$5)`, entityA, entityB, entityC, tenantID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($1::uuid,$2::uuid,'PERSON','Group CRO','ACTIVE',$3)`, principalID, tenantID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO role_templates(id,tenant_id,code,name,description,responsibilities,capabilities,valid_from) VALUES
		($1::uuid,$2::uuid,'GROUP_CRO','Group CRO','',ARRAY['OBSERVER'],ARRAY['OVERSIGHT_READ'],$3)`, roleID, tenantID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO org_positions(id,tenant_id,legal_entity_id,code,title,occupant_principal_id,department_path,valid_from) VALUES
		($1::uuid,$4::uuid,$5::uuid,'CRO-A','CRO A',$8::uuid,ARRAY[]::text[],$9),
		($2::uuid,$4::uuid,$6::uuid,'CRO-B','CRO B',$8::uuid,ARRAY[]::text[],$9),
		($3::uuid,$4::uuid,$7::uuid,'RISK-C','Risk C',$8::uuid,ARRAY['BANK','RISK'],$9)`, positionA, positionB, positionC, tenantID, entityA, entityB, entityC, principalID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO position_role_bindings(id,tenant_id,position_id,role_template_id,valid_from) VALUES
		($1::uuid,$4::uuid,$5::uuid,$8::uuid,$9),
		($2::uuid,$4::uuid,$6::uuid,$8::uuid,$9),
		($3::uuid,$4::uuid,$7::uuid,$8::uuid,$9)`, bindingA, bindingB, bindingC, tenantID, positionA, positionB, positionC, roleID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}

	resolver := NewPostgresResolver(pool)
	page, err := resolver.ResolveOversightLegalEntities(ctx, "group-oversight-test", principalID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if page.HasMore || page.TenantID != tenantID || page.TenantName != "Group Oversight Test" ||
		len(page.Items) != 2 || page.Items[0].ID != entityA || page.Items[1].ID != entityB {
		t.Fatalf("authorized Group scopes=%#v", page)
	}

	if _, err := pool.Exec(ctx, `UPDATE position_role_bindings SET valid_until=$2 WHERE id=$1::uuid`, bindingB, now); err != nil {
		t.Fatal(err)
	}
	page, err = resolver.ResolveOversightLegalEntities(ctx, "group-oversight-test", principalID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != entityA {
		t.Fatalf("expired Group access remained visible: %#v", page)
	}
}
