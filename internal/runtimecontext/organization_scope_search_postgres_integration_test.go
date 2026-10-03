//go:build postgres && postgresintegration

package runtimecontext

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestOrganizationScopeSearchScalesBeyondCompactHierarchy(t *testing.T) {
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
		tenantID        = "8f300000-0000-4000-8000-000000000001"
		entityID        = "8f300000-0000-4000-8000-000000000002"
		globalPrincipal = "8f300000-0000-4000-8000-000000000003"
		localPrincipal  = "8f300000-0000-4000-8000-000000000004"
		globalPosition  = "8f300000-0000-4000-8000-000000000005"
		localPosition   = "8f300000-0000-4000-8000-000000000006"
		roleID          = "8f300000-0000-4000-8000-000000000007"
		rootScopeID     = "8f300000-0000-4000-8000-000000000008"
	)
	cleanup := func(cleanCtx context.Context) {
		_, _ = pool.Exec(cleanCtx, `DELETE FROM position_role_bindings WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM org_positions WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM role_templates WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM principals WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM organization_scopes WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM legal_entities WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	}
	cleanup(ctx)
	t.Cleanup(func() { cleanup(context.Background()) })

	now := time.Now().UTC().Truncate(time.Second)
	if _, err = pool.Exec(ctx, `
		INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'scope-scale-test','Scope Scale Test');
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from)
		VALUES($2::uuid,$1::uuid,'SCALE-NG','Scale Nigeria','NG',$8);
		INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES
			($3::uuid,$1::uuid,'PERSON','Global reader','ACTIVE',$8),
			($4::uuid,$1::uuid,'PERSON','Local reader','ACTIVE',$8);
		INSERT INTO role_templates(id,tenant_id,code,name,capabilities,valid_from)
		VALUES($6::uuid,$1::uuid,'SCOPE_READER','Scope reader',ARRAY['CONFIG_READ','OVERSIGHT_READ'],$8);
		INSERT INTO organization_scopes(
			id,tenant_id,legal_entity_id,code,name,kind,department_path,origin,status,valid_from
		) VALUES($7::uuid,$1::uuid,$2::uuid,'BANK','Bank','BUSINESS_UNIT',ARRAY['BANK'],'MANAGED','ACTIVE',$8);
		INSERT INTO organization_scopes(
			tenant_id,legal_entity_id,parent_scope_id,code,name,kind,department_path,origin,status,valid_from
		)
		SELECT $1::uuid,$2::uuid,$7::uuid,
		       'SITE' || lpad(value::text,5,'0'),
		       'Site ' || lpad(value::text,5,'0'),
		       'BRANCH',
		       ARRAY['BANK','SITE' || lpad(value::text,5,'0')],
		       'MANAGED','ACTIVE',$8
		FROM generate_series(1,19999) value;
		INSERT INTO org_positions(
			id,tenant_id,legal_entity_id,code,title,occupant_principal_id,department_path,valid_from
		) VALUES($5::uuid,$1::uuid,$2::uuid,'GLOBAL','Global reader',$3::uuid,ARRAY[]::text[],$8);
		INSERT INTO position_role_bindings(tenant_id,position_id,role_template_id,valid_from)
		VALUES($1::uuid,$5::uuid,$6::uuid,$8)
	`, pgx.QueryExecModeSimpleProtocol, tenantID, entityID, globalPrincipal, localPrincipal, globalPosition, roleID, rootScopeID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}

	var localScopeID string
	if err = pool.QueryRow(ctx, `
		SELECT id::text FROM organization_scopes
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND code='SITE00001'
	`, tenantID, entityID).Scan(&localScopeID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `
		INSERT INTO org_positions(
			id,tenant_id,legal_entity_id,code,title,occupant_principal_id,department_path,organization_scope_id,valid_from
		) VALUES($1::uuid,$2::uuid,$3::uuid,'LOCAL','Local reader',$4::uuid,ARRAY['BANK','SITE00001'],$5::uuid,$6);
		INSERT INTO position_role_bindings(tenant_id,position_id,role_template_id,valid_from)
		VALUES($2::uuid,$1::uuid,$7::uuid,$6)
	`, pgx.QueryExecModeSimpleProtocol, localPosition, tenantID, entityID, localPrincipal, localScopeID, now.Add(-time.Hour), roleID); err != nil {
		t.Fatal(err)
	}

	resolver := NewPostgresResolver(pool)
	globalScope := Scope{TenantID: "scope-scale-test", LegalEntityID: "SCALE-NG", PrincipalID: globalPrincipal}
	hierarchy, err := resolver.ResolveHierarchy(ctx, globalScope)
	if err != nil {
		t.Fatal(err)
	}
	if hierarchy.State != HierarchyComplete || !hierarchy.OrganizationScopesTruncated || len(hierarchy.OrganizationScopes) != maxOrganizationScopes {
		t.Fatalf("compact hierarchy state=%q truncated=%v count=%d", hierarchy.State, hierarchy.OrganizationScopesTruncated, len(hierarchy.OrganizationScopes))
	}

	page, err := resolver.SearchOrganizationScopes(ctx, globalScope, "SITE19999", 20)
	if err != nil {
		t.Fatal(err)
	}
	if page.HasMore || len(page.Items) != 1 || page.Items[0].Code != "SITE19999" || !page.Items[0].Filterable {
		t.Fatalf("global search page = %#v", page)
	}

	localScope := Scope{TenantID: "scope-scale-test", LegalEntityID: "SCALE-NG", PrincipalID: localPrincipal}
	forbiddenPage, err := resolver.SearchOrganizationScopes(ctx, localScope, "SITE19999", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(forbiddenPage.Items) != 0 {
		t.Fatalf("local reader discovered sibling scope: %#v", forbiddenPage.Items)
	}

	var plan []byte
	if err = pool.QueryRow(ctx, `
		EXPLAIN (FORMAT JSON)
		SELECT id
		FROM organization_scopes
		WHERE tenant_id=$1::uuid
		  AND legal_entity_id=$2::uuid
		  AND status='ACTIVE'
		  AND valid_until IS NULL
		  AND search_document @@ websearch_to_tsquery('simple'::regconfig,'SITE19999')
		LIMIT 20
	`, tenantID, entityID).Scan(&plan); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(plan, []byte("organization_scopes_search_idx")) {
		t.Fatalf("organization search plan did not use search index: %s", plan)
	}
}
