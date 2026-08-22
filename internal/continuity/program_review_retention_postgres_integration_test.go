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

func TestAcceptedProgramReviewPreventsBaselineSnapshotDeletion(t *testing.T) {
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
		tenantID    = "94444444-4444-7444-8444-444444444441"
		principalID = "94444444-4444-7444-8444-444444444442"
	)
	_, _ = pool.Exec(ctx, `DELETE FROM program_review_checkpoints WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() {
		cleanup := context.Background()
		_, _ = pool.Exec(cleanup, `DELETE FROM program_review_checkpoints WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanup, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	})
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'review-retention','Review Retention')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,external_ref,display_name) VALUES($1::uuid,$2::uuid,'PERSON','review-retention-actor','Review Retention Actor')`, principalID, tenantID); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	repo := NewCurrentPostgresRepository(pool)
	service := NewServiceWithClock(repo, func() time.Time { return now })
	program, err := service.CreateProgram(ctx, CreateProgramInput{
		TenantID:       "review-retention",
		Code:           "RETENTION",
		Name:           "Review retention programme",
		Type:           "REGULATORY",
		OwningFunction: "Compliance",
		Scope:          json.RawMessage(`{}`),
		EffectiveFrom:  now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.refreshProgramCurrent(ctx, "review-retention", program.Program.ID, "TEST_SETUP", ""); err != nil {
		t.Fatal(err)
	}
	program, err = service.GetProgram(ctx, "review-retention", program.Program.ID)
	if err != nil {
		t.Fatal(err)
	}
	if program.CurrentState == nil {
		t.Fatal("expected current Program state")
	}
	digest, err := service.ProgramReviewDigest(ctx, "review-retention", program.Program.ID, principalID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.AcceptProgramReview(ctx, AcceptProgramReviewInput{
		TenantID:                  "review-retention",
		ProgramID:                 program.Program.ID,
		PrincipalID:               principalID,
		ExpectedProgramVersion:    program.Program.Version,
		ExpectedProjectionVersion: digest.CurrentProjectionVersion,
	}); err != nil {
		t.Fatal(err)
	}

	// The checkpoint stores the canonical projection version internally even
	// though the client round-trips the actor-safe semantic version.
	_, err = pool.Exec(ctx, `DELETE FROM program_state_snapshots WHERE tenant_id=$1::uuid AND program_id=$2::uuid AND projection_version=$3`, tenantID, program.Program.ID, program.CurrentState.ProjectionVersion)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23503" {
		t.Fatalf("accepted review baseline snapshot must be protected by FK, got %v", err)
	}
}
