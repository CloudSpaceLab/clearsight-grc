//go:build postgres && postgresintegration

package continuity

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresProgramStateTemporalAndSourceTruth(t *testing.T) {
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
		tenantID  = "86666666-6666-7666-8666-666666666661"
		programID = "86666666-6666-7666-8666-666666666662"
		sourceA   = "86666666-6666-7666-8666-666666666663"
		sourceB   = "86666666-6666-7666-8666-666666666664"
		sourceC   = "86666666-6666-7666-8666-666666666665"
		reqA      = "86666666-6666-7666-8666-666666666666"
		reqB      = "86666666-6666-7666-8666-666666666667"
		reqFuture = "86666666-6666-7666-8666-666666666668"
		contract  = "86666666-6666-7666-8666-666666666669"
	)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID) })
	now := time.Now().UTC().Truncate(time.Second)
	end := now.Add(90 * 24 * time.Hour)

	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'state-truth-test','State Truth Test')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO programs(id,tenant_id,code,name,program_type,status,owning_function,scope,effective_from,effective_until,version) VALUES($1::uuid,$2::uuid,'STATE','State truth','COMPLIANCE','PAUSED','Compliance','{}'::jsonb,$3,$4,3)`, programID, tenantID, now.Add(-24*time.Hour), end); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO evidence_sources(id,tenant_id,code,name,source_type,authority_class,expected_freshness_minutes,health) VALUES
		($1::uuid,$4::uuid,'A','Source A','SYSTEM','SYSTEM_OF_RECORD',60,'CURRENT'),
		($2::uuid,$4::uuid,'B','Source B','SYSTEM','SYSTEM_OF_RECORD',60,'DEGRADED'),
		($3::uuid,$4::uuid,'C','Future source','SYSTEM','SYSTEM_OF_RECORD',60,'DEGRADED')`, sourceA, sourceB, sourceC, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO program_requirements(id,tenant_id,program_id,source_id,code,title,statement,modality,status,effective_from) VALUES
		($1::uuid,$5::uuid,$6::uuid,$7::uuid,'A','A','A','MUST','APPROVED',$8),
		($2::uuid,$5::uuid,$6::uuid,$9::uuid,'B','B','B','MUST','APPROVED',$8),
		($3::uuid,$5::uuid,$6::uuid,$10::uuid,'FUTURE','Future','Future','MUST','APPROVED',$11)`, reqA, reqB, reqFuture, tenantID, programID, sourceA, now.Add(-time.Hour), sourceB, sourceC, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	repo := NewPostgresRepository(pool)
	state, err := repo.CurrentProgramSourceState(ctx, "state-truth-test", programID)
	if err != nil {
		t.Fatal(err)
	}
	if state.Required != 2 || state.Current || !state.Known {
		t.Fatalf("future requirement polluted source denominator or degraded source was ignored: %#v", state)
	}
	if _, err := pool.Exec(ctx, `UPDATE evidence_sources SET health='CURRENT' WHERE id=$1::uuid`, sourceB); err != nil {
		t.Fatal(err)
	}
	state, err = repo.CurrentProgramSourceState(ctx, "state-truth-test", programID)
	if err != nil {
		t.Fatal(err)
	}
	if state.Required != 2 || !state.Current {
		t.Fatalf("all current effective sources were not recognized: %#v", state)
	}

	// Normalized projection preserves the configured period when resuming even
	// if the legacy command payload attempts to write NULL.
	if _, err := pool.Exec(ctx, `UPDATE programs SET status='ACTIVE',effective_until=NULL WHERE id=$1::uuid`, programID); err != nil {
		t.Fatal(err)
	}
	var storedEnd *time.Time
	if err := pool.QueryRow(ctx, `SELECT effective_until FROM programs WHERE id=$1::uuid`, programID).Scan(&storedEnd); err != nil {
		t.Fatal(err)
	}
	if storedEnd == nil || !storedEnd.Equal(end) {
		t.Fatalf("resume erased configured Program period: %v want %v", storedEnd, end)
	}

	if _, err := pool.Exec(ctx, `INSERT INTO evidence_contracts(id,tenant_id,program_id,requirement_id,code,name,claim,acceptable_source_ids,population_scope,freshness_minutes,minimum_coverage,contradiction_policy,failure_action,status) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,'EVIDENCE','Evidence','Evidence must remain current','[]'::jsonb,'{}'::jsonb,60,1,'REVIEW','FLAG','ACTIVE')`, contract, tenantID, programID, reqA); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO evidence_assessments(tenant_id,program_id,contract_id,conclusion,coverage,basis,assessed_at,valid_until) VALUES($1::uuid,$2::uuid,$3::uuid,'SUPPORTED',1,'{}'::jsonb,$4,$5)`, tenantID, programID, contract, now, now.Add(2*time.Hour)); err == nil {
		t.Fatal("assessment validity beyond the contract freshness boundary was accepted")
	}
	if _, err := pool.Exec(ctx, `INSERT INTO evidence_assessments(tenant_id,program_id,contract_id,conclusion,coverage,basis,assessed_at,valid_until) VALUES($1::uuid,$2::uuid,$3::uuid,'SUPPORTED',1,'{}'::jsonb,$4,NULL)`, tenantID, programID, contract, now); err != nil {
		t.Fatal(err)
	}
	var validUntil time.Time
	if err := pool.QueryRow(ctx, `SELECT valid_until FROM evidence_assessments WHERE tenant_id=$1::uuid AND program_id=$2::uuid ORDER BY created_at DESC LIMIT 1`, tenantID, programID).Scan(&validUntil); err != nil {
		t.Fatal(err)
	}
	if !validUntil.Equal(now.Add(time.Hour)) {
		t.Fatalf("server-derived assessment validity=%s want=%s", validUntil, now.Add(time.Hour))
	}
}

func TestPostgresProgramSummaryExposesProjectionFreshness(t *testing.T) {
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
		tenantID  = "87777777-7777-7777-8777-777777777771"
		programID = "87777777-7777-7777-8777-777777777772"
		snapshot  = "87777777-7777-7777-8777-777777777773"
	)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID) })
	now := time.Now().UTC().Truncate(time.Second)
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'summary-truth-test','Summary Truth Test')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO programs(id,tenant_id,code,name,program_type,status,owning_function,scope,effective_from,version) VALUES($1::uuid,$2::uuid,'SUMMARY','Summary truth','COMPLIANCE','ACTIVE','Compliance','{}'::jsonb,$3,5)`, programID, tenantID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	reasons := make([]map[string]string, 8)
	for index := range reasons {
		reasons[index] = map[string]string{"code": "R", "summary": "reason"}
	}
	rawReasons, _ := json.Marshal(reasons)
	if _, err := pool.Exec(ctx, `INSERT INTO program_state_snapshots(id,tenant_id,program_id,overall_state,dimensions,reasons,open_matter_count,generated_at,program_version,projection_version) VALUES($1::uuid,$2::uuid,$3::uuid,'AT_RISK',$4::jsonb,$5::jsonb,0,$6,4,7)`, snapshot, tenantID, programID, `{"interpretation":"CURRENT","applicability":"CURRENT","control_design":"CURRENT","implementation":"CURRENT","evidence_sufficiency":"CURRENT","operating_effectiveness":"CURRENT","exception":"AT_RISK","assurance":"CURRENT","deadline":"CURRENT","source_quality":"CURRENT"}`, string(rawReasons), now); err != nil {
		t.Fatal(err)
	}

	page, err := NewPostgresRepository(pool).ListProgramSummaries(ctx, "summary-truth-test", SummaryQuery{Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("summary count=%d", len(page.Items))
	}
	item := page.Items[0]
	if item.ProgramVersion != 5 || item.AssessedProgramVersion != 4 || item.ProjectionVersion != 7 || !item.ProjectionStale {
		t.Fatalf("projection freshness metadata is wrong: %#v", item)
	}
	if item.ReasonsTotal != 8 || item.ReasonsOmitted != 2 || len(item.Reasons) != 6 {
		t.Fatalf("reason completeness metadata is wrong: %#v", item)
	}
}
