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
	entityA := seedPostgresTestLegalEntity(t, ctx, pool, tenantA, "ENTITY-A")
	_ = seedPostgresTestLegalEntity(t, ctx, pool, tenantB, "ENTITY-B")
	ctx = WithTrustedSystemEntityScope(ctx, "program-review-a", entityA)
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
		LegalEntityID:  entityA,
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
	firstDigest, err := service.ProgramReviewDigest(ctx, "program-review-a", program.Program.ID, principalA)
	if err != nil {
		t.Fatal(err)
	}
	// A blocked checkpoint must release its transaction rather than indefinitely
	// holding future session and authority locks.
	stalled, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = stalled.Exec(ctx, `SELECT id FROM programs WHERE id=$1::uuid FOR UPDATE`, program.Program.ID); err != nil {
		t.Fatal(err)
	}
	stalledResult := make(chan error, 1)
	go func() {
		_, e := service.AcceptProgramReview(ctx, AcceptProgramReviewInput{TenantID: "program-review-a", ProgramID: program.Program.ID, PrincipalID: principalA, ExpectedProgramVersion: program.Program.Version, ExpectedProjectionVersion: firstDigest.CurrentProjectionVersion})
		stalledResult <- e
	}()
	select {
	case e := <-stalledResult:
		if !errors.Is(e, context.DeadlineExceeded) {
			t.Fatalf("blocked checkpoint: %v", e)
		}
	case <-time.After(7 * time.Second):
		_ = stalled.Rollback(ctx)
		t.Fatal("blocked checkpoint did not time out")
	}
	if err = stalled.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	var partialCheckpoints, partialEvents, partialOutbox int
	if err = pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM program_review_checkpoints WHERE tenant_id=$1::uuid), (SELECT count(*) FROM continuity_events WHERE tenant_id=$1::uuid AND event_type=$2), (SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND event_type=$2)`, tenantA, EventProgramReviewAccepted).Scan(&partialCheckpoints, &partialEvents, &partialOutbox); err != nil || partialCheckpoints != 0 || partialEvents != 0 || partialOutbox != 0 {
		t.Fatalf("timed-out checkpoint=%d event=%d outbox=%d err=%v", partialCheckpoints, partialEvents, partialOutbox, err)
	}

	blocker, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = blocker.Exec(ctx, `SELECT id FROM programs WHERE id=$1::uuid FOR UPDATE`, program.Program.ID); err != nil {
		t.Fatal(err)
	}
	type reviewResult struct {
		digest ProgramReviewDigest
		err    error
	}
	result := make(chan reviewResult, 1)
	go func() {
		digest, e := service.AcceptProgramReview(ctx, AcceptProgramReviewInput{
			TenantID:                  "program-review-a",
			ProgramID:                 program.Program.ID,
			PrincipalID:               principalA,
			ExpectedProgramVersion:    program.Program.Version,
			ExpectedProjectionVersion: firstDigest.CurrentProjectionVersion,
		})
		result <- reviewResult{digest, e}
	}()
	// The receipt time must describe the database write after the Program lock,
	// not the earlier service-clock value captured before waiting.
	waitUntil := time.Now().Add(3 * time.Second)
	for {
		var waiting bool
		if err = pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND query LIKE '%FOR UPDATE OF p%')`).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		if time.Now().After(waitUntil) {
			_ = blocker.Rollback(ctx)
			t.Fatal("review did not reach the Program lock")
		}
		time.Sleep(10 * time.Millisecond)
	}
	var beforeRelease time.Time
	if err = pool.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&beforeRelease); err != nil {
		t.Fatal(err)
	}
	if err = blocker.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	got := <-result
	accepted, err := got.digest, got.err
	if err != nil {
		t.Fatal(err)
	}
	if accepted.State != "CURRENT" || accepted.Checkpoint == nil {
		t.Fatalf("unexpected accepted digest: %#v", accepted)
	}
	if accepted.Checkpoint.AcceptedAt.Before(beforeRelease) {
		t.Fatalf("checkpoint predates write boundary: %s < %s", accepted.Checkpoint.AcceptedAt, beforeRelease)
	}

	// Re-accepting the exact same actor-visible versions is idempotent rather
	// than manufacturing duplicate acknowledgement history.
	again, err := service.AcceptProgramReview(ctx, AcceptProgramReviewInput{
		TenantID:                  "program-review-a",
		ProgramID:                 program.Program.ID,
		PrincipalID:               principalA,
		ExpectedProgramVersion:    program.Program.Version,
		ExpectedProjectionVersion: accepted.CurrentProjectionVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	if again.Checkpoint == nil || again.Checkpoint.ID != accepted.Checkpoint.ID {
		t.Fatalf("same actor-visible baseline created duplicate checkpoint: first=%#v second=%#v", accepted.Checkpoint, again.Checkpoint)
	}
	var checkpointCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM program_review_checkpoints WHERE tenant_id=$1::uuid AND program_id=$2::uuid AND principal_id=$3::uuid`, tenantA, program.Program.ID, principalA).Scan(&checkpointCount); err != nil {
		t.Fatal(err)
	}
	if checkpointCount != 1 {
		t.Fatalf("expected one idempotent checkpoint, got %d", checkpointCount)
	}
	var reviewEventCount, reviewOutboxCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM continuity_events WHERE aggregate_type='PROGRAM_REVIEW' AND aggregate_id=$1::uuid AND event_type=$2`, accepted.Checkpoint.ID, EventProgramReviewAccepted).Scan(&reviewEventCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE aggregate_type='PROGRAM_REVIEW' AND aggregate_id=$1::uuid AND event_type=$2`, accepted.Checkpoint.ID, EventProgramReviewAccepted).Scan(&reviewOutboxCount); err != nil {
		t.Fatal(err)
	}
	if reviewEventCount != 1 || reviewOutboxCount != 1 {
		t.Fatalf("review checkpoint event/outbox counts = %d/%d, want 1/1", reviewEventCount, reviewOutboxCount)
	}
	var eventAt, outboxAt time.Time
	var eventPayloadAt, outboxPayloadAt time.Time
	if err := pool.QueryRow(ctx, `SELECT e.occurred_at, o.occurred_at, (e.payload->>'accepted_at')::timestamptz, (o.payload->>'accepted_at')::timestamptz FROM continuity_events e JOIN outbox_events o ON o.aggregate_id=e.aggregate_id AND o.event_type=e.event_type WHERE e.aggregate_type='PROGRAM_REVIEW' AND e.aggregate_id=$1::uuid`, accepted.Checkpoint.ID).Scan(&eventAt, &outboxAt, &eventPayloadAt, &outboxPayloadAt); err != nil {
		t.Fatal(err)
	}
	for _, recordedAt := range []time.Time{eventAt, outboxAt, eventPayloadAt, outboxPayloadAt} {
		if !recordedAt.Equal(accepted.Checkpoint.AcceptedAt) {
			t.Fatalf("checkpoint and audit timestamp differ: %s != %s", recordedAt, accepted.Checkpoint.AcceptedAt)
		}
	}

	// Tenant binding is database-enforced even if a caller tried to pair a
	// Program from one bank with an actor from another. Stored checkpoints keep
	// canonical projection versions for exact historical reconstruction.
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
		ExpectedProjectionVersion: accepted.CurrentProjectionVersion,
	}); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale expected versions should fail closed, got %v", err)
	}
	if updated.CurrentState == nil || updated.CurrentState.ProgramVersion != updated.Program.Version {
		t.Fatalf("updated Program was not projected to current state: %#v", updated.CurrentState)
	}
}
