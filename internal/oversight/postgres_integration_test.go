//go:build postgres && postgresintegration

package oversight

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresProjectionExcludesRestrictedAndUnknownMatterScopes(t *testing.T) {
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
		tenantID = "8a646464-6464-7464-8464-646464646401"
		entityID = "8a646464-6464-7464-8464-646464646402"
	)
	cleanup := func(cleanCtx context.Context) {
		_, _ = pool.Exec(cleanCtx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	}
	cleanup(ctx)
	t.Cleanup(func() { cleanup(context.Background()) })

	now := time.Now().UTC().Truncate(time.Second)
	mustOversightExec(t, ctx, pool, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'oversight-scope-test','Oversight scope test')`, tenantID)
	mustOversightExec(t, ctx, pool, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2::uuid,'SCOPE-NG','Scope Bank Nigeria','NG',$3)`, entityID, tenantID, now.Add(-time.Hour))
	for _, matter := range []struct {
		id, reference, title, scope string
	}{
		{"8a646464-6464-7464-8464-646464646411", "SCOPE-VISIBLE", "Visible control gap", `{"access":"INTERNAL"}`},
		{"8a646464-6464-7464-8464-646464646412", "SCOPE-RESTRICTED", "Restricted control gap", `{"access":"RESTRICTED"}`},
		{"8a646464-6464-7464-8464-646464646413", "SCOPE-UNKNOWN", "Unknown-scope control gap", `{"access":{"unexpected":true}}`},
	} {
		mustOversightExec(t, ctx, pool, `INSERT INTO matters(id,tenant_id,legal_entity_id,reference,matter_type,status,priority,title,summary,scope,created_at,updated_at) VALUES($1::uuid,$2::uuid,$3::uuid,$4,'CONTROL_GAP','TRIAGE',5,$5,'Scope boundary test',$6::jsonb,$7,$7)`, matter.id, tenantID, entityID, matter.reference, matter.title, matter.scope, now.Add(-24*time.Hour))
	}

	repository := NewPostgresRepository(pool)
	value, err := repository.build(ctx, Scope{TenantID: tenantID, LegalEntityID: entityID}, now)
	if err != nil {
		t.Fatal(err)
	}
	if value.Coverage.Population != 3 || value.Coverage.Excluded == nil || *value.Coverage.Excluded != 1 || value.Coverage.Unknown == nil || *value.Coverage.Unknown != 1 {
		t.Fatalf("coverage=%#v", value.Coverage)
	}
	if value.Counts.CriticalHigh != 1 || len(value.Interventions) != 1 || value.Interventions[0].Title != "Visible control gap" {
		t.Fatalf("restricted or unknown record leaked: counts=%#v interventions=%#v", value.Counts, value.Interventions)
	}
	if inserted, err := repository.store(ctx, value, now.Truncate(refreshInterval)); err != nil || !inserted {
		t.Fatalf("store projection inserted=%t err=%v", inserted, err)
	}
	loaded, err := NewService(repository).Get(ctx, Scope{TenantID: tenantID, LegalEntityID: entityID})
	if err != nil || loaded.Counts.CriticalHigh != 1 || loaded.Coverage.Excluded == nil || *loaded.Coverage.Excluded != 1 {
		t.Fatalf("loaded projection=%#v err=%v", loaded, err)
	}
}

func mustOversightExec(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(ctx, query, args...); err != nil {
		t.Fatal(err)
	}
}
