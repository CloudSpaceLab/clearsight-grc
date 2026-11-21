//go:build postgres && postgresintegration

package continuity

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	platformid "github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresRequirementReviewIndependentDecision(t *testing.T) {
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
		v, e := platformid.NewUUIDv7()
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	tenant, entity, owner, checker := uuid(), uuid(), uuid(), uuid()
	suffix := strings.ToUpper(tenant[len(tenant)-8:])
	slug := "req-review-" + strings.ToLower(suffix)
	exec := func(q string, args ...any) {
		t.Helper()
		if _, e := pool.Exec(ctx, q, args...); e != nil {
			t.Fatal(e)
		}
	}
	exec("INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,'Requirement review test')", tenant, slug)
	exec("INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES($1::uuid,$2::uuid,$3,'Entity','NG')", entity, tenant, "RR-"+suffix)
	exec("INSERT INTO principals(id,tenant_id,kind,display_name) VALUES($1::uuid,$3::uuid,'PERSON','Owner'),($2::uuid,$3::uuid,'PERSON','Reviewer')", owner, checker, tenant)
	ctx = WithTrustedSystemEntityScope(ctx, slug, entity)
	service := NewService(NewCurrentPostgresRepository(pool))
	now := time.Now().UTC().Add(-time.Minute)
	program, err := service.CreateProgram(ctx, CreateProgramInput{TenantID: slug, LegalEntityID: entity, Code: "RR-" + suffix, Name: "Requirement review test", Type: "REGULATORY", OwningFunction: "Risk", OwnerPrincipalID: owner, AuthorityPrincipalID: checker, EffectiveFrom: now, ActorID: owner})
	if err != nil {
		t.Fatal(err)
	}
	drafted, err := service.AddRequirement(ctx, AddRequirementInput{TenantID: slug, ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, Code: "DRAFT-REQ", Title: "Review access", Statement: "Review privileged access", Status: RequirementDraft, ActorID: owner})
	if err != nil {
		t.Fatal(err)
	}
	if len(drafted.Requirements) != 1 {
		t.Fatalf("draft missing: %+v", drafted.Requirements)
	}
	review, proposed, err := service.ProposeRequirementReview(ctx, ProposeRequirementReviewInput{TenantID: slug, ProgramID: program.Program.ID, RequirementID: drafted.Requirements[0].ID, ExpectedVersion: drafted.Program.Version, ActorID: owner, Rationale: "Review this draft"})
	if err != nil {
		t.Fatal(err)
	}
	page, err := service.ListRequirementReviews(ctx, slug, program.Program.ID, "", 100)
	if err != nil || len(page) != 1 || page[0].ID != review.ID {
		t.Fatalf("review list unavailable: %+v %v", page, err)
	}
	if _, err = service.ListRequirementReviews(ctx, "other-tenant", program.Program.ID, "", 100); err == nil {
		t.Fatal("cross-tenant list was allowed")
	}
	if review.Status != "PENDING" || proposed.Requirements[0].Status != RequirementDraft {
		t.Fatalf("invalid proposal: %+v %+v", review, proposed.Requirements)
	}
	_, _, err = service.DecideRequirementReview(ctx, DecideRequirementReviewInput{TenantID: slug, ProgramID: program.Program.ID, ReviewID: review.ID, ExpectedVersion: proposed.Program.Version, ActorID: owner, Rationale: "Self approval"}, true)
	if err == nil {
		t.Fatal("owner self approval allowed")
	}
	decided, approved, err := service.DecideRequirementReview(ctx, DecideRequirementReviewInput{TenantID: slug, ProgramID: program.Program.ID, ReviewID: review.ID, ExpectedVersion: proposed.Program.Version, ActorID: checker, Rationale: "Independent review"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if decided.Status != "APPROVED" || approved.Requirements[0].Status != RequirementApproved {
		t.Fatalf("invalid approval: %+v %+v", decided, approved.Requirements)
	}
	var state string
	if err = pool.QueryRow(ctx, "SELECT status FROM program_requirements WHERE id=$1::uuid", drafted.Requirements[0].ID).Scan(&state); err != nil || state != "APPROVED" {
		t.Fatalf("projection: %s %v", state, err)
	}
	_, _, err = service.DecideRequirementReview(ctx, DecideRequirementReviewInput{TenantID: slug, ProgramID: program.Program.ID, ReviewID: review.ID, ExpectedVersion: approved.Program.Version, ActorID: checker, Rationale: "Repeat"}, true)
	if err == nil {
		t.Fatal("duplicate decision allowed")
	}
}
