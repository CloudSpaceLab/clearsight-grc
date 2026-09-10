//go:build postgres && postgresintegration

package thirdparty

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresDemoMatterArchiveFiltersWorkBeforePagination(t *testing.T) {
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
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	// The integration database is disposable; retained curation history is
	// deliberately not deleted through the application schema.
	exec(`TRUNCATE tenants CASCADE`)
	const tenant = "99999999-9999-7999-8999-999999999991"
	const entity = "99999999-9999-7999-8999-999999999992"
	const owner = "99999999-9999-7999-8999-999999999993"
	exec(`INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'archive-matter-test','Archive matter test')`, tenant)
	exec(`INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2::uuid,'BANK','Bank','NG',now()-interval '1 day')`, entity, tenant)
	exec(`INSERT INTO principals(id,tenant_id,kind,display_name) VALUES($1::uuid,$2::uuid,'PERSON','Sample owner')`, owner, tenant)
	ids := []string{"99999999-9999-7999-8999-999999999994", "99999999-9999-7999-8999-999999999995"}
	retainedTaskIDs := map[string]bool{}
	for i, id := range ids {
		exec(`INSERT INTO matters(id,tenant_id,legal_entity_id,reference,matter_type,status,priority,title,summary,scope,known_facts,missing_facts,contradictions,owner_principal_id,created_at,updated_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4,'CONTROL_GAP','TRIAGE',$5,'Sample finding','Sample history','{"access":"INTERNAL"}','{}','[]','[]',$6::uuid,now(),now())`, id, tenant, entity, fmt.Sprintf("ARCHIVE-%d", i), i+1, owner)
		var action, request string
		if err := pool.QueryRow(ctx, `INSERT INTO matter_actions(tenant_id,matter_id,title,description,owner_principal_id,status,created_at,updated_at) VALUES($1::uuid,$2::uuid,'Sample action','Sample action',$3::uuid,'PLANNED',now(),now()) RETURNING id::text`, tenant, id, owner).Scan(&action); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `INSERT INTO capture_requests(tenant_id,legal_entity_id,subject_type,subject_id,title,purpose,why_you,sensitivity,audience_type,recipient_type,recipient_principal_id,recipient_state,recipient_revision,estimated_minutes,deadline,known_facts,fields,status,created_by,version,created_at,updated_at)
		VALUES($1::uuid,$2::uuid,'MATTER',$3,'Sample evidence','Review evidence','Sample owner','INTERNAL','INTERNAL','INTERNAL_PRINCIPAL',$4::uuid,'ASSIGNED',1,3,now()+interval '1 day','{}','[{"id":"confirm","label":"Confirm","type":"text","required":true}]','READY',$4::uuid,1,now(),now()) RETURNING id::text`, tenant, entity, id, owner).Scan(&request); err != nil {
			t.Fatal(err)
		}
		for _, subject := range []struct{ kind, typ, id string }{{"MATTER_LIFECYCLE", "MATTER", id}, {"MATTER_ACTION", "MATTER_ACTION", action}, {"EVIDENCE_REQUEST", "EVIDENCE_REQUEST", request}} {
			var instance string
			if err := pool.QueryRow(ctx, `INSERT INTO workflow_instances(tenant_id,kind,subject_type,subject_id,state,policy_version) VALUES($1::uuid,$2,$3,$4::uuid,'ACTIVE','test') RETURNING id::text`, tenant, subject.kind, subject.typ, subject.id).Scan(&instance); err != nil {
				t.Fatal(err)
			}
			var taskID string
			if err := pool.QueryRow(ctx, `INSERT INTO workflow_tasks(tenant_id,workflow_id,step_key,responsibility,principal_id,title,status,updated_at) VALUES($1::uuid,$2::uuid,'review','ACCOUNTABLE_OWNER',$3::uuid,'Review sample','READY',now()+$4*interval '1 hour') RETURNING id::text`, tenant, instance, owner, i).Scan(&taskID); err != nil {
				t.Fatal(err)
			}
			if i == 0 {
				retainedTaskIDs[taskID] = true
			}
		}
	}
	actor := identity.WithActor(ctx, identity.Actor{TenantID: tenant, LegalEntityID: entity, PrincipalID: owner})
	current := continuity.NewCurrentPostgresRepository(pool)
	summaries := continuity.NewPostgresRepository(pool)
	work := workflow.NewPostgresRepository(pool)
	filter := workflow.ListFilter{TenantID: tenant, LegalEntityID: entity, PrincipalID: owner, ActiveOnly: true, Limit: 10}
	before, err := work.List(ctx, filter)
	if err != nil || len(before) != 6 {
		t.Fatalf("fixture work=%d err=%v", len(before), err)
	}
	exec(`INSERT INTO demo_record_archives(tenant_id,legal_entity_id,record_type,record_id,reason,source_manifest,archived_by,archived_at) VALUES($1::uuid,$2::uuid,'MATTER',$3::uuid,'Excluded sample','test-manifest',$4::uuid,now())`, tenant, entity, ids[1], owner)
	page, err := summaries.ListMatterSummaries(actor, tenant, continuity.SummaryQuery{Status: "OPEN", Limit: 1})
	if err != nil || len(page.Items) != 1 || page.Items[0].Matter.ID != ids[0] || page.NextCursor != "" {
		t.Fatalf("curated summaries=%#v err=%v", page, err)
	}
	rows, err := current.ListMatters(actor, tenant, "OPEN", 1)
	if err != nil || len(rows) != 1 || rows[0].Matter.ID != ids[0] {
		t.Fatalf("curated current list=%#v err=%v", rows, err)
	}
	exact, err := current.GetMatter(actor, tenant, ids[1])
	if err != nil || string(exact.Matter.Status) != "TRIAGE" || exact.Matter.Version != 1 {
		t.Fatalf("exact archive history changed=%#v err=%v", exact.Matter, err)
	}
	filter.Limit = 3
	remaining, err := work.List(ctx, filter)
	if err != nil || len(remaining) != 3 {
		t.Fatalf("curated work=%#v err=%v", remaining, err)
	}
	for _, task := range remaining {
		if !retainedTaskIDs[task.ID] {
			t.Fatalf("archived task retained=%#v", task)
		}
	}
	exec(`UPDATE demo_record_archives SET restored_at=now(),restored_by=$4::uuid,restoration_reason='Restore sample' WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND record_id=$3::uuid`, tenant, entity, ids[1], owner)
	filter.Limit = 10
	restored, err := work.List(ctx, filter)
	if err != nil || len(restored) != 6 {
		t.Fatalf("restored work=%d err=%v", len(restored), err)
	}
}
