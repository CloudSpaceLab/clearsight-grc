//go:build postgres && postgresintegration

package access

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestOrganizationPositionGovernanceRequiresIndependentApprovalAndProtectsHierarchy(t *testing.T) {
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
		tenantID   = "8a555555-5555-7555-8555-555555555551"
		entityID   = "8a555555-5555-7555-8555-555555555552"
		otherID    = "8a555555-5555-7555-8555-555555555553"
		makerID    = "8a555555-5555-7555-8555-555555555554"
		checkerID  = "8a555555-5555-7555-8555-555555555555"
		occupantA  = "8a555555-5555-7555-8555-555555555556"
		occupantB  = "8a555555-5555-7555-8555-555555555557"
		scopeID    = "8a555555-5555-7555-8555-555555555558"
		roleID     = "8a555555-5555-7555-8555-555555555559"
		otherPosID = "8a555555-5555-7555-8555-555555555560"
	)
	cleanup := func(cleanCtx context.Context) {
		_, _ = pool.Exec(cleanCtx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	}
	cleanup(ctx)
	t.Cleanup(func() { cleanup(context.Background()) })

	now := time.Now().UTC().Truncate(time.Second)
	mustAdminExec(t, ctx, pool, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'position-governance-test','Position Governance Test')`, tenantID)
	mustAdminExec(t, ctx, pool, `
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES
			($1::uuid,$3::uuid,'BANK-NG','Bank NG','NG',$4),
			($2::uuid,$3::uuid,'BANK-GH','Bank GH','GH',$4)`,
		entityID, otherID, tenantID, now.Add(-time.Hour))
	mustAdminExec(t, ctx, pool, `
		INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES
			($1::uuid,$5::uuid,'PERSON','Maker','ACTIVE',$6),
			($2::uuid,$5::uuid,'PERSON','Checker','ACTIVE',$6),
			($3::uuid,$5::uuid,'PERSON','Alice','ACTIVE',$6),
			($4::uuid,$5::uuid,'PERSON','Bob','ACTIVE',$6)`,
		makerID, checkerID, occupantA, occupantB, tenantID, now.Add(-time.Hour))
	mustAdminExec(t, ctx, pool, `
		INSERT INTO organization_scopes(
			id,tenant_id,legal_entity_id,code,name,kind,department_path,origin,status,valid_from
		) VALUES($1::uuid,$2::uuid,$3::uuid,'RISK','Risk','DEPARTMENT',ARRAY['BANK','RISK'],'MANAGED','ACTIVE',$4)`,
		scopeID, tenantID, entityID, now.Add(-time.Hour))
	mustAdminExec(t, ctx, pool, `
		INSERT INTO role_templates(id,tenant_id,code,name,capabilities,valid_from)
		VALUES($1::uuid,$2::uuid,'WORKSPACE_READER','Workspace reader',ARRAY['CONFIG_READ'],$3)`,
		roleID, tenantID, now.Add(-time.Hour))
	mustAdminExec(t, ctx, pool, `
		INSERT INTO org_positions(
			id,tenant_id,legal_entity_id,code,title,occupant_principal_id,department_path,valid_from,version
		) VALUES($1::uuid,$2::uuid,$3::uuid,'GH_HEAD','Ghana head',$4::uuid,ARRAY[]::text[],$5,1)`,
		otherPosID, tenantID, otherID, occupantA, now.Add(-time.Hour))

	admin := NewPostgresAdministrator(pool)

	rootRevision, err := admin.ProposeOrganizationPosition(ctx, ProposeOrganizationPositionInput{
		TenantID: tenantID, LegalEntityID: entityID, Operation: OrganizationPositionCreate,
		Code: "CRO", Title: "Chief Risk Officer", FunctionName: "Risk", OrganizationScopeID: scopeID,
		OccupantPrincipalID: occupantA, ActorID: makerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := admin.ApproveOrganizationPosition(ctx, DecideOrganizationPositionInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: rootRevision.ID, ActorID: makerID, Rationale: "self",
	}); !errors.Is(err, ErrAdminMakerChecker) {
		t.Fatalf("maker approved own position revision: %v", err)
	}
	if err := admin.ApproveOrganizationPosition(ctx, DecideOrganizationPositionInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: rootRevision.ID, ActorID: checkerID, Rationale: "hierarchy reviewed",
	}); err != nil {
		t.Fatal(err)
	}

	var rootID, rootScopeID string
	var rootPath []string
	if err := pool.QueryRow(ctx, `
		SELECT id::text,COALESCE(organization_scope_id::text,''),department_path
		FROM org_positions
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND code='CRO' AND valid_until IS NULL`,
		tenantID, entityID).Scan(&rootID, &rootScopeID, &rootPath); err != nil {
		t.Fatal(err)
	}
	if rootID != rootRevision.PositionID {
		t.Fatalf("stable position id=%s revision=%s", rootID, rootRevision.PositionID)
	}
	if rootScopeID != scopeID || len(rootPath) != 2 || rootPath[0] != "BANK" || rootPath[1] != "RISK" {
		t.Fatalf("canonical position area id=%s path=%#v", rootScopeID, rootPath)
	}

	childRevision, err := admin.ProposeOrganizationPosition(ctx, ProposeOrganizationPositionInput{
		TenantID: tenantID, LegalEntityID: entityID, Operation: OrganizationPositionCreate,
		Code: "RISK_MANAGER", Title: "Risk Manager", FunctionName: "Risk", OrganizationScopeID: scopeID,
		ParentPositionID: rootID, OccupantPrincipalID: occupantB, ActorID: makerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := admin.ApproveOrganizationPosition(ctx, DecideOrganizationPositionInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: childRevision.ID, ActorID: checkerID, Rationale: "reporting line reviewed",
	}); err != nil {
		t.Fatal(err)
	}
	childID := childRevision.PositionID

	if _, err := admin.ProposeOrganizationPosition(ctx, ProposeOrganizationPositionInput{
		TenantID: tenantID, LegalEntityID: entityID, PositionID: rootID, Operation: OrganizationPositionUpdate,
		Title: "Chief Risk Officer", FunctionName: "Risk", OrganizationScopeID: scopeID,
		ParentPositionID: childID, OccupantPrincipalID: occupantA, ExpectedVersion: 1, ActorID: makerID,
	}); !errors.Is(err, ErrAdminConflict) {
		t.Fatalf("cycle proposal error=%v", err)
	}

	if _, err := admin.ProposeOrganizationPosition(ctx, ProposeOrganizationPositionInput{
		TenantID: tenantID, LegalEntityID: entityID, PositionID: childID, Operation: OrganizationPositionUpdate,
		Title: "Risk Manager", FunctionName: "Risk", OrganizationScopeID: scopeID,
		ParentPositionID: otherPosID, OccupantPrincipalID: occupantB, ExpectedVersion: 1, ActorID: makerID,
	}); !errors.Is(err, ErrAdminInvalid) {
		t.Fatalf("cross-entity parent proposal error=%v", err)
	}

	stale, err := admin.ProposeOrganizationPosition(ctx, ProposeOrganizationPositionInput{
		TenantID: tenantID, LegalEntityID: entityID, PositionID: childID, Operation: OrganizationPositionUpdate,
		Title: "Senior Risk Manager", FunctionName: "Risk", OrganizationScopeID: scopeID,
		ParentPositionID: rootID, OccupantPrincipalID: occupantA, ExpectedVersion: 1, ActorID: makerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	mustAdminExec(t, ctx, pool, `
		UPDATE org_positions SET title='Risk Manager v2',version=2
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid`,
		tenantID, entityID, childID)
	if err := admin.ApproveOrganizationPosition(ctx, DecideOrganizationPositionInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: stale.ID, ActorID: checkerID, Rationale: "stale check",
	}); !errors.Is(err, ErrAdminConflict) {
		t.Fatalf("stale position revision error=%v", err)
	}
	if err := admin.RejectOrganizationPosition(ctx, DecideOrganizationPositionInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: stale.ID, ActorID: checkerID, Rationale: "replace stale proposal",
	}); err != nil {
		t.Fatal(err)
	}

	mustAdminExec(t, ctx, pool, `
		INSERT INTO responsibility_assignments(
			tenant_id,legal_entity_id,position_id,responsibility,object_type,valid_from,policy_version
		) VALUES($1::uuid,$2::uuid,$3::uuid,'ACCOUNTABLE_OWNER','MATTER',$4,'position-route:v1');
		INSERT INTO authority_grants(
			tenant_id,legal_entity_id,position_id,decision_type,valid_from,policy_version
		) VALUES($1::uuid,$2::uuid,$3::uuid,'matter.action.add',$4,'position-route:v1')`,
		tenantID, entityID, childID, now.Add(-time.Hour))
	vacancy, err := admin.ProposeOrganizationPosition(ctx, ProposeOrganizationPositionInput{
		TenantID: tenantID, LegalEntityID: entityID, PositionID: childID, Operation: OrganizationPositionUpdate,
		Title: "Risk Manager v2", FunctionName: "Risk", OrganizationScopeID: scopeID,
		ParentPositionID: rootID, ExpectedVersion: 2, ActorID: makerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if vacancy.Impact.ResponsibilityAssignments != 1 || vacancy.Impact.AuthorityGrants != 1 {
		t.Fatalf("vacancy route impact=%#v", vacancy.Impact)
	}
	if err := admin.ApproveOrganizationPosition(ctx, DecideOrganizationPositionInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: vacancy.ID, ActorID: checkerID, Rationale: "leave role vacant",
	}); !errors.Is(err, ErrAdminConflict) {
		t.Fatalf("vacancy with governed routes error=%v", err)
	}
	if err := admin.RejectOrganizationPosition(ctx, DecideOrganizationPositionInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: vacancy.ID, ActorID: checkerID, Rationale: "routes need coverage first",
	}); err != nil {
		t.Fatal(err)
	}
	mustAdminExec(t, ctx, pool, `
		UPDATE responsibility_assignments
		SET valid_until=$4
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND position_id=$3::uuid AND valid_until IS NULL;
		UPDATE authority_grants
		SET valid_until=$4
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND position_id=$3::uuid AND valid_until IS NULL`,
		tenantID, entityID, childID, now)

	rootRetire, err := admin.ProposeOrganizationPosition(ctx, ProposeOrganizationPositionInput{
		TenantID: tenantID, LegalEntityID: entityID, PositionID: rootID, Operation: OrganizationPositionRetire,
		ExpectedVersion: 1, ActorID: makerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := admin.ApproveOrganizationPosition(ctx, DecideOrganizationPositionInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: rootRetire.ID, ActorID: checkerID, Rationale: "retire root",
	}); !errors.Is(err, ErrAdminConflict) {
		t.Fatalf("retired parent with active child: %v", err)
	}
	if err := admin.RejectOrganizationPosition(ctx, DecideOrganizationPositionInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: rootRetire.ID, ActorID: checkerID, Rationale: "child still active",
	}); err != nil {
		t.Fatal(err)
	}

	mustAdminExec(t, ctx, pool, `
		INSERT INTO position_role_bindings(tenant_id,position_id,role_template_id,scope,valid_from)
		VALUES($1::uuid,$2::uuid,$3::uuid,jsonb_build_object('legal_entity_id',$4::text),$5)`,
		tenantID, childID, roleID, entityID, now.Add(-time.Hour))
	childRetire, err := admin.ProposeOrganizationPosition(ctx, ProposeOrganizationPositionInput{
		TenantID: tenantID, LegalEntityID: entityID, PositionID: childID, Operation: OrganizationPositionRetire,
		ExpectedVersion: 2, ActorID: makerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if childRetire.Impact.ActiveRoleBindings != 1 {
		t.Fatalf("retirement impact=%#v", childRetire.Impact)
	}
	if err := admin.ApproveOrganizationPosition(ctx, DecideOrganizationPositionInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: childRetire.ID, ActorID: checkerID, Rationale: "vacated position reviewed",
	}); err != nil {
		t.Fatal(err)
	}
	var positionRetired, roleRetired bool
	if err := pool.QueryRow(ctx, `
		SELECT valid_until IS NOT NULL FROM org_positions WHERE id=$1::uuid`, childID).Scan(&positionRetired); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT valid_until IS NOT NULL FROM position_role_bindings
		WHERE tenant_id=$1::uuid AND position_id=$2::uuid AND role_template_id=$3::uuid`,
		tenantID, childID, roleID).Scan(&roleRetired); err != nil {
		t.Fatal(err)
	}
	if !positionRetired || !roleRetired {
		t.Fatalf("position retired=%v role retired=%v", positionRetired, roleRetired)
	}

	if _, err := pool.Exec(ctx, `
		UPDATE org_positions SET parent_position_id=$1::uuid
		WHERE tenant_id=$2::uuid AND legal_entity_id=$3::uuid AND id=$4::uuid`,
		otherPosID, tenantID, entityID, rootID); err == nil {
		t.Fatal("direct cross-entity reporting write was accepted")
	}

	overview, err := admin.Overview(ctx, tenantID, entityID, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(overview.OrganizationPositionRevisions) != 0 {
		t.Fatalf("unexpected pending position revisions: %#v", overview.OrganizationPositionRevisions)
	}
	var decisionCount int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM governance_decisions
		WHERE tenant_id=$1::uuid AND object_type IN ('ORGANIZATION_POSITION','ORGANIZATION_POSITION_REVISION')`,
		tenantID).Scan(&decisionCount); err != nil {
		t.Fatal(err)
	}
	if decisionCount < 7 {
		t.Fatalf("position governance decision count=%d", decisionCount)
	}
}
