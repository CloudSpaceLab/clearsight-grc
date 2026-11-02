//go:build postgres && postgresintegration

package autonomy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresAutomationPolicyPageDeduplicatesBeforeLimitAndKeepsTenantScope(t *testing.T) {
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
	const tenantID = "9a200000-0000-7000-8000-000000000001"
	const otherID = "9a200000-0000-7000-8000-000000000002"
	cleanup := func() {
		if _, err := pool.Exec(ctx, `DELETE FROM automation_policies WHERE tenant_id IN ($1::uuid,$2::uuid)`, tenantID, otherID); err != nil {
			t.Error(err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM tenants WHERE id IN ($1::uuid,$2::uuid)`, tenantID, otherID); err != nil {
			t.Error(err)
		}
	}
	cleanup()
	defer cleanup()
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES
		($1::uuid,'automation-page-test','Automation page test'),
		($2::uuid,'automation-page-other','Other automation page test')`, tenantID, otherID); err != nil {
		t.Fatal(err)
	}
	// There are 400 revisions but only 200 policy codes. The latest revision is
	// deliberately a draft, so neither raw rows nor active rows are this list's population.
	if _, err := pool.Exec(ctx, `INSERT INTO automation_policies(tenant_id,code,name,action_class,eligibility,blast_radius_limit,verification_contract,status,version)
		SELECT $1::uuid, 'POLICY-'||lpad(i::text,3,'0'), 'Policy '||(200-i)::text,
		'REVERSIBLE_WRITE','{"materiality_max":2}'::jsonb,'{"max_records":25}'::jsonb,'{"method":"source_recheck"}'::jsonb,
		CASE WHEN revision=1 THEN 'ACTIVE' ELSE 'DRAFT' END, revision
		FROM generate_series(0,199) i CROSS JOIN generate_series(1,2) revision`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO automation_policies(tenant_id,code,name,action_class,eligibility,blast_radius_limit,verification_contract,version)
		VALUES($1::uuid,'000-FOREIGN','Other tenant','REVERSIBLE_WRITE','{}','{}','{}',99)`, otherID); err != nil {
		t.Fatal(err)
	}
	repo := NewPostgresRepository(pool)
	service := NewService(repo)
	for _, tenant := range []string{tenantID, "automation-page-test"} {
		page, err := service.ListAutomationPolicyPage(ctx, tenant)
		if err != nil {
			t.Fatal(err)
		}
		if page.TenantID != tenant || page.ReturnedCount != 200 || len(page.Items) != 200 || page.Truncated {
			t.Fatalf("revisions or foreign records distorted the exact-200 population: %#v", page)
		}
		for i, policy := range page.Items {
			if policy.Code != fmt.Sprintf("POLICY-%03d", i) || policy.Version != 2 || policy.Status != AutomationPolicyDraft || policy.TenantID != "automation-page-test" {
				t.Fatalf("latest ordered tenant policy %d = %#v", i, policy)
			}
			var radius struct {
				MaxRecords int `json:"max_records"`
			}
			var check struct {
				Method string `json:"method"`
			}
			if json.Unmarshal(policy.BlastRadiusLimit, &radius) != nil || json.Unmarshal(policy.VerificationContract, &check) != nil || radius.MaxRecords != 25 || check.Method != "source_recheck" {
				t.Fatalf("guardrails lost during list read: %#v", policy)
			}
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO automation_policies(tenant_id,code,name,action_class,eligibility,blast_radius_limit,verification_contract,version)
		VALUES($1::uuid,'POLICY-200','Final policy','REVERSIBLE_WRITE','{}','{}','{}',1)`, tenantID); err != nil {
		t.Fatal(err)
	}
	page, err := service.ListAutomationPolicyPage(ctx, tenantID)
	if err != nil || !page.Truncated || page.ReturnedCount != 200 || len(page.Items) != 200 || page.Items[199].Code != "POLICY-199" {
		t.Fatalf("201st code not disclosed by bounded page: count=%d truncated=%t err=%v", page.ReturnedCount, page.Truncated, err)
	}
	stored, err := repo.ListAutomationPolicies(ctx, tenantID)
	if err != nil || len(stored) != 201 || stored[200].Code != "POLICY-200" {
		t.Fatalf("sentinel row missing: count=%d err=%v", len(stored), err)
	}
}
