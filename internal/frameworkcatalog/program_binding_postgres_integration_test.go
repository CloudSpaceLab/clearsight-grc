//go:build postgres && postgresintegration

package frameworkcatalog

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresProgramFrameworkAdoptionScopeAndReviewGuards(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	newID := func() string {
		t.Helper()
		value, err := id.NewUUIDv7()
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	exec := func(statement string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, statement, args...); err != nil {
			t.Fatal(err)
		}
	}
	tenant, entityA, entityB := newID(), newID(), newID()
	programA, programB := newID(), newID()
	requirementA, requirementB, draftRequirement := newID(), newID(), newID()
	maker, checker := newID(), newID()
	suffix := strings.ToUpper(tenant[len(tenant)-8:])
	now := time.Date(2026, 10, 15, 8, 0, 0, 0, time.UTC)
	start := now.Add(-time.Hour)

	exec("INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,$3)", tenant, "framework-"+strings.ToLower(suffix), "Framework fixture")
	exec(`INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES
		($1::uuid,$3::uuid,$4,'Entity A','NG'),
		($2::uuid,$3::uuid,$5,'Entity B','NG')`,
		entityA, entityB, tenant, "FWA-"+suffix, "FWB-"+suffix)
	exec(`INSERT INTO principals(id,tenant_id,kind,display_name) VALUES
		($1::uuid,$3::uuid,'PERSON','Framework proposer'),
		($2::uuid,$3::uuid,'PERSON','Independent reviewer')`,
		maker, checker, tenant)
	exec(`INSERT INTO programs(
		id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,
		scope,effective_from,created_at,updated_at,version) VALUES
		($1::uuid,$3::uuid,$4::uuid,$5,'Program A','REGULATORY','ACTIVE','Risk','{}'::jsonb,$8,$8,$8,1),
		($2::uuid,$3::uuid,$6::uuid,$7,'Program B','REGULATORY','ACTIVE','Risk','{}'::jsonb,$8,$8,$8,1)`,
		programA, programB, tenant, entityA, "FRAME-A-"+suffix, entityB, "FRAME-B-"+suffix, start)
	exec(`INSERT INTO program_requirements(
		id,tenant_id,program_id,code,title,statement,modality,status,effective_from,created_at,version) VALUES
		($1::uuid,$4::uuid,$5::uuid,'A','Access','Review access','MUST','APPROVED',$7,$7,1),
		($2::uuid,$4::uuid,$6::uuid,'B','Other','Review other','MUST','APPROVED',$7,$7,1),
		($3::uuid,$4::uuid,$5::uuid,'C','Draft','Await approval','MUST','DRAFT',$7,$7,1)`,
		requirementA, requirementB, draftRequirement, tenant, programA, programB, start)

	editionInput := validTestPublication()
	editionInput.FrameworkCode = "ADOPT-" + suffix
	edition, err := NewPostgresRepository(pool).Publish(ctx, editionInput, "independent-reference-approval")
	if err != nil {
		t.Fatal(err)
	}
	var clause string
	if err := pool.QueryRow(ctx,
		"SELECT id::text FROM framework_catalog_clauses WHERE edition_id=$1::uuid AND code='A'",
		edition.ID).Scan(&clause); err != nil {
		t.Fatal(err)
	}
	var adoption string
	err = pool.QueryRow(ctx, `INSERT INTO program_framework_adoptions(
		tenant_id,legal_entity_id,program_id,edition_id,publisher_code,framework_code,edition_code,
		status,proposed_by,rationale,proposed_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,'PENDING',$8::uuid,$9,$10)
		RETURNING id::text`,
		tenant, entityA, programA, edition.ID,
		edition.PublisherCode, edition.FrameworkCode, edition.EditionCode,
		maker, "Review the published edition", now).Scan(&adoption)
	if err != nil {
		t.Fatal(err)
	}

	// A caller cannot self-approve or claim ACTIVE without reviewer facts.
	if _, err := pool.Exec(ctx, `UPDATE program_framework_adoptions
		SET status='ACTIVE',reviewed_by=$2::uuid,reviewed_at=$3,effective_from=$3
		WHERE id=$1::uuid`, adoption, maker, now); err == nil {
		t.Fatal("maker approved own Program adoption")
	}
	if _, err := pool.Exec(ctx, `UPDATE program_framework_adoptions
		SET status='ACTIVE' WHERE id=$1::uuid`, adoption); err == nil {
		t.Fatal("active adoption without independent review accepted")
	}

	// Even an explicitly provided tenant cannot change the legal-entity scope.
	_, err = pool.Exec(ctx, `INSERT INTO program_framework_adoptions(
		tenant_id,legal_entity_id,program_id,edition_id,publisher_code,framework_code,edition_code,
		status,proposed_by,rationale,proposed_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,'PENDING',$8::uuid,'wrong entity',$9)`,
		tenant, entityB, programA, edition.ID,
		edition.PublisherCode, edition.FrameworkCode, edition.EditionCode, maker, now)
	if err == nil {
		t.Fatal("cross-entity Program adoption accepted")
	}
	exec(`UPDATE program_framework_adoptions
		SET status='ACTIVE',reviewed_by=$2::uuid,reviewed_at=$3,effective_from=$3
		WHERE id=$1::uuid`, adoption, checker, now)

	insertMapping := func(requirement, clauseID, adoptionID string) error {
		_, err := pool.Exec(ctx, `INSERT INTO framework_clause_requirement_links(
			tenant_id,legal_entity_id,program_id,adoption_id,edition_id,clause_id,
			requirement_id,mapping_kind,rationale,mapped_by,reviewed_by,reviewed_at,effective_from)
			VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6::uuid,
			       $7::uuid,'EXACT','Reviewed against source',$8::uuid,$9::uuid,$10,$10)`,
			tenant, entityA, programA, adoptionID, edition.ID, clauseID,
			requirement, maker, checker, now)
		return err
	}
	if err := insertMapping(requirementA, clause, adoption); err != nil {
		t.Fatal(err)
	}
	// Editing the mapping to claim self-review remains prohibited at the
	// persistence boundary, independently of future HTTP authority checks.
	if _, err := pool.Exec(ctx, `INSERT INTO framework_clause_requirement_links(
		tenant_id,legal_entity_id,program_id,adoption_id,edition_id,clause_id,
		requirement_id,mapping_kind,rationale,mapped_by,reviewed_by,reviewed_at,effective_from)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6::uuid,
		       $7::uuid,'EXACT','Self-approved mapping',$8::uuid,$8::uuid,$9,$9)`,
		tenant, entityA, programA, adoption, edition.ID, clause, draftRequirement, maker, now); err == nil {
		t.Fatal("mapping maker self-approval accepted")
	}
	if err := insertMapping(requirementA, clause, adoption); err == nil {
		t.Fatal("duplicate active clause mapping accepted")
	}
	if err := insertMapping(requirementB, clause, adoption); err == nil {
		t.Fatal("cross-Program Requirement mapping accepted")
	}
	if err := insertMapping(draftRequirement, clause, adoption); err == nil {
		t.Fatal("draft Requirement treated as approved")
	}

	// A second published edition's clause cannot be attributed to this edition.
	otherInput := validTestPublication()
	otherInput.FrameworkCode = "OTHER-" + suffix
	other, err := NewPostgresRepository(pool).Publish(ctx, otherInput, "second-reviewed-reference")
	if err != nil {
		t.Fatal(err)
	}
	var otherClause string
	if err := pool.QueryRow(ctx, "SELECT id::text FROM framework_catalog_clauses WHERE edition_id=$1::uuid AND code='A'", other.ID).Scan(&otherClause); err != nil {
		t.Fatal(err)
	}
	if err := insertMapping(requirementA, otherClause, adoption); err == nil {
		t.Fatal("wrong edition clause mapped into an adopted standard")
	}

	// The currently active published standard cannot be activated twice.
	_, err = pool.Exec(ctx, `INSERT INTO program_framework_adoptions(
		tenant_id,legal_entity_id,program_id,edition_id,publisher_code,framework_code,edition_code,
		status,proposed_by,reviewed_by,rationale,proposed_at,reviewed_at,effective_from)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,
		'ACTIVE',$8::uuid,$9::uuid,'duplicate active',$10,$10,$10)`,
		tenant, entityA, programA, edition.ID,
		edition.PublisherCode, edition.FrameworkCode, edition.EditionCode, maker, checker, now)
	if err == nil {
		t.Fatal("duplicate active edition accepted")
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM framework_clause_requirement_links
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND program_id=$3::uuid
		  AND retired_at IS NULL`, tenant, entityA, programA).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected one governed mapping, got %d", count)
	}
}
