//go:build postgres && postgresintegration

package workflow

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestEvidenceRequestProjectorConvergesRecipientVisibilityWithoutDuplicateWork(t *testing.T) {
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
		tenantID          = "95555555-5555-7555-8555-555555555551"
		entityID          = "95555555-5555-7555-8555-555555555558"
		creatorID         = "95555555-5555-7555-8555-555555555552"
		recipientA        = "95555555-5555-7555-8555-555555555553"
		recipientB        = "95555555-5555-7555-8555-555555555554"
		matterID          = "95555555-5555-7555-8555-555555555555"
		requestID         = "95555555-5555-7555-8555-555555555556"
		terminalRequestID = "95555555-5555-7555-8555-555555555557"
	)
	// Actor visibility uses database now(), so keep this integration fixture
	// relative to the executing database clock rather than a wall-clock date.
	now := time.Now().UTC().Truncate(time.Second)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID) })

	mustExecEvidenceWork(t, ctx, pool, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'evidence-work-test','Evidence Work Test')`, tenantID)
	mustExecEvidenceWork(t, ctx, pool, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2::uuid,'ENTITY-A','Entity A','NG',$3)`, entityID, tenantID, now.Add(-24*time.Hour))
	mustExecEvidenceWork(t, ctx, pool, `
		INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES
		($1::uuid,$4::uuid,'PERSON','Requester','ACTIVE',$5),
		($2::uuid,$4::uuid,'PERSON','Recipient A','ACTIVE',$5),
		($3::uuid,$4::uuid,'PERSON','Recipient B','ACTIVE',$5)`,
		creatorID, recipientA, recipientB, tenantID, now.Add(-24*time.Hour))
	mustExecEvidenceWork(t, ctx, pool, `
		INSERT INTO matters(id,tenant_id,legal_entity_id,reference,matter_type,status,priority,title,summary,scope,known_facts,missing_facts,contradictions,created_at,updated_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,'MAT-EVIDENCE-WORK','CONTROL_GAP','TRIAGE',4,'Restricted evidence matter','Recipient visibility drives work',$4::jsonb,'{}'::jsonb,'[]'::jsonb,'[]'::jsonb,$5,$5)`,
		matterID, tenantID, entityID, restrictedScope(recipientA), now)
	mustExecEvidenceWork(t, ctx, pool, `
		INSERT INTO capture_requests(
			id,tenant_id,legal_entity_id,subject_type,subject_id,title,purpose,why_you,sensitivity,audience_type,
			recipient_type,recipient_principal_id,recipient_state,recipient_revision,
			estimated_minutes,deadline,known_facts,fields,status,created_by,version,created_at,updated_at
		) VALUES(
			$1::uuid,$2::uuid,$3::uuid,'MATTER',$4,'Provide current evidence','Confirm the current control evidence.','You are the current respondent.','RESTRICTED','INTERNAL',
			'INTERNAL_PRINCIPAL',$5::uuid,'ASSIGNED',1,3,$6,'{}'::jsonb,'[{"id":"confirm","label":"Confirm","type":"text","required":true}]'::jsonb,'READY',$7::uuid,1,$8,$8
		)`, requestID, tenantID, entityID, matterID, recipientA, now.Add(2*time.Hour), creatorID, now)
	mustExecEvidenceWork(t, ctx, pool, `
		INSERT INTO capture_requests(
			id,tenant_id,legal_entity_id,subject_type,subject_id,title,purpose,why_you,sensitivity,audience_type,
			recipient_type,recipient_principal_id,recipient_state,recipient_revision,
			estimated_minutes,deadline,known_facts,fields,status,created_by,version,created_at,updated_at
		) VALUES(
			$1::uuid,$2::uuid,$3::uuid,'MATTER',$4,'Historical submitted request','Already complete.','Historical.','RESTRICTED','INTERNAL',
			'INTERNAL_PRINCIPAL',$5::uuid,'ASSIGNED',1,3,$6,'{}'::jsonb,'[{"id":"confirm","label":"Confirm","type":"text","required":true}]'::jsonb,'SUBMITTED',$7::uuid,2,$8,$8
		)`, terminalRequestID, tenantID, entityID, matterID, recipientA, now.Add(time.Hour), creatorID, now.Add(-time.Hour))

	repo := NewPostgresRepository(pool)
	projector := &EvidenceRequestProjector{Repo: repo}
	if processed, err := projector.Maintain(ctx, now, 20); err != nil || processed != 1 {
		t.Fatalf("initial evidence work projection processed=%d err=%v", processed, err)
	}
	service := NewService(repo)
	assignedA, err := service.List(ctx, ListFilter{TenantID: "evidence-work-test", PrincipalID: recipientA, ActiveOnly: true, VisibleActorWorkOnly: true, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(assignedA) != 1 || assignedA[0].WorkflowKind != EvidenceRequestWorkflowKind || assignedA[0].EvidenceRequestID != requestID || !ActorWorkVisibleTo(assignedA[0], recipientA) {
		t.Fatalf("canonical evidence request was not visible as one actor item: %#v", assignedA)
	}
	if assignedA[0].TenantID != tenantID {
		t.Fatalf("workflow Task tenant identity = %q, want canonical UUID %q", assignedA[0].TenantID, tenantID)
	}
	workflowID, taskID := assignedA[0].WorkflowID, assignedA[0].ID
	assertEvidenceWorkCardinality(t, ctx, pool, tenantID, 1, 1)

	wrongAt := now.Add(time.Minute)
	mustExecEvidenceWork(t, ctx, pool, `
		UPDATE capture_requests
		SET recipient_state='REASSIGNMENT_REQUIRED',recipient_issue_reason='Wrong owner',version=version+1,updated_at=$2
		WHERE id=$1::uuid`, requestID, wrongAt)
	if processed, err := projector.Maintain(ctx, wrongAt, 20); err != nil || processed != 1 {
		t.Fatalf("wrong-recipient convergence processed=%d err=%v", processed, err)
	}
	assignedA, err = service.List(ctx, ListFilter{TenantID: "evidence-work-test", PrincipalID: recipientA, ActiveOnly: true, VisibleActorWorkOnly: true, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(assignedA) != 0 {
		t.Fatalf("wrong-recipient work remained assigned: %#v", assignedA)
	}
	assertEvidenceProjectedIdentity(t, ctx, pool, tenantID, workflowID, taskID, "CANCELLED", "")

	reassignedAt := now.Add(2 * time.Minute)
	mustExecEvidenceWork(t, ctx, pool, `UPDATE matters SET scope=$2::jsonb,updated_at=$3 WHERE id=$1::uuid`, matterID, restrictedScope(recipientB), reassignedAt)
	mustExecEvidenceWork(t, ctx, pool, `
		UPDATE capture_requests
		SET recipient_principal_id=$2::uuid,recipient_state='ASSIGNED',recipient_revision=recipient_revision+1,recipient_issue_reason='',version=version+1,updated_at=$3
		WHERE id=$1::uuid`, requestID, recipientB, reassignedAt)
	if processed, err := projector.Maintain(ctx, reassignedAt, 20); err != nil || processed != 1 {
		t.Fatalf("reassignment convergence processed=%d err=%v", processed, err)
	}
	assignedB, err := service.List(ctx, ListFilter{TenantID: "evidence-work-test", PrincipalID: recipientB, ActiveOnly: true, VisibleActorWorkOnly: true, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(assignedB) != 1 || assignedB[0].WorkflowID != workflowID || assignedB[0].ID != taskID || assignedB[0].Status != StatusReady {
		t.Fatalf("reassignment duplicated or failed to move actor work: %#v", assignedB)
	}
	assertEvidenceWorkCardinality(t, ctx, pool, tenantID, 1, 1)

	visibilityLostAt := now.Add(3 * time.Minute)
	malformed := fmt.Sprintf("{\"access\":\"RESTRICTED\",\"allowed_principal_ids\":[\"%s\",7]}", recipientB)
	mustExecEvidenceWork(t, ctx, pool, `UPDATE matters SET scope=$2::jsonb,updated_at=$3 WHERE id=$1::uuid`, matterID, malformed, visibilityLostAt)
	if processed, err := projector.Maintain(ctx, visibilityLostAt, 20); err != nil || processed != 1 {
		t.Fatalf("visibility-loss convergence processed=%d err=%v", processed, err)
	}
	assignedB, err = service.List(ctx, ListFilter{TenantID: "evidence-work-test", PrincipalID: recipientB, ActiveOnly: true, VisibleActorWorkOnly: true, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(assignedB) != 0 {
		t.Fatalf("restricted visibility loss leaked evidence work: %#v", assignedB)
	}
	assertEvidenceProjectedIdentity(t, ctx, pool, tenantID, workflowID, taskID, "CANCELLED", "")

	restoredAt := now.Add(4 * time.Minute)
	mustExecEvidenceWork(t, ctx, pool, `UPDATE matters SET scope=$2::jsonb,updated_at=$3 WHERE id=$1::uuid`, matterID, restrictedScope(recipientB), restoredAt)
	if processed, err := projector.Maintain(ctx, restoredAt, 20); err != nil || processed != 1 {
		t.Fatalf("visibility recovery processed=%d err=%v", processed, err)
	}
	assignedB, err = service.List(ctx, ListFilter{TenantID: "evidence-work-test", PrincipalID: recipientB, ActiveOnly: true, VisibleActorWorkOnly: true, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(assignedB) != 1 || assignedB[0].WorkflowID != workflowID || assignedB[0].ID != taskID {
		t.Fatalf("visibility recovery did not restore the same projection identity: %#v", assignedB)
	}
	assertEvidenceWorkCardinality(t, ctx, pool, tenantID, 1, 1)
}

func restrictedScope(principalID string) string {
	return fmt.Sprintf("{\"access\":\"RESTRICTED\",\"allowed_principal_ids\":[\"%s\"]}", principalID)
}

func assertEvidenceProjectedIdentity(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, workflowID, taskID, status, principalID string) {
	t.Helper()
	var gotWorkflowID, gotTaskID, gotStatus, gotPrincipal string
	if err := pool.QueryRow(ctx, `
		SELECT wi.id::text,wt.id::text,wt.status,COALESCE(wt.principal_id::text,'')
		FROM workflow_instances wi
		JOIN workflow_tasks wt ON wt.workflow_id=wi.id
		WHERE wi.tenant_id=$1::uuid AND wi.kind='EVIDENCE_REQUEST'`, tenantID).Scan(&gotWorkflowID, &gotTaskID, &gotStatus, &gotPrincipal); err != nil {
		t.Fatal(err)
	}
	if gotWorkflowID != workflowID || gotTaskID != taskID || gotStatus != status || gotPrincipal != principalID {
		t.Fatalf("projected identity/state changed unexpectedly workflow=%s task=%s status=%s principal=%s", gotWorkflowID, gotTaskID, gotStatus, gotPrincipal)
	}
}

func assertEvidenceWorkCardinality(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID string, workflows, tasks int) {
	t.Helper()
	var workflowCount, taskCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM workflow_instances WHERE tenant_id=$1::uuid AND kind='EVIDENCE_REQUEST'`, tenantID).Scan(&workflowCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM workflow_tasks wt
		JOIN workflow_instances wi ON wi.id=wt.workflow_id
		WHERE wi.tenant_id=$1::uuid AND wi.kind='EVIDENCE_REQUEST'`, tenantID).Scan(&taskCount); err != nil {
		t.Fatal(err)
	}
	if workflowCount != workflows || taskCount != tasks {
		t.Fatalf("unexpected evidence work cardinality workflows=%d tasks=%d", workflowCount, taskCount)
	}
}

func mustExecEvidenceWork(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(ctx, query, args...); err != nil {
		t.Fatal(err)
	}
}
