//go:build postgres && postgresintegration

package workflow

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMatterActionProjectorIsIdempotentAndKeepsTaskDerived(t *testing.T) {
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
		tenantID  = "96666666-6666-7666-8666-666666666661"
		ownerID   = "96666666-6666-7666-8666-666666666662"
		matterID  = "96666666-6666-7666-8666-666666666663"
		actionID  = "96666666-6666-7666-8666-666666666664"
		action2ID = "96666666-6666-7666-8666-666666666665"
	)
	occurred := time.Date(2026, 8, 7, 18, 30, 0, 0, time.UTC)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID) })
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'action-work-test','Action Work Test')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,external_ref,display_name) VALUES($1::uuid,$2::uuid,'PERSON','action-owner','Action Owner')`, ownerID, tenantID); err != nil {
		t.Fatal(err)
	}
	scope, _ := json.Marshal(map[string]any{"access": "RESTRICTED", "allowed_principal_ids": []string{ownerID}})
	if _, err := pool.Exec(ctx, `
		INSERT INTO matters(id,tenant_id,reference,matter_type,status,priority,title,summary,scope,known_facts,missing_facts,contradictions,owner_principal_id,created_at,updated_at)
		VALUES($1::uuid,$2::uuid,'MAT-WORK-1','CONTROL_GAP','ACTION_IN_PROGRESS',5,'Material action matter','Test actor work projection',$3::jsonb,'{}'::jsonb,'[]'::jsonb,'[]'::jsonb,$4::uuid,$5,$5)`,
		matterID, tenantID, string(scope), ownerID, occurred); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO matter_actions(id,tenant_id,matter_id,title,description,owner_principal_id,status,created_at,updated_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,'Restore access review','Restore the review control',$4::uuid,'PLANNED',$5,$5)`,
		actionID, tenantID, matterID, ownerID, occurred); err != nil {
		t.Fatal(err)
	}

	repo := NewPostgresRepository(pool)
	projector := &MatterActionProjector{Repo: repo}
	payload, _ := json.Marshal(matterActionPayload{ID: actionID, TenantID: "action-work-test", MatterID: matterID, Title: "Restore access review", OwnerPrincipalID: ownerID, Status: "PLANNED"})
	event := workflowruntime.OutboxEvent{ID: "96001", TenantID: "action-work-test", AggregateType: "MATTER", AggregateID: matterID, EventType: "ACTION_ADDED", Payload: payload, OccurredAt: occurred}
	if err := projector.Publish(ctx, event); err != nil {
		t.Fatal(err)
	}
	if err := projector.Publish(ctx, event); err != nil {
		t.Fatalf("duplicate projection event must be idempotent: %v", err)
	}

	service := NewService(repo)
	tasks, err := service.List(ctx, ListFilter{
		TenantID: "action-work-test", PrincipalID: ownerID, WorkflowKind: MatterActionWorkflowKind,
		ActiveOnly: true, VisibleMatterWorkOnly: true, Limit: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected exactly one derived task after duplicate delivery, got %d", len(tasks))
	}
	task := tasks[0]
	if task.Status != StatusReady || task.DueAt != nil || task.Context["matter_id"] != matterID || task.Context["action_id"] != actionID {
		t.Fatalf("unexpected projected task: %#v", task)
	}
	if task.Context["command_name"] != "matter.action.transition" || task.Context["subresource_type"] != "ACTION" || task.Context["subresource_id"] != actionID || task.Context["allowed_targets"] != "IN_PROGRESS,BLOCKED,CANCELLED" {
		t.Fatalf("Matter Action task is not executable from canonical packet context: %#v", task.Context)
	}
	if task.WorkflowKind != MatterActionWorkflowKind || task.MatterID != matterID || task.MatterPriority != 5 || !MatterActionVisibleTo(task, ownerID) {
		t.Fatalf("canonical Matter metadata was not joined into Task read: %#v", task)
	}

	implementedAt := occurred.Add(time.Hour)
	if _, err := pool.Exec(ctx, `UPDATE matter_actions SET status='IMPLEMENTED',implemented_at=$2,updated_at=$2 WHERE id=$1::uuid`, actionID, implementedAt); err != nil {
		t.Fatal(err)
	}
	implementedPayload, _ := json.Marshal(matterActionPayload{ID: actionID, TenantID: "action-work-test", MatterID: matterID, Title: "Restore access review", OwnerPrincipalID: ownerID, Status: "IMPLEMENTED"})
	implemented := workflowruntime.OutboxEvent{ID: "96002", TenantID: "action-work-test", AggregateType: "MATTER", AggregateID: matterID, EventType: "ACTION_STATE_CHANGED", Payload: implementedPayload, OccurredAt: implementedAt}
	if err := projector.Publish(ctx, implemented); err != nil {
		t.Fatal(err)
	}

	activeDue := occurred.Add(2 * time.Hour)
	if _, err := pool.Exec(ctx, `
		INSERT INTO matter_actions(id,tenant_id,matter_id,title,description,owner_principal_id,status,due_at,created_at,updated_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,'Complete second action','Second active item',$4::uuid,'PLANNED',$5,$6,$6)`,
		action2ID, tenantID, matterID, ownerID, activeDue, occurred.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	secondPayload, _ := json.Marshal(matterActionPayload{ID: action2ID, TenantID: "action-work-test", MatterID: matterID, Title: "Complete second action", OwnerPrincipalID: ownerID, Status: "PLANNED", DueAt: &activeDue})
	second := workflowruntime.OutboxEvent{ID: "96003", TenantID: "action-work-test", AggregateType: "MATTER", AggregateID: matterID, EventType: "ACTION_ADDED", Payload: secondPayload, OccurredAt: occurred.Add(-time.Hour)}
	if err := projector.Publish(ctx, second); err != nil {
		t.Fatal(err)
	}

	// The newer completed task must be filtered before LIMIT so the older active
	// task remains visible to Today. Matter access is also evaluated before LIMIT.
	active, err := service.List(ctx, ListFilter{
		TenantID: "action-work-test", PrincipalID: ownerID, WorkflowKind: MatterActionWorkflowKind,
		ActiveOnly: true, VisibleMatterWorkOnly: true, Limit: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 1 || active[0].Context["action_id"] != action2ID || active[0].Status != StatusReady {
		t.Fatalf("terminal work displaced active work before limit: %#v", active)
	}

	allTasks, err := service.List(ctx, ListFilter{TenantID: "action-work-test", PrincipalID: ownerID, WorkflowKind: MatterActionWorkflowKind, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(allTasks) != 2 {
		t.Fatalf("expected two projected tasks including terminal history, got %#v", allTasks)
	}

	var receipts, workflows, taskRows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM inbox_receipts WHERE tenant_id=$1::uuid AND consumer=$2`, tenantID, matterActionProjectionConsumer).Scan(&receipts); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM workflow_instances WHERE tenant_id=$1::uuid AND kind='MATTER_ACTION'`, tenantID).Scan(&workflows); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM workflow_tasks wt JOIN workflow_instances wi ON wi.id=wt.workflow_id WHERE wi.tenant_id=$1::uuid AND wi.kind='MATTER_ACTION'`, tenantID).Scan(&taskRows); err != nil {
		t.Fatal(err)
	}
	if receipts != 3 || workflows != 2 || taskRows != 2 {
		t.Fatalf("projection cardinality receipts=%d workflows=%d tasks=%d", receipts, workflows, taskRows)
	}
}

func TestMatterActionProjectorReassignsCurrentTask(t *testing.T) {
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
		tenantID = "97666666-6666-7766-8766-666666666661"
		oldOwner = "97666666-6666-7766-8766-666666666662"
		newOwner = "97666666-6666-7766-8766-666666666663"
		matterID = "97666666-6666-7766-8766-666666666664"
		actionID = "97666666-6666-7766-8766-666666666665"
	)
	occurred := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID) })
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'action-reassignment-test','Action Reassignment Test')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name) VALUES
		($1::uuid,$3::uuid,'PERSON','Previous performer'),
		($2::uuid,$3::uuid,'PERSON','Current performer')`, oldOwner, newOwner, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO matters(id,tenant_id,reference,matter_type,status,priority,title,summary,scope,known_facts,missing_facts,contradictions,owner_principal_id,created_at,updated_at)
		VALUES($1::uuid,$2::uuid,'MAT-REASSIGN-1','CONTROL_GAP','ACTION_IN_PROGRESS',4,'Reassign action','Test current workflow ownership','{}'::jsonb,'{}'::jsonb,'[]'::jsonb,'[]'::jsonb,$3::uuid,$4,$4)`, matterID, tenantID, oldOwner, occurred); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO matter_actions(id,tenant_id,matter_id,title,description,owner_principal_id,status,created_at,updated_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,'Update checklist','Map every section',$4::uuid,'PLANNED',$5,$5)`, actionID, tenantID, matterID, oldOwner, occurred); err != nil {
		t.Fatal(err)
	}

	projector := &MatterActionProjector{Repo: NewPostgresRepository(pool)}
	addedPayload, _ := json.Marshal(matterActionPayload{ID: actionID, TenantID: "action-reassignment-test", MatterID: matterID, Title: "Update checklist", OwnerPrincipalID: oldOwner, Status: "PLANNED"})
	if err := projector.Publish(ctx, workflowruntime.OutboxEvent{ID: "97001", TenantID: "action-reassignment-test", AggregateType: "MATTER", AggregateID: matterID, EventType: "ACTION_ADDED", Payload: addedPayload, OccurredAt: occurred}); err != nil {
		t.Fatal(err)
	}
	reassignedPayload, _ := json.Marshal(map[string]any{
		"action":                      map[string]any{"id": actionID, "tenant_id": "action-reassignment-test", "matter_id": matterID, "title": "Update checklist", "owner_principal_id": newOwner, "status": "PLANNED"},
		"previous_owner_principal_id": oldOwner, "owner_principal_id": newOwner, "rationale": "Assign the current process owner.",
	})
	if err := projector.Publish(ctx, workflowruntime.OutboxEvent{ID: "97002", TenantID: "action-reassignment-test", AggregateType: "MATTER", AggregateID: matterID, EventType: "ACTION_ASSIGNED", Payload: reassignedPayload, OccurredAt: occurred.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}

	var principalID string
	var taskVersion int64
	var taskCount int
	if err := pool.QueryRow(ctx, `SELECT wt.principal_id::text,wt.version FROM workflow_tasks wt JOIN workflow_instances wi ON wi.id=wt.workflow_id WHERE wi.tenant_id=$1::uuid AND wi.kind='MATTER_ACTION' AND wi.subject_id=$2::uuid`, tenantID, actionID).Scan(&principalID, &taskVersion); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM workflow_tasks wt JOIN workflow_instances wi ON wi.id=wt.workflow_id WHERE wi.tenant_id=$1::uuid AND wi.kind='MATTER_ACTION' AND wi.subject_id=$2::uuid`, tenantID, actionID).Scan(&taskCount); err != nil {
		t.Fatal(err)
	}
	if principalID != newOwner || taskVersion != 2 || taskCount != 1 {
		t.Fatalf("reassignment did not converge: principal=%s version=%d tasks=%d", principalID, taskVersion, taskCount)
	}
	service := NewService(NewPostgresRepository(pool))
	oldTasks, err := service.List(ctx, ListFilter{TenantID: "action-reassignment-test", PrincipalID: oldOwner, WorkflowKind: MatterActionWorkflowKind, ActiveOnly: true, VisibleMatterWorkOnly: true, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	newTasks, err := service.List(ctx, ListFilter{TenantID: "action-reassignment-test", PrincipalID: newOwner, WorkflowKind: MatterActionWorkflowKind, ActiveOnly: true, VisibleMatterWorkOnly: true, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(oldTasks) != 0 || len(newTasks) != 1 {
		t.Fatalf("active work did not move to the new performer: old=%d new=%d", len(oldTasks), len(newTasks))
	}
}
