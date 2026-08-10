//go:build postgres && postgresintegration

package access

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/scimapi"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestIdentityAccessAdminRevokesSourceDerivedEligibilityWithoutDeletingPrincipal(t *testing.T) {
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
		tenantID   = "8a333333-3333-7333-8333-333333333331"
		entityID   = "8a333333-3333-7333-8333-333333333332"
		adminID    = "8a333333-3333-7333-8333-333333333333"
		roleID     = "8a333333-3333-7333-8333-333333333334"
	)
	cleanup := func(cleanCtx context.Context) {
		_, _ = pool.Exec(cleanCtx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	}
	cleanup(ctx)
	t.Cleanup(func() { cleanup(context.Background()) })

	now := time.Now().UTC().Truncate(time.Second)
	mustAdminExec(t, ctx, pool, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'eia-admin-test','EIA Admin Test')`, tenantID)
	mustAdminExec(t, ctx, pool, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2::uuid,'BANK-NG','Bank NG','NG',$3)`, entityID, tenantID, now.Add(-time.Hour))
	mustAdminExec(t, ctx, pool, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($1::uuid,$2::uuid,'PERSON','Identity administrator','ACTIVE',$3)`, adminID, tenantID, now.Add(-time.Hour))
	mustAdminExec(t, ctx, pool, `INSERT INTO role_templates(id,tenant_id,code,name,responsibilities,capabilities,valid_from) VALUES($1::uuid,$2::uuid,'RISK_REVIEWER','Risk reviewer',ARRAY['REVIEWER'],ARRAY['IDENTITY_READ'],$3)`, roleID, tenantID, now.Add(-time.Hour))

	admin := NewPostgresAdministrator(pool)
	token, digest, err := NewProvisioningToken()
	if err != nil || token == "" {
		t.Fatalf("generate token: %v", err)
	}
	source, err := admin.CreateSCIMSource(ctx, CreateSCIMSourceInput{
		TenantID: tenantID, Code: "ENTRA", IdentityIssuer: "https://login.example.test", SubjectAttribute: "externalId", ActorID: adminID,
	}, digest[:])
	if err != nil {
		t.Fatal(err)
	}

	scimRepo := scimapi.NewPostgresRepository(pool)
	scimSource, err := scimRepo.AuthenticateSource(ctx, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	user, err := scimRepo.CreateUser(ctx, scimSource, scimapi.User{ExternalID: "alice-subject", UserName: "alice@example.test", DisplayName: "Alice", Active: true})
	if err != nil {
		t.Fatal(err)
	}
	group, err := scimRepo.CreateGroup(ctx, scimSource, scimapi.Group{ExternalID: "risk-reviewers", DisplayName: "Risk Reviewers", Members: []scimapi.GroupMember{{UserID: user.ID}}})
	if err != nil {
		t.Fatal(err)
	}

	binding, err := admin.CreateGroupRoleBinding(ctx, CreateGroupRoleBindingInput{
		TenantID: tenantID, GroupID: group.ID, RoleTemplateID: roleID, LegalEntityID: entityID,
		DepartmentPath: []string{"BANK", "RISK"}, ActorID: adminID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := admin.CreateGroupRoleBinding(ctx, CreateGroupRoleBindingInput{
		TenantID: tenantID, GroupID: group.ID, RoleTemplateID: roleID, LegalEntityID: entityID,
		DepartmentPath: []string{"BANK", "RISK"}, ActorID: adminID,
	}); !errors.Is(err, ErrAdminConflict) {
		t.Fatalf("duplicate active mapping must conflict, got %v", err)
	}

	resolver := NewPostgresResolver(pool)
	resolved, err := resolver.ResolvePrincipal(ctx, tenantID, user.PrincipalID, entityID)
	if err != nil {
		t.Fatalf("source-derived access should resolve before revoke: %v", err)
	}
	actor := identity.Actor{DepartmentGrants: resolved.DepartmentGrants}
	if !identity.HasDepartmentPermission(actor, []string{"BANK", "RISK"}, identity.PermissionIdentityRead) {
		t.Fatalf("expected exact department identity read grant: %#v", resolved.DepartmentGrants)
	}

	overview, err := admin.Overview(ctx, tenantID, entityID, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(overview.Sources) != 1 || overview.Sources[0].ID != source.ID || overview.Sources[0].Status != "ACTIVE" {
		t.Fatalf("unexpected source overview: %#v", overview.Sources)
	}
	if len(overview.Bindings) != 1 || overview.Bindings[0].ID != binding.ID {
		t.Fatalf("unexpected binding overview: %#v", overview.Bindings)
	}

	if err := admin.RevokeSCIMSource(ctx, tenantID, source.ID, adminID); err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.ResolvePrincipal(ctx, tenantID, user.PrincipalID, entityID); !errors.Is(err, ErrPrincipalUnavailable) {
		t.Fatalf("revoked source must remove group-derived entity eligibility, got %v", err)
	}
	var principalStatus string
	if err := pool.QueryRow(ctx, `SELECT status FROM principals WHERE id=$1::uuid`, user.PrincipalID).Scan(&principalStatus); err != nil {
		t.Fatal(err)
	}
	if principalStatus != "ACTIVE" {
		t.Fatalf("source revoke must preserve historical principal state, got %s", principalStatus)
	}
	if _, err := scimRepo.AuthenticateSource(ctx, digest[:]); !errors.Is(err, scimapi.ErrNotFound) {
		t.Fatalf("revoked source token must stop provisioning authentication, got %v", err)
	}
	if err := admin.RetireGroupRoleBinding(ctx, tenantID, binding.ID, adminID); err != nil {
		t.Fatal(err)
	}
	var auditCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE tenant_id=$1::uuid AND purpose='IDENTITY_ACCESS_ADMIN'`, tenantID).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount < 4 {
		t.Fatalf("expected audited source/binding administration, got %d events", auditCount)
	}
}

func mustAdminExec(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(ctx, query, args...); err != nil {
		t.Fatal(err)
	}
}
