//go:build postgres && postgresintegration

package workflow

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMatterEscalationRoleAndGroupGuardsNarrowCurrentAuthorityCandidates(t *testing.T) {
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
		tenantID       = "98888888-8888-7888-8888-888888888801"
		entityID       = "98888888-8888-7888-8888-888888888802"
		complianceID   = "98888888-8888-7888-8888-888888888803"
		auditorAID     = "98888888-8888-7888-8888-888888888804"
		auditorBID     = "98888888-8888-7888-8888-888888888805"
		complianceRole = "98888888-8888-7888-8888-888888888806"
		auditorRole    = "98888888-8888-7888-8888-888888888807"
		supervisorRole = "98888888-8888-7888-8888-888888888808"
		compliancePos  = "98888888-8888-7888-8888-888888888809"
		auditorAPos    = "98888888-8888-7888-8888-888888888810"
		auditorBPos    = "98888888-8888-7888-8888-888888888811"
		complianceBind = "98888888-8888-7888-8888-888888888812"
		auditorABind   = "98888888-8888-7888-8888-888888888813"
		auditorBBind   = "98888888-8888-7888-8888-888888888814"
		supervisorBind = "98888888-8888-7888-8888-888888888815"
		scimSourceID   = "98888888-8888-7888-8888-888888888816"
		scimUserID     = "98888888-8888-7888-8888-888888888817"
		auditorGroupID = "98888888-8888-7888-8888-888888888818"
		policyID       = "98888888-8888-7888-8888-888888888819"
		policyVersion  = "98888888-8888-7888-8888-888888888820"
	)
	const tenantSlug = "eia5-escalation-guards"
	now := time.Date(2026, 8, 11, 3, 0, 0, 0, time.UTC)

	cleanupEscalationGuardFixture(ctx, pool, tenantID, policyID)
	t.Cleanup(func() { cleanupEscalationGuardFixture(context.Background(), pool, tenantID, policyID) })

	mustExecEscalation(t, ctx, pool, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,'EIA5 escalation guards')`, tenantID, tenantSlug)
	mustExecEscalation(t, ctx, pool, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2::uuid,'BANK-NG','Bank NG','NG',$3)`, entityID, tenantID, now.Add(-24*time.Hour))
	mustExecEscalation(t, ctx, pool, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES
		($1::uuid,$4::uuid,'PERSON','Compliance officer','ACTIVE',$5),
		($2::uuid,$4::uuid,'PERSON','Network auditor','ACTIVE',$5),
		($3::uuid,$4::uuid,'PERSON','General auditor','ACTIVE',$5)`, complianceID, auditorAID, auditorBID, tenantID, now.Add(-24*time.Hour))
	mustExecEscalation(t, ctx, pool, `INSERT INTO role_templates(id,tenant_id,code,name,responsibilities,valid_from) VALUES
		($1::uuid,$4::uuid,'COMPLIANCE_OFFICER','Compliance officer',ARRAY['ACCOUNTABLE_OWNER'],$5),
		($2::uuid,$4::uuid,'AUDITOR','Auditor',ARRAY['ESCALATION_OWNER'],$5),
		($3::uuid,$4::uuid,'SUPERVISOR','Supervisor',ARRAY['ESCALATION_OWNER'],$5)`, complianceRole, auditorRole, supervisorRole, tenantID, now.Add(-24*time.Hour))
	mustExecEscalation(t, ctx, pool, `INSERT INTO org_positions(id,tenant_id,legal_entity_id,code,title,occupant_principal_id,department_path,valid_from) VALUES
		($1::uuid,$7::uuid,$8::uuid,'COMPLIANCE','Compliance officer',$4::uuid,ARRAY['BANK','COMPLIANCE'],$9),
		($2::uuid,$7::uuid,$8::uuid,'NETWORK-AUDITOR','Network auditor',$5::uuid,ARRAY['BANK','AUDIT'],$9),
		($3::uuid,$7::uuid,$8::uuid,'GENERAL-AUDITOR','General auditor',$6::uuid,ARRAY['BANK','AUDIT'],$9)`, compliancePos, auditorAPos, auditorBPos, complianceID, auditorAID, auditorBID, tenantID, entityID, now.Add(-24*time.Hour))
	mustExecEscalation(t, ctx, pool, `INSERT INTO position_role_bindings(id,tenant_id,position_id,role_template_id,valid_from) VALUES
		($1::uuid,$7::uuid,$4::uuid,$5::uuid,$8),
		($2::uuid,$7::uuid,$6::uuid,$9::uuid,$8),
		($3::uuid,$7::uuid,$10::uuid,$9::uuid,$8)`, complianceBind, auditorABind, auditorBBind, compliancePos, complianceRole, auditorAPos, tenantID, now.Add(-24*time.Hour), auditorRole, auditorBPos)

	mustExecEscalation(t, ctx, pool, `INSERT INTO scim_sources(id,tenant_id,code,token_hash,status) VALUES($1::uuid,$2::uuid,'ENTRA',decode(repeat('ab',32),'hex'),'ACTIVE')`, scimSourceID, tenantID)
	mustExecEscalation(t, ctx, pool, `INSERT INTO scim_users(id,tenant_id,source_id,principal_id,external_id,user_name,active) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,'auditor-a','auditor-a@example.test',true)`, scimUserID, tenantID, scimSourceID, auditorAID)
	mustExecEscalation(t, ctx, pool, `INSERT INTO directory_groups(id,tenant_id,source_id,external_id,display_name) VALUES($1::uuid,$2::uuid,$3::uuid,'network-auditors','Network Auditors')`, auditorGroupID, tenantID, scimSourceID)
	mustExecEscalation(t, ctx, pool, `INSERT INTO directory_group_members(tenant_id,group_id,scim_user_id) VALUES($1::uuid,$2::uuid,$3::uuid)`, tenantID, auditorGroupID, scimUserID)

	definition, err := json.Marshal(map[string]any{
		"rules": []map[string]any{{
			"id": "auditor-route", "legal_entity_id": "BANK-NG", "object_type": "MATTER", "object_id": "*",
			"responsibility": "ESCALATION_OWNER", "decision_type": "matter.guard", "priority": 100,
			"selector": map[string]any{"kind": "ROLE", "ref": "AUDITOR"},
		}},
		"escalations": []map[string]any{{
			"id": "compliance-overdue", "trigger": "OVERDUE", "steps": []map[string]any{{
				"after": "0s", "responsibility": "ESCALATION_OWNER", "source_roles": []string{"COMPLIANCE_OFFICER"},
				"targets": map[string]any{"roles": []string{"SUPERVISOR"}, "groups": []string{auditorGroupID}},
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	mustExecEscalation(t, ctx, pool, `INSERT INTO routing_policies(id,tenant_id,code,name,status,current_version,approved_at,version) VALUES($1::uuid,$2::uuid,'EIA5-GUARDS','EIA5 guards','DRAFT',1,$3,1)`, policyID, tenantID, now.Add(-time.Hour))
	mustExecEscalation(t, ctx, pool, `INSERT INTO routing_policy_versions(id,policy_id,version,definition,checksum,effective_from,approved_at) VALUES($1::uuid,$2::uuid,1,$3::jsonb,'eia5-guards',$4,$4)`, policyVersion, policyID, string(definition), now.Add(-time.Hour))
	mustExecEscalation(t, ctx, pool, `UPDATE routing_policies SET status='ACTIVE' WHERE id=$1::uuid`, policyID)

	runtimeRepo := workflowruntime.NewPostgresRepository(pool)
	current := now
	coordinator := &MatterEscalationCoordinator{
		Repo: NewPostgresRepository(pool), Runtime: runtimeRepo,
		Authority: authority.NewEffectivePostgresService(pool), Continuity: continuity.NewService(continuity.NewCurrentPostgresRepository(pool)),
		Now: func() time.Time { return current },
	}

	firstTask := createEscalationGuardWork(t, ctx, pool, tenantID, complianceID, "831", now)
	if processed, err := coordinator.Maintain(ctx, now, 20); err != nil || processed != 1 {
		t.Fatalf("schedule group-constrained escalation: processed=%d err=%v", processed, err)
	}
	current = now
	if err := coordinator.Publish(ctx, fireGuardEscalationTimer(t, ctx, pool, tenantID, tenantSlug, firstTask, "group", now)); err != nil {
		t.Fatal(err)
	}
	assertEscalatedTask(t, ctx, pool, firstTask, "ESCALATION_OWNER", auditorAID, "0", "ROUTED")

	// Removing the directory source must remove group eligibility immediately.
	mustExecEscalation(t, ctx, pool, `UPDATE scim_sources SET status='REVOKED',updated_at=$2 WHERE id=$1::uuid`, scimSourceID, now.Add(time.Minute))
	// The alternate target is then admitted by a current SUPERVISOR role.
	mustExecEscalation(t, ctx, pool, `INSERT INTO position_role_bindings(id,tenant_id,position_id,role_template_id,valid_from) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5)`, supervisorBind, tenantID, auditorBPos, supervisorRole, now.Add(time.Minute))
	secondTask := createEscalationGuardWork(t, ctx, pool, tenantID, complianceID, "841", now.Add(2*time.Minute))
	current = now.Add(2 * time.Minute)
	if processed, err := coordinator.Maintain(ctx, current, 20); err != nil || processed != 1 {
		t.Fatalf("schedule role-constrained escalation: processed=%d err=%v", processed, err)
	}
	if err := coordinator.Publish(ctx, fireGuardEscalationTimer(t, ctx, pool, tenantID, tenantSlug, secondTask, "role", current)); err != nil {
		t.Fatal(err)
	}
	assertEscalatedTask(t, ctx, pool, secondTask, "ESCALATION_OWNER", auditorBID, "0", "ROUTED")

	// A source-role restriction is evaluated at fire time too. Removing the
	// source role fails closed before any target candidate can be assigned.
	mustExecEscalation(t, ctx, pool, `UPDATE position_role_bindings SET valid_until=$2 WHERE id=$1::uuid`, complianceBind, now.Add(3*time.Minute))
	thirdTask := createEscalationGuardWork(t, ctx, pool, tenantID, complianceID, "851", now.Add(4*time.Minute))
	current = now.Add(4 * time.Minute)
	if processed, err := coordinator.Maintain(ctx, current, 20); err != nil || processed != 1 {
		t.Fatalf("schedule source-role-constrained escalation: processed=%d err=%v", processed, err)
	}
	if err := coordinator.Publish(ctx, fireGuardEscalationTimer(t, ctx, pool, tenantID, tenantSlug, thirdTask, "source", current)); err != nil {
		t.Fatal(err)
	}
	var status, principalID, reason string
	if err := pool.QueryRow(ctx, `SELECT status,COALESCE(principal_id::text,''),COALESCE(context->>'escalation_status','') FROM workflow_tasks WHERE id=$1::uuid`, thirdTask).Scan(&status, &principalID, &reason); err != nil {
		t.Fatal(err)
	}
	if status != "READY" || principalID != complianceID || reason != "SOURCE_ROLE_NOT_ALLOWED" {
		t.Fatalf("source role guard did not fail closed: status=%s principal=%s reason=%s", status, principalID, reason)
	}
}

func createEscalationGuardWork(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, principalID, suffix string, due time.Time) string {
	t.Helper()
	matterID := "98888888-8888-7888-8888-888888888" + suffix[:1] + "1"
	workflowID := "98888888-8888-7888-8888-888888888" + suffix[:1] + "2"
	taskID := "98888888-8888-7888-8888-888888888" + suffix[:1] + "3"
	mustExecEscalation(t, ctx, pool, `INSERT INTO matters(id,tenant_id,reference,matter_type,status,priority,title,summary,scope,known_facts,missing_facts,contradictions,due_at,created_at,updated_at)
		VALUES($1::uuid,$2::uuid,$3,'EXCEPTION','ASSESSMENT',4,'Guard test','Escalation guard test','{"access":"INTERNAL"}'::jsonb,'{}'::jsonb,'[]'::jsonb,'[]'::jsonb,$4,$5,$5)`, matterID, tenantID, "MAT-GUARD-"+suffix, due, due.Add(-time.Minute))
	mustExecEscalation(t, ctx, pool, `INSERT INTO workflow_instances(id,tenant_id,kind,subject_type,subject_id,state,policy_version,due_at,created_at,updated_at,version)
		VALUES($1::uuid,$2::uuid,$3,'MATTER',$4::uuid,'ACTIVE','matter-work-v1',$5,$6,$6,1)`, workflowID, tenantID, MatterLifecycleWorkflowKind, matterID, due, due.Add(-time.Minute))
	contextValue, _ := json.Marshal(map[string]string{
		"type": "MATTER_WORK", "matter_id": matterID, "work_requirement_key": "assessment:owner",
		"decision_type": "matter.guard", "materiality": "4", "authority_policy_version": "EIA5-GUARDS:v1",
	})
	mustExecEscalation(t, ctx, pool, `INSERT INTO workflow_tasks(id,tenant_id,workflow_id,step_key,responsibility,principal_id,title,status,due_at,context,created_at,updated_at,version)
		VALUES($1::uuid,$2::uuid,$3::uuid,'assessment:owner','ACCOUNTABLE_OWNER',$4::uuid,'Resolve guarded escalation','READY',$5,$6::jsonb,$7,$7,1)`, taskID, tenantID, workflowID, principalID, due, string(contextValue), due.Add(-time.Minute))
	return taskID
}

func fireGuardEscalationTimer(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, tenantSlug, taskID, eventSuffix string, at time.Time) workflowruntime.OutboxEvent {
	t.Helper()
	var workflowID string
	var payload []byte
	if err := pool.QueryRow(ctx, `UPDATE workflow_timers SET state='FIRED',fired_at=$3 WHERE id=(SELECT id FROM workflow_timers WHERE tenant_id=$1::uuid AND task_id=$2::uuid AND timer_type='MATTER_ESCALATION' AND state='READY' ORDER BY due_at,id LIMIT 1) RETURNING workflow_id::text,payload`, tenantID, taskID, at).Scan(&workflowID, &payload); err != nil {
		t.Fatal(err)
	}
	return workflowruntime.OutboxEvent{
		ID: "eia5-guard-" + eventSuffix, TenantID: tenantSlug, AggregateType: "WORKFLOW", AggregateID: workflowID,
		EventType: "WorkflowTimerFired", Payload: payload, OccurredAt: at,
	}
}

func cleanupEscalationGuardFixture(ctx context.Context, pool *pgxpool.Pool, tenantID, policyID string) {
	_, _ = pool.Exec(ctx, `DELETE FROM workflow_timers WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM inbox_receipts WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM workflow_events WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM workflow_tasks WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM workflow_instances WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM effective_authority_routes WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM routing_policy_versions WHERE policy_id=$1::uuid`, policyID)
	_, _ = pool.Exec(ctx, `DELETE FROM routing_policies WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM directory_group_members WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM directory_group_role_bindings WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM directory_groups WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM scim_users WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM scim_sources WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM position_role_bindings WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM org_positions WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM role_templates WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM matters WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM principals WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM legal_entities WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
}
