//go:build postgres && postgresintegration

package oversight

import (
	"context"
	"os"
	"strings"
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
	value, err := repository.build(ctx, Scope{TenantID: tenantID, LegalEntityID: entityID}, now, now.Add(-90*24*time.Hour))
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
	var snapshotID, membershipRevision string
	if err := pool.QueryRow(ctx, `
		SELECT id::text,COALESCE(metric_membership_revision,'')
		FROM oversight_snapshots
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid
		ORDER BY generated_at DESC,id DESC LIMIT 1`, tenantID, entityID).Scan(&snapshotID, &membershipRevision); err != nil {
		t.Fatal(err)
	}
	if membershipRevision != MetricSnapshotDrillDefinitionRevision {
		t.Fatalf("membership revision=%q", membershipRevision)
	}
	var exactCritical int
	if err := pool.QueryRow(ctx, `
		SELECT count(*)
		FROM oversight_snapshot_metric_memberships
		WHERE oversight_snapshot_id=$1::uuid
		  AND metric_id=$2
		  AND definition_revision=$3`,
		snapshotID, MetricCriticalHighOpen, MetricSnapshotDrillDefinitionRevision,
	).Scan(&exactCritical); err != nil {
		t.Fatal(err)
	}
	if exactCritical != value.Counts.CriticalHigh || exactCritical != 1 {
		t.Fatalf("critical membership=%d count=%d", exactCritical, value.Counts.CriticalHigh)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE oversight_snapshot_metric_memberships
		SET state='CHANGED'
		WHERE oversight_snapshot_id=$1::uuid AND metric_id=$2`,
		snapshotID, MetricCriticalHighOpen,
	); err == nil {
		t.Fatal("retained metric membership mutation was accepted")
	}
	var storedBefore int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM oversight_snapshots WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid`, tenantID, entityID).Scan(&storedBefore); err != nil {
		t.Fatal(err)
	}
	customStart := now.Add(-180 * 24 * time.Hour)
	custom, err := repository.BuildPeriod(ctx, Scope{TenantID: tenantID, LegalEntityID: entityID}, customStart, now)
	if err != nil {
		t.Fatal(err)
	}
	if custom.Counts.CriticalHigh != value.Counts.CriticalHigh || !custom.PeriodStart.Equal(customStart) || !custom.PeriodEnd.Equal(now) {
		t.Fatalf("custom period changed current posture or effective range: %#v", custom)
	}
	var storedAfter int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM oversight_snapshots WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid`, tenantID, entityID).Scan(&storedAfter); err != nil {
		t.Fatal(err)
	}
	if storedAfter != storedBefore {
		t.Fatalf("on-demand reporting period persisted a snapshot: before=%d after=%d", storedBefore, storedAfter)
	}
	loaded, err := NewService(repository).Get(ctx, Scope{TenantID: tenantID, LegalEntityID: entityID})
	if err != nil || loaded.Counts.CriticalHigh != 1 || loaded.Coverage.Excluded == nil || *loaded.Coverage.Excluded != 1 {
		t.Fatalf("loaded projection=%#v err=%v", loaded, err)
	}
	if loaded.SnapshotID != snapshotID {
		t.Fatalf("loaded exact source id=%q want=%q", loaded.SnapshotID, snapshotID)
	}
	var operatorID string
	if err := pool.QueryRow(ctx, `INSERT INTO principals(tenant_id,kind,display_name) VALUES($1::uuid,'PERSON','Demo curator') RETURNING id::text`, tenantID).Scan(&operatorID); err != nil {
		t.Fatal(err)
	}
	mustOversightExec(t, ctx, pool, `INSERT INTO demo_record_archives(tenant_id,legal_entity_id,record_type,record_id,reason,source_manifest,archived_by,archived_at) VALUES($1::uuid,$2::uuid,'MATTER','8a646464-6464-7464-8464-646464646411','Excluded sample','test-manifest',$3::uuid,now())`, tenantID, entityID, operatorID)
	curated, err := repository.build(ctx, Scope{TenantID: tenantID, LegalEntityID: entityID}, now, now.Add(-90*24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if curated.Coverage.Population != 2 || curated.Counts.CriticalHigh != 0 || len(curated.Interventions) != 0 || len(curated.Pressure) != 0 {
		t.Fatalf("archived sample affected oversight: %#v", curated)
	}
}

func TestPostgresProjectionFiltersAuthorizedOrganizationScopeDescendantsWithoutSiblingLeakage(t *testing.T) {
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
		tenantID       = "8a686868-6868-7868-8868-686868686801"
		entityID       = "8a686868-6868-7868-8868-686868686802"
		riskScope      = "8a686868-6868-7868-8868-686868686803"
		opsScope       = "8a686868-6868-7868-8868-686868686804"
		riskChildScope = "8a686868-6868-7868-8868-686868686805"
	)
	cleanup := func(cleanCtx context.Context) {
		_, _ = pool.Exec(cleanCtx, `DELETE FROM matters WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM organization_scopes WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	}
	cleanup(ctx)
	t.Cleanup(func() { cleanup(context.Background()) })

	now := time.Now().UTC().Truncate(time.Second)
	mustOversightExec(t, ctx, pool, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'oversight-org-scope','Oversight organization scope')`, tenantID)
	mustOversightExec(t, ctx, pool, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2::uuid,'ORG-NG','Organization Bank Nigeria','NG',$3)`, entityID, tenantID, now.Add(-time.Hour))
	mustOversightExec(t, ctx, pool, `INSERT INTO organization_scopes(id,tenant_id,legal_entity_id,code,name,kind,department_path,origin,status,valid_from) VALUES
		($1::uuid,$4::uuid,$5::uuid,'RISK','Risk','DEPARTMENT',ARRAY['BANK','RISK'],'MANAGED','ACTIVE',$6),
		($2::uuid,$4::uuid,$5::uuid,'OPS','Operations','DEPARTMENT',ARRAY['BANK','OPERATIONS'],'MANAGED','ACTIVE',$6),
		($3::uuid,$4::uuid,$5::uuid,'RISK-OPS','Risk Operations','DEPARTMENT',ARRAY['BANK','RISK','OPERATIONS'],'MANAGED','ACTIVE',$6)`, riskScope, opsScope, riskChildScope, tenantID, entityID, now.Add(-time.Hour))
	for _, item := range []struct {
		id, ref, title, organizationScope string
	}{
		{"8a686868-6868-7868-8868-686868686811", "ORG-RISK", "Risk exact issue", riskScope},
		{"8a686868-6868-7868-8868-686868686812", "ORG-OPS", "Operations sibling issue", opsScope},
		{"8a686868-6868-7868-8868-686868686813", "ORG-UNATTRIBUTED", "Unattributed issue", ""},
		{"8a686868-6868-7868-8868-686868686814", "ORG-RISK-OPS", "Risk child issue", riskChildScope},
	} {
		mustOversightExec(t, ctx, pool, `INSERT INTO matters(id,tenant_id,legal_entity_id,organization_scope_id,reference,matter_type,status,priority,title,summary,scope,created_at,updated_at)
			VALUES($1::uuid,$2::uuid,$3::uuid,NULLIF($4,'')::uuid,$5,'CONTROL_GAP','TRIAGE',5,$6,'Organization scope boundary','{"access":"INTERNAL"}'::jsonb,$7,$7)`,
			item.id, tenantID, entityID, item.organizationScope, item.ref, item.title, now.Add(-24*time.Hour))
	}

	value, err := NewPostgresRepository(pool).BuildPeriod(ctx, Scope{
		TenantID: tenantID, LegalEntityID: entityID, OrganizationScopeID: riskScope, OrganizationScopeIDs: []string{riskScope, riskChildScope},
	}, now.Add(-90*24*time.Hour), now)
	if err != nil {
		t.Fatal(err)
	}
	if value.OrganizationScopeID != riskScope || value.Coverage.Population != 2 || value.Coverage.Unknown != nil {
		t.Fatalf("scoped coverage leaked legal-entity attribution: scope=%s coverage=%#v", value.OrganizationScopeID, value.Coverage)
	}
	if value.Counts.CriticalHigh != 2 || len(value.Interventions) != 2 {
		t.Fatalf("sibling or unattributed Matter leaked: counts=%#v interventions=%#v", value.Counts, value.Interventions)
	}
	for _, item := range value.Interventions {
		if item.Title == "Operations sibling issue" || item.Title == "Unattributed issue" {
			t.Fatalf("forbidden Matter leaked: %#v", item)
		}
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

	value, err := NewPostgresRepository(pool).build(ctx, Scope{TenantID: tenantID, LegalEntityID: entityID}, now, now.Add(-90*24*time.Hour))
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

func TestReportingPeriodScalesAcrossLargeEntityAndSnapshotHistory(t *testing.T) {
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
		tenantID      = "8a676767-6767-7767-8767-676767676701"
		entityID      = "8a676767-6767-7767-8767-676767676702"
		matterCount   = 20_000
		snapshotCount = 20_000
	)
	cleanup := func(cleanCtx context.Context) {
		_, _ = pool.Exec(cleanCtx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	}
	cleanup(ctx)
	t.Cleanup(func() { cleanup(context.Background()) })

	now := time.Now().UTC().Truncate(time.Second)
	mustOversightExec(t, ctx, pool, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'oversight-period-load','Oversight period load')`, tenantID)
	mustOversightExec(t, ctx, pool, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2::uuid,'LOAD-NG','Load Bank Nigeria','NG',$3)`, entityID, tenantID, now.Add(-2*365*24*time.Hour))
	mustOversightExec(t, ctx, pool, `
		INSERT INTO matters(
			id,tenant_id,legal_entity_id,reference,matter_type,status,priority,title,summary,scope,
			due_at,closed_at,closure_reason,created_at,updated_at
		)
		SELECT md5('oversight-period-matter-' || gs::text)::uuid,$1::uuid,$2::uuid,
		       'LOAD-' || lpad(gs::text,6,'0'),'CONTROL_GAP',
		       CASE WHEN gs % 10 = 0 THEN 'CLOSED' ELSE 'TRIAGE' END,
		       CASE WHEN gs % 5 = 0 THEN 5 ELSE 3 END,
		       'Oversight load matter ' || gs,'Synthetic reporting-period load row','{"access":"INTERNAL"}'::jsonb,
		       CASE WHEN gs % 4 = 0 THEN $3::timestamptz-interval '2 days' ELSE $3::timestamptz+interval '2 days' END,
		       CASE WHEN gs % 10 = 0 THEN $3::timestamptz-(gs % 180 + 1)*interval '1 day' ELSE NULL END,
		       CASE WHEN gs % 10 = 0 THEN 'Closed in synthetic load history' ELSE '' END,
		       $3::timestamptz-(gs % 300 + 1)*interval '1 day',
		       $3::timestamptz
		FROM generate_series(1,$4) gs`, tenantID, entityID, now, matterCount)
	mustOversightExec(t, ctx, pool, `
		INSERT INTO oversight_snapshots(
			tenant_id,legal_entity_id,period_start,period_end,refresh_slot,generated_at,projection_version,
			source_high_water,coverage_population,coverage_excluded,coverage_unknown,payload
		)
		SELECT $1::uuid,$2::uuid,
		       ($3::timestamptz-gs*interval '5 minutes')-interval '90 days',
		       $3::timestamptz-gs*interval '5 minutes',
		       $3::timestamptz-gs*interval '5 minutes',
		       $3::timestamptz-gs*interval '5 minutes',
		       $4,'{}'::jsonb,$5,0,0,'{}'::jsonb
		FROM generate_series(0,$6-1) gs`, tenantID, entityID, now, ProjectionVersion, matterCount, snapshotCount)
	mustOversightExec(t, ctx, pool, `ANALYZE matters`)
	mustOversightExec(t, ctx, pool, `ANALYZE oversight_snapshots`)

	var plan []byte
	if err := pool.QueryRow(ctx, `
		EXPLAIN (FORMAT JSON)
		SELECT os.generated_at
		FROM oversight_snapshots os
		WHERE os.tenant_id=$1::uuid AND os.legal_entity_id=$2::uuid
		ORDER BY os.generated_at DESC,os.id DESC
		LIMIT 1`, tenantID, entityID).Scan(&plan); err != nil {
		t.Fatal(err)
	}
	planText := string(plan)
	if !strings.Contains(planText, "\"Node Type\": \"Index Scan\"") || strings.Contains(planText, "\"Node Type\": \"Seq Scan\"") || strings.Contains(planText, "\"Node Type\": \"Sort\"") {
		t.Fatalf("latest snapshot plan is not an ordered indexed read: %s", plan)
	}
	var latestIndexDefinition string
	if err := pool.QueryRow(ctx, `
		SELECT indexdef FROM pg_indexes
		WHERE schemaname='public' AND tablename='oversight_snapshots' AND indexname='oversight_snapshots_latest_idx'`).Scan(&latestIndexDefinition); err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"tenant_id", "legal_entity_id", "generated_at DESC", "id DESC"} {
		if !strings.Contains(latestIndexDefinition, required) {
			t.Fatalf("latest snapshot index is missing %q: %s", required, latestIndexDefinition)
		}
	}

	repository := NewPostgresRepository(pool)
	latestCtx, cancelLatest := context.WithTimeout(ctx, 2*time.Second)
	latest, err := repository.Latest(latestCtx, Scope{TenantID: tenantID, LegalEntityID: entityID})
	cancelLatest()
	if err != nil {
		t.Fatalf("latest snapshot failed under %d-row history: %v", snapshotCount, err)
	}
	if !latest.GeneratedAt.Equal(now) {
		t.Fatalf("latest generated_at=%s want=%s", latest.GeneratedAt, now)
	}

	periodCtx, cancelPeriod := context.WithTimeout(ctx, 10*time.Second)
	value, err := repository.BuildPeriod(periodCtx, Scope{TenantID: tenantID, LegalEntityID: entityID}, now.Add(-180*24*time.Hour), now)
	cancelPeriod()
	if err != nil {
		t.Fatalf("180-day reporting period failed under %d-matter load: %v", matterCount, err)
	}
	if value.Coverage.Population != matterCount || value.Counts.CriticalHigh != 2_000 || value.Counts.Overdue != 4_000 || value.Counts.Unassigned != 18_000 {
		t.Fatalf("unexpected large-entity posture: coverage=%#v counts=%#v", value.Coverage, value.Counts)
	}
	if !value.PeriodStart.Equal(now.Add(-180*24*time.Hour)) || !value.PeriodEnd.Equal(now) {
		t.Fatalf("effective period=%s..%s", value.PeriodStart, value.PeriodEnd)
	}
}

func mustOversightExec(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(ctx, query, args...); err != nil {
		t.Fatal(err)
	}
}
