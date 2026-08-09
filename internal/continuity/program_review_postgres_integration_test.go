//go:build postgres && postgresintegration

package continuity

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestProgramReviewPostgresKeepsActorTenantAndVersionTruth(t *testing.T) {
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
		tenantA    = "91111111-1111-7111-8111-111111111111"
		tenantB    = "92222222-2222-7222-8222-222222222222"
		principalA = "91111111-1111-7111-8111-111111111112"
		principalB = "92222222-2222-7222-8222-222222222223"
	)
	for _, tenantID := range []string{tenantA, tenantB} {
		_, _ = pool.Exec(ctx, `DELETE FROM program_review_checkpoints WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	}
	t.Cleanup(func() {
		cleanup := context.Background()
		for _, tenantID := range []string{tenantA, tenantB} {
			_, _ = pool.Exec(cleanup, `DELETE FROM program_review_checkpoints WHERE tenant_id=$1::uuid`, tenantID)
			_, _ = pool.Exec(cleanup, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
		}
	})
	if _, err := pool.Exec(ctx, `
		INSERT INTO tenants(id,slug,name) VALUES
		($1::uuid,'program-review-a','Program Review A'),
		($2::uuid,'program-review-b','Program Review B')`, tenantA, tenantB); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO principals(id,tenant_id,kind,external_ref,display_name) VALUES
		($1::uuid,$3::uuid,'PERSON','reviewer-a','Reviewer A'),
		($2::uuid,$4::uuid,'PERSON','reviewer-b','Reviewer B')`, principalA, principalB, tenantA, tenantB); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	repo := NewCurrentPostgresRepository(pool)
	service := NewServiceWithClock(repo, func() time.Time { return now })
	program, err := service.CreateProgram(ctx, CreateProgramInput{
		TenantID:       "program-review-a",
		Code:           "PRIVACY",
		Name:           "Privacy Programme",
		Type:           "PRIVACY",
		OwningFunction: "Privacy",
		Scope:          json.RawMessage(`{}`),
		EffectiveFrom:  now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.refreshProgramCurrent(ctx, "program-review-a", program.Program.ID, "TEST_SETUP", ""); err != nil {
		t.Fatal(err)
	}
	program, err = service.GetProgram(ctx, "program-review-a", program.Program.ID)
	if err != nil {
		t.Fatal(err)
	}
	if program.CurrentState == nil || program.CurrentState.ProjectionVersion < 1 {
		t.Fatalf("expected worker-projected current state before accepting review: %#v", program.CurrentState)
	}

	accepted, err := service.AcceptProgramReview(ctx, AcceptProgramReviewInput{
		TenantID:                  "program-review-a",
		ProgramID:                 program.Program.ID,
		PrincipalID:               principalA,
		ExpectedProgramVersion:    program.Program.Version,
		ExpectedProjectionVersion: program.CurrentState.ProjectionVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	if accepted.State != "CURRENT" || accepted.Checkpoint == nil {
		t.Fatalf("unexpected accepted digest: %#v", accepted)
	}

	// Re-accepting the exact same canonical versions is idempotent rather than
	// manufacturing duplicate acknowledgement history.
	again, err := service.AcceptProgramReview(ctx, AcceptProgramReviewInput{
		TenantID:                  "program-review-a",
		ProgramID:                 program.Program.ID,
		PrincipalID:               principalA,
		ExpectedProgramVersion:    program.Program.Version,
		ExpectedProjectionVersion: program.CurrentState.ProjectionVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	if again.Checkpoint == nil || again.Checkpoint.ID != accepted.Checkpoint.ID {
		t.Fatalf("same canonical baseline created duplicate checkpoint: first=%#v second=%#v", accepted.Checkpoint, again.Checkpoint)
	}
	var checkpointCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM program_review_checkpoints WHERE tenant_id=$1::uuid AND program_id=$2::uuid AND principal_id=$3::uuid`, tenantA, program.Program.ID, principalA).Scan(&checkpointCount); err != nil {
		t.Fatal(err)
	}
	if checkpointCount != 1 {
		t.Fatalf("expected one idempotent checkpoint, got %d", checkpointCount)
	}

	// Tenant binding is database-enforced even if a caller tried to pair a
	// Program from one bank with an actor from another.
	_, err = pool.Exec(ctx, `
		INSERT INTO program_review_checkpoints(tenant_id,program_id,principal_id,program_version,projection_version)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5)`, tenantB, program.Program.ID, principalB, program.Program.Version, program.CurrentState.ProjectionVersion)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23503" {
		t.Fatalf("cross-tenant review checkpoint must fail its tenant FK, got %v", err)
	}

	now = now.Add(time.Minute)
	updated, err := service.AddRequirement(ctx, AddRequirementInput{
		TenantID:        "program-review-a",
		ProgramID:       program.Program.ID,
		ExpectedVersion: program.Program.Version,
		Code:            "RETENTION",
		Title:           "Maintain approved retention rules",
		Statement:       "Retention rules must remain approved and current.",
		Modality:        "MUST",
		Status:          RequirementApproved,
		EffectiveFrom:   now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.refreshProgramCurrent(ctx, "program-review-a", updated.Program.ID, "TEST_REQUIREMENT_CHANGE", ""); err != nil {
		t.Fatal(err)
	}
	updated, err = service.GetProgram(ctx, "program-review-a", updated.Program.ID)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := service.ProgramReviewDigest(ctx, "program-review-a", program.Program.ID, principalA)
	if err != nil {
		t.Fatal(err)
	}
	if !changed.ReviewRequired || changed.State != "CHANGED" || changed.ChangesTotal == 0 {
		t.Fatalf("canonical Program change was not surfaced: %#v", changed)
	}
	if _, err := service.AcceptProgramReview(ctx, AcceptProgramReviewInput{
		TenantID:                  "program-review-a",
		ProgramID:                 program.Program.ID,
		PrincipalID:               principalA,
		ExpectedProgramVersion:    program.Program.Version,
		ExpectedProjectionVersion: program.CurrentState.ProjectionVersion,
	}); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale expected versions should fail closed, got %v", err)
	}
	if updated.CurrentState == nil || updated.CurrentState.ProgramVersion != updated.Program.Version {
		t.Fatalf("updated Program was not projected to current state: %#v", updated.CurrentState)
	}
}
