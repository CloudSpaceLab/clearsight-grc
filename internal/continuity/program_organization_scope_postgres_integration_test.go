//go:build postgres && postgresintegration

package continuity

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresProgramOrganizationScopeIsExactAndFilterable(t *testing.T) {
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
		tenantID         = "91222222-2222-7222-8222-222222222221"
		entityA          = "91222222-2222-7222-8222-222222222222"
		entityB          = "91222222-2222-7222-8222-222222222223"
		parentScope      = "91222222-2222-7222-8222-222222222224"
		childScope       = "91222222-2222-7222-8222-222222222225"
		siblingScope     = "91222222-2222-7222-8222-222222222226"
		otherEntityScope = "91222222-2222-7222-8222-222222222227"
	)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID) })

	now := time.Date(2026, 10, 3, 13, 0, 0, 0, time.UTC)
	if _, err = pool.Exec(ctx, `
		INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'program-scope-test','Program Scope Test');
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES
			($2::uuid,$1::uuid,'ENTITY-A','Entity A','NG',$8),
			($3::uuid,$1::uuid,'ENTITY-B','Entity B','GH',$8);
		INSERT INTO organization_scopes(
			id,tenant_id,legal_entity_id,parent_scope_id,code,name,kind,department_path,origin,status,valid_from
		) VALUES
			($4::uuid,$1::uuid,$2::uuid,NULL,'RISK','Risk','DEPARTMENT',ARRAY['BANK','RISK'],'MANAGED','ACTIVE',$8),
			($5::uuid,$1::uuid,$2::uuid,$4::uuid,'RISK-OPS','Risk Operations','DEPARTMENT',ARRAY['BANK','RISK','OPERATIONS'],'MANAGED','ACTIVE',$8),
			($6::uuid,$1::uuid,$2::uuid,NULL,'FINANCE','Finance','DEPARTMENT',ARRAY['BANK','FINANCE'],'MANAGED','ACTIVE',$8),
			($7::uuid,$1::uuid,$3::uuid,NULL,'OTHER-RISK','Other Risk','DEPARTMENT',ARRAY['BANK','RISK'],'MANAGED','ACTIVE',$8)
	`, pgx.QueryExecModeSimpleProtocol, tenantID, entityA, entityB, parentScope, childScope, siblingScope, otherEntityScope, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}

	actor := identity.WithActor(ctx, identity.Actor{TenantID: "program-scope-test", LegalEntityID: "ENTITY-A", PrincipalID: "scope-reader"})
	service := NewServiceWithClock(NewPostgresRepository(pool), func() time.Time { return now })
	created := make(map[string]ProgramAggregate)
	for _, spec := range []struct {
		code  string
		scope string
	}{
		{code: "PARENT", scope: parentScope},
		{code: "CHILD", scope: childScope},
		{code: "SIBLING", scope: siblingScope},
		{code: "UNATTRIBUTED"},
	} {
		value, createErr := service.CreateProgram(actor, CreateProgramInput{
			TenantID: "program-scope-test", LegalEntityID: entityB, OrganizationScopeID: spec.scope,
			Code: spec.code, Name: spec.code, Type: "ASSURANCE", OwningFunction: "Risk",
			Scope: json.RawMessage(`{}`), EffectiveFrom: now,
		})
		if createErr != nil {
			t.Fatalf("create %s: %v", spec.code, createErr)
		}
		created[spec.code] = value
	}
	if created["PARENT"].Program.LegalEntityID != entityA || created["PARENT"].Program.OrganizationScopeID != parentScope {
		t.Fatalf("canonical Program scope = %#v", created["PARENT"].Program)
	}

	if _, err = pool.Exec(ctx, `
		INSERT INTO programs(
			tenant_id,legal_entity_id,organization_scope_id,code,name,program_type,status,owning_function,scope,effective_from,created_at,updated_at,version
		) VALUES($1::uuid,$2::uuid,$3::uuid,'CROSS-SCOPE','Cross scope','ASSURANCE','DRAFT','Risk','{}'::jsonb,$4,$4,$4,1)
	`, tenantID, entityA, otherEntityScope, now); err == nil {
		t.Fatal("cross-entity Program organization scope was accepted")
	}

	page, err := service.ListProgramSummaries(actor, "program-scope-test", SummaryQuery{
		OrganizationScopeID:  parentScope,
		OrganizationScopeIDs: []string{parentScope, childScope},
		Limit:                 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if page.OrganizationScopeID != parentScope || len(page.Items) != 2 {
		t.Fatalf("organization scoped Program page = %#v", page)
	}
	seen := map[string]bool{}
	for _, item := range page.Items {
		seen[item.Program.Code] = true
		if item.Program.OrganizationScopeID != parentScope && item.Program.OrganizationScopeID != childScope {
			t.Fatalf("unexpected Program scope %q in %#v", item.Program.OrganizationScopeID, page)
		}
	}
	if !seen["PARENT"] || !seen["CHILD"] || seen["SIBLING"] || seen["UNATTRIBUTED"] {
		t.Fatalf("organization scope membership = %#v", seen)
	}

	detail, err := service.GetProgram(actor, "program-scope-test", created["SIBLING"].Program.ID)
	if err != nil || detail.Program.ID != created["SIBLING"].Program.ID {
		t.Fatalf("exact Program drill unexpectedly depended on list scope: %#v err=%v", detail.Program, err)
	}
}
