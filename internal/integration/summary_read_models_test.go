//go:build postgres && postgresintegration

package integration_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestOperationalSummaryReadModels(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err = pool.Exec(ctx, `TRUNCATE tenants CASCADE`); err != nil {
		t.Fatal(err)
	}
	const tenantID = "55555555-5555-7555-8555-555555555551"
	if _, err = pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'summary-bank','Summary Bank')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `
		INSERT INTO programs(id,tenant_id,code,name,program_type,status,owning_function,jurisdiction,scope,effective_from,created_at,updated_at,version)
		SELECT uuidv7(),$1::uuid,format('PRG-%s',lpad(g::text,4,'0')),format('Operational program %s',g),'COMPLIANCE',
			CASE WHEN g % 9 = 0 THEN 'PAUSED' ELSE 'ACTIVE' END,'Compliance','Nigeria','{}'::jsonb,
			clock_timestamp()-interval '1 year',clock_timestamp()-g*interval '1 minute',clock_timestamp()-g*interval '1 minute',1
		FROM generate_series(1,250) g`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `
		INSERT INTO program_state_snapshots(id,tenant_id,program_id,overall_state,dimensions,reasons,open_matter_count,generated_at,program_version)
		SELECT uuidv7(),tenant_id,id,CASE WHEN code='PRG-0019' THEN 'EVIDENCE_INSUFFICIENT' ELSE 'CURRENT' END,
			'{}'::jsonb,CASE WHEN code='PRG-0019' THEN '[{"code":"EVIDENCE_NOT_ASSESSED","summary":"Evidence has not been assessed."}]'::jsonb ELSE '[]'::jsonb END,
			0,updated_at,version
		FROM programs WHERE tenant_id=$1::uuid`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `
		INSERT INTO matters(id,tenant_id,reference,matter_type,status,priority,title,summary,scope,known_facts,missing_facts,contradictions,closed_at,closure_reason,created_at,updated_at,version)
		SELECT uuidv7(),$1::uuid,format('MAT-%s',lpad(g::text,4,'0')),'CONTROL_GAP',
			CASE WHEN g % 11 = 0 THEN 'CLOSED' ELSE 'ASSESSMENT' END,(g % 5)+1,
			format('Operational issue %s',g),'A bounded issue used to test the operational list read.','{}'::jsonb,'{}'::jsonb,'[]'::jsonb,'[]'::jsonb,
			CASE WHEN g % 11 = 0 THEN clock_timestamp()-g*interval '1 minute' ELSE NULL END,
			CASE WHEN g % 11 = 0 THEN 'Test closure' ELSE '' END,
			clock_timestamp()-g*interval '1 minute',clock_timestamp()-g*interval '1 minute',1
		FROM generate_series(1,300) g`, tenantID); err != nil {
		t.Fatal(err)
	}

	service := continuity.NewService(continuity.NewPostgresRepository(pool))
	programs, err := service.ListProgramSummaries(ctx, "summary-bank", continuity.SummaryQuery{Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(programs.Items) != 50 || programs.NextCursor == "" {
		t.Fatalf("unexpected program page size=%d cursor=%q", len(programs.Items), programs.NextCursor)
	}
	second, err := service.ListProgramSummaries(ctx, "summary-bank", continuity.SummaryQuery{Limit: 50, Cursor: programs.NextCursor})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, item := range programs.Items {
		seen[item.Program.ID] = true
	}
	for _, item := range second.Items {
		if seen[item.Program.ID] {
			t.Fatalf("program %s appeared on two pages", item.Program.ID)
		}
	}
	search, err := service.ListProgramSummaries(ctx, "summary-bank", continuity.SummaryQuery{Limit: 20, Search: "operational program 19"})
	if err != nil {
		t.Fatal(err)
	}
	if len(search.Items) != 1 || search.Items[0].Program.Code != "PRG-0019" {
		t.Fatalf("unexpected program search result: %#v", search.Items)
	}

	matters, err := service.ListMatterSummaries(ctx, "summary-bank", continuity.SummaryQuery{Limit: 50, Status: "OPEN"})
	if err != nil {
		t.Fatal(err)
	}
	if len(matters.Items) != 50 || matters.NextCursor == "" {
		t.Fatalf("unexpected matter page size=%d cursor=%q", len(matters.Items), matters.NextCursor)
	}
	matterSearch, err := service.ListMatterSummaries(ctx, "summary-bank", continuity.SummaryQuery{Limit: 20, Search: "operational issue 23"})
	if err != nil {
		t.Fatal(err)
	}
	if len(matterSearch.Items) != 1 || matterSearch.Items[0].Matter.Reference != "MAT-0023" {
		t.Fatalf("unexpected matter search result: %s", fmt.Sprint(matterSearch.Items))
	}
}
