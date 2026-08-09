//go:build postgres && postgresintegration

package continuity

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresMatterSummariesMatchCanonicalRestrictedVisibilityBeforePagination(t *testing.T) {
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
		tenantID   = "95555555-5555-7555-8555-555555555551"
		principalA = "95555555-5555-7555-8555-555555555552"
		principalB = "95555555-5555-7555-8555-555555555553"
		hiddenID   = "95555555-5555-7555-8555-555555555554"
		mixedID    = "95555555-5555-7555-8555-555555555555"
		visibleID  = "95555555-5555-7555-8555-555555555556"
	)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID) })
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'matter-summary-visibility','Matter Summary Visibility')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO principals(id,tenant_id,kind,external_ref,display_name) VALUES
		($1::uuid,$3::uuid,'PERSON','summary-a','Summary A'),
		($2::uuid,$3::uuid,'PERSON','summary-b','Summary B')`, principalA, principalB, tenantID); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	if _, err := pool.Exec(ctx, `
		INSERT INTO matters(
			id,tenant_id,reference,matter_type,status,priority,title,summary,scope,
			known_facts,missing_facts,contradictions,created_at,updated_at,version
		) VALUES
		($1::uuid,$4::uuid,'VIS-001','AUTHORITY_REQUEST','ASSESSMENT',5,'Hidden restricted','hidden restricted','{"access":"RESTRICTED","allowed_principal_ids":["95555555-5555-7555-8555-555555555553"]}'::jsonb,'{}','[]','[]',$7,$7,1),
		($2::uuid,$4::uuid,'VIS-002','AUTHORITY_REQUEST','ASSESSMENT',4,'Mixed secret','mixed-secret','{"access":"RESTRICTED","allowed_principal_ids":["95555555-5555-7555-8555-555555555552",42]}'::jsonb,'{}','[]','[]',$7,$7,1),
		($3::uuid,$4::uuid,'VIS-003','REGULATORY_CHANGE','ASSESSMENT',3,'Visible internal','visible internal','{"access":"INTERNAL"}'::jsonb,'{}','[]','[]',$7,$7,1)`, hiddenID, mixedID, visibleID, tenantID, principalA, principalB, now); err != nil {
		t.Fatal(err)
	}

	repo := NewPostgresRepository(pool)
	actorA := identity.WithActor(ctx, identity.Actor{TenantID: "matter-summary-visibility", PrincipalID: principalA})
	page, err := repo.ListMatterSummaries(actorA, "matter-summary-visibility", SummaryQuery{Status: "OPEN", Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].Matter.ID != visibleID || page.NextCursor != "" {
		t.Fatalf("malformed/restricted rows affected actor-visible pagination: %#v cursor=%q", page.Items, page.NextCursor)
	}

	search, err := repo.ListMatterSummaries(actorA, "matter-summary-visibility", SummaryQuery{Search: "mixed-secret", Status: "OPEN", Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(search.Items) != 0 || search.NextCursor != "" {
		t.Fatalf("search leaked malformed restricted record existence: %#v cursor=%q", search.Items, search.NextCursor)
	}

	actorB := identity.WithActor(ctx, identity.Actor{TenantID: "matter-summary-visibility", PrincipalID: principalB})
	allowed, err := repo.ListMatterSummaries(actorB, "matter-summary-visibility", SummaryQuery{Status: "OPEN", Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(allowed.Items) != 2 || allowed.Items[0].Matter.ID != hiddenID || allowed.Items[1].Matter.ID != visibleID {
		t.Fatalf("canonical allowed principal lost restricted visibility: %#v", allowed.Items)
	}
}
