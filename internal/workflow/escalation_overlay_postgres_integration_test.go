//go:build postgres && postgresintegration

package workflow

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestEscalationOverlayDoesNotSurviveMaterialRoutingChange(t *testing.T) {
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
		tenantID   = "97777777-7777-7777-8777-777777777731"
		workflowID = "97777777-7777-7777-8777-777777777732"
		taskID     = "97777777-7777-7777-8777-777777777733"
		subjectID  = "97777777-7777-7777-8777-777777777734"
	)
	cleanup := func(cleanCtx context.Context) {
		_, _ = pool.Exec(cleanCtx, `DELETE FROM workflow_tasks WHERE id=$1::uuid`, taskID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM workflow_instances WHERE id=$1::uuid`, workflowID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	}
	cleanup(ctx)
	t.Cleanup(func() { cleanup(context.Background()) })

	now := time.Now().UTC().Truncate(time.Second)
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'overlay-semantic-test','Overlay semantic test')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO workflow_instances(id,tenant_id,kind,subject_type,subject_id,state,policy_version,due_at,created_at,updated_at)
		VALUES($1::uuid,$2::uuid,'MATTER_LIFECYCLE','MATTER',$3::uuid,'ACTIVE','matter-work-v1',$4,$4,$4)`, workflowID, tenantID, subjectID, now); err != nil {
		t.Fatal(err)
	}
	oldContext := `{
		"escalation_active":"true","work_requirement_key":"assessment:owner","escalation_policy_version":"BANK:v1",
		"authority_policy_version":"BANK:v1","decision_type":"matter.test","materiality":"3","command_name":"matter.test",
		"target_status":"ASSESSMENT","allowed_targets":"DECISION_REQUIRED","sequence_policy_version":"SEQ:v1"
	}`
	if _, err := pool.Exec(ctx, `
		INSERT INTO workflow_tasks(id,tenant_id,workflow_id,step_key,responsibility,title,status,due_at,context,created_at,updated_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,'assessment:owner','ESCALATION_OWNER','Resolve assessment','ESCALATED',$4,$5::jsonb,$4,$4)`, taskID, tenantID, workflowID, now, oldContext); err != nil {
		t.Fatal(err)
	}

	newContext := `{
		"work_requirement_key":"assessment:owner","authority_policy_version":"BANK:v1",
		"decision_type":"matter.test","materiality":"4","command_name":"matter.test",
		"target_status":"ASSESSMENT","allowed_targets":"DECISION_REQUIRED","sequence_policy_version":"SEQ:v1"
	}`
	if _, err := pool.Exec(ctx, `
		UPDATE workflow_tasks
		SET responsibility='ACCOUNTABLE_OWNER',status='READY',context=$2::jsonb,updated_at=$3,version=version+1
		WHERE id=$1::uuid`, taskID, newContext, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	var responsibility, status, escalationActive, materiality string
	if err := pool.QueryRow(ctx, `SELECT responsibility,status,COALESCE(context->>'escalation_active',''),COALESCE(context->>'materiality','') FROM workflow_tasks WHERE id=$1::uuid`, taskID).
		Scan(&responsibility, &status, &escalationActive, &materiality); err != nil {
		t.Fatal(err)
	}
	if responsibility != "ACCOUNTABLE_OWNER" || status != "READY" || escalationActive != "" || materiality != "4" {
		t.Fatalf("material routing change incorrectly preserved escalation overlay: responsibility=%s status=%s active=%q materiality=%s", responsibility, status, escalationActive, materiality)
	}
}
