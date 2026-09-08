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
