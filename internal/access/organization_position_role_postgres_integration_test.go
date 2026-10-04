//go:build postgres && postgresintegration

package access

import (
	"context"
	"errors"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestOrganizationWorkspaceRoleGovernanceKeepsMaterialRolesLocked(t *testing.T) {
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
		tenantID           = "8a666666-6666-7666-8666-666666666601"
		entityID           = "8a666666-6666-7666-8666-666666666602"
		makerID            = "8a666666-6666-7666-8666-666666666603"
		checkerID          = "8a666666-6666-7666-8666-666666666604"
		occupantID         = "8a666666-6666-7666-8666-666666666605"
		positionID         = "8a666666-6666-7666-8666-666666666606"
		safeRoleID         = "8a666666-6666-7666-8666-666666666610"
		responsibilityRole = "8a666666-6666-7666-8666-666666666611"
		grantRole          = "8a666666-6666-7666-8666-666666666612"
		routeRole          = "8a666666-6666-7666-8666-666666666613"
		segregationRole    = "8a666666-6666-7666-8666-666666666614"
		escalationRole     = "8a666666-6666-7666-8666-666666666615"
		policyID           = "8a666666-6666-7666-8666-666666666620"
		policyVersionID    = "8a666666-6666-7666-8666-666666666621"
	)
	cleanup := func(cleanCtx context.Context) {
		_, _ = pool.Exec(cleanCtx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	}
	cleanup(ctx)
	t.Cleanup(func() { cleanup(context.Background()) })

	now := time.Now().UTC().Truncate(time.Second)
	mustAdminExec(t, ctx, pool, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'workspace-role-test','Workspace Role Test')`, tenantID)
	mustAdminExec(t, ctx, pool, `
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from)
		VALUES($1::uuid,$2::uuid,'BANK-NG','Bank NG','NG',$3)`,
		entityID, tenantID, now.Add(-time.Hour))
	mustAdminExec(t, ctx, pool, `
		INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES
			($1::uuid,$4::uuid,'PERSON','Maker','ACTIVE',$5),
			($2::uuid,$4::uuid,'PERSON','Checker','ACTIVE',$5),
			($3::uuid,$4::uuid,'PERSON','Workspace user','ACTIVE',$5)`,
		makerID, checkerID, occupantID, tenantID, now.Add(-time.Hour))
	mustAdminExec(t, ctx, pool, `
		INSERT INTO role_templates(id,tenant_id,code,name,capabilities,valid_from,version) VALUES
			($1::uuid,$7::uuid,'WORKSPACE_READER','Workspace reader',ARRAY['CONFIG_READ'],$8,1),
			($2::uuid,$7::uuid,'RESP_ROLE','Responsibility role',ARRAY['OVERSIGHT_READ'],$8,1),
			($3::uuid,$7::uuid,'GRANT_ROLE','Authority grant role',ARRAY['CONFIG_WRITE'],$8,1),
			($4::uuid,$7::uuid,'ROUTE_ROLE','Authority route role',ARRAY['OVERSIGHT_READ'],$8,1),
			($5::uuid,$7::uuid,'SEG_ROLE','Segregation role',ARRAY['CONFIG_READ'],$8,1),
			($6::uuid,$7::uuid,'ESC_ROLE','Escalation role',ARRAY['OVERSIGHT_READ'],$8,1)`,
		safeRoleID, responsibilityRole, grantRole, routeRole, segregationRole, escalationRole, tenantID, now.Add(-time.Hour))
	mustAdminExec(t, ctx, pool, `
		INSERT INTO org_positions(
			id,tenant_id,legal_entity_id,code,title,occupant_principal_id,department_path,valid_from,version
		) VALUES($1::uuid,$2::uuid,$3::uuid,'RISK_ANALYST','Risk Analyst',$4::uuid,ARRAY[]::text[],$5,3)`,
		positionID, tenantID, entityID, occupantID, now.Add(-time.Hour))

	mustAdminExec(t, ctx, pool, `
		INSERT INTO responsibility_assignments(
			tenant_id,legal_entity_id,role_template_id,responsibility,object_type,valid_from,policy_version
		) VALUES($1::uuid,$2::uuid,$3::uuid,'REVIEWER','MATTER',$4,'test:v1')`,
		tenantID, entityID, responsibilityRole, now.Add(-time.Hour))
	mustAdminExec(t, ctx, pool, `
		INSERT INTO authority_grants(
			tenant_id,legal_entity_id,role_template_id,decision_type,valid_from,policy_version
		) VALUES($1::uuid,$2::uuid,$3::uuid,'matter.test',$4,'test:v1')`,
		tenantID, entityID, grantRole, now.Add(-time.Hour))
	mustAdminExec(t, ctx, pool, `
		INSERT INTO segregation_rules(tenant_id,code,responsibility,prohibited_role_code,status,valid_from)
		VALUES($1::uuid,'NO-SEG-ROLE','ACCOUNTABLE_OWNER','SEG_ROLE','ACTIVE',$2)`,
		tenantID, now.Add(-time.Hour))

	definition := `{
		"rules":[{
			"id":"route-role","legal_entity_id":"`+entityID+`","object_type":"MATTER","object_id":"*",
			"responsibility":"REVIEWER","decision_type":"matter.test","priority":100,
			"selector":{"kind":"ROLE","ref":"ROUTE_ROLE"}
		}],
		"escalations":[{
			"id":"overdue","trigger":"OVERDUE","steps":[{
				"after":"0s","responsibility":"ESCALATION_OWNER",
				"source_roles":["ESC_ROLE"],"targets":{"roles":[],"groups":[]}
			}]
		}]
	}`
	mustAdminExec(t, ctx, pool, `
		INSERT INTO routing_policies(id,tenant_id,legal_entity_id,code,name,status,current_version,version)
		VALUES($1::uuid,$2::uuid,$3::uuid,'ROLE-LOCKS','Role locks','DRAFT',1,1)`,
		policyID, tenantID, entityID)
	mustAdminExec(t, ctx, pool, `
		INSERT INTO routing_policy_versions(
			id,policy_id,legal_entity_id,version,definition,checksum,effective_from,approved_at
		) VALUES($1::uuid,$2::uuid,$3::uuid,1,$4::jsonb,'workspace-role-test',$5,$5)`,
		policyVersionID, policyID, entityID, definition, now.Add(-time.Hour))
	mustAdminExec(t, ctx, pool, `UPDATE routing_policies SET status='ACTIVE',approved_at=$2 WHERE id=$1::uuid`, policyID, now.Add(-time.Minute))

	admin := NewPostgresAdministrator(pool)
	overview, err := admin.Overview(ctx, tenantID, entityID, 50)
	if err != nil {
		t.Fatal(err)
	}
	roles := make(map[string]RoleTemplateSummary, len(overview.Roles))
	for _, role := range overview.Roles {
		roles[role.Code] = role
	}
	if !roles["WORKSPACE_READER"].OrganizationEditable || len(roles["WORKSPACE_READER"].OrganizationLockReasons) != 0 {
		t.Fatalf("workspace role unexpectedly locked: %#v", roles["WORKSPACE_READER"])
	}
	for code, reason := range map[string]string{
		"RESP_ROLE": "RESPONSIBILITY_ASSIGNMENT",
		"GRANT_ROLE": "AUTHORITY_GRANT",
		"ROUTE_ROLE": "AUTHORITY_ROUTE",
		"SEG_ROLE": "SEGREGATION_RULE",
		"ESC_ROLE": "ESCALATION_ROUTE",
	} {
		role := roles[code]
		if role.OrganizationEditable || !slices.Contains(role.OrganizationLockReasons, reason) {
			t.Fatalf("%s lock reasons=%#v", code, role.OrganizationLockReasons)
		}
	}

	if _, err := admin.ProposeOrganizationPositionRole(ctx, ProposeOrganizationPositionRoleInput{
		TenantID: tenantID, LegalEntityID: entityID, PositionID: positionID,
		RoleTemplateID: responsibilityRole, Operation: OrganizationPositionRoleAdd,
		ExpectedPositionVersion: 3, ActorID: makerID,
	}); !errors.Is(err, ErrAdminConflict) {
		t.Fatalf("material role proposal error=%v", err)
	}

	add, err := admin.ProposeOrganizationPositionRole(ctx, ProposeOrganizationPositionRoleInput{
		TenantID: tenantID, LegalEntityID: entityID, PositionID: positionID,
		RoleTemplateID: safeRoleID, Operation: OrganizationPositionRoleAdd,
		ExpectedPositionVersion: 3, ActorID: makerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	var activeBindings int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM position_role_bindings
		WHERE tenant_id=$1::uuid AND position_id=$2::uuid AND role_template_id=$3::uuid AND valid_until IS NULL`,
		tenantID, positionID, safeRoleID).Scan(&activeBindings); err != nil {
		t.Fatal(err)
	}
	if activeBindings != 0 {
		t.Fatalf("workspace role activated before approval: %d", activeBindings)
	}
	if err := admin.ApproveOrganizationPositionRole(ctx, DecideOrganizationPositionRoleInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: add.ID, ActorID: makerID, Rationale: "self",
	}); !errors.Is(err, ErrAdminMakerChecker) {
		t.Fatalf("maker approved own role revision: %v", err)
	}
	if err := admin.ApproveOrganizationPositionRole(ctx, DecideOrganizationPositionRoleInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: add.ID, ActorID: checkerID, Rationale: "workspace role reviewed",
	}); err != nil {
		t.Fatal(err)
	}

	var bindingID, bindingEntity string
	var positionVersion int64
	if err := pool.QueryRow(ctx, `
		SELECT binding.id::text,COALESCE(binding.scope->>'legal_entity_id',''),position.version
		FROM position_role_bindings binding
		JOIN org_positions position ON position.id=binding.position_id
		WHERE binding.tenant_id=$1::uuid AND binding.position_id=$2::uuid
		  AND binding.role_template_id=$3::uuid AND binding.valid_until IS NULL`,
		tenantID, positionID, safeRoleID).Scan(&bindingID, &bindingEntity, &positionVersion); err != nil {
		t.Fatal(err)
	}
	if bindingEntity != entityID || positionVersion != 4 {
		t.Fatalf("workspace binding entity=%q position version=%d", bindingEntity, positionVersion)
	}

	removeStale, err := admin.ProposeOrganizationPositionRole(ctx, ProposeOrganizationPositionRoleInput{
		TenantID: tenantID, LegalEntityID: entityID, PositionID: positionID,
		RoleTemplateID: safeRoleID, Operation: OrganizationPositionRoleRemove,
		ExpectedPositionVersion: 4, ActorID: makerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	mustAdminExec(t, ctx, pool, `UPDATE role_templates SET version=2 WHERE id=$1::uuid`, safeRoleID)
	if err := admin.ApproveOrganizationPositionRole(ctx, DecideOrganizationPositionRoleInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: removeStale.ID, ActorID: checkerID, Rationale: "stale role version",
	}); !errors.Is(err, ErrAdminConflict) {
		t.Fatalf("stale role revision error=%v", err)
	}
	if err := admin.RejectOrganizationPositionRole(ctx, DecideOrganizationPositionRoleInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: removeStale.ID, ActorID: checkerID, Rationale: "replace stale role proposal",
	}); err != nil {
		t.Fatal(err)
	}

	remove, err := admin.ProposeOrganizationPositionRole(ctx, ProposeOrganizationPositionRoleInput{
		TenantID: tenantID, LegalEntityID: entityID, PositionID: positionID,
		RoleTemplateID: safeRoleID, Operation: OrganizationPositionRoleRemove,
		ExpectedPositionVersion: 4, ActorID: makerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	mustAdminExec(t, ctx, pool, `
		INSERT INTO responsibility_assignments(
			tenant_id,legal_entity_id,role_template_id,responsibility,object_type,valid_from,policy_version
		) VALUES($1::uuid,$2::uuid,$3::uuid,'PERFORMER','MATTER_ACTION',$4,'temporary:v1')`,
		tenantID, entityID, safeRoleID, now.Add(-time.Minute))
	if err := admin.ApproveOrganizationPositionRole(ctx, DecideOrganizationPositionRoleInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: remove.ID, ActorID: checkerID, Rationale: "revalidate material route",
	}); !errors.Is(err, ErrAdminConflict) {
		t.Fatalf("materialized role approval error=%v", err)
	}
	mustAdminExec(t, ctx, pool, `
		DELETE FROM responsibility_assignments
		WHERE tenant_id=$1::uuid AND role_template_id=$2::uuid AND policy_version='temporary:v1'`,
		tenantID, safeRoleID)
	if err := admin.ApproveOrganizationPositionRole(ctx, DecideOrganizationPositionRoleInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: remove.ID, ActorID: checkerID, Rationale: "workspace role removal reviewed",
	}); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT count(*),(SELECT version FROM org_positions WHERE id=$2::uuid)
		FROM position_role_bindings
		WHERE tenant_id=$1::uuid AND id=$3::uuid AND valid_until IS NULL`,
		tenantID, positionID, bindingID).Scan(&activeBindings, &positionVersion); err != nil {
		t.Fatal(err)
	}
	if activeBindings != 0 || positionVersion != 5 {
		t.Fatalf("workspace removal active=%d position version=%d", activeBindings, positionVersion)
	}

	var decisionCount int
	if err := pool.QueryRow(ctx, `
		SELECT count(*)
		FROM governance_decisions
		WHERE tenant_id=$1::uuid
		  AND object_type IN ('ORGANIZATION_POSITION_ROLE_REVISION','ORGANIZATION_POSITION_ROLE_BINDING')`,
		tenantID).Scan(&decisionCount); err != nil {
		t.Fatal(err)
	}
	if decisionCount < 5 {
		t.Fatalf("workspace role decision history=%d", decisionCount)
	}
}
