//go:build postgres && postgresintegration

package continuity

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresFrameworkAdoptionEventAndCurrentProjection(t *testing.T) {
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
	uuid := func() string {
		t.Helper()
		v, err := id.NewUUIDv7()
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	tenant, entity, owner, checker, editionID := uuid(), uuid(), uuid(), uuid(), uuid()
	suffix := strings.ToUpper(tenant[len(tenant)-8:])
	slug := "framework-event-" + strings.ToLower(suffix)
	now := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, q, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec("INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,$3)", tenant, slug, "Framework event fixture")
	exec("INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES($1::uuid,$2::uuid,$3,'Entity','NG')", entity, tenant, "FWE-"+suffix)
	exec(`INSERT INTO principals(id,tenant_id,kind,display_name) VALUES
		($1::uuid,$3::uuid,'PERSON','Program owner'),
		($2::uuid,$3::uuid,'PERSON','Independent authority')`, owner, checker, tenant)
	release := fmt.Sprintf("%x", sha256.Sum256([]byte(editionID)))
	source := strings.Repeat("a", 64)
	code := "FWE-" + suffix
	exec(`INSERT INTO framework_catalog_editions(
		id,publisher_code,framework_code,edition_code,title,jurisdiction,source_url,
		source_sha256,source_published_on,license_basis,release_sha256,approval_reference)
		VALUES($1::uuid,'EXAMPLE',$2,'2026','Event fixture','GLOBAL',
		'https://example.org/framework-test',$3,'2026-10-01','CITATIONS_ONLY',$4,'reviewed-fixture')`,
		editionID, code, source, release)
	ctx = WithTrustedSystemEntityScope(ctx, slug, entity)
	service := NewService(NewCurrentPostgresRepository(pool))
	service.now = func() time.Time { return now }
	program, err := service.CreateProgram(ctx, CreateProgramInput{
		TenantID: slug, LegalEntityID: entity, Code: "FWE-PROGRAM-" + suffix,
		Name: "Security Program", Type: "REGULATORY", OwningFunction: "Risk",
		OwnerPrincipalID: owner, AuthorityPrincipalID: checker,
		EffectiveFrom: now, ActorID: owner,
	})
	if err != nil {
		t.Fatal(err)
	}
	service.ConfigureFrameworkEditionLookup(fixtureEditionLookup{value: FrameworkEditionReference{
		ID: editionID, PublisherCode: "EXAMPLE", FrameworkCode: code,
		EditionCode: "2026", ReleaseSHA256: release,
	}})
	service.now = func() time.Time { return now.Add(time.Minute) }
	proposed, err := service.ProposeFrameworkAdoption(ctx, ProposeFrameworkAdoptionInput{
		TenantID: slug, ProgramID: program.Program.ID, EditionID: editionID,
		ExpectedVersion: program.Program.Version, ActorID: owner,
		Rationale: "Review published edition and its requirements.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(proposed.FrameworkAdoptions) != 1 || proposed.FrameworkAdoptions[0].Status != FrameworkAdoptionPending {
		t.Fatalf("current Program lost pending adoption: %+v", proposed.FrameworkAdoptions)
	}
	adoptionID := proposed.FrameworkAdoptions[0].ID
	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM program_framework_adoptions
		WHERE id=$1::uuid AND tenant_id=$2::uuid AND legal_entity_id=$3::uuid`,
		adoptionID, tenant, entity).Scan(&status); err != nil || status != "PENDING" {
		t.Fatalf("uncommitted or incorrectly scoped adoption: %q %v", status, err)
	}

	service.now = func() time.Time { return now.Add(2 * time.Minute) }
	approved, err := service.ApproveFrameworkAdoption(ctx, DecideFrameworkAdoptionInput{
		TenantID: slug, ProgramID: program.Program.ID, AdoptionID: adoptionID,
		ExpectedVersion: proposed.Program.Version, ActorID: checker,
		Rationale: "Approved after independent review.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(approved.FrameworkAdoptions) != 1 || approved.FrameworkAdoptions[0].Status != FrameworkAdoptionActive ||
		approved.FrameworkAdoptions[0].ReviewedBy != checker || approved.FrameworkAdoptions[0].Version != 2 {
		t.Fatalf("current Program and SQL projection diverged: %+v", approved.FrameworkAdoptions)
	}
	var activeVersion int64
	if err := pool.QueryRow(ctx, `SELECT status,version FROM program_framework_adoptions
		WHERE id=$1::uuid`, adoptionID).Scan(&status, &activeVersion); err != nil ||
		status != "ACTIVE" || activeVersion != 2 {
		t.Fatalf("not committed with Program event: %s v%d %v", status, activeVersion, err)
	}
	history, err := service.ProgramAt(ctx, slug, program.Program.ID, now.Add(time.Minute))
	if err != nil || len(history.FrameworkAdoptions) != 1 ||
		history.FrameworkAdoptions[0].Status != FrameworkAdoptionPending {
		t.Fatalf("historical proposal was overwritten: %+v %v", history, err)
	}
	events, err := service.repo.ProgramEvents(ctx, slug, program.Program.ID, nil)
	if err != nil || len(events) < 3 ||
		events[len(events)-1].Type != EventProgramFrameworkAdoptionApproved {
		t.Fatalf("no durable approval event: %+v %v", events, err)
	}
	if len(approved.Requirements) > 0 || approved.Program.CalculationPolicy != nil {
		t.Fatal("edition approval manufactured obligations or a score")
	}
}
