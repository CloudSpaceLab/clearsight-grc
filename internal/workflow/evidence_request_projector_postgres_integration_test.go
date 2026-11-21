//go:build postgres && postgresintegration

package workflow

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
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
	t.Cleanup(pool.Close)

	const (
		tenantID          = "9f555555-5555-7555-8555-555555555551"
		entityID          = "9f555555-5555-7555-8555-555555555558"
		creatorID         = "9f555555-5555-7555-8555-555555555552"
		recipientA        = "9f555555-5555-7555-8555-555555555553"
		recipientB        = "9f555555-5555-7555-8555-555555555554"
		matterID          = "9f555555-5555-7555-8555-555555555555"
		requestID         = "9f555555-5555-7555-8555-555555555556"
		terminalRequestID = "9f555555-5555-7555-8555-555555555557"
	)
	// Actor visibility uses database now(), so keep this integration fixture
	// relative to the executing database clock rather than a wall-clock date.
	now := time.Now().UTC().Truncate(time.Second)
	cleanupEvidenceWorkTenant(t, pool, tenantID)
	t.Cleanup(func() { cleanupEvidenceWorkTenant(t, pool, tenantID) })

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

func TestProgramFormWorkRespectsCurrentRestrictedScope(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	const tenant = "9f666666-6666-7666-8666-666666666661"
	const entity = "9f666666-6666-7666-8666-666666666662"
	const owner = "9f666666-6666-7666-8666-666666666663"
	const recipient = "9f666666-6666-7666-8666-666666666664"
	const program = "9f666666-6666-7666-8666-666666666665"
	const request = "9f666666-6666-7666-8666-666666666666"
	now := time.Now().UTC().Truncate(time.Second)
	cleanupEvidenceWorkTenant(t, pool, tenant)
	t.Cleanup(func() { cleanupEvidenceWorkTenant(t, pool, tenant) })
	mustExecEvidenceWork(t, ctx, pool, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'program-response-work','Bank')`, tenant)
	mustExecEvidenceWork(t, ctx, pool, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2::uuid,'BANK','Bank','NG',$3)`, entity, tenant, now.Add(-time.Hour))
	mustExecEvidenceWork(t, ctx, pool, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($1::uuid,$3::uuid,'PERSON','Owner','ACTIVE',$4),($2::uuid,$3::uuid,'PERSON','Respondent','ACTIVE',$4)`, owner, recipient, tenant, now.Add(-time.Hour))
	mustExecEvidenceWork(t, ctx, pool, `INSERT INTO programs(id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,scope,effective_from) VALUES($1::uuid,$2::uuid,$3::uuid,'SECURE','Restricted Program','COMPLIANCE','ACTIVE','Risk',$4::jsonb,$5)`, program, tenant, entity, restrictedScope(recipient), now.Add(-time.Hour))
	mustExecEvidenceWork(t, ctx, pool, `INSERT INTO capture_requests(id,tenant_id,legal_entity_id,subject_type,subject_id,title,purpose,why_you,sensitivity,audience_type,recipient_type,recipient_principal_id,recipient_state,recipient_revision,estimated_minutes,deadline,known_facts,fields,status,created_by,version,created_at,updated_at) VALUES($1::uuid,$2::uuid,$3::uuid,'PROGRAM',$4,'Provide a response','Review control results.','Assigned respondent.','RESTRICTED','INTERNAL','INTERNAL_PRINCIPAL',$5::uuid,'ASSIGNED',1,3,$6,'{}','[{"id":"answer","label":"Answer","type":"text"}]','READY',$7::uuid,1,$8,$8)`, request, tenant, entity, program, recipient, now.Add(time.Hour), owner, now)
	repo := NewPostgresRepository(pool)
	projector := &EvidenceRequestProjector{Repo: repo}
	service := NewService(repo)
	if count, err := projector.Maintain(ctx, now, 20); err != nil || count != 1 {
		t.Fatalf("initial projection=%d err=%v", count, err)
	}
	query := ActorWorkPageQuery{TenantID: tenant, LegalEntityID: entity, PrincipalID: recipient, Limit: 10}
	page, err := service.ListActorWorkPage(ctx, query)
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("allowed work=%+v err=%v", page, err)
	}
	mustExecEvidenceWork(t, ctx, pool, `UPDATE programs SET scope=$2::jsonb WHERE id=$1::uuid`, program, restrictedScope(owner))
	page, err = service.ListActorWorkPage(ctx, query)
	if err != nil || len(page.Items) != 0 {
		t.Fatalf("restricted work leaked before projection=%+v err=%v", page, err)
	}
	counts, err := service.ReadActorWorkDueCounts(ctx, tenant, entity, recipient)
	if err != nil || counts.Total != 0 {
		t.Fatalf("restricted count=%+v err=%v", counts, err)
	}
	if count, err := projector.Maintain(ctx, now, 20); err != nil || count != 1 {
		t.Fatalf("scope loss projection=%d err=%v", count, err)
	}
	mustExecEvidenceWork(t, ctx, pool, `UPDATE programs SET scope=$2::jsonb WHERE id=$1::uuid`, program, restrictedScope(recipient))
	if count, err := projector.Maintain(ctx, now, 20); err != nil || count != 1 {
		t.Fatalf("scope restored projection=%d err=%v", count, err)
	}
	page, err = service.ListActorWorkPage(ctx, query)
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("restored work=%+v err=%v", page, err)
	}
	mustExecEvidenceWork(t, ctx, pool, `UPDATE programs SET scope=$2::jsonb WHERE id=$1::uuid`, program, fmt.Sprintf(`{"access":"RESTRICTED","allowed_principal_ids":["%s",7]}`, recipient))
	page, err = service.ListActorWorkPage(ctx, query)
	if err != nil || len(page.Items) != 0 {
		t.Fatalf("malformed restricted scope=%+v err=%v", page, err)
	}
	if count, err := projector.Maintain(ctx, now, 20); err != nil || count != 1 {
		t.Fatalf("malformed scope projection=%d err=%v", count, err)
	}
	mustExecEvidenceWork(t, ctx, pool, `DELETE FROM programs WHERE id=$1::uuid`, program)
	page, err = service.ListActorWorkPage(ctx, query)
	if err != nil || len(page.Items) != 0 {
		t.Fatalf("missing program work=%+v err=%v", page, err)
	}
	mustExecEvidenceWork(t, ctx, pool, `UPDATE capture_requests SET subject_id='invalid-program-id' WHERE id=$1::uuid`, request)
	if _, err := projector.Maintain(ctx, now, 20); err != nil {
		t.Fatalf("malformed program identifier: %v", err)
	}
	page, err = service.ListActorWorkPage(ctx, query)
	if err != nil || len(page.Items) != 0 {
		t.Fatalf("malformed program work=%+v err=%v", page, err)
	}
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

func cleanupEvidenceWorkTenant(t *testing.T, pool *pgxpool.Pool, tenant string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `BEGIN; SET LOCAL session_replication_role='replica'; SELECT set_config('clearsight.test_cleanup_tenant',$1,true);
DO $$ DECLARE tab record; BEGIN
 FOR tab IN SELECT c.table_schema,c.table_name FROM information_schema.columns c JOIN information_schema.tables t ON t.table_schema=c.table_schema AND t.table_name=c.table_name WHERE c.table_schema='public' AND c.column_name='tenant_id' AND t.table_type='BASE TABLE' LOOP
 EXECUTE format('DELETE FROM %I.%I WHERE tenant_id=current_setting(''clearsight.test_cleanup_tenant'')::uuid',tab.table_schema,tab.table_name);
 END LOOP;
END $$;
DELETE FROM tenants WHERE id=$1::uuid; COMMIT`, pgx.QueryExecModeSimpleProtocol, tenant)
	if err != nil {
		t.Fatalf("cleanup test tenant: %v", err)
	}
}
