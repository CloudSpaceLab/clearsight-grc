//go:build postgres && postgresintegration

package access

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/organization"
	"github.com/CloudSpaceLab/clearsight-grc/internal/scimapi"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestIdentityAccessAdminRevokesSourceDerivedGrantWithoutDeletingPrincipal(t *testing.T) {
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
		tenantID = "8a333333-3333-7333-8333-333333333331"
		entityID = "8a333333-3333-7333-8333-333333333332"
		adminID  = "8a333333-3333-7333-8333-333333333333"
		roleID   = "8a333333-3333-7333-8333-333333333334"
		croID    = "8a333333-3333-7333-8333-333333333335"
		ownerID  = "8a333333-3333-7333-8333-333333333336"
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
	mustAdminExec(t, ctx, pool, `INSERT INTO org_positions(id,tenant_id,legal_entity_id,code,title,function_name,department_path,occupant_principal_id,valid_from,version) VALUES($1::uuid,$2::uuid,$3::uuid,'CRO','Chief Risk Officer','Risk',ARRAY['BANK','RISK'],$4::uuid,$5,3)`, croID, tenantID, entityID, adminID, now.Add(-time.Hour))
	mustAdminExec(t, ctx, pool, `INSERT INTO position_role_bindings(tenant_id,position_id,role_template_id,valid_from) VALUES($1::uuid,$2::uuid,$3::uuid,$4)`, tenantID, croID, roleID, now.Add(-time.Hour))

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
	mustAdminExec(t, ctx, pool, `INSERT INTO org_positions(id,tenant_id,legal_entity_id,code,title,function_name,parent_position_id,department_path,occupant_principal_id,valid_from,version) VALUES($1::uuid,$2::uuid,$3::uuid,'PROGRAM_OWNER','Program Owner','Risk Operations',$4::uuid,ARRAY['BANK','RISK','OPERATIONS'],$5::uuid,$6,4)`, ownerID, tenantID, entityID, croID, user.PrincipalID, now.Add(-time.Hour))
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
	if overview.Bindings[0].OrganizationScopeID == "" {
		t.Fatalf("department-scoped group mapping must have stable organization scope: %#v", overview.Bindings[0])
	}
	if len(overview.Positions) != 2 {
		t.Fatalf("expected two active positions, got %#v", overview.Positions)
	}
	if len(overview.OrganizationScopes) != 3 || overview.OrganizationScopesTruncated {
		t.Fatalf("expected stable organization scope inventory, got %#v truncated=%v", overview.OrganizationScopes, overview.OrganizationScopesTruncated)
	}
	if overview.Positions[0].OrganizationScopeID == "" || overview.Positions[1].OrganizationScopeID == "" {
		t.Fatalf("positions must resolve stable organization scopes: %#v", overview.Positions)
	}
	if overview.Positions[1].ParentPositionID != croID || overview.Positions[1].OccupantName != "Alice" || overview.Positions[1].Version != 4 {
		t.Fatalf("expected exact reporting-line inventory, got %#v", overview.Positions[1])
	}

	if err := admin.RevokeSCIMSource(ctx, tenantID, source.ID, adminID); err != nil {
		t.Fatal(err)
	}
	revoked, err := resolver.ResolvePrincipal(ctx, tenantID, user.PrincipalID, entityID)
	if err != nil {
		t.Fatalf("active position must preserve entity eligibility after source revoke: %v", err)
	}
	if identity.HasDepartmentPermission(identity.Actor{DepartmentGrants: revoked.DepartmentGrants}, []string{"BANK", "RISK"}, identity.PermissionIdentityRead) {
		t.Fatalf("revoked source must remove its group-derived permission: %#v", revoked.DepartmentGrants)
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
	var decisionCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM governance_decisions WHERE tenant_id=$1::uuid AND object_type IN ('SCIM_SOURCE','DIRECTORY_GROUP_ROLE_BINDING')`, tenantID).Scan(&decisionCount); err != nil {
		t.Fatal(err)
	}
	if decisionCount < 4 {
		t.Fatalf("expected governed source/binding administration history, got %d decisions", decisionCount)
	}
}


func TestOrganizationScopeManagementRequiresIndependentApprovalAndProtectsLiveReferences(t *testing.T) {
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
		tenantID  = "8a444444-4444-7444-8444-444444444441"
		entityID  = "8a444444-4444-7444-8444-444444444442"
		makerID   = "8a444444-4444-7444-8444-444444444443"
		checkerID = "8a444444-4444-7444-8444-444444444444"
		positionID = "8a444444-4444-7444-8444-444444444445"
	)
	cleanup := func(cleanCtx context.Context) {
		_, _ = pool.Exec(cleanCtx, `DELETE FROM organization_scope_revisions WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM org_positions WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM organization_scopes WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM principals WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM legal_entities WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	}
	cleanup(ctx)
	t.Cleanup(func() { cleanup(context.Background()) })

	now := time.Now().UTC().Truncate(time.Second)
	mustAdminExec(t, ctx, pool, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'org-scope-admin','Org Scope Admin')`, tenantID)
	mustAdminExec(t, ctx, pool, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2::uuid,'BANK-NG','Bank NG','NG',$3)`, entityID, tenantID, now.Add(-time.Hour))
	mustAdminExec(t, ctx, pool, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES
		($1::uuid,$3::uuid,'PERSON','Maker','ACTIVE',$4),
		($2::uuid,$3::uuid,'PERSON','Checker','ACTIVE',$4)`, makerID, checkerID, tenantID, now.Add(-time.Hour))
	mustAdminExec(t, ctx, pool, `INSERT INTO org_positions(id,tenant_id,legal_entity_id,code,title,department_path,occupant_principal_id,valid_from)
		VALUES($1::uuid,$2::uuid,$3::uuid,'RISK-HEAD','Risk head',ARRAY['BANK','RISK'],$4::uuid,$5)`,
		positionID, tenantID, entityID, makerID, now.Add(-time.Hour))

	var riskScopeID string
	if err := pool.QueryRow(ctx, `
		SELECT id::text FROM organization_scopes
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND department_path=ARRAY['BANK','RISK']::text[]
		  AND status='ACTIVE' AND valid_until IS NULL`, tenantID, entityID).Scan(&riskScopeID); err != nil {
		t.Fatal(err)
	}

	admin := NewPostgresAdministrator(pool)
	revision, err := admin.ProposeOrganizationScope(ctx, ProposeOrganizationScopeInput{
		TenantID: tenantID, LegalEntityID: entityID, Operation: OrganizationScopeCreate,
		ParentScopeID: riskScopeID, Code: "LAGOS_ISLAND", Name: "Lagos Island",
		Kind: organization.ScopeKindBranch, ActorID: makerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if revision.Status != "PENDING" || revision.ScopeID == "" || revision.MakerID != makerID {
		t.Fatalf("unexpected revision: %#v", revision)
	}
	if err := admin.ApproveOrganizationScope(ctx, DecideOrganizationScopeInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: revision.ID, ActorID: makerID, Rationale: "self",
	}); !errors.Is(err, ErrAdminMakerChecker) {
		t.Fatalf("maker approval error=%v", err)
	}
	if err := admin.ApproveOrganizationScope(ctx, DecideOrganizationScopeInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: revision.ID, ActorID: checkerID, Rationale: "checked",
	}); err != nil {
		t.Fatal(err)
	}

	var childID string
	var childVersion int64
	var childPath []string
	if err := pool.QueryRow(ctx, `
		SELECT id::text,version,department_path FROM organization_scopes
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND code='LAGOS_ISLAND' AND status='ACTIVE'`,
		tenantID, entityID).Scan(&childID, &childVersion, &childPath); err != nil {
		t.Fatal(err)
	}
	if childID != revision.ScopeID || strings.Join(childPath, "/") != "BANK/RISK/LAGOS_ISLAND" {
		t.Fatalf("created scope id=%s path=%v revision=%#v", childID, childPath, revision)
	}

	livePositionID := "8a444444-4444-7444-8444-444444444446"
	mustAdminExec(t, ctx, pool, `INSERT INTO org_positions(id,tenant_id,legal_entity_id,code,title,organization_scope_id,department_path,valid_from)
		VALUES($1::uuid,$2::uuid,$3::uuid,'LAGOS-OPS','Lagos operations',$4::uuid,ARRAY['WRONG'],$5)`,
		livePositionID, tenantID, entityID, childID, now)

	retire, err := admin.ProposeOrganizationScope(ctx, ProposeOrganizationScopeInput{
		TenantID: tenantID, LegalEntityID: entityID, ScopeID: childID, Operation: OrganizationScopeRetire,
		ExpectedVersion: childVersion, ActorID: makerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if retire.Impact.Positions != 1 {
		t.Fatalf("retirement impact=%#v", retire.Impact)
	}
	if err := admin.ApproveOrganizationScope(ctx, DecideOrganizationScopeInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: retire.ID, ActorID: checkerID, Rationale: "checked",
	}); !errors.Is(err, ErrAdminConflict) {
		t.Fatalf("retire with active position error=%v", err)
	}
	if err := admin.RejectOrganizationScope(ctx, DecideOrganizationScopeInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: retire.ID, ActorID: checkerID, Rationale: "active position",
	}); err != nil {
		t.Fatal(err)
	}

	overview, err := admin.Overview(ctx, tenantID, entityID, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(overview.OrganizationScopeRevisions) != 0 {
		t.Fatalf("pending revisions after rejection=%#v", overview.OrganizationScopeRevisions)
	}
}

func mustAdminExec(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(ctx, query, args...); err != nil {
		t.Fatal(err)
	}
}
