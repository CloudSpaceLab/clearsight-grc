//go:build postgres && postgresintegration

package scimapi

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/access"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSCIMProvisioningFeedsScopedLocalAccess(t *testing.T) {
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
		tenantID  = "8a222222-2222-7222-8222-222222222221"
		entityID  = "8a222222-2222-7222-8222-222222222222"
		roleID    = "8a222222-2222-7222-8222-222222222223"
		sourceID  = "8a222222-2222-7222-8222-222222222224"
		bindingID = "8a222222-2222-7222-8222-222222222225"
	)
	cleanup := func(cleanCtx context.Context) {
		_, _ = pool.Exec(cleanCtx, `DELETE FROM directory_group_role_bindings WHERE id=$1::uuid`, bindingID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM directory_groups WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM scim_users WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM principal_identities WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM principals WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM scim_sources WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM role_templates WHERE id=$1::uuid`, roleID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM legal_entities WHERE id=$1::uuid`, entityID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	}
	cleanup(ctx)
	t.Cleanup(func() { cleanup(context.Background()) })

	now := time.Now().UTC().Truncate(time.Second)
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'eia-scim-test','EIA SCIM Test')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2::uuid,'BANK-NG','Bank NG','NG',$3)`, entityID, tenantID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO role_templates(id,tenant_id,code,name,description,responsibilities,capabilities,valid_from) VALUES($1::uuid,$2::uuid,'RISK_REVIEWER','Risk reviewer','',ARRAY['REVIEWER'],ARRAY['CONFIG_READ'],$3)`, roleID, tenantID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	token := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	digest := HashToken(token)
	if _, err := pool.Exec(ctx, `INSERT INTO scim_sources(id,tenant_id,code,token_hash,identity_issuer,subject_attribute) VALUES($1::uuid,$2::uuid,'entra',$3,'https://issuer.example','externalId')`, sourceID, tenantID, digest[:]); err != nil {
		t.Fatal(err)
	}

	repository := NewPostgresRepository(pool)
	source, err := repository.AuthenticateSource(ctx, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	user, err := repository.CreateUser(ctx, source, User{ExternalID: "oidc-alice", UserName: "alice@example.com", DisplayName: "Alice", Active: true})
	if err != nil {
		t.Fatal(err)
	}
	group, err := repository.CreateGroup(ctx, source, Group{ExternalID: "risk-group", DisplayName: "Risk Reviewers", Members: []GroupMember{{UserID: user.ID}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO directory_group_role_bindings(id,tenant_id,group_id,role_template_id,legal_entity_id,department_path,valid_from)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,ARRAY['BANK','RISK'],$6)`,
		bindingID, tenantID, group.ID, roleID, entityID, now.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}

	resolver := access.NewPostgresResolver(pool)
	resolved, err := resolver.ResolveOIDC(ctx, "eia-scim-test", "BANK-NG", "https://issuer.example", "oidc-alice")
	if err != nil {
		t.Fatalf("provisioned group binding should make legal entity eligible: %v", err)
	}
	actor := identity.Actor{PermissionCodes: resolved.PermissionCodes, DepartmentGrants: resolved.DepartmentGrants}
	if identity.HasPermission(actor, identity.PermissionConfigRead) {
		t.Fatal("department-scoped directory role leaked into global permission")
	}
	if !identity.HasDepartmentPermission(actor, []string{"BANK", "RISK"}, identity.PermissionConfigRead) {
		t.Fatalf("expected exact department capability, got %#v", resolved.DepartmentGrants)
	}
	if len(resolved.RoleCodes) != 0 {
		t.Fatalf("department directory role leaked into global role codes: %#v", resolved.RoleCodes)
	}

	user.Active = false
	if _, err := repository.ReplaceUser(ctx, source, user.ID, user); err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.ResolveOIDC(ctx, "eia-scim-test", "BANK-NG", "https://issuer.example", "oidc-alice"); !errors.Is(err, access.ErrIdentityNotProvisioned) {
		t.Fatalf("deactivated SCIM user must lose OIDC access, got %v", err)
	}

	user.Active = true
	if _, err := repository.ReplaceUser(ctx, source, user.ID, user); err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.ResolveOIDC(ctx, "eia-scim-test", "BANK-NG", "https://issuer.example", "oidc-alice"); err != nil {
		t.Fatalf("reactivated user should regain governed eligibility: %v", err)
	}
	if err := repository.DeleteGroup(ctx, source, group.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.ResolvePrincipal(ctx, "eia-scim-test", user.PrincipalID, "BANK-NG"); !errors.Is(err, access.ErrPrincipalUnavailable) {
		t.Fatalf("deleted group must remove group-derived legal entity eligibility, got %v", err)
	}
}
