//go:build postgres && postgresintegration

package evidence

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"strings"
	"testing"
	"time"
)

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
