//go:build postgres && postgresintegration

package access

import (
	"context"
	"errors"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresResolverKeepsDepartmentCapabilitiesScoped(t *testing.T) {
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
		tenantID        = "8a111111-1111-7111-8111-111111111111"
		entityID        = "8a111111-1111-7111-8111-111111111112"
		principalID     = "8a111111-1111-7111-8111-111111111113"
		globalRoleID    = "8a111111-1111-7111-8111-111111111114"
		paymentRoleID   = "8a111111-1111-7111-8111-111111111115"
		globalPosID     = "8a111111-1111-7111-8111-111111111116"
		paymentPosID    = "8a111111-1111-7111-8111-111111111117"
		globalBindID    = "8a111111-1111-7111-8111-111111111118"
		paymentBindID   = "8a111111-1111-7111-8111-111111111119"
		identityID      = "8a111111-1111-7111-8111-111111111120"
		secondEntityID  = "8a111111-1111-7111-8111-111111111121"
		deniedEntityID  = "8a111111-1111-7111-8111-111111111122"
		secondPosID     = "8a111111-1111-7111-8111-111111111123"
		secondBindID    = "8a111111-1111-7111-8111-111111111124"
	)
	cleanup := func(cleanCtx context.Context) {
		_, _ = pool.Exec(cleanCtx, `DELETE FROM position_role_bindings WHERE id IN ($1::uuid,$2::uuid,$3::uuid)`, globalBindID, paymentBindID, secondBindID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM org_positions WHERE id IN ($1::uuid,$2::uuid,$3::uuid)`, globalPosID, paymentPosID, secondPosID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM principal_identities WHERE id=$1::uuid`, identityID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM role_templates WHERE id IN ($1::uuid,$2::uuid)`, globalRoleID, paymentRoleID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM principals WHERE id=$1::uuid`, principalID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM legal_entities WHERE id IN ($1::uuid,$2::uuid,$3::uuid)`, entityID, secondEntityID, deniedEntityID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	}
	cleanup(ctx)
	t.Cleanup(func() { cleanup(context.Background()) })

	now := time.Now().UTC().Truncate(time.Second)
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'eia-access-test','EIA Access Test')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES
		($1::uuid,$4::uuid,'BANK-NG','Bank NG','NG',$5),
		($2::uuid,$4::uuid,'BANK-GH','Bank GH','GH',$5),
		($3::uuid,$4::uuid,'BANK-KE','Bank KE','KE',$5)`, entityID, secondEntityID, deniedEntityID, tenantID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($1::uuid,$2::uuid,'PERSON','Alice Reviewer','ACTIVE',$3)`, principalID, tenantID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO role_templates(id,tenant_id,code,name,description,responsibilities,capabilities,valid_from) VALUES
		($1::uuid,$3::uuid,'BANK_READER','Bank reader','',ARRAY['OBSERVER'],ARRAY['CONFIG_READ'],$4),
		($2::uuid,$3::uuid,'PAYMENT_REVIEWER','Payment reviewer','',ARRAY['REVIEWER'],ARRAY['CONFIG_WRITE','EVIDENCE_REVIEW'],$4)`, globalRoleID, paymentRoleID, tenantID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO org_positions(id,tenant_id,legal_entity_id,code,title,occupant_principal_id,department_path,valid_from) VALUES
		($1::uuid,$4::uuid,$5::uuid,'BANK-READER-NG','Bank reader NG',$6::uuid,ARRAY[]::text[],$7),
		($2::uuid,$4::uuid,$5::uuid,'PAY-REVIEW','Payments reviewer',$6::uuid,ARRAY['BANK','OPERATIONS','PAYMENTS'],$7),
		($3::uuid,$4::uuid,$8::uuid,'BANK-READER-GH','Bank reader GH',$6::uuid,ARRAY[]::text[],$7)`, globalPosID, paymentPosID, secondPosID, tenantID, entityID, principalID, now.Add(-time.Hour), secondEntityID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO position_role_bindings(id,tenant_id,position_id,role_template_id,valid_from) VALUES
		($1::uuid,$4::uuid,$5::uuid,$6::uuid,$9),
		($2::uuid,$4::uuid,$7::uuid,$8::uuid,$9),
		($3::uuid,$4::uuid,$10::uuid,$6::uuid,$9)`, globalBindID, paymentBindID, secondBindID, tenantID, globalPosID, globalRoleID, paymentPosID, paymentRoleID, now.Add(-time.Hour), secondPosID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO principal_identities(id,tenant_id,principal_id,issuer,subject) VALUES($1::uuid,$2::uuid,$3::uuid,'https://issuer.example','alice-subject')`, identityID, tenantID, principalID); err != nil {
		t.Fatal(err)
	}

	resolver := NewPostgresResolver(pool)
	resolved, err := resolver.ResolveOIDC(ctx, "eia-access-test", "BANK-NG", "https://issuer.example", "alice-subject")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(resolved.RoleCodes, "BANK_READER") || slices.Contains(resolved.RoleCodes, "PAYMENT_REVIEWER") {
		t.Fatalf("department role leaked into global role codes: %#v", resolved.RoleCodes)
	}
	actor := identity.Actor{PermissionCodes: resolved.PermissionCodes, DepartmentGrants: resolved.DepartmentGrants}
	if !identity.HasPermission(actor, identity.PermissionConfigRead) {
		t.Fatalf("expected legal-entity-wide CONFIG_READ, got %#v", resolved.PermissionCodes)
	}
	if identity.HasPermission(actor, identity.PermissionConfigWrite) {
		t.Fatalf("department CONFIG_WRITE leaked into global permissions: %#v", resolved.PermissionCodes)
	}
	payments := []string{"BANK", "OPERATIONS", "PAYMENTS"}
	if !identity.HasDepartmentPermission(actor, payments, identity.PermissionConfigWrite) {
		t.Fatalf("expected exact Payments CONFIG_WRITE grant: %#v", resolved.DepartmentGrants)
	}
	if identity.HasDepartmentPermission(actor, []string{"BANK", "OPERATIONS"}, identity.PermissionConfigWrite) {
		t.Fatal("parent department inherited a child capability")
	}

	second, err := resolver.ResolveOIDC(ctx, "eia-access-test", "BANK-GH", "https://issuer.example", "alice-subject")
	if err != nil {
		t.Fatalf("same OIDC identity should resolve another authorized legal entity: %v", err)
	}
	if second.LegalEntityID != "BANK-GH" || !slices.Contains(second.RoleCodes, "BANK_READER") || len(second.DepartmentGrants) != 0 {
		t.Fatalf("unexpected second legal-entity resolution: %#v", second)
	}
	if _, err := resolver.ResolveOIDC(ctx, "eia-access-test", "BANK-KE", "https://issuer.example", "alice-subject"); !errors.Is(err, ErrPrincipalUnavailable) {
		t.Fatalf("identity must not enter a legal entity without a current position, got %v", err)
	}

	if _, err := pool.Exec(ctx, `UPDATE principals SET status='INACTIVE' WHERE id=$1::uuid`, principalID); err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.ResolvePrincipal(ctx, "eia-access-test", principalID, "BANK-NG"); !errors.Is(err, ErrPrincipalUnavailable) {
		t.Fatalf("expected deactivated principal to lose access, got %v", err)
	}
}
