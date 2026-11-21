//go:build postgres && postgresintegration

package evidence

import (
	"context"
	"fmt"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5"
	"testing"
	"time"
)

func TestPostgresProgramActivityScopePagingAndDocumentHistory(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	id := func() string { return mustResponseWorkspaceID(t) }
	tenant, entity, owner, reviewer, program, issue, request, submission, artifact, schedule := id(), id(), id(), id(), id(), id(), id(), id(), id(), id()
	now := time.Now().UTC().Truncate(time.Second)
	t.Cleanup(func() {
		tx, err := pool.Begin(context.Background())
		if err != nil {
			return
		}
		defer tx.Rollback(context.Background())
		if _, err = tx.Exec(context.Background(), `SET LOCAL session_replication_role='replica'`); err != nil {
			t.Error(err)
			return
		}
		for _, table := range []string{"capture_submission_reviews", "capture_submissions", "capture_artifacts", "capture_requests", "continuity_events", "outbox_events", "program_assessment_schedules", "matter_links", "matters", "programs", "principals", "legal_entities", "tenants"} {
			column := "tenant_id"
			if table == "tenants" {
				column = "id"
			}
			if _, err = tx.Exec(context.Background(), fmt.Sprintf("DELETE FROM %s WHERE %s=$1::uuid", table, column), tenant); err != nil {
				t.Error(err)
				return
			}
		}
		if err = tx.Commit(context.Background()); err != nil {
			t.Error(err)
		}
	})
	_, err := pool.Exec(ctx, `
 INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'activity-'||$1::text,'Activity test');
 INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($2::uuid,$1::uuid,'TEST','Activity bank','NG',$11);
 INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($3::uuid,$1::uuid,'PERSON','Activity owner','ACTIVE',$11),($4::uuid,$1::uuid,'PERSON','Activity reviewer','ACTIVE',$11);
 INSERT INTO programs(id,tenant_id,legal_entity_id,code,name,program_type,owning_function,owner_principal_id,authority_principal_id,effective_from) VALUES($5::uuid,$1::uuid,$2::uuid,'ACTIVITY','Activity Program','COMPLIANCE','Risk',$3::uuid,$4::uuid,$11);
 INSERT INTO matters(id,tenant_id,legal_entity_id,reference,matter_type,priority,title,summary,scope) VALUES($6::uuid,$1::uuid,$2::uuid,'ACT-ISSUE','CONTROL_GAP',3,'Restricted issue','Test issue',jsonb_build_object('access','RESTRICTED','allowed_principal_ids',jsonb_build_array($3::text)));
 INSERT INTO matter_links(tenant_id,matter_id,program_id) VALUES($1::uuid,$6::uuid,$5::uuid);
 INSERT INTO continuity_events(tenant_id,aggregate_type,aggregate_id,aggregate_version,event_type,payload,actor_type,actor_id,occurred_at) VALUES($1::uuid,'MATTER',$6::uuid,1,'MATTER_VERIFICATION_RECORDED','{"outcome":"PASSED","private":"must not escape"}'::jsonb,'PERSON',$3::uuid,$11);
 INSERT INTO capture_requests(id,tenant_id,legal_entity_id,subject_type,subject_id,title,purpose,why_you,sensitivity,audience_type,estimated_minutes,deadline,fields,created_by,created_at) VALUES($7::uuid,$1::uuid,$2::uuid,'PROGRAM',$5::text,'Test response','Test','Test','INTERNAL','INTERNAL',1,$11::timestamptz+interval '1 day','[{"id":"file","label":"Record","type":"file"}]',$3::uuid,$11);
 INSERT INTO capture_artifacts(id,tenant_id,request_id,file_name,media_type,size_bytes,sha256,storage_key,created_by,created_at) VALUES($9::uuid,$1::uuid,$7::uuid,'test-record.pdf','application/pdf',1,repeat('a',64),'activity/'||$9::text,$3::uuid,$11);
 INSERT INTO capture_submissions(id,tenant_id,request_id,submitted_by,channel,answers,submitted_at) VALUES($8::uuid,$1::uuid,$7::uuid,$3::uuid,'INTERNAL',jsonb_build_object('file',jsonb_build_object('artifact_ids',jsonb_build_array($9::text))),$11);
 INSERT INTO capture_submission_reviews(tenant_id,legal_entity_id,request_id,submission_id,reviewed_by,status,rationale,groups,version,created_at) VALUES($1::uuid,$2::uuid,$7::uuid,$8::uuid,$4::uuid,'REVIEWED','Checked',jsonb_build_array(jsonb_build_object('group_id','g','outcome','REVIEWED','artifact_ids',jsonb_build_array($9::text))),1,$11);
 INSERT INTO program_assessment_schedules(id,tenant_id,legal_entity_id,program_id,owner_id,form_selection,first_period_start,time_zone,interval_months,response_days,cycle_limit,next_attempt_at,rationale) VALUES($10::uuid,$1::uuid,$2::uuid,$5::uuid,$3::uuid,'{}'::jsonb,'2026-11-01','UTC',1,10,2,$11,'Test schedule');
 INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at) VALUES($1::uuid,'PROGRAM_ASSESSMENT_SCHEDULE',$10::uuid,'ProgramAssessmentScheduleApproved',jsonb_build_object('actor_id',$4::text,'version',2,'state','ACTIVE'),$11);
 INSERT INTO continuity_events(tenant_id,aggregate_type,aggregate_id,aggregate_version,event_type,payload,actor_type,actor_id,occurred_at) SELECT $1::uuid,'PROGRAM',$5::uuid,g,'PROGRAM_EDITED','{}'::jsonb,'PERSON',$3::uuid,$11 FROM generate_series(1,401) g;
 `, pgx.QueryExecModeSimpleProtocol, tenant, entity, owner, reviewer, program, issue, request, submission, artifact, schedule, now)
	if err != nil {
		t.Fatal(err)
	}
	repo := NewPostgresRepository(pool)
	actor := identity.Actor{TenantID: tenant, LegalEntityID: entity, PrincipalID: owner}
	query := ProgramActivityQuery{Limit: 37}
	seen := map[string]bool{}
	categories := map[string]bool{}
	count := 0
	for {
		page, e := repo.ProgramActivity(ctx, actor, program, query, now.Add(time.Second))
		if e != nil {
			t.Fatal(e)
		}
		for _, event := range page.Items {
			if seen[event.ID] {
				t.Fatal("duplicate across pages")
			}
			seen[event.ID] = true
			categories[event.Category] = true
			count++
			if _, ok := event.Details["private"]; ok {
				t.Fatal("raw payload leaked")
			}
			if event.Category == "DOCUMENT" && event.Event == "DOCUMENT_SUBMITTED" && (event.SubmissionID != submission || event.ArtifactID != artifact || event.FieldID != "file") {
				t.Fatalf("wrong document link: %#v", event)
			}
		}
		if page.NextCursor == "" {
			break
		}
		query.Cursor = page.NextCursor
	}
	if count != 408 || len(categories) != 5 {
		t.Fatalf("events=%d categories=%v", count, categories)
	}
	q := ProgramActivityQuery{Search: "Activity reviewer", Category: "DOCUMENT", EventType: "DOCUMENT_REVIEWED"}
	page, err := repo.ProgramActivity(ctx, actor, program, q, now.Add(time.Second))
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("actor/category filter: %#v %v", page, err)
	}
	actor.PrincipalID = reviewer
	page, err = repo.ProgramActivity(ctx, actor, program, ProgramActivityQuery{Category: "ISSUE"}, now.Add(time.Second))
	if err != nil || len(page.Items) != 0 {
		t.Fatalf("restricted issue exposed: %#v %v", page, err)
	}
	actor.LegalEntityID = id()
	page, err = repo.ProgramActivity(ctx, actor, program, ProgramActivityQuery{}, now.Add(time.Second))
	if err != nil || len(page.Items) != 0 {
		t.Fatalf("cross-entity exposed: %#v %v", page, err)
	}
	actor.LegalEntityID = entity
	actor.TenantID = id()
	page, err = repo.ProgramActivity(ctx, actor, program, ProgramActivityQuery{}, now.Add(time.Second))
	if err != nil || len(page.Items) != 0 {
		t.Fatalf("cross-tenant exposed: %#v %v", page, err)
	}
}
