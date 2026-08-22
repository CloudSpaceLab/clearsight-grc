//go:build postgres && postgresintegration

package continuity

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresProgramSummaryHidesRestrictedMatterState(t *testing.T) {
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
		tenantID   = "f1666666-6666-4666-8666-666666666661"
		programID  = "f1666666-6666-4666-8666-666666666662"
		matterID   = "f1666666-6666-4666-8666-666666666663"
		linkID     = "f1666666-6666-4666-8666-666666666664"
		snapshotID = "f1666666-6666-4666-8666-666666666665"
		principalA = "f1666666-6666-4666-8666-666666666666"
		principalB = "f1666666-6666-4666-8666-666666666667"
	)
	cleanupProgramStateVisibilityTenant(ctx, pool, tenantID)
	t.Cleanup(func() { cleanupProgramStateVisibilityTenant(context.Background(), pool, tenantID) })
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'program-state-visibility','Program State Visibility')`, tenantID); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	if _, err := pool.Exec(ctx, `
		INSERT INTO programs(id,tenant_id,code,name,program_type,status,owning_function,jurisdiction,scope,effective_from,created_at,updated_at,version)
		VALUES($1::uuid,$2::uuid,'PSV','Program state visibility','COMPLIANCE','ACTIVE','Compliance','','{}'::jsonb,$3,$3,$3,1)`, programID, tenantID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO matters(id,tenant_id,reference,matter_type,status,priority,title,summary,scope,known_facts,missing_facts,contradictions,created_at,updated_at,version)
		VALUES($1::uuid,$2::uuid,'PSV-MAT','AUTHORITY_REQUEST','ASSESSMENT',4,'Restricted issue','Restricted issue',
		'{"access":"RESTRICTED","allowed_principal_ids":["f1666666-6666-4666-8666-666666666667"]}'::jsonb,'{}','[]','[]',$3,$3,1)`, matterID, tenantID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO matter_links(id,tenant_id,matter_id,program_id,relationship,created_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,'AFFECTS',$5)`, linkID, tenantID, matterID, programID, now); err != nil {
		t.Fatal(err)
	}

	dimensions := allCurrentDimensions()
	dimensions.Exception = StateAtRisk
	dimensionsJSON, _ := json.Marshal(dimensions)
	reasonsJSON, _ := json.Marshal([]StateReason{{Code: "OPEN_MATTERS", Summary: "1 open issue(s) or change(s) affect this program."}})
	if _, err := pool.Exec(ctx, `
		INSERT INTO program_state_snapshots(id,tenant_id,program_id,overall_state,dimensions,reasons,open_matter_count,trigger_type,trigger_id,generated_at,program_version,projection_version)
		VALUES($1::uuid,$2::uuid,$3::uuid,'AT_RISK',$4::jsonb,$5::jsonb,1,'MATTER_STATE_CHANGED',$6,$7,1,1)`, snapshotID, tenantID, programID, string(dimensionsJSON), string(reasonsJSON), matterID, now); err != nil {
		t.Fatal(err)
	}

	repo := NewPostgresRepository(pool)
	actorA := identity.WithActor(ctx, identity.Actor{TenantID: tenantID, PrincipalID: principalA})
	pageA, err := repo.ListProgramSummaries(actorA, tenantID, SummaryQuery{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(pageA.Items) != 1 {
		t.Fatalf("actor A Program summary count=%d", len(pageA.Items))
	}
	gotA := pageA.Items[0]
	if gotA.OpenMatterCount != 0 || gotA.OverallState != StateCurrent || hasReasonCode(gotA.Reasons, "OPEN_MATTERS") {
		t.Fatalf("restricted Matter leaked into actor A Program summary: %#v", gotA)
	}
	if gotA.ProjectionVersion < 1 || gotA.ProjectionVersion == 1 {
		t.Fatalf("actor A did not receive a semantic projection version: %d", gotA.ProjectionVersion)
	}

	actorB := identity.WithActor(ctx, identity.Actor{TenantID: tenantID, PrincipalID: principalB})
	pageB, err := repo.ListProgramSummaries(actorB, tenantID, SummaryQuery{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(pageB.Items) != 1 {
		t.Fatalf("actor B Program summary count=%d", len(pageB.Items))
	}
	gotB := pageB.Items[0]
	if gotB.OpenMatterCount != 1 || gotB.OverallState != StateAtRisk || !hasReasonCode(gotB.Reasons, "OPEN_MATTERS") {
		t.Fatalf("authorized principal lost restricted Matter state: %#v", gotB)
	}
	if gotA.ProjectionVersion == gotB.ProjectionVersion {
		t.Fatal("different visible Program states share one semantic projection version")
	}

	wrongTenant := identity.WithActor(ctx, identity.Actor{TenantID: "other-bank", PrincipalID: principalB})
	wrongPage, err := repo.ListProgramSummaries(wrongTenant, "program-state-visibility", SummaryQuery{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(wrongPage.Items) != 0 {
		t.Fatalf("Program summary trusted caller tenant over verified actor tenant: %#v", wrongPage.Items)
	}
}

func TestPostgresVisibleOpenMatterCountsRespectHistoricalAccess(t *testing.T) {
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
		tenantID   = "f1777777-7777-4777-8777-777777777771"
		programID  = "f1777777-7777-4777-8777-777777777772"
		matterID   = "f1777777-7777-4777-8777-777777777773"
		linkID     = "f1777777-7777-4777-8777-777777777774"
		createdEvt = "f1777777-7777-4777-8777-777777777775"
		changedEvt = "f1777777-7777-4777-8777-777777777776"
		principalA = "f1777777-7777-4777-8777-777777777777"
		principalB = "f1777777-7777-4777-8777-777777777778"
	)
	cleanupProgramStateVisibilityTenant(ctx, pool, tenantID)
	t.Cleanup(func() { cleanupProgramStateVisibilityTenant(context.Background(), pool, tenantID) })
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'program-state-history','Program State History')`, tenantID); err != nil {
		t.Fatal(err)
	}

	t1 := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	t2 := t1.Add(2 * time.Hour)
	if _, err := pool.Exec(ctx, `
		INSERT INTO programs(id,tenant_id,code,name,program_type,status,owning_function,jurisdiction,scope,effective_from,created_at,updated_at,version)
		VALUES($1::uuid,$2::uuid,'PSH','Program state history','COMPLIANCE','ACTIVE','Compliance','','{}'::jsonb,$3,$3,$3,1)`, programID, tenantID, t1); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO matters(id,tenant_id,reference,matter_type,status,priority,title,summary,scope,known_facts,missing_facts,contradictions,created_at,updated_at,version)
		VALUES($1::uuid,$2::uuid,'PSH-MAT','AUTHORITY_REQUEST','ASSESSMENT',4,'Historical issue','Historical issue',
		'{"access":"RESTRICTED","allowed_principal_ids":["f1777777-7777-4777-8777-777777777778"]}'::jsonb,'{}','[]','[]',$3,$4,2)`, matterID, tenantID, t1, t2); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO matter_links(id,tenant_id,matter_id,program_id,relationship,created_at) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,'AFFECTS',$5)`, linkID, tenantID, matterID, programID, t1); err != nil {
		t.Fatal(err)
	}

	createdMatter := Matter{ID: matterID, TenantID: tenantID, Status: MatterAssessment, Scope: json.RawMessage(`{"access":"INTERNAL"}`), CreatedAt: t1, UpdatedAt: t1, Version: 1}
	changedMatter := createdMatter
	changedMatter.Scope = json.RawMessage(`{"access":"RESTRICTED","allowed_principal_ids":["f1777777-7777-4777-8777-777777777778"]}`)
	changedMatter.UpdatedAt = t2
	changedMatter.Version = 2
	createdPayload, _ := json.Marshal(createdMatter)
	changedPayload, _ := json.Marshal(changedMatter)
	if _, err := pool.Exec(ctx, `
		INSERT INTO continuity_events(id,tenant_id,aggregate_type,aggregate_id,aggregate_version,event_type,payload,actor_type,occurred_at) VALUES
		($1::uuid,$3::uuid,'MATTER',$4::uuid,1,'MATTER_CREATED',$5::jsonb,'SYSTEM',$7),
		($2::uuid,$3::uuid,'MATTER',$4::uuid,2,'MATTER_STATE_CHANGED',$6::jsonb,'SYSTEM',$8)`, createdEvt, changedEvt, tenantID, matterID, string(createdPayload), string(changedPayload), t1, t2); err != nil {
		t.Fatal(err)
	}

	repo := NewPostgresRepository(pool)
	before := t1.Add(time.Hour)
	beforeCounts, err := repo.VisibleOpenMatterCounts(ctx, tenantID, []string{programID}, principalA, &before)
	if err != nil {
		t.Fatal(err)
	}
	if beforeCounts[programID] != 1 {
		t.Fatalf("historically visible Matter count=%d, want 1", beforeCounts[programID])
	}
	after := t2.Add(time.Hour)
	afterCounts, err := repo.VisibleOpenMatterCounts(ctx, tenantID, []string{programID}, principalA, &after)
	if err != nil {
		t.Fatal(err)
	}
	if afterCounts[programID] != 0 {
		t.Fatalf("historically restricted Matter count=%d, want 0", afterCounts[programID])
	}
	allowedCounts, err := repo.VisibleOpenMatterCounts(ctx, tenantID, []string{programID}, principalB, &after)
	if err != nil {
		t.Fatal(err)
	}
	if allowedCounts[programID] != 1 {
		t.Fatalf("allowed historical Matter count=%d, want 1", allowedCounts[programID])
	}
}

func cleanupProgramStateVisibilityTenant(ctx context.Context, pool *pgxpool.Pool, tenantID string) {
	for _, query := range []string{
		`DELETE FROM program_review_checkpoints WHERE tenant_id=$1::uuid`,
		`DELETE FROM program_state_work_items WHERE tenant_id=$1::uuid`,
		`DELETE FROM program_state_snapshots WHERE tenant_id=$1::uuid`,
		`DELETE FROM outbox_events WHERE tenant_id=$1::uuid`,
		`DELETE FROM continuity_events WHERE tenant_id=$1::uuid`,
		`DELETE FROM matter_links WHERE tenant_id=$1::uuid`,
		`DELETE FROM matters WHERE tenant_id=$1::uuid`,
		`DELETE FROM programs WHERE tenant_id=$1::uuid`,
		`DELETE FROM principals WHERE tenant_id=$1::uuid`,
		`DELETE FROM tenants WHERE id=$1::uuid`,
	} {
		_, _ = pool.Exec(ctx, query, tenantID)
	}
}
