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

func TestMatterLifecycleProjectorRoutesCompletesAndReconcilesResponseWork(t *testing.T) {
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
		tenantID    = "97777777-7777-7777-8777-777777777701"
		entityID    = "97777777-7777-7777-8777-777777777702"
		principalA  = "97777777-7777-7777-8777-777777777703"
		principalB  = "97777777-7777-7777-8777-777777777704"
		programID   = "97777777-7777-7777-8777-777777777705"
		matterID    = "97777777-7777-7777-8777-777777777706"
		responseAck = "97777777-7777-7777-8777-777777777707"
		responseFix = "97777777-7777-7777-8777-777777777708"
		responseAmb = "97777777-7777-7777-8777-777777777709"
		ackRouteA   = "97777777-7777-7777-8777-777777777710"
		fixRouteA   = "97777777-7777-7777-8777-777777777711"
		fixRouteB   = "97777777-7777-7777-8777-777777777712"
		eventAck1   = "97777777-7777-7777-8777-777777777713"
		eventAck2   = "97777777-7777-7777-8777-777777777714"
		eventFix    = "97777777-7777-7777-8777-777777777715"
		eventAmb    = "97777777-7777-7777-8777-777777777716"
	)

	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID) })
	now := time.Date(2026, 8, 7, 20, 0, 0, 0, time.UTC)
	seedMatterLifecycleWork(t, ctx, pool, tenantID, entityID, principalA, principalB, programID, matterID, responseAck, responseFix, responseAmb, ackRouteA, fixRouteA, fixRouteB, now)

	repo := NewPostgresRepository(pool)
	continuityService := continuity.NewService(continuity.NewCurrentPostgresRepository(pool))
	projector := &MatterLifecycleProjector{
		Repo: repo, Continuity: continuityService,
		Authority: authority.NewEffectivePostgresService(pool),
	}

	publishLifecycleTestEvent(t, ctx, projector, eventAck1, "lifecycle-work-test", matterID, continuity.EventResponsePackageStateChanged, now)
	service := NewService(repo)
	tasks, err := service.List(ctx, ListFilter{
		TenantID: "lifecycle-work-test", PrincipalID: principalA,
		SupportedMatterWorkOnly: true, ActiveOnly: true, VisibleMatterWorkOnly: true, Limit: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	ackTask := taskForResponse(t, tasks, responseAck)
	if ackTask.Status != StatusReady || ackTask.Responsibility != string(authority.ResponsibilityAcknowledger) || ackTask.Context["target_status"] != string(continuity.ResponseAcknowledged) || ackTask.Context["routing_state"] != "DIRECT" {
		t.Fatalf("unexpected acknowledgement task: %#v", ackTask)
	}
	if ackTask.MatterID != matterID || ackTask.MatterPriority != 5 || ackTask.WorkflowKind != MatterResponseWorkflowKind {
		t.Fatalf("acknowledgement task lost canonical Matter metadata: %#v", ackTask)
	}
	initialAckVersion := ackTask.Version

	// Duplicate delivery must be consumed idempotently and must not churn the
	// Task version.
	publishLifecycleTestEvent(t, ctx, projector, eventAck1, "lifecycle-work-test", matterID, continuity.EventResponsePackageStateChanged, now)
	ackTask = loadResponseTask(t, ctx, service, "lifecycle-work-test", principalA, responseAck)
	if ackTask.Version != initialAckVersion {
		t.Fatalf("duplicate event changed task version: got %d want %d", ackTask.Version, initialAckVersion)
	}

	// Ambiguous APPROVED response state must not invent transmitter/signatory
	// work from status alone.
	publishLifecycleTestEvent(t, ctx, projector, eventAmb, "lifecycle-work-test", matterID, continuity.EventResponsePackageStateChanged, now.Add(time.Minute))
	var ambiguousWork int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM workflow_instances WHERE tenant_id=$1::uuid AND kind='MATTER_RESPONSE' AND subject_id=$2::uuid`, tenantID, responseAmb).Scan(&ambiguousWork); err != nil {
		t.Fatal(err)
	}
	if ambiguousWork != 0 {
		t.Fatalf("ambiguous response state created %d workflow projections", ambiguousWork)
	}

	// REJECTED is unambiguous: the proposer must rework the package.
	publishLifecycleTestEvent(t, ctx, projector, eventFix, "lifecycle-work-test", matterID, continuity.EventResponsePackageStateChanged, now.Add(2*time.Minute))
	fixTask := loadResponseTask(t, ctx, service, "lifecycle-work-test", principalA, responseFix)
	if fixTask.Status != StatusReady || fixTask.Responsibility != string(authority.ResponsibilityProposer) || fixTask.Context["target_status"] != string(continuity.ResponseDraft) {
		t.Fatalf("unexpected response rework task: %#v", fixTask)
	}

	// Authority changes converge through the bounded maintainer without a new
	// Matter event. The first pass also proves the empty cursor is UUID-safe.
	if _, err := pool.Exec(ctx, `UPDATE responsibility_assignments SET valid_until=$2 WHERE id=$1::uuid`, fixRouteA, now.Add(30*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE responsibility_assignments SET valid_from=$2 WHERE id=$1::uuid`, fixRouteB, now.Add(30*time.Minute)); err != nil {
		t.Fatal(err)
	}
	processed, err := projector.Maintain(ctx, now.Add(time.Hour), 10)
	if err != nil {
		t.Fatal(err)
	}
	if processed == 0 {
		t.Fatal("expected lifecycle maintainer to process current work")
	}
	fixTaskB := loadResponseTask(t, ctx, service, "lifecycle-work-test", principalB, responseFix)
	if fixTaskB.PrincipalID != principalB || fixTaskB.Version <= fixTask.Version {
		t.Fatalf("authority change did not reassign projected work: before=%#v after=%#v", fixTask, fixTaskB)
	}
	reassignedVersion := fixTaskB.Version

	// Exhaust/reset the cursor, then reconcile the same desired state again.
	if _, err := projector.Maintain(ctx, now.Add(time.Hour+time.Minute), 10); err != nil {
		t.Fatal(err)
	}
	if _, err := projector.Maintain(ctx, now.Add(time.Hour+2*time.Minute), 10); err != nil {
		t.Fatal(err)
	}
	fixTaskB = loadResponseTask(t, ctx, service, "lifecycle-work-test", principalB, responseFix)
	if fixTaskB.Version != reassignedVersion {
		t.Fatalf("unchanged authority caused projection version churn: got %d want %d", fixTaskB.Version, reassignedVersion)
	}

	// Fulfilling the acknowledgement requirement completes its prior projected
	// Task while preserving the actual assignee for history.
	if _, err := pool.Exec(ctx, `UPDATE response_packages SET status='ACKNOWLEDGED',acknowledged_at=$2,updated_at=$2,version=version+1 WHERE id=$1::uuid`, responseAck, now.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	publishLifecycleTestEvent(t, ctx, projector, eventAck2, "lifecycle-work-test", matterID, continuity.EventResponsePackageStateChanged, now.Add(2*time.Hour))
	var terminalStatus Status
	var terminalPrincipal string
	if err := pool.QueryRow(ctx, `
		SELECT wt.status,COALESCE(wt.principal_id::text,'')
		FROM workflow_tasks wt
		JOIN workflow_instances wi ON wi.id=wt.workflow_id
		WHERE wi.tenant_id=$1::uuid AND wi.kind='MATTER_RESPONSE' AND wi.subject_id=$2::uuid`, tenantID, responseAck).Scan(&terminalStatus, &terminalPrincipal); err != nil {
		t.Fatal(err)
	}
	if terminalStatus != StatusCompleted || terminalPrincipal != principalA {
		t.Fatalf("fulfilled work lost terminal assignee/history: status=%s principal=%s", terminalStatus, terminalPrincipal)
	}
	activeAfter, err := service.List(ctx, ListFilter{
		TenantID: "lifecycle-work-test", PrincipalID: principalA,
		SupportedMatterWorkOnly: true, ActiveOnly: true, VisibleMatterWorkOnly: true, Limit: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range activeAfter {
		if task.Context["response_id"] == responseAck {
			t.Fatalf("fulfilled acknowledgement remained in active actor work: %#v", task)
		}
	}
}

func seedMatterLifecycleWork(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, entityID, principalA, principalB, programID, matterID, responseAck, responseFix, responseAmb, ackRouteA, fixRouteA, fixRouteB string, now time.Time) {
	t.Helper()
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'lifecycle-work-test','Lifecycle Work Test')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2::uuid,'BANK-NG','Bank NG','NG',$3)`, entityID, tenantID, now.Add(-24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES
		($1::uuid,$3::uuid,'PERSON','Lifecycle A','ACTIVE',$4),
		($2::uuid,$3::uuid,'PERSON','Lifecycle B','ACTIVE',$4)`, principalA, principalB, tenantID, now.Add(-24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO programs(id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,owner_principal_id,jurisdiction,scope,effective_from,created_at,updated_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,'LIFE','Lifecycle Program','REGULATORY','ACTIVE','Compliance',$4::uuid,'NG','{}'::jsonb,$5,$5,$5)`,
		programID, tenantID, entityID, principalA, now.Add(-24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	scope, _ := json.Marshal(map[string]any{"access": "RESTRICTED", "allowed_principal_ids": []string{principalA, principalB}})
	if _, err := pool.Exec(ctx, `
		INSERT INTO matters(id,tenant_id,reference,matter_type,status,priority,title,summary,scope,known_facts,missing_facts,contradictions,owner_principal_id,created_at,updated_at)
		VALUES($1::uuid,$2::uuid,'MAT-LIFE-1','AUTHORITY_REQUEST','RESPONSE_PREPARATION',5,'Authority response','Lifecycle work projection test',$3::jsonb,'{}'::jsonb,'[]'::jsonb,'[]'::jsonb,$4::uuid,$5,$5)`,
		matterID, tenantID, string(scope), principalA, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO matter_links(tenant_id,matter_id,program_id,relationship) VALUES($1::uuid,$2::uuid,$3::uuid,'AFFECTS')`, tenantID, matterID, programID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO response_packages(id,tenant_id,matter_id,purpose,audience,status,manifest,transmitted_at,created_at,updated_at) VALUES
		($1::uuid,$4::uuid,$5::uuid,'Acknowledge response','Regulator','TRANSMITTED','[]'::jsonb,$6,$6,$6),
		($2::uuid,$4::uuid,$5::uuid,'Rework response','Regulator','REJECTED','[]'::jsonb,NULL,$6,$6),
		($3::uuid,$4::uuid,$5::uuid,'Approved response','Regulator','APPROVED','[]'::jsonb,NULL,$6,$6)`,
		responseAck, responseFix, responseAmb, tenantID, matterID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO responsibility_assignments(id,tenant_id,legal_entity_id,principal_id,responsibility,object_type,object_id,scope,priority,valid_from,policy_version,decision_type) VALUES
		($1::uuid,$7::uuid,$8::uuid,$9::uuid,'ACKNOWLEDGEMENT_RECORDER','MATTER',$10::uuid,'{}'::jsonb,100,$11,'ack:v1','matter.response.transition'),
		($2::uuid,$7::uuid,$8::uuid,$9::uuid,'PROPOSER','MATTER',$10::uuid,'{}'::jsonb,100,$11,'fix:a','matter.response.transition'),
		($3::uuid,$7::uuid,$8::uuid,$12::uuid,'PROPOSER','MATTER',$10::uuid,'{}'::jsonb,100,$13,'fix:b','matter.response.transition')`,
		ackRouteA, fixRouteA, fixRouteB, responseAck, responseFix, responseAmb, tenantID, entityID, principalA, matterID, now.Add(-24*time.Hour), principalB, now.Add(24*time.Hour)); err != nil {
		t.Fatal(err)
	}
}

func publishLifecycleTestEvent(t *testing.T, ctx context.Context, projector *MatterLifecycleProjector, eventID, tenant, matterID, eventType string, at time.Time) {
	t.Helper()
	if err := projector.Publish(ctx, workflowruntime.OutboxEvent{
		ID: eventID, TenantID: tenant, AggregateType: "MATTER", AggregateID: matterID,
		EventType: eventType, Payload: json.RawMessage(`{}`), OccurredAt: at,
	}); err != nil {
		t.Fatal(err)
	}
}

func taskForResponse(t *testing.T, tasks []Task, responseID string) Task {
	t.Helper()
	for _, task := range tasks {
		if task.Context["response_id"] == responseID {
			return task
		}
	}
	t.Fatalf("response task %s not found in %#v", responseID, tasks)
	return Task{}
}

func loadResponseTask(t *testing.T, ctx context.Context, service *Service, tenant, principal, responseID string) Task {
	t.Helper()
	tasks, err := service.List(ctx, ListFilter{TenantID: tenant, PrincipalID: principal, SupportedMatterWorkOnly: true, ActiveOnly: true, VisibleMatterWorkOnly: true, Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	return taskForResponse(t, tasks, responseID)
}
