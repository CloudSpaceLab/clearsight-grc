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

func TestMatterEscalationExecutesOrderedDepartmentSequenceAndCancelsOnCompletion(t *testing.T) {
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
		tenantID      = "97777777-7777-7777-8777-777777777701"
		entityID      = "97777777-7777-7777-8777-777777777702"
		ownerID       = "97777777-7777-7777-8777-777777777703"
		parentRiskID  = "97777777-7777-7777-8777-777777777704"
		otherRiskID   = "97777777-7777-7777-8777-777777777705"
		croID         = "97777777-7777-7777-8777-777777777706"
		ownerRoleID   = "97777777-7777-7777-8777-777777777707"
		riskRoleID    = "97777777-7777-7777-8777-777777777708"
		croRoleID     = "97777777-7777-7777-8777-777777777709"
		ownerPosID    = "97777777-7777-7777-8777-777777777710"
		parentPosID   = "97777777-7777-7777-8777-777777777711"
		otherPosID    = "97777777-7777-7777-8777-777777777712"
		croPosID      = "97777777-7777-7777-8777-777777777713"
		ownerBindID   = "97777777-7777-7777-8777-777777777714"
		parentBindID  = "97777777-7777-7777-8777-777777777715"
		otherBindID   = "97777777-7777-7777-8777-777777777716"
		croBindID     = "97777777-7777-7777-8777-777777777717"
		matterID      = "97777777-7777-7777-8777-777777777718"
		policyID      = "97777777-7777-7777-8777-777777777719"
		policyVersion = "97777777-7777-7777-8777-777777777720"
		workflowID    = "97777777-7777-7777-8777-777777777721"
		taskID        = "97777777-7777-7777-8777-777777777722"
	)
	const tenantSlug = "eia4-escalation-test"
	now := time.Date(2026, 8, 10, 18, 0, 0, 0, time.UTC)

	cleanupEscalationFixture(ctx, pool, tenantID, policyID)
	t.Cleanup(func() { cleanupEscalationFixture(context.Background(), pool, tenantID, policyID) })

	mustExecEscalation(t, ctx, pool, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,'EIA4 escalation test')`, tenantID, tenantSlug)
	mustExecEscalation(t, ctx, pool, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2::uuid,'BANK-NG','Bank NG','NG',$3)`, entityID, tenantID, now.Add(-24*time.Hour))
	mustExecEscalation(t, ctx, pool, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES
		($1::uuid,$5::uuid,'PERSON','Operations owner','ACTIVE',$6),
		($2::uuid,$5::uuid,'PERSON','Parent risk manager','ACTIVE',$6),
		($3::uuid,$5::uuid,'PERSON','Other risk manager','ACTIVE',$6),
		($4::uuid,$5::uuid,'PERSON','Chief risk officer','ACTIVE',$6)`,
		ownerID, parentRiskID, otherRiskID, croID, tenantID, now.Add(-24*time.Hour))
	mustExecEscalation(t, ctx, pool, `INSERT INTO role_templates(id,tenant_id,code,name,responsibilities,valid_from) VALUES
		($1::uuid,$4::uuid,'DEPT_OWNER','Department owner',ARRAY['ACCOUNTABLE_OWNER'],$5),
		($2::uuid,$4::uuid,'RISK_MANAGER','Risk manager',ARRAY['ESCALATION_OWNER'],$5),
		($3::uuid,$4::uuid,'CRO','Chief risk officer',ARRAY['AUTHORIZER'],$5)`,
		ownerRoleID, riskRoleID, croRoleID, tenantID, now.Add(-24*time.Hour))
	mustExecEscalation(t, ctx, pool, `INSERT INTO org_positions(id,tenant_id,legal_entity_id,code,title,occupant_principal_id,department_path,valid_from) VALUES
		($1::uuid,$9::uuid,$10::uuid,'OPS-OWNER','Operations owner',$5::uuid,ARRAY['BANK','RISK','OPS'],$11),
		($2::uuid,$9::uuid,$10::uuid,'RISK-PARENT','Parent risk manager',$6::uuid,ARRAY['BANK','RISK'],$11),
		($3::uuid,$9::uuid,$10::uuid,'OPS-RISK','Other risk manager',$7::uuid,ARRAY['BANK','OPERATIONS'],$11),
		($4::uuid,$9::uuid,$10::uuid,'CRO','Chief risk officer',$8::uuid,ARRAY['BANK'],$11)`,
		ownerPosID, parentPosID, otherPosID, croPosID, ownerID, parentRiskID, otherRiskID, croID, tenantID, entityID, now.Add(-24*time.Hour))
	mustExecEscalation(t, ctx, pool, `INSERT INTO position_role_bindings(id,tenant_id,position_id,role_template_id,valid_from) VALUES
		($1::uuid,$9::uuid,$5::uuid,$6::uuid,$10),
		($2::uuid,$9::uuid,$7::uuid,$8::uuid,$10),
		($3::uuid,$9::uuid,$11::uuid,$8::uuid,$10),
		($4::uuid,$9::uuid,$12::uuid,$13::uuid,$10)`,
		ownerBindID, parentBindID, otherBindID, croBindID,
		ownerPosID, ownerRoleID, parentPosID, riskRoleID, tenantID, now.Add(-24*time.Hour), otherPosID, croPosID, croRoleID)

	definition, err := json.Marshal(map[string]any{
		"rules": []map[string]any{
			{"id": "owner-route", "legal_entity_id": entityID, "object_type": "MATTER", "object_id": "*", "responsibility": "ACCOUNTABLE_OWNER", "decision_type": "matter.test", "priority": 100, "selector": map[string]any{"kind": "ROLE", "ref": "DEPT_OWNER"}},
			{"id": "risk-route", "legal_entity_id": entityID, "object_type": "MATTER", "object_id": "*", "responsibility": "ESCALATION_OWNER", "decision_type": "matter.test", "priority": 90, "selector": map[string]any{"kind": "ROLE", "ref": "RISK_MANAGER"}},
			{"id": "cro-route", "legal_entity_id": entityID, "object_type": "MATTER", "object_id": "*", "responsibility": "AUTHORIZER", "decision_type": "matter.test", "priority": 80, "selector": map[string]any{"kind": "ROLE", "ref": "CRO"}},
		},
		"escalations": []map[string]any{{
			"id": "overdue-matter", "trigger": "OVERDUE", "steps": []map[string]any{
				{"after": "0s", "responsibility": "ACCOUNTABLE_OWNER", "department_levels_up": 0},
				{"after": "1m", "responsibility": "ESCALATION_OWNER", "department_levels_up": 1},
				{"after": "2m", "responsibility": "AUTHORIZER"},
				{"after": "3m", "responsibility": "SIGNATORY"},
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	mustExecEscalation(t, ctx, pool, `INSERT INTO routing_policies(id,tenant_id,legal_entity_id,code,name,status,current_version,approved_at,version) VALUES($1::uuid,$2::uuid,$3::uuid,'EIA4-ESCALATION','EIA4 escalation','DRAFT',1,$4,1)`, policyID, tenantID, entityID, now.Add(-time.Hour))
	mustExecEscalation(t, ctx, pool, `INSERT INTO routing_policy_versions(id,policy_id,legal_entity_id,version,definition,checksum,effective_from,approved_at) VALUES($1::uuid,$2::uuid,$3::uuid,1,$4::jsonb,'eia4',$5,$5)`, policyVersion, policyID, entityID, string(definition), now.Add(-time.Hour))
	mustExecEscalation(t, ctx, pool, `UPDATE routing_policies SET status='ACTIVE' WHERE id=$1::uuid`, policyID)

	mustExecEscalation(t, ctx, pool, `INSERT INTO matters(id,tenant_id,legal_entity_id,reference,matter_type,status,priority,title,summary,scope,known_facts,missing_facts,contradictions,due_at,created_at,updated_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,'MAT-EIA4-1','EXCEPTION','ASSESSMENT',4,'Escalation test','Overdue work must escalate','{"access":"INTERNAL"}'::jsonb,'{}'::jsonb,'[]'::jsonb,'[]'::jsonb,$4,$5,$5)`, matterID, tenantID, entityID, now, now.Add(-time.Hour))
	mustExecEscalation(t, ctx, pool, `INSERT INTO workflow_instances(id,tenant_id,kind,subject_type,subject_id,state,policy_version,due_at,created_at,updated_at,version)
		VALUES($1::uuid,$2::uuid,$3,'MATTER',$4::uuid,'ACTIVE','matter-work-v1',$5,$6,$6,1)`, workflowID, tenantID, MatterLifecycleWorkflowKind, matterID, now, now.Add(-time.Hour))
	baseContext := map[string]string{
		"type": "MATTER_WORK", "matter_id": matterID, "work_requirement_key": "assessment:owner",
		"decision_type": "matter.test", "materiality": "4", "authority_policy_version": "EIA4-ESCALATION:v1",
	}
	rawContext, _ := json.Marshal(baseContext)
	mustExecEscalation(t, ctx, pool, `INSERT INTO workflow_tasks(id,tenant_id,workflow_id,step_key,responsibility,principal_id,title,status,due_at,context,created_at,updated_at,version)
		VALUES($1::uuid,$2::uuid,$3::uuid,'assessment:owner','ACCOUNTABLE_OWNER',$4::uuid,'Resolve overdue assessment','READY',$5,$6::jsonb,$7,$7,1)`,
		taskID, tenantID, workflowID, ownerID, now, string(rawContext), now.Add(-time.Hour))

	runtimeRepo := workflowruntime.NewPostgresRepository(pool)
	current := now
	coordinator := &MatterEscalationCoordinator{
		Repo: NewPostgresRepository(pool), Runtime: runtimeRepo,
		Authority:  authority.NewEffectivePostgresService(pool),
		Continuity: continuity.NewService(continuity.NewCurrentPostgresRepository(pool)),
		Now:        func() time.Time { return current },
	}
	if processed, err := coordinator.Maintain(ctx, now.Add(-time.Second), 20); err != nil || processed != 1 {
		t.Fatalf("schedule first escalation: processed=%d err=%v", processed, err)
	}

	firstEvent := fireEscalationTimer(t, ctx, pool, tenantID, tenantSlug, taskID, 0, now)
	current = now
	if err := coordinator.Publish(ctx, firstEvent); err != nil {
		t.Fatal(err)
	}
	assertEscalatedTask(t, ctx, pool, taskID, "ACCOUNTABLE_OWNER", ownerID, "0", "ROUTED")

	secondEvent := fireEscalationTimer(t, ctx, pool, tenantID, tenantSlug, taskID, 1, now.Add(time.Minute))
	current = now.Add(time.Minute)
	if err := coordinator.Publish(ctx, secondEvent); err != nil {
		t.Fatal(err)
	}
	assertEscalatedTask(t, ctx, pool, taskID, "ESCALATION_OWNER", parentRiskID, "1", "ROUTED")

	// The normal lifecycle projector may reconcile current work between escalation
	// levels. The trigger must preserve the active overlay when the canonical
	// requirement, due date and authority policy are unchanged.
	reprojected, _ := json.Marshal(baseContext)
	mustExecEscalation(t, ctx, pool, `UPDATE workflow_tasks SET responsibility='ACCOUNTABLE_OWNER',principal_id=$2::uuid,status='READY',context=$3::jsonb,updated_at=$4 WHERE id=$1::uuid`, taskID, ownerID, string(reprojected), current.Add(time.Second))
	assertEscalatedTask(t, ctx, pool, taskID, "ESCALATION_OWNER", parentRiskID, "1", "ROUTED")

	var beforeReplay int64
	if err := pool.QueryRow(ctx, `SELECT version FROM workflow_tasks WHERE id=$1::uuid`, taskID).Scan(&beforeReplay); err != nil {
		t.Fatal(err)
	}
	if err := coordinator.Publish(ctx, secondEvent); err != nil {
		t.Fatal(err)
	}
	var afterReplay int64
	if err := pool.QueryRow(ctx, `SELECT version FROM workflow_tasks WHERE id=$1::uuid`, taskID).Scan(&afterReplay); err != nil {
		t.Fatal(err)
	}
	if afterReplay != beforeReplay {
		t.Fatalf("replayed escalation event changed task version: before=%d after=%d", beforeReplay, afterReplay)
	}

	thirdEvent := fireEscalationTimer(t, ctx, pool, tenantID, tenantSlug, taskID, 2, now.Add(2*time.Minute))
	current = now.Add(2 * time.Minute)
	if err := coordinator.Publish(ctx, thirdEvent); err != nil {
		t.Fatal(err)
	}
	assertEscalatedTask(t, ctx, pool, taskID, "AUTHORIZER", croID, "2", "ROUTED")

	var pendingStep int
	if err := pool.QueryRow(ctx, `SELECT (payload->>'step_index')::int FROM workflow_timers WHERE tenant_id=$1::uuid AND task_id=$2::uuid AND timer_type='MATTER_ESCALATION' AND state='READY'`, tenantID, taskID).Scan(&pendingStep); err != nil {
		t.Fatal(err)
	}
	if pendingStep != 3 {
		t.Fatalf("expected only next escalation level 3 to remain pending, got %d", pendingStep)
	}

	mustExecEscalation(t, ctx, pool, `UPDATE workflow_tasks SET status='COMPLETED',completed_at=$2,updated_at=$2,version=version+1 WHERE id=$1::uuid`, taskID, current.Add(time.Second))
	if _, err := coordinator.Maintain(ctx, current.Add(2*time.Second), 20); err != nil {
		t.Fatal(err)
	}
	var state string
	if err := pool.QueryRow(ctx, `SELECT state FROM workflow_timers WHERE tenant_id=$1::uuid AND task_id=$2::uuid AND timer_type='MATTER_ESCALATION' AND payload->>'step_index'='3'`, tenantID, taskID).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != "CANCELLED" {
		t.Fatalf("completion did not cancel pending escalation timer: %s", state)
	}
}

func fireEscalationTimer(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, tenantSlug, taskID string, step int, at time.Time) workflowruntime.OutboxEvent {
	t.Helper()
	var timerID, workflowID string
	var payload []byte
	err := pool.QueryRow(ctx, `
		UPDATE workflow_timers SET state='FIRED',fired_at=$4
		WHERE id=(
			SELECT id FROM workflow_timers
			WHERE tenant_id=$1::uuid AND task_id=$2::uuid AND timer_type='MATTER_ESCALATION' AND state='READY' AND payload->>'step_index'=$3
			ORDER BY due_at,id LIMIT 1
		)
		RETURNING id::text,workflow_id::text,payload`, tenantID, taskID, string(rune('0'+step)), at).Scan(&timerID, &workflowID, &payload)
	if err != nil {
		t.Fatalf("fire escalation step %d: %v", step, err)
	}
	return workflowruntime.OutboxEvent{
		ID:       "97777777-7777-7777-8777-7777777778" + string(rune('0'+step)),
		TenantID: tenantSlug, AggregateType: "WORKFLOW", AggregateID: workflowID,
		EventType: "WorkflowTimerFired", Payload: payload, OccurredAt: at,
	}
}

func assertEscalatedTask(t *testing.T, ctx context.Context, pool *pgxpool.Pool, taskID, responsibility, principalID, stepIndex, status string) {
	t.Helper()
	var gotResponsibility, gotPrincipal, gotStatus, gotStep, gotEscalationStatus string
	if err := pool.QueryRow(ctx, `SELECT responsibility,COALESCE(principal_id::text,''),status,COALESCE(context->>'escalation_step_index',''),COALESCE(context->>'escalation_status','') FROM workflow_tasks WHERE id=$1::uuid`, taskID).
		Scan(&gotResponsibility, &gotPrincipal, &gotStatus, &gotStep, &gotEscalationStatus); err != nil {
		t.Fatal(err)
	}
	if gotResponsibility != responsibility || gotPrincipal != principalID || gotStatus != "ESCALATED" || gotStep != stepIndex || gotEscalationStatus != status {
		t.Fatalf("unexpected escalation task state: responsibility=%s principal=%s status=%s step=%s escalation_status=%s", gotResponsibility, gotPrincipal, gotStatus, gotStep, gotEscalationStatus)
	}
}

func cleanupEscalationFixture(ctx context.Context, pool *pgxpool.Pool, tenantID, policyID string) {
	_, _ = pool.Exec(ctx, `DELETE FROM workflow_timers WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM inbox_receipts WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM workflow_events WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM workflow_tasks WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM workflow_instances WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM effective_authority_routes WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM routing_policy_versions WHERE policy_id=$1::uuid`, policyID)
	_, _ = pool.Exec(ctx, `DELETE FROM routing_policies WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM position_role_bindings WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM org_positions WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM role_templates WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM matters WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM principals WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM legal_entities WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
}

func mustExecEscalation(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(ctx, query, args...); err != nil {
		t.Fatal(err)
	}
}
