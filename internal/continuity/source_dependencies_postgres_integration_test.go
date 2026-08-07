//go:build postgres && postgresintegration

package continuity

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresSourceDependenciesIncludeRequirementsAndEvidenceContracts(t *testing.T) {
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
		tenantID            = "33333333-3333-7333-8333-333333333331"
		programID           = "33333333-3333-7333-8333-333333333332"
		requirementID       = "33333333-3333-7333-8333-333333333333"
		contractID          = "33333333-3333-7333-8333-333333333334"
		requirementSourceID = "33333333-3333-7333-8333-333333333335"
		contractSourceID    = "33333333-3333-7333-8333-333333333336"
	)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID) })

	now := time.Now().UTC()
	statements := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'source-dependency-test','Source Dependency Test')`, []any{tenantID}},
		{`INSERT INTO evidence_sources(id,tenant_id,code,name,source_type,authority_class,expected_freshness_minutes,health,status)
		  VALUES($1::uuid,$3::uuid,'REQ-SOURCE','Requirement Source','REGULATORY','AUTHORITATIVE',1440,'CURRENT','ACTIVE'),
		        ($2::uuid,$3::uuid,'CONTRACT-SOURCE','Contract Source','SYSTEM','SYSTEM_OF_RECORD',60,'DEGRADED','ACTIVE')`,
			[]any{requirementSourceID, contractSourceID, tenantID}},
		{`INSERT INTO programs(id,tenant_id,code,name,program_type,status,owning_function,scope,effective_from,version)
		  VALUES($1::uuid,$2::uuid,'SOURCE-DEPS','Source Dependency Program','CONTINUOUS','ACTIVE','Compliance','{}'::jsonb,$3,1)`,
			[]any{programID, tenantID, now}},
		{`INSERT INTO program_requirements(id,tenant_id,program_id,source_id,code,title,statement,modality,status,effective_from,version)
		  VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,'REQ-1','Source-backed requirement','Maintain the governed source.','MUST','APPROVED',$5,1)`,
			[]any{requirementID, tenantID, programID, requirementSourceID, now}},
		{`INSERT INTO evidence_contracts(id,tenant_id,program_id,requirement_id,code,name,claim,acceptable_source_ids,population_scope,freshness_minutes,minimum_coverage,independence_required,contradiction_policy,failure_action,status,version)
		  VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,'EC-1','Source contract','Source remains current.','[]'::jsonb,'{}'::jsonb,60,1,false,'REVIEW','MATTER','ACTIVE',1)`,
			[]any{contractID, tenantID, programID, requirementID}},
		{`INSERT INTO evidence_contract_sources(tenant_id,program_id,contract_id,source_id)
		  VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid)`,
			[]any{tenantID, programID, contractID, contractSourceID}},
	}
	for _, statement := range statements {
		if _, err := pool.Exec(ctx, statement.query, statement.args...); err != nil {
			t.Fatal(err)
		}
	}

	repo := NewPostgresRepository(pool)
	for _, sourceID := range []string{requirementSourceID, contractSourceID} {
		programs, err := repo.ProgramIDsForEvidenceSource(ctx, "source-dependency-test", sourceID)
		if err != nil {
			t.Fatal(err)
		}
		if len(programs) != 1 || programs[0] != programID {
			t.Fatalf("source %s resolved unexpected programs: %#v", sourceID, programs)
		}
	}

	current, err := repo.EvidenceSourcesCurrentForProgram(ctx, "source-dependency-test", programID)
	if err != nil {
		t.Fatal(err)
	}
	if current {
		t.Fatal("program source health was current while one active contract source was degraded")
	}
	if _, err := pool.Exec(ctx, `UPDATE evidence_sources SET health='CURRENT' WHERE id=$1::uuid`, contractSourceID); err != nil {
		t.Fatal(err)
	}
	current, err = repo.EvidenceSourcesCurrentForProgram(ctx, "source-dependency-test", programID)
	if err != nil {
		t.Fatal(err)
	}
	if !current {
		t.Fatal("program source health did not recover after all requirement and contract sources became current")
	}
}
