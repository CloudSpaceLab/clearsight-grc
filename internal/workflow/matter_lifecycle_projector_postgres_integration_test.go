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

func TestMatterLifecycleProjectorConvergesCurrentAuthorityAndVisibility(t *testing.T) {
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
		tenantID    = "9f777777-7777-7777-8777-777777777771"
		entityID    = "9f777777-7777-7777-8777-777777777772"
		firstID     = "9f777777-7777-7777-8777-777777777773"
		secondID    = "9f777777-7777-7777-8777-777777777774"
		matterID    = "9f777777-7777-7777-8777-777777777775"
		responseID  = "9f777777-7777-7777-8777-777777777776"
		emptyMatter = "9f777777-7777-7777-8777-777777777777"
		ownerMatter = "9f777777-7777-7777-8777-777777777778"
	)
	now := time.Date(2026, 8, 7, 22, 45, 0, 0, time.UTC)
	oldEventTime := now.Add(-2 * time.Hour)

	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID) })
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'matter-work-test','Matter Work Test')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2::uuid,'BANK-NG','Bank NG','NG',$3)`, entityID, tenantID, now.Add(-24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES
		($1::uuid,$3::uuid,'PERSON','First recorder','ACTIVE',$4),
		($2::uuid,$3::uuid,'PERSON','Second recorder','ACTIVE',$4)`, firstID, secondID, tenantID, now.Add(-24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO matters(id,tenant_id,legal_entity_id,reference,matter_type,status,priority,title,summary,scope,known_facts,missing_facts,contradictions,created_at,updated_at)
		VALUES
		($1::uuid,$3::uuid,$4::uuid,'MAT-WORK-2','AUTHORITY_REQUEST','RESPONSE_PREPARATION',4,'Respond to authority','Await acknowledgement','{"access":"INTERNAL"}'::jsonb,'{}'::jsonb,'[]'::jsonb,'[]'::jsonb,$5,$5),
		($2::uuid,$3::uuid,$4::uuid,'MAT-WORK-EMPTY','CONTROL_GAP','DRAFT',2,'No lifecycle work','No current work','{"access":"INTERNAL"}'::jsonb,'{}'::jsonb,'[]'::jsonb,'[]'::jsonb,$5,$5),
		($6::uuid,$3::uuid,$4::uuid,'MAT-WORK-OWNER','CONTROL_GAP','TRIAGE',4,'Restore an unavailable source','Confirm scope and owner','{"access":"INTERNAL"}'::jsonb,'{}'::jsonb,'[]'::jsonb,'[]'::jsonb,$5,$5)`, matterID, emptyMatter, tenantID, entityID, oldEventTime, ownerMatter); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE matters SET owner_principal_id=$2::uuid WHERE id=$1::uuid`, ownerMatter, secondID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO response_packages(id,tenant_id,matter_id,purpose,audience,status,manifest,transmitted_at,created_at,updated_at,version)
		VALUES($1::uuid,$2::uuid,$3::uuid,'NDPC response','NDPC','TRANSMITTED','[]'::jsonb,$4,$5,$4,2)`, responseID, tenantID, matterID, now.Add(-30*time.Minute), oldEventTime); err != nil {
		t.Fatal(err)
	}

	if _, err := pool.Exec(ctx, `INSERT INTO responsibility_assignments(tenant_id,legal_entity_id,principal_id,responsibility,object_type,object_id,priority,valid_from,valid_until,policy_version,decision_type) VALUES
		($1::uuid,$2::uuid,$3::uuid,'ACKNOWLEDGEMENT_RECORDER','MATTER',$5::uuid,100,$6,$7,'assignment:old','matter.response.transition'),
		($1::uuid,$2::uuid,$4::uuid,'ACKNOWLEDGEMENT_RECORDER','MATTER',$5::uuid,100,$7,NULL,'assignment:current','matter.response.transition'),
		($1::uuid,$2::uuid,$4::uuid,'ACCOUNTABLE_OWNER','MATTER',$8::uuid,100,$7,NULL,'assignment:owner','matter.transition')`,
		tenantID, entityID, firstID, secondID, matterID, oldEventTime.Add(-time.Hour), now.Add(-time.Hour), ownerMatter); err != nil {
		t.Fatal(err)
	}

	repo := NewPostgresRepository(pool)
	continuityService := continuity.NewService(continuity.NewCurrentPostgresRepository(pool))
	projector := &MatterLifecycleProjector{Repo: repo, Continuity: continuityService, Authority: authority.NewEffectivePostgresService(pool), Now: func() time.Time { return now }}
	event := workflowruntime.OutboxEvent{ID: "97001", TenantID: "matter-work-test", AggregateType: "MATTER", AggregateID: matterID, EventType: "RESPONSE_PACKAGE_STATE_CHANGED", OccurredAt: oldEventTime}
	if err := projector.Publish(ctx, event); err != nil {
		t.Fatal(err)
	}

	service := NewService(repo)
	tasks, err := service.List(ctx, ListFilter{TenantID: "matter-work-test", PrincipalID: secondID, ActiveOnly: true, VisibleMatterWorkOnly: true, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected one current acknowledgement task for the current actor, got %#v", tasks)
	}
	task := tasks[0]
	if task.WorkflowKind != MatterLifecycleWorkflowKind || task.StepKey != "response:"+responseID+":ACKNOWLEDGED" || task.Status != StatusReady || task.PrincipalID != secondID {
		t.Fatalf("unexpected current lifecycle task: %#v", task)
	}
	if task.Context["primary_action"] != "Record acknowledgement" || task.Context["routing_status"] != "DIRECT" || task.Context["target_status"] != "ACKNOWLEDGED" {
		t.Fatalf("unexpected lifecycle task context: %#v", task.Context)
	}
	oldTasks, err := service.List(ctx, ListFilter{TenantID: "matter-work-test", PrincipalID: firstID, ActiveOnly: true, VisibleMatterWorkOnly: true, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(oldTasks) != 0 {
		t.Fatalf("expired authority assignment remained actor-visible: %#v", oldTasks)
	}

	var workflowCount, eventCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM workflow_instances wi JOIN tenants t ON t.id=wi.tenant_id WHERE t.slug='matter-work-test' AND wi.kind=$1 AND wi.subject_id=$2::uuid`, MatterLifecycleWorkflowKind, matterID).Scan(&workflowCount); err != nil {
		t.Fatal(err)
	}
	if workflowCount != 1 {
		t.Fatalf("expected one deterministic lifecycle workflow, got %d", workflowCount)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM workflow_events we JOIN workflow_instances wi ON wi.id=we.workflow_id JOIN tenants t ON t.id=we.tenant_id WHERE t.slug='matter-work-test' AND wi.kind=$1 AND wi.subject_id=$2::uuid AND we.event_type='WORK_REQUIREMENTS_RECONCILED'`, MatterLifecycleWorkflowKind, matterID).Scan(&eventCount); err != nil {
		t.Fatal(err)
	}
	if eventCount != 1 {
		t.Fatalf("expected one material reconciliation event, got %d", eventCount)
	}

	if err := projector.Publish(ctx, event); err != nil {
		t.Fatalf("duplicate delivery must be idempotent: %v", err)
	}
	var afterDuplicate int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM workflow_events we JOIN workflow_instances wi ON wi.id=we.workflow_id JOIN tenants t ON t.id=we.tenant_id WHERE t.slug='matter-work-test' AND wi.kind=$1 AND wi.subject_id=$2::uuid AND we.event_type='WORK_REQUIREMENTS_RECONCILED'`, MatterLifecycleWorkflowKind, matterID).Scan(&afterDuplicate); err != nil {
		t.Fatal(err)
	}
	if afterDuplicate != eventCount {
		t.Fatalf("duplicate delivery emitted a duplicate reconciliation event: before=%d after=%d", eventCount, afterDuplicate)
	}

	if err := projector.ReconcileMatter(ctx, "matter-work-test", emptyMatter, now); err != nil {
		t.Fatal(err)
	}
	var emptyCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM workflow_instances wi JOIN tenants t ON t.id=wi.tenant_id WHERE t.slug='matter-work-test' AND wi.kind=$1 AND wi.subject_id=$2::uuid`, MatterLifecycleWorkflowKind, emptyMatter).Scan(&emptyCount); err != nil {
		t.Fatal(err)
	}
	if emptyCount != 0 {
		t.Fatalf("empty Matter created lifecycle Workflow bloat: %d", emptyCount)
	}

	if _, err := projector.Maintain(ctx, now, 50); err != nil {
		t.Fatal(err)
	}
	ownerTasks, err := service.List(ctx, ListFilter{TenantID: "matter-work-test", PrincipalID: secondID, ActiveOnly: true, VisibleMatterWorkOnly: true, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	var ownerTask *Task
	for index := range ownerTasks {
		if ownerTasks[index].MatterID == ownerMatter {
			ownerTask = &ownerTasks[index]
			break
		}
	}
	if ownerTask == nil || ownerTask.StepKey != "matter:"+ownerMatter+":ASSESSMENT" || ownerTask.Context["primary_action"] != "Confirm scope and owner" || ownerTask.Context["target_status"] != "ASSESSMENT" {
		t.Fatalf("assigned owner initial-review handoff was not projected: %#v", ownerTasks)
	}

	restricted, _ := json.Marshal(map[string]any{"access": "RESTRICTED", "allowed_principal_ids": []string{firstID}})
	if _, err := pool.Exec(ctx, `UPDATE matters SET scope=$2::jsonb,updated_at=$3 WHERE id=$1::uuid`, matterID, string(restricted), now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := projector.ReconcileMatter(ctx, "matter-work-test", matterID, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	var principal string
	var status Status
	var routing string
	if err := pool.QueryRow(ctx, `SELECT COALESCE(wt.principal_id::text,''),wt.status,wt.context->>'routing_status' FROM workflow_tasks wt JOIN workflow_instances wi ON wi.id=wt.workflow_id JOIN tenants t ON t.id=wt.tenant_id WHERE t.slug='matter-work-test' AND wi.kind=$1 AND wi.subject_id=$2::uuid AND wt.step_key=$3`, MatterLifecycleWorkflowKind, matterID, "response:"+responseID+":ACKNOWLEDGED").Scan(&principal, &status, &routing); err != nil {
		t.Fatal(err)
	}
	if principal != "" || status != StatusBlocked || routing != "NO_VISIBLE_CANDIDATE" {
		t.Fatalf("restricted Matter assignment did not fail closed: principal=%q status=%s routing=%s", principal, status, routing)
	}
}
