//go:build postgres && postgresintegration

package evidence

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5"
	"strings"
	"testing"
	"time"
)

func TestPostgresReviewerDiscoveryDoesNotBypassRestrictedProgram(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	tenant, entity, owner, reviewer, form, program := mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t)
	now := time.Now().UTC()
	prefix := "program-review-scope-" + tenant
	setupResponseWorkspaceFixture(t, ctx, pool, tenant, prefix, entity, owner, form, now)
	seedCompletedResponseRows(t, ctx, pool, tenant, entity, owner, form, prefix, 1, now)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM responsibility_assignments WHERE tenant_id=$1::uuid; DELETE FROM programs WHERE tenant_id=$1::uuid`, pgx.QueryExecModeSimpleProtocol, tenant)
		cleanupResponseWorkspaceTenant(context.Background(), pool, tenant)
	})
	_, err := pool.Exec(ctx, `
 INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($4::uuid,$1::uuid,'PERSON','Risk reviewer','ACTIVE',$7);
 INSERT INTO programs(id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,jurisdiction,scope,effective_from)
 VALUES($6::uuid,$1::uuid,$2::uuid,'PRIVATE','Restricted program','COMPLIANCE','ACTIVE','Risk','NG',jsonb_build_object('access','RESTRICTED','allowed_principal_ids',jsonb_build_array($3::text)),$7);
 UPDATE capture_form_distributions SET subject_type='PROGRAM',subject_id=$6::uuid WHERE tenant_id=$1::uuid;
 UPDATE capture_requests SET subject_type='PROGRAM',subject_id=$6 WHERE tenant_id=$1::uuid;
 INSERT INTO responsibility_assignments(tenant_id,legal_entity_id,principal_id,responsibility,object_type,object_id,decision_type,scope,priority,valid_from,policy_version)
 VALUES($1::uuid,$2::uuid,$4::uuid,'REVIEWER','FORM_RESPONSE',md5($5||':response:1')::uuid,'forms.response.review','{}',10,$7,'route-v1')`, pgx.QueryExecModeSimpleProtocol, tenant, entity, owner, reviewer, prefix, program, now.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	store := NewPostgresDistributionStore(NewPostgresRepository(pool), nil)
	query := CompletedResponseQuery{TenantID: tenant, LegalEntityID: entity, PrincipalID: reviewer, Sort: ResponseSortNewest, CurrentOnly: true, Limit: 1}
	page, err := store.ListCompletedResponses(ctx, query)
	if err != nil || len(page.Items) != 0 || page.NextCursor != "" {
		t.Fatalf("restricted reviewer page=%+v err=%v", page, err)
	}
	var response string
	if err = pool.QueryRow(ctx, `SELECT md5($1||':response:1')::uuid::text`, prefix).Scan(&response); err != nil {
		t.Fatal(err)
	}
	if _, _, err = store.GetCompletedResponse(ctx, tenant, entity, reviewer, response); !errors.Is(err, ErrNotFound) {
		t.Fatalf("restricted detail error=%v", err)
	}
	query.PrincipalID = owner
	page, err = store.ListCompletedResponses(ctx, query)
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("program owner page=%+v err=%v", page, err)
	}
}

func TestPostgresReviewerDiscoveryFiltersBeforePagesAndCounts(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	tenant := mustResponseWorkspaceID(t)
	entity := mustResponseWorkspaceID(t)
	owner := mustResponseWorkspaceID(t)
	reviewer := mustResponseWorkspaceID(t)
	form := mustResponseWorkspaceID(t)
	vendor := mustResponseWorkspaceID(t)
	rel := mustResponseWorkspaceID(t)
	now := time.Now().UTC()
	prefix := "reviewer-discovery-" + tenant
	setupResponseWorkspaceFixture(t, ctx, pool, tenant, prefix, entity, owner, form, now)
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM responsibility_assignments WHERE tenant_id=$1::uuid`, tenant)
		cleanupVendorFormsFixture(context.Background(), pool, tenant)
	}()
	seedCompletedResponseRows(t, ctx, pool, tenant, entity, owner, form, prefix, 4, now)
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, append([]any{pgx.QueryExecModeSimpleProtocol}, args...)...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($1::uuid,$2::uuid,'PERSON','Bank reviewer','ACTIVE',$3)`, reviewer, tenant, now.Add(-time.Hour))
	exec(`INSERT INTO third_parties(id,tenant_id,legal_name,status,created_at,updated_at,version) VALUES($1::uuid,$2::uuid,'Sample vendor','ACTIVE',$6,$6,1);INSERT INTO third_party_relationships(id,tenant_id,legal_entity_id,vendor_id,service_name,business_owner_principal_id,criticality,privacy_role,status,created_at,updated_at,version) VALUES($3::uuid,$2::uuid,$4::uuid,$1::uuid,'Processing',$5::uuid,'IMPORTANT','PROCESSOR','PROPOSED',$6,$6,1)`, vendor, tenant, rel, entity, owner, now)
	exec(`UPDATE capture_form_distributions SET subject_type='VENDOR_RELATIONSHIP',subject_id=$2::uuid WHERE tenant_id=$1::uuid;
 UPDATE capture_requests SET subject_type='VENDOR_RELATIONSHIP',subject_id=$2,fields='[{"id":"report","label":"Report","type":"short_text","assessment":{"mode":"MANUAL","reviewer_role":"Risk","weight":100,"rubric":[{"id":"poor","label":"Incomplete","points":80}]}}]' WHERE tenant_id=$1::uuid;
 UPDATE capture_submissions SET answers='{"report":{"text":"private supplied answer"}}' WHERE tenant_id=$1::uuid;
 UPDATE capture_requests SET origin_type='THIRD_PARTY_WORK',origin_id=md5($3||':restricted')::uuid::text,origin_version=1 WHERE id=md5($3||':request:4')::uuid;
 INSERT INTO capture_requests SELECT (jsonb_populate_record(NULL::capture_requests,to_jsonb(req)||jsonb_build_object('id',md5($3||':pending')::uuid,'status','IN_PROGRESS','updated_at',$4::timestamptz+interval '1 hour'))).* FROM capture_requests req WHERE id=md5($3||':request:1')::uuid`, tenant, rel, prefix, now)
	exec(`INSERT INTO responsibility_assignments(tenant_id,legal_entity_id,principal_id,responsibility,object_type,object_id,decision_type,scope,priority,valid_from,policy_version) SELECT $1::uuid,$2::uuid,$3::uuid,'REVIEWER','FORM_RESPONSE',md5($4||':response:'||g)::uuid,'forms.response.assess','{}',10,$5,'route-v1' FROM unnest(ARRAY[1,2,4]) g`, tenant, entity, reviewer, prefix, now.Add(-time.Hour))
	store := NewPostgresDistributionStore(NewPostgresRepository(pool), nil)
	q := CompletedResponseQuery{TenantID: tenant, LegalEntityID: entity, PrincipalID: reviewer, Sort: ResponseSortNewest, CurrentOnly: true, Limit: 1}
	first, err := store.ListCompletedResponses(ctx, q)
	if err != nil || len(first.Items) != 1 || first.NextCursor == "" {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	q.Cursor = first.NextCursor
	second, err := store.ListCompletedResponses(ctx, q)
	if err != nil || len(second.Items) != 1 || second.NextCursor != "" || second.Items[0].ID == first.Items[0].ID {
		t.Fatalf("second=%+v err=%v", second, err)
	}
	if _, _, err = store.GetCompletedResponse(ctx, tenant, entity, reviewer, first.Items[0].ID); err != nil {
		t.Fatalf("discovered row unavailable: %v", err)
	}
	vq := VendorFormsQuery{TenantID: tenant, LegalEntityID: entity, PrincipalID: reviewer, RelationshipIDs: []string{rel}, Limit: 1}
	page, err := store.ListVendorForms(ctx, vq)
	if err != nil || len(page.Items) != 1 || page.NextCursor == "" || page.Items[0].ResponseID == "" {
		t.Fatalf("vendor=%+v err=%v", page, err)
	}
	raw, _ := json.Marshal(page)
	if strings.Contains(string(raw), "private supplied answer") {
		t.Fatal("list leaked answers")
	}
	sums, err := store.VendorFormSummaries(ctx, vq)
	if err != nil || sums[0].SubmittedForms != 2 || sums[0].OutstandingForms != 0 {
		t.Fatalf("reviewer counts=%+v err=%v", sums, err)
	}
	vq.PrincipalID = owner
	sums, err = store.VendorFormSummaries(ctx, vq)
	if err != nil || sums[0].SubmittedForms != 3 || sums[0].OutstandingForms != 1 {
		t.Fatalf("owner counts=%+v err=%v", sums, err)
	}
	// A whole-response reviewer must discover a response even when no question has a rubric.
	exec(`UPDATE capture_requests SET fields='[{"id":"report","label":"Report","type":"short_text"}]' WHERE id=md5($1||':request:3')::uuid;
INSERT INTO responsibility_assignments(tenant_id,legal_entity_id,principal_id,responsibility,object_type,object_id,decision_type,scope,priority,valid_from,policy_version) VALUES($2::uuid,$3::uuid,$4::uuid,'REVIEWER','FORM_RESPONSE',md5($1||':response:3')::uuid,'forms.response.review','{}',10,$5,'route-v1')`, prefix, tenant, entity, reviewer, now.Add(-time.Hour))
	q.Cursor = ""
	q.Limit = 10
	reviewPage, err := store.ListCompletedResponses(ctx, q)
	if err != nil || len(reviewPage.Items) != 3 {
		t.Fatalf("whole-response review discovery=%+v err=%v", reviewPage, err)
	}
	var reviewResponse string
	if err := pool.QueryRow(ctx, `SELECT md5($1||':response:3')::uuid::text`, prefix).Scan(&reviewResponse); err != nil {
		t.Fatal(err)
	}
	permitted := true
	service := NewDistributionService(store).WithResponseReviewAuthorizer(func(_ context.Context, actor identity.Actor, summary CompletedResponseSummary) (string, error) {
		if permitted && actor.PrincipalID == reviewer && summary.ID == reviewResponse {
			return "policy:rule", nil
		}
		return "", ErrAssessmentForbidden
	})
	service.demoResponseReviewDecisions = true // Exercise the unexposed transaction in isolation.
	reviewCtx := identity.WithActor(ctx, identity.Actor{TenantID: tenant, LegalEntityID: entity, PrincipalID: reviewer, ExpiresAt: now.Add(time.Hour)})
	history, historyErr := service.GetResponseReviews(reviewCtx, tenant, entity, reviewer, reviewResponse, 0, 10)
	if historyErr != nil || !history.MayReview || len(history.Items) != 0 {
		t.Fatalf("review-only history=%+v err=%v", history, historyErr)
	}
	permitted = false
	if _, historyErr = service.GetResponseReviews(reviewCtx, tenant, entity, reviewer, reviewResponse, 0, 10); historyErr == nil {
		t.Fatal("revoked reviewer read response history")
	}
	exec(`UPDATE capture_submissions SET submitted_by=$2::uuid WHERE id=md5($1||':submission:3')::uuid`, prefix, reviewer)
	reviewPage, err = store.ListCompletedResponses(ctx, q)
	if err != nil || len(reviewPage.Items) != 2 {
		t.Fatalf("whole-response self-review=%+v err=%v", reviewPage, err)
	}
	exec(`UPDATE capture_submissions SET submitted_by=$2::uuid WHERE id=md5($1||':submission:3')::uuid;
UPDATE responsibility_assignments SET valid_until=$3 WHERE tenant_id=$4::uuid AND decision_type='forms.response.review'`, prefix, owner, now.Add(-time.Minute), tenant)
	reviewPage, err = store.ListCompletedResponses(ctx, q)
	if err != nil || len(reviewPage.Items) != 2 {
		t.Fatalf("whole-response revoked=%+v err=%v", reviewPage, err)
	}
	q.Limit = 1
	exec(`UPDATE capture_requests SET recipient_type='INTERNAL_PRINCIPAL',recipient_principal_id=$2::uuid,recipient_state='ASSIGNED',recipient_revision=1 WHERE id=md5($1||':request:2')::uuid;
UPDATE capture_submissions SET submitted_by=NULL,channel='MAGIC_LINK' WHERE id=md5($1||':submission:2')::uuid`, prefix, reviewer)
	q.Cursor = ""
	assignedPage, err := store.ListCompletedResponses(ctx, q)
	if err != nil || len(assignedPage.Items) != 1 || assignedPage.NextCursor != "" {
		t.Fatalf("assigned respondent discovery=%+v err=%v", assignedPage, err)
	}
	exec(`UPDATE capture_requests SET recipient_type=NULL,recipient_principal_id=NULL,recipient_state='LEGACY_UNASSIGNED',recipient_revision=0 WHERE id=md5($1||':request:2')::uuid`, prefix)
	exec(`UPDATE capture_submissions SET submitted_by=$2::uuid WHERE id=md5($1||':submission:2')::uuid`, prefix, reviewer)
	vq.PrincipalID = reviewer
	sums, err = store.VendorFormSummaries(ctx, vq)
	if err != nil || sums[0].SubmittedForms != 1 {
		t.Fatalf("self-review counts=%+v err=%v", sums, err)
	}
	exec(`UPDATE responsibility_assignments SET valid_until=$2 WHERE tenant_id=$1::uuid`, tenant, now.Add(-time.Minute))
	vq.PrincipalID = reviewer
	sums, err = store.VendorFormSummaries(ctx, vq)
	if err != nil || sums[0].SubmittedForms != 0 || sums[0].OutstandingForms != 0 {
		t.Fatalf("revoked counts=%+v err=%v", sums, err)
	}
	q.Cursor = ""
	first, err = store.ListCompletedResponses(ctx, q)
	if err != nil || len(first.Items) != 0 {
		t.Fatalf("revoked=%+v err=%v", first, err)
	}
}
