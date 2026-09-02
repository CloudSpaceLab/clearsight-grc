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

func TestPostgresProjectionAttributesHistoryToExactOwnerIntervals(t *testing.T) {
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
		tenantID = "8a656565-6565-7565-8565-656565656501"
		entityID = "8a656565-6565-7565-8565-656565656502"
		firstID  = "8a656565-6565-7565-8565-656565656503"
		secondID = "8a656565-6565-7565-8565-656565656504"
		matterID = "8a656565-6565-7565-8565-656565656505"
	)
	cleanup := func(cleanCtx context.Context) {
		_, _ = pool.Exec(cleanCtx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	}
	cleanup(ctx)
	t.Cleanup(func() { cleanup(context.Background()) })

	now := time.Now().UTC().Truncate(time.Second)
	openedAt, reassignedAt, returnedAt := now.Add(-100*time.Hour), now.Add(-60*time.Hour), now.Add(-50*time.Hour)
	blockedAt, reopenedAt, closedAt := now.Add(-40*time.Hour), now.Add(-30*time.Hour), now.Add(-10*time.Hour)
	mustOversightExec(t, ctx, pool, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'oversight-owner-interval-test','Oversight owner interval test')`, tenantID)
	mustOversightExec(t, ctx, pool, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2::uuid,'INTERVAL-NG','Interval Bank Nigeria','NG',$3)`, entityID, tenantID, openedAt)
	mustOversightExec(t, ctx, pool, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($1::uuid,$3::uuid,'PERSON','First owner','ACTIVE',$4),($2::uuid,$3::uuid,'PERSON','Second owner','ACTIVE',$4)`, firstID, secondID, tenantID, openedAt)
	mustOversightExec(t, ctx, pool, `INSERT INTO matters(id,tenant_id,legal_entity_id,reference,matter_type,status,priority,title,summary,scope,owner_principal_id,due_at,closed_at,closure_reason,reopen_count,created_at,updated_at,version) VALUES($1::uuid,$2::uuid,$3::uuid,'INTERVAL-1','CONTROL_GAP','CLOSED',4,'Restore source access','Owner interval projection test','{"access":"INTERNAL"}'::jsonb,$4::uuid,$5,$6,'Verified after reassignment',1,$7,$6,7)`, matterID, tenantID, entityID, secondID, now, closedAt, openedAt)
	for _, event := range []struct {
		version int
		type_   string
		payload string
		at      time.Time
	}{
		{1, "MATTER_CREATED", `{}`, openedAt},
		{2, "MATTER_OWNER_CHANGED", `{"previous_owner_principal_id":"` + firstID + `","owner_principal_id":"` + secondID + `"}`, reassignedAt},
		{3, "DECISION_ADDED", `{"status":"RETURNED"}`, returnedAt},
		{4, "ACTION_STATE_CHANGED", `{"id":"8a656565-6565-7565-8565-656565656506","status":"BLOCKED"}`, blockedAt},
		{5, "MATTER_STATE_CHANGED", `{"status":"ASSESSMENT","reopen_count":1}`, reopenedAt},
		{6, "ACTION_STATE_CHANGED", `{"id":"8a656565-6565-7565-8565-656565656506","status":"IMPLEMENTED"}`, now.Add(-20 * time.Hour)},
		{7, "MATTER_STATE_CHANGED", `{"status":"CLOSED","reopen_count":1}`, closedAt},
	} {
		mustOversightExec(t, ctx, pool, `INSERT INTO continuity_events(tenant_id,aggregate_type,aggregate_id,aggregate_version,event_type,payload,actor_type,actor_id,occurred_at) VALUES($1::uuid,'MATTER',$2::uuid,$3,$4,$5::jsonb,'PERSON',$6::uuid,$7)`, tenantID, matterID, event.version, event.type_, event.payload, secondID, event.at)
	}

	value, err := NewPostgresRepository(pool).build(ctx, Scope{TenantID: tenantID, LegalEntityID: entityID}, now)
	if err != nil {
		t.Fatal(err)
	}
	performance := make(map[string]Performance, len(value.Performance))
	for _, item := range value.Performance {
		performance[item.OwnerID] = item
	}
	first, ok := performance[firstID]
	if !ok || first.MeasurementSamples != 1 || first.MedianHours == nil || *first.MedianHours != 40 || first.Reassigned == nil || *first.Reassigned != 1 {
		t.Fatalf("first owner interval=%#v present=%t", first, ok)
	}
	second, ok := performance[secondID]
	if !ok || second.MeasurementSamples != 1 || second.MedianHours == nil || *second.MedianHours != 30 || second.Returned == nil || *second.Returned != 1 || second.Blocked != 1 || second.BlockedHours != 20 || second.Reopened != 1 {
		t.Fatalf("second owner interval=%#v present=%t", second, ok)
	}
}

func mustOversightExec(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(ctx, query, args...); err != nil {
		t.Fatal(err)
	}
}
