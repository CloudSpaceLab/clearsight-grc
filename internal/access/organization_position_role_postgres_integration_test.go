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

func TestPositionWorkspaceRoleGovernanceKeepsMaterialRolesLocked(t *testing.T) {
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
		tenantID       = "8a666666-6666-7666-8666-666666666601"
		entityID       = "8a666666-6666-7666-8666-666666666602"
		makerID        = "8a666666-6666-7666-8666-666666666603"
		checkerID      = "8a666666-6666-7666-8666-666666666604"
		positionID     = "8a666666-6666-7666-8666-666666666605"
		safeRoleID     = "8a666666-6666-7666-8666-666666666610"
		declaredRoleID = "8a666666-6666-7666-8666-666666666611"
		assignedRoleID = "8a666666-6666-7666-8666-666666666612"
		authorityRoleID = "8a666666-6666-7666-8666-666666666613"
		routingRoleID  = "8a666666-6666-7666-8666-666666666614"
		segRoleID      = "8a666666-6666-7666-8666-666666666615"
		policyID       = "8a666666-6666-7666-8666-666666666620"
		policyVersionID = "8a666666-6666-7666-8666-666666666621"
	)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID) })

	now := time.Now().UTC().Truncate(time.Second)
	mustAdminExec(t, ctx, pool, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'workspace-role-test','Workspace Role Test')`, tenantID)
	mustAdminExec(t, ctx, pool, `
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from)
		VALUES($1::uuid,$2::uuid,'BANK-NG','Bank NG','NG',$3)`, entityID, tenantID, now.Add(-time.Hour))
	mustAdminExec(t, ctx, pool, `
		INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES
			($1::uuid,$3::uuid,'PERSON','Maker','ACTIVE',$4),
			($2::uuid,$3::uuid,'PERSON','Checker','ACTIVE',$4)`,
		makerID, checkerID, tenantID, now.Add(-time.Hour))
	mustAdminExec(t, ctx, pool, `
		INSERT INTO org_positions(id,tenant_id,legal_entity_id,code,title,department_path,valid_from,version)
		VALUES($1::uuid,$2::uuid,$3::uuid,'RISK_ANALYST','Risk Analyst',ARRAY[]::text[],$4,1)`,
		positionID, tenantID, entityID, now.Add(-time.Hour))
	mustAdminExec(t, ctx, pool, `
		INSERT INTO role_templates(id,tenant_id,code,name,responsibilities,capabilities,valid_from,version) VALUES
			($1::uuid,$7::uuid,'WORKSPACE_READER','Workspace reader',ARRAY[]::text[],ARRAY['CONFIG_READ'],$8,1),
			($2::uuid,$7::uuid,'BUSINESS_REVIEWER','Business reviewer',ARRAY['REVIEWER'],ARRAY['CONFIG_READ'],$8,1),
			($3::uuid,$7::uuid,'ROUTED_OWNER','Routed owner',ARRAY[]::text[],ARRAY['CONFIG_READ'],$8,1),
			($4::uuid,$7::uuid,'DECISION_ROLE','Decision role',ARRAY[]::text[],ARRAY['CONFIG_READ'],$8,1),
			($5::uuid,$7::uuid,'ESCALATION_ROLE','Escalation role',ARRAY[]::text[],ARRAY['CONFIG_READ'],$8,1),
			($6::uuid,$7::uuid,'CONFLICT_ROLE','Conflict role',ARRAY[]::text[],ARRAY['CONFIG_READ'],$8,1)`,
		safeRoleID, declaredRoleID, assignedRoleID, authorityRoleID, routingRoleID, segRoleID, tenantID, now.Add(-time.Hour))
	mustAdminExec(t, ctx, pool, `
		INSERT INTO responsibility_assignments(
			tenant_id,legal_entity_id,role_template_id,responsibility,object_type,priority,valid_from,policy_version
		) VALUES($1::uuid,$2::uuid,$3::uuid,'ACCOUNTABLE_OWNER','MATTER',100,$4,'fixture:v1')`,
		tenantID, entityID, assignedRoleID, now.Add(-time.Hour))
	mustAdminExec(t, ctx, pool, `
		INSERT INTO authority_grants(
			tenant_id,legal_entity_id,role_template_id,decision_type,limits,valid_from,policy_version
		) VALUES($1::uuid,$2::uuid,$3::uuid,'matter.action.add','{}'::jsonb,$4,'fixture:v1')`,
		tenantID, entityID, authorityRoleID, now.Add(-time.Hour))
	mustAdminExec(t, ctx, pool, `
		INSERT INTO segregation_rules(
			tenant_id,code,responsibility,prohibited_role_code,status,valid_from
		) VALUES($1::uuid,'NO-CONFLICT','ACCOUNTABLE_OWNER','CONFLICT_ROLE','ACTIVE',$2)`,
		tenantID, now.Add(-time.Hour))
	mustAdminExec(t, ctx, pool, `
		INSERT INTO routing_policies(
			id,tenant_id,legal_entity_id,code,name,status,current_version,maker_id,checker_id,approved_at,version
		) VALUES($1::uuid,$2::uuid,$3::uuid,'workspace-role-test','Workspace role test','ACTIVE',1,$4::uuid,$5::uuid,$6,1)`,
		policyID, tenantID, entityID, makerID, checkerID, now.Add(-30*time.Minute))
	mustAdminExec(t, ctx, pool, `
		INSERT INTO routing_policy_versions(
			id,policy_id,legal_entity_id,version,definition,checksum,created_by,approved_by,created_at,approved_at,effective_from
		) VALUES(
			$1::uuid,$2::uuid,$3::uuid,1,
			jsonb_build_object('rules',jsonb_build_array(jsonb_build_object(
				'id','workspace-role-rule','legal_entity_id',$3::text,'responsibility','ESCALATION_OWNER',
				'selector',jsonb_build_object('kind','ROLE','ref','ESCALATION_ROLE')
			))),
			'workspace-role-v1',$4::uuid,$5::uuid,$6,$6,$6
		)`,
		policyVersionID, policyID, entityID, makerID, checkerID, now.Add(-30*time.Minute))

	admin := NewPostgresAdministrator(pool)
	overview, err := admin.Overview(ctx, tenantID, entityID, 50)
	if err != nil {
		t.Fatal(err)
	}
	roles := make(map[string]RoleTemplateSummary, len(overview.Roles))
	for _, role := range overview.Roles {
		roles[role.Code] = role
	}
	if role := roles["WORKSPACE_READER"]; !role.WorkspaceEditable || role.WorkspaceLockReason != "" {
		t.Fatalf("safe workspace role = %#v", role)
	}
	for code, reason := range map[string]string{
		"BUSINESS_REVIEWER": "Defines business responsibilities",
		"ROUTED_OWNER": "Used by responsibility routing",
		"DECISION_ROLE": "Used by decision authority",
		"ESCALATION_ROLE": "Used by routing or escalation",
		"CONFLICT_ROLE": "Used by segregation rules",
	} {
		role := roles[code]
		if role.WorkspaceEditable || role.WorkspaceLockReason != reason {
			t.Fatalf("%s classification = %#v", code, role)
		}
	}

	if _, err := admin.ProposeOrganizationPositionRole(ctx, ProposeOrganizationPositionRoleInput{
		TenantID: tenantID, LegalEntityID: entityID, PositionID: positionID, RoleTemplateID: declaredRoleID,
		Operation: OrganizationPositionRoleAdd, ExpectedPositionVersion: 1, ActorID: makerID,
	}); !errors.Is(err, ErrAdminInvalid) {
		t.Fatalf("material role proposal error=%v", err)
	}

	staleMaterial, err := admin.ProposeOrganizationPositionRole(ctx, ProposeOrganizationPositionRoleInput{
		TenantID: tenantID, LegalEntityID: entityID, PositionID: positionID, RoleTemplateID: safeRoleID,
		Operation: OrganizationPositionRoleAdd, ExpectedPositionVersion: 1, ActorID: makerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := admin.ApproveOrganizationPositionRole(ctx, DecideOrganizationPositionRoleInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: staleMaterial.ID, ActorID: makerID, Rationale: "self",
	}); !errors.Is(err, ErrAdminMakerChecker) {
		t.Fatalf("maker approved own workspace role revision: %v", err)
	}
	mustAdminExec(t, ctx, pool, `
		INSERT INTO authority_grants(
			tenant_id,legal_entity_id,role_template_id,decision_type,limits,valid_from,policy_version
		) VALUES($1::uuid,$2::uuid,$3::uuid,'matter.outcome.record','{}'::jsonb,$4,'fixture:stale')`,
		tenantID, entityID, safeRoleID, now)
	if err := admin.ApproveOrganizationPositionRole(ctx, DecideOrganizationPositionRoleInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: staleMaterial.ID, ActorID: checkerID, Rationale: "authority changed",
	}); !errors.Is(err, ErrAdminConflict) {
		t.Fatalf("new material reference did not stale proposal: %v", err)
	}
	if err := admin.RejectOrganizationPositionRole(ctx, DecideOrganizationPositionRoleInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: staleMaterial.ID, ActorID: checkerID, Rationale: "material role now",
	}); err != nil {
		t.Fatal(err)
	}
	mustAdminExec(t, ctx, pool, `
		UPDATE authority_grants SET valid_until=clock_timestamp()
		WHERE tenant_id=$1::uuid AND role_template_id=$2::uuid AND policy_version='fixture:stale'`,
		tenantID, safeRoleID)

	staleTemplate, err := admin.ProposeOrganizationPositionRole(ctx, ProposeOrganizationPositionRoleInput{
		TenantID: tenantID, LegalEntityID: entityID, PositionID: positionID, RoleTemplateID: safeRoleID,
		Operation: OrganizationPositionRoleAdd, ExpectedPositionVersion: 1, ActorID: makerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	mustAdminExec(t, ctx, pool, `UPDATE role_templates SET version=2 WHERE id=$1::uuid`, safeRoleID)
	if err := admin.ApproveOrganizationPositionRole(ctx, DecideOrganizationPositionRoleInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: staleTemplate.ID, ActorID: checkerID, Rationale: "template changed",
	}); !errors.Is(err, ErrAdminConflict) {
		t.Fatalf("role template version did not stale proposal: %v", err)
	}
	if err := admin.RejectOrganizationPositionRole(ctx, DecideOrganizationPositionRoleInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: staleTemplate.ID, ActorID: checkerID, Rationale: "replace stale proposal",
	}); err != nil {
		t.Fatal(err)
	}

	addRevision, err := admin.ProposeOrganizationPositionRole(ctx, ProposeOrganizationPositionRoleInput{
		TenantID: tenantID, LegalEntityID: entityID, PositionID: positionID, RoleTemplateID: safeRoleID,
		Operation: OrganizationPositionRoleAdd, ExpectedPositionVersion: 1, ActorID: makerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := admin.ApproveOrganizationPositionRole(ctx, DecideOrganizationPositionRoleInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: addRevision.ID, ActorID: checkerID, Rationale: "workspace access reviewed",
	}); err != nil {
		t.Fatal(err)
	}

	var (
		bindingID     string
		bindingScope  string
		positionVersion int64
	)
	if err := pool.QueryRow(ctx, `
		SELECT binding.id::text,binding.scope->>'legal_entity_id',position.version
		FROM position_role_bindings binding
		JOIN org_positions position ON position.tenant_id=binding.tenant_id AND position.id=binding.position_id
		WHERE binding.tenant_id=$1::uuid AND binding.position_id=$2::uuid
		  AND binding.role_template_id=$3::uuid AND binding.valid_until IS NULL`,
		tenantID, positionID, safeRoleID).Scan(&bindingID, &bindingScope, &positionVersion); err != nil {
		t.Fatal(err)
	}
	if bindingScope != entityID || positionVersion != 2 {
		t.Fatalf("approved binding scope=%q position version=%d", bindingScope, positionVersion)
	}

	retireRevision, err := admin.ProposeOrganizationPositionRole(ctx, ProposeOrganizationPositionRoleInput{
		TenantID: tenantID, LegalEntityID: entityID, PositionID: positionID, RoleTemplateID: safeRoleID,
		Operation: OrganizationPositionRoleRetire, ExpectedPositionVersion: 2, ActorID: makerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := admin.ApproveOrganizationPositionRole(ctx, DecideOrganizationPositionRoleInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: retireRevision.ID, ActorID: checkerID, Rationale: "workspace role no longer required",
	}); err != nil {
		t.Fatal(err)
	}
	var activeBindingCount int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM position_role_bindings
		WHERE tenant_id=$1::uuid AND position_id=$2::uuid AND role_template_id=$3::uuid
		  AND valid_from<=clock_timestamp() AND (valid_until IS NULL OR clock_timestamp()<valid_until)`,
		tenantID, positionID, safeRoleID).Scan(&activeBindingCount); err != nil {
		t.Fatal(err)
	}
	if activeBindingCount != 0 {
		t.Fatalf("active workspace role bindings=%d", activeBindingCount)
	}
	if err := pool.QueryRow(ctx, `SELECT version FROM org_positions WHERE id=$1::uuid`, positionID).Scan(&positionVersion); err != nil {
		t.Fatal(err)
	}
	if positionVersion != 3 {
		t.Fatalf("position version after role retirement=%d", positionVersion)
	}

	var decisionCount int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM governance_decisions
		WHERE tenant_id=$1::uuid
		  AND object_type IN ('ORGANIZATION_POSITION_ROLE_BINDING','ORGANIZATION_POSITION_ROLE_REVISION')`,
		tenantID).Scan(&decisionCount); err != nil {
		t.Fatal(err)
	}
	if decisionCount < 8 {
		t.Fatalf("workspace role governance decisions=%d", decisionCount)
	}
}
