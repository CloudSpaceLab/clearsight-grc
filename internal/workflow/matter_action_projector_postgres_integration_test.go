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
		tenantID = "96666666-6666-7666-8666-666666666661"
		ownerID  = "96666666-6666-7666-8666-666666666662"
		matterID = "96666666-6666-7666-8666-666666666663"
		actionID = "96666666-6666-7666-8666-666666666664"
	)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID) })
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'action-work-test','Action Work Test')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,external_ref,display_name) VALUES($1::uuid,$2::uuid,'PERSON','action-owner','Action Owner')`, ownerID, tenantID); err != nil {
		t.Fatal(err)
	}

	repo := NewPostgresRepository(pool)
	projector := &MatterActionProjector{Repo: repo}
	occurred := time.Date(2026, 8, 7, 18, 30, 0, 0, time.UTC)
	payload, _ := json.Marshal(matterActionPayload{ID: actionID, TenantID: "action-work-test", MatterID: matterID, Title: "Restore access review", OwnerPrincipalID: ownerID, Status: "PLANNED"})
	event := workflowruntime.OutboxEvent{ID: 96001, TenantID: "action-work-test", AggregateType: "MATTER", AggregateID: matterID, EventType: "ACTION_ADDED", Payload: payload, OccurredAt: occurred}
	if err := projector.Publish(ctx, event); err != nil {
		t.Fatal(err)
	}
	if err := projector.Publish(ctx, event); err != nil {
		t.Fatalf("duplicate projection event must be idempotent: %v", err)
	}

	service := NewService(repo)
	tasks, err := service.List(ctx, ListFilter{TenantID: "action-work-test", PrincipalID: ownerID, Limit: 20})
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

	implementedPayload, _ := json.Marshal(matterActionPayload{ID: actionID, TenantID: "action-work-test", MatterID: matterID, Title: "Restore access review", OwnerPrincipalID: ownerID, Status: "IMPLEMENTED"})
	implemented := workflowruntime.OutboxEvent{ID: 96002, TenantID: "action-work-test", AggregateType: "MATTER", AggregateID: matterID, EventType: "ACTION_STATE_CHANGED", Payload: implementedPayload, OccurredAt: occurred.Add(time.Hour)}
	if err := projector.Publish(ctx, implemented); err != nil {
		t.Fatal(err)
	}
	tasks, err = service.List(ctx, ListFilter{TenantID: "action-work-test", PrincipalID: ownerID, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || tasks[0].Status != StatusCompleted || tasks[0].CompletedAt == nil {
		t.Fatalf("implemented Matter Action did not complete its derived task: %#v", tasks)
	}

	var receipts, workflows, taskRows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM inbox_receipts WHERE tenant_id=$1::uuid AND consumer=$2`, tenantID, matterActionProjectionConsumer).Scan(&receipts); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM workflow_instances WHERE tenant_id=$1::uuid AND kind='MATTER_ACTION' AND subject_id=$2::uuid`, tenantID, actionID).Scan(&workflows); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM workflow_tasks wt JOIN workflow_instances wi ON wi.id=wt.workflow_id WHERE wi.tenant_id=$1::uuid AND wi.kind='MATTER_ACTION' AND wi.subject_id=$2::uuid`, tenantID, actionID).Scan(&taskRows); err != nil {
		t.Fatal(err)
	}
	if receipts != 2 || workflows != 1 || taskRows != 1 {
		t.Fatalf("projection cardinality receipts=%d workflows=%d tasks=%d", receipts, workflows, taskRows)
	}
}
