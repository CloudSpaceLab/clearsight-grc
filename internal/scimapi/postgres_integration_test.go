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

	var tenantID, entityID, roleID, sourceID, bindingID string
	if err := pool.QueryRow(ctx, `SELECT gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text`).Scan(&tenantID, &entityID, &roleID, &sourceID, &bindingID); err != nil {
		t.Fatal(err)
	}
	tenantSlug := "scim-test-" + tenantID
	// Creating a legal entity writes immutable scope lineage. Preserve that
	// lineage, entity and tenant; unique IDs make the fixture safe to repeat.
	cleanup := func(cleanCtx context.Context) {
		for _, table := range []string{"directory_group_role_bindings", "directory_group_members", "directory_groups", "scim_users", "principal_identities", "principals", "scim_sources", "role_templates"} {
			if _, err := pool.Exec(cleanCtx, `DELETE FROM `+table+` WHERE tenant_id=$1::uuid`, tenantID); err != nil {
				t.Errorf("clean up %s: %v", table, err)
			}
		}
	}
	defer cleanup(context.Background())

	now := time.Now().UTC().Truncate(time.Second)
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,'EIA SCIM Test')`, tenantID, tenantSlug); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2::uuid,'BANK-NG','Bank NG','NG',$3)`, entityID, tenantID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO role_templates(id,tenant_id,code,name,description,responsibilities,capabilities,valid_from) VALUES($1::uuid,$2::uuid,'RISK_REVIEWER','Risk reviewer','',ARRAY['REVIEWER'],ARRAY['CONFIG_READ'],$3)`, roleID, tenantID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	token := "scim-test-token-" + tenantID
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
	resolved, err := resolver.ResolveOIDC(ctx, tenantSlug, "BANK-NG", "https://issuer.example", "oidc-alice")
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
	if _, err := resolver.ResolveOIDC(ctx, tenantSlug, "BANK-NG", "https://issuer.example", "oidc-alice"); !errors.Is(err, access.ErrIdentityNotProvisioned) {
		t.Fatalf("deactivated SCIM user must lose OIDC access, got %v", err)
	}

	user.Active = true
	if _, err := repository.ReplaceUser(ctx, source, user.ID, user); err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.ResolveOIDC(ctx, tenantSlug, "BANK-NG", "https://issuer.example", "oidc-alice"); err != nil {
		t.Fatalf("reactivated user should regain governed eligibility: %v", err)
	}
	if err := repository.DeleteGroup(ctx, source, group.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.ResolvePrincipal(ctx, tenantSlug, user.PrincipalID, "BANK-NG"); !errors.Is(err, access.ErrPrincipalUnavailable) {
		t.Fatalf("deleted group must remove group-derived legal entity eligibility, got %v", err)
	}
}

func TestSCIMProvisioningDoesNotDependOnUnusedAuthorityEpoch(t *testing.T) {
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
	const tenant = "8a333333-3333-7333-8333-333333333331"
	const sourceID = "8a333333-3333-7333-8333-333333333332"
	cleanup := func() {
		for _, table := range []string{"directory_group_members", "directory_groups", "scim_users", "principal_identities", "principals", "scim_sources"} {
			_, _ = pool.Exec(ctx, `DELETE FROM `+table+` WHERE tenant_id=$1::uuid`, tenant)
		}
		_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenant)
	}
	cleanup()
	defer cleanup()
	if _, err = pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'scim-fence-test','SCIM fence test')`, tenant); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO scim_sources(id,tenant_id,code,token_hash,subject_attribute) VALUES($1::uuid,$2::uuid,'scim-fence',$3,'userName')`, sourceID, tenant, make([]byte, 32)); err != nil {
		t.Fatal(err)
	}
	// No active command uses the unfinished epoch protocol. A missing epoch
	// must not prevent directory provisioning or access revocation.
	if _, err = pool.Exec(ctx, `DELETE FROM tenant_authority_epochs WHERE tenant_id=$1::uuid`, tenant); err != nil {
		t.Fatal(err)
	}
	repo := NewPostgresRepository(pool)
	source := Source{ID: sourceID, TenantID: tenant}
	user, err := repo.CreateUser(ctx, source, User{UserName: "scim-fence@example.com", Active: true})
	if err != nil {
		t.Fatal(err)
	}
	user.Active = false
	if _, err = repo.ReplaceUser(ctx, source, user.ID, user); err != nil {
		t.Fatal(err)
	}
	current, err := repo.GetUser(ctx, source, user.ID)
	if err != nil || current.Active {
		t.Fatalf("suspension must take effect: %#v, %v", current, err)
	}
	group, err := repo.CreateGroup(ctx, source, Group{DisplayName: "Risk reviewers", Members: []GroupMember{{UserID: user.ID}}})
	if err != nil {
		t.Fatal(err)
	}
	group.Members = nil
	if _, err = repo.ReplaceGroup(ctx, source, group.ID, group); err != nil {
		t.Fatal(err)
	}
	if err = repo.DeleteGroup(ctx, source, group.ID); err != nil {
		t.Fatal(err)
	}
	if err = repo.DeleteUser(ctx, source, user.ID); err != nil {
		t.Fatal(err)
	}
}
