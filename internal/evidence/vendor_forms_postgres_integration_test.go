//go:build postgres && postgresintegration

package evidence

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
	"testing"
	"time"
)

func cleanupVendorFormsFixture(ctx context.Context, pool *pgxpool.Pool, tenant string) {
	for _, table := range []string{"capture_distribution_creation_receipts", "third_party_relationships", "third_parties"} {
		_, _ = pool.Exec(ctx, "DELETE FROM "+table+" WHERE tenant_id=$1::uuid", tenant)
	}
	cleanupResponseWorkspaceTenant(ctx, pool, tenant)
}

func TestPostgresVendorFormsSubmittedRevisionOverridesOpenRequest(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	tenant, entity, owner := mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t)
	form, vendor, relationship := mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t)
	now := time.Now().UTC()
	prefix := "vendor-submitted-" + tenant
	setupResponseWorkspaceFixture(t, ctx, pool, tenant, prefix, entity, owner, form, now)
	t.Cleanup(func() { cleanupVendorFormsFixture(context.Background(), pool, tenant) })
	seedCompletedResponseRows(t, ctx, pool, tenant, entity, owner, form, prefix, 1, now)
	if _, err := pool.Exec(ctx, `
 INSERT INTO third_parties(id,tenant_id,legal_name,status,created_at,updated_at,version)
 VALUES($1::uuid,$2::uuid,'Sample vendor','ACTIVE',$6,$6,1);
 INSERT INTO third_party_relationships(id,tenant_id,legal_entity_id,vendor_id,service_name,business_owner_principal_id,criticality,privacy_role,status,created_at,updated_at,version)
 VALUES($3::uuid,$2::uuid,$4::uuid,$1::uuid,'Processing',$5::uuid,'IMPORTANT','PROCESSOR','PROPOSED',$6,$6,1);
 UPDATE capture_form_distributions SET subject_type='VENDOR_RELATIONSHIP',subject_id=$3::uuid,status='OPEN' WHERE tenant_id=$2::uuid;
 UPDATE capture_requests SET subject_type='VENDOR_RELATIONSHIP',subject_id=$3,status='IN_PROGRESS',deadline=$6::timestamptz-interval '1 day' WHERE tenant_id=$2::uuid;
 UPDATE capture_submissions SET answers='{"score":{"text":"1"}}' WHERE tenant_id=$2::uuid;
 UPDATE capture_response_revisions SET score_result='{"state":"FINAL","final":true,"band":"HIGH","coverage":1}' WHERE tenant_id=$2::uuid`,
		pgx.QueryExecModeSimpleProtocol, vendor, tenant, relationship, entity, owner, now); err != nil {
		t.Fatal(err)
	}
	var requestID, responseID, distributionID string
	if err := pool.QueryRow(ctx, `SELECT req.id::text,r.id::text,req.distribution_id::text
 FROM capture_requests req JOIN capture_submissions sub ON sub.request_id=req.id
 JOIN capture_response_revisions r ON r.submission_id=sub.id WHERE req.tenant_id=$1::uuid`, tenant).Scan(&requestID, &responseID, &distributionID); err != nil {
		t.Fatal(err)
	}
	store := NewPostgresDistributionStore(NewPostgresRepository(pool), postgresTestRecipientProtector{})
	pending, err := store.CreateDistribution(ctx, CreateDistributionInput{
		TenantID: tenant, LegalEntityID: entity, FormTemplateID: form, FormTemplateVersion: 1,
		SubjectType: "VENDOR_RELATIONSHIP", SubjectID: relationship, Title: "Security follow-up", Purpose: "Confirm supplier security controls",
		AccessPolicy: AccessDirectEmailOTP, EstimatedMinutes: 5, Deadline: now.Add(time.Hour), RouteExpiresAt: now.Add(time.Hour), CreatedBy: owner,
		Recipients: []DistributionRecipientInput{{Role: RecipientTo, Type: RecipientExternalAudience, Address: "vendor@example.test", AudienceHint: "Vendor contact"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE capture_requests SET deadline=$2::timestamptz-interval '1 day' WHERE distribution_id=$1::uuid`, pending.Distribution.ID, now); err != nil {
		t.Fatal(err)
	}
	q := VendorFormsQuery{TenantID: tenant, LegalEntityID: entity, PrincipalID: owner, RelationshipIDs: []string{relationship}, Limit: 25}
	page, err := store.ListVendorForms(ctx, q)
	if err != nil || len(page.Items) != 2 {
		t.Fatalf("page=%+v err=%v", page, err)
	}
	foundSubmitted := false
	for _, row := range page.Items {
		if row.RequestID == requestID {
			foundSubmitted = true
			if row.ResponseState != "SUBMITTED" || row.ResponseID != responseID || !row.Current || row.ResponseCurrency != "CURRENT" {
				t.Fatalf("submitted response must override open reusable request: %+v", row)
			}
		}
	}
	if !foundSubmitted {
		t.Fatal("submitted request missing from vendor forms")
	}
	summary, err := store.VendorFormSummaries(ctx, q)
	if err != nil || len(summary) != 1 || summary[0].SubmittedForms != 1 || summary[0].OutstandingForms != 1 || summary[0].OverdueForms != 1 || summary[0].AssessedForms != 1 || summary[0].HighestConcern != "HIGH" {
		t.Fatalf("summary=%+v err=%v", summary, err)
	}
	for _, filter := range []string{"AWAITING_VENDOR", "OVERDUE", "WITH_RISKS", "HIGH_RISK"} {
		q.Filter = filter
		filtered, err := store.ListVendorForms(ctx, q)
		wantResponse := filter == "WITH_RISKS" || filter == "HIGH_RISK"
		wantDistribution, wantSubmitted, wantOutstanding := pending.Distribution.ID, 0, 1
		if wantResponse {
			wantDistribution, wantSubmitted, wantOutstanding = distributionID, 1, 0
		}
		if err != nil || len(filtered.Items) != 1 || filtered.Items[0].DistributionID != wantDistribution {
			t.Fatalf("filter %s: page=%+v err=%v", filter, filtered, err)
		}
		counts, err := store.VendorFormSummaries(ctx, q)
		if err != nil || len(counts) != 1 || counts[0].SubmittedForms != wantSubmitted || counts[0].OutstandingForms != wantOutstanding || counts[0].OverdueForms != wantOutstanding {
			t.Fatalf("filter %s: summary=%+v err=%v", filter, counts, err)
		}
	}
	for _, state := range []string{"REVOKED", "SUPERSEDED"} {
		if _, err := pool.Exec(ctx, `UPDATE capture_form_distributions SET status=$2 WHERE id=$1::uuid`, distributionID, state); err != nil {
			t.Fatal(err)
		}
		q.Filter = ""
		retired, err := store.ListVendorForms(ctx, q)
		if err != nil || len(retired.Items) != 2 {
			t.Fatalf("retired page=%+v err=%v", retired, err)
		}
		foundRetired := false
		for _, row := range retired.Items {
			if row.RequestID == requestID {
				foundRetired = true
				if row.ResponseState != state || row.ResponseID != responseID || row.Current || row.ResponseCurrency != "HISTORICAL" {
					t.Fatalf("retirement must override submitted response: %+v", row)
				}
			}
		}
		if !foundRetired {
			t.Fatal("retired response missing from history")
		}
		counts, err := store.VendorFormSummaries(ctx, q)
		if err != nil || len(counts) != 1 || counts[0].SubmittedForms != 0 || counts[0].OutstandingForms != 1 || counts[0].OverdueForms != 1 || counts[0].HighestConcern != "" {
			t.Fatalf("retired summary=%+v err=%v", counts, err)
		}
		q.Filter = "HIGH_RISK"
		risks, err := store.ListVendorForms(ctx, q)
		if err != nil || len(risks.Items) != 0 {
			t.Fatalf("retired response included in current risks: %+v err=%v", risks, err)
		}
	}
	var requestState string
	if err := pool.QueryRow(ctx, `SELECT status FROM capture_requests WHERE id=$1::uuid`, requestID).Scan(&requestState); err != nil || requestState != "IN_PROGRESS" {
		t.Fatalf("reusable request changed: state=%s err=%v", requestState, err)
	}
}

func TestPostgresVendorFormsScopeProgressPaginationAndRetry(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	const tenant = "8a111111-1111-7111-8111-111111111111"
	const entity = "8a111111-1111-7111-8111-111111111112"
	const owner = "8a111111-1111-7111-8111-111111111113"
	const other = "8a111111-1111-7111-8111-111111111114"
	const form = "8a111111-1111-7111-8111-111111111115"
	const vendor = "8a111111-1111-7111-8111-111111111116"
	const rel = "8a111111-1111-7111-8111-111111111117"
	now := time.Now().UTC()
	cleanupVendorFormsFixture(ctx, pool, tenant)
	setupDistributionFixture(t, ctx, pool, tenant, entity, owner, other, form, now)
	defer cleanupVendorFormsFixture(context.Background(), pool, tenant)
	_, err := pool.Exec(ctx, `INSERT INTO third_parties(id,tenant_id,legal_name,status,created_at,updated_at,version) VALUES($1::uuid,$2::uuid,'Sample vendor','ACTIVE',$6,$6,1); INSERT INTO third_party_relationships(id,tenant_id,legal_entity_id,vendor_id,service_name,business_owner_principal_id,criticality,privacy_role,status,created_at,updated_at,version) VALUES($3::uuid,$2::uuid,$4::uuid,$1::uuid,'Processing',$5::uuid,'IMPORTANT','PROCESSOR','PROPOSED',$6,$6,1)`, pgx.QueryExecModeSimpleProtocol, vendor, tenant, rel, entity, owner, now)
	if err != nil {
		t.Fatal(err)
	}
	store := NewPostgresDistributionStore(NewPostgresRepository(pool), postgresTestRecipientProtector{})
	input := CreateDistributionInput{TenantID: tenant, LegalEntityID: entity, FormTemplateID: form, FormTemplateVersion: 1, SubjectType: "VENDOR_RELATIONSHIP", SubjectID: rel, Title: "Security review", Purpose: "Review supplier security", AccessPolicy: AccessDirectEmailOTP, EstimatedMinutes: 10, Deadline: now.Add(72 * time.Hour), RouteExpiresAt: now.Add(48 * time.Hour), CreatedBy: owner, Recipients: []DistributionRecipientInput{{Role: RecipientTo, Type: RecipientExternalAudience, Address: "vendor@example.test", AudienceHint: "Vendor contact"}}, IdempotencyKey: "test:batch:1"}
	first, err := store.CreateDistribution(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	retry, err := store.CreateDistribution(ctx, input)
	if err != nil || first.Distribution.ID != retry.Distribution.ID {
		t.Fatalf("retry=%+v err=%v", retry, err)
	}
	input.IdempotencyKey = "test:batch:2"
	if _, err = store.CreateDistribution(ctx, input); err != nil {
		t.Fatal(err)
	}
	q := VendorFormsQuery{TenantID: tenant, LegalEntityID: entity, PrincipalID: owner, RelationshipIDs: []string{rel}, Limit: 1}
	page, err := store.ListVendorForms(ctx, q)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.NextCursor == "" || page.Items[0].RequiredCount == nil || *page.Items[0].RequiredCount != 1 {
		t.Fatalf("page=%+v", page)
	}
	q.Cursor = page.NextCursor
	next, err := store.ListVendorForms(ctx, q)
	if err != nil || len(next.Items) != 1 || next.Items[0].RequestID == page.Items[0].RequestID {
		t.Fatalf("next=%+v err=%v", next, err)
	}
	summaries, err := store.VendorFormSummaries(ctx, q)
	if err != nil || len(summaries) != 1 || summaries[0].OutstandingForms != 2 {
		t.Fatalf("summaries=%+v err=%v", summaries, err)
	}
	raw, _ := json.Marshal(page)
	if strings.Contains(string(raw), "vendor@example.test") {
		t.Fatal("protected recipient leaked")
	}
	q.PrincipalID = other
	q.Cursor = ""
	denied, err := store.ListVendorForms(ctx, q)
	if err != nil || len(denied.Items) != 0 {
		t.Fatalf("unauthorized page=%+v err=%v", denied, err)
	}
	summary, err := store.VendorFormSummaries(ctx, q)
	if err != nil || summary[0].OutstandingForms != 0 {
		t.Fatalf("unauthorized count=%+v err=%v", summary, err)
	}
}
