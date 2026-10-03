//go:build postgres && postgresintegration

package access

import (
	"context"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresResolveLegalEntityAccessExcludesDepartmentGrants(t *testing.T) {
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
		tenantID    = "8f500000-0000-4000-8000-000000000001"
		principalID = "8f500000-0000-4000-8000-000000000002"
		roleID      = "8f500000-0000-4000-8000-000000000003"
		entityA     = "8f500000-0000-4000-8000-000000000010"
		entityB     = "8f500000-0000-4000-8000-000000000011"
		entityC     = "8f500000-0000-4000-8000-000000000012"
		positionA   = "8f500000-0000-4000-8000-000000000020"
		positionB   = "8f500000-0000-4000-8000-000000000021"
	)
	cleanup := func(cleanCtx context.Context) {
		_, _ = pool.Exec(cleanCtx, `DELETE FROM position_role_bindings WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM org_positions WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM role_templates WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM principals WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM legal_entities WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	}
	cleanup(ctx)
	t.Cleanup(func() { cleanup(context.Background()) })

	validFrom := time.Now().UTC().Add(-time.Hour)
	if _, err = pool.Exec(ctx, `
		INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'entity-access-test','Entity Access Test');
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES
			($4::uuid,$1::uuid,'ENTITY-A','Entity A','NG',$9),
			($5::uuid,$1::uuid,'ENTITY-B','Entity B','GH',$9),
			($6::uuid,$1::uuid,'ENTITY-C','Entity C','KE',$9);
		INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from)
		VALUES($2::uuid,$1::uuid,'PERSON','Group reader','ACTIVE',$9);
		INSERT INTO role_templates(id,tenant_id,code,name,capabilities,valid_from)
		VALUES($3::uuid,$1::uuid,'ENTITY_OVERSIGHT','Entity oversight',ARRAY['OVERSIGHT_READ'],$9);
		INSERT INTO org_positions(id,tenant_id,legal_entity_id,code,title,occupant_principal_id,department_path,valid_from) VALUES
			($7::uuid,$1::uuid,$4::uuid,'A-GLOBAL','A global',$2::uuid,ARRAY[]::text[],$9),
			($8::uuid,$1::uuid,$5::uuid,'B-RISK','B risk',$2::uuid,ARRAY['BANK','RISK'],$9);
		INSERT INTO position_role_bindings(tenant_id,position_id,role_template_id,valid_from) VALUES
			($1::uuid,$7::uuid,$3::uuid,$9),
			($1::uuid,$8::uuid,$3::uuid,$9);
	`, pgx.QueryExecModeSimpleProtocol,
		tenantID, principalID, roleID, entityA, entityB, entityC, positionA, positionB, validFrom); err != nil {
		t.Fatal(err)
	}

	values, err := NewPostgresResolver(pool).ResolveLegalEntityAccess(ctx, "entity-access-test", principalID, []string{entityA, entityB, entityC})
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 3 {
		t.Fatalf("entity access count = %d values=%#v", len(values), values)
	}
	permissions := make(map[string][]string, len(values))
	for _, value := range values {
		permissions[value.LegalEntityID] = value.PermissionCodes
	}
	if !slices.Contains(permissions[entityA], identity.PermissionOversightRead) {
		t.Fatalf("global entity permission missing: %#v", permissions)
	}
	if slices.Contains(permissions[entityB], identity.PermissionOversightRead) {
		t.Fatalf("department grant leaked as entity-wide permission: %#v", permissions)
	}
	if len(permissions[entityC]) != 0 {
		t.Fatalf("unassigned entity gained permissions: %#v", permissions)
	}
}
