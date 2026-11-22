//go:build postgres && postgresintegration

package continuity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/federation"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5"
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
	if _, err = pool.Exec(ctx, `INSERT INTO responsibility_assignments(tenant_id,legal_entity_id,principal_id,responsibility,object_type,object_id,scope,priority,valid_from,policy_version) VALUES($1::uuid,$2::uuid,$3::uuid,'REVIEWER','PROGRAM',$4::uuid,jsonb_build_object('decision_type','program.review.accept'),10,$5,'review:v1')`, tenantA, entityA, principalA, program.Program.ID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	ctx = checkpointOIDCContext(t, pool, "program-review-a", entityA, principalA, time.Now().Add(time.Hour))
	const restrictedMatter = "91111111-1111-7111-8111-111111111113"
	const restrictedLink = "91111111-1111-7111-8111-111111111114"
	if _, err = pool.Exec(ctx, `INSERT INTO matters(id,tenant_id,legal_entity_id,reference,matter_type,status,priority,title,summary,scope,known_facts,missing_facts,contradictions,created_at,updated_at,version) VALUES($1::uuid,$2::uuid,$3::uuid,'REVIEW-RACE','AUTHORITY_REQUEST','ASSESSMENT',4,'Review race','Review race',jsonb_build_object('access','RESTRICTED','allowed_principal_ids',jsonb_build_array($4::text)),'{}','[]','[]',$5,$5,1); INSERT INTO matter_links(id,tenant_id,matter_id,program_id,relationship,created_at) VALUES($6::uuid,$2::uuid,$1::uuid,$7::uuid,'AFFECTS',$5)`, pgx.QueryExecModeSimpleProtocol, restrictedMatter, tenantA, entityA, principalB, now, restrictedLink, program.Program.ID); err != nil {
		t.Fatal(err)
	}
	firstDigest, err := service.ProgramReviewDigest(ctx, "program-review-a", program.Program.ID, principalA)
	if err != nil {
		t.Fatal(err)
	}
	forgedCheckpoint := ProgramReviewCheckpoint{ID: "91111111-1111-7111-8111-111111111118", TenantID: "program-review-a", ProgramID: program.Program.ID, PrincipalID: principalA, ProgramVersion: program.Program.Version, ProjectionVersion: program.CurrentState.ProjectionVersion, visibleReceiptPresent: true, visibleProjectionVersion: firstDigest.CurrentProjectionVersion}
	forgedEvent, err := newEvent(forgedCheckpoint.TenantID, "PROGRAM_REVIEW", forgedCheckpoint.ID, 1, EventProgramReviewAccepted, programReviewEventValue(forgedCheckpoint), ActorPerson, principalB, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.RecordProgramReview(ctx, forgedCheckpoint, forgedEvent); !errors.Is(err, ErrNotFound) {
		t.Fatalf("forged event actor accepted: %v", err)
	}
	orderedSessionAction := func(t *testing.T, principal string, projection int64, action string) ProgramReviewDigest {
		t.Helper()
		fixture := newCheckpointOIDCFixture(t, pool, "program-review-a", entityA, principal, time.Now().Add(time.Hour))
		actor, _ := identity.FromContext(fixture.ctx)
		proof, e := federation.VerifiedOIDCSession(fixture.ctx, actor)
		if e != nil {
			t.Fatal(e)
		}
		block, e := pool.Begin(ctx)
		if e != nil {
			t.Fatal(e)
		}
		defer block.Rollback(context.Background())
		if _, e = block.Exec(ctx, `SELECT id FROM programs WHERE id=$1::uuid FOR UPDATE`, program.Program.ID); e != nil {
			t.Fatal(e)
		}
		type outcome struct {
			digest ProgramReviewDigest
			err    error
		}
		done := make(chan outcome, 1)
		input := AcceptProgramReviewInput{TenantID: "program-review-a", ProgramID: program.Program.ID, PrincipalID: principal, ExpectedProgramVersion: program.Program.Version, ExpectedProjectionVersion: projection}
		go func() { digest, err := service.AcceptProgramReview(fixture.ctx, input); done <- outcome{digest, err} }()
		deadline := time.Now().Add(3 * time.Second)
		for {
			short, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
			_, e = pool.Exec(short, `UPDATE web_sessions SET expiry=expiry WHERE token=$1`, proof.TokenHash)
			cancel()
			if errors.Is(e, context.DeadlineExceeded) {
				break
			}
			if e != nil || time.Now().After(deadline) {
				t.Fatalf("checkpoint did not hold session lock: %v", e)
			}
			time.Sleep(10 * time.Millisecond)
		}
		actionDone := make(chan error, 1)
		go func() {
			if action == "rotation" {
				_, err := fixture.service.SwitchScope(fixture.ctx, actor, entityA)
				actionDone <- err
				return
			}
			w := httptest.NewRecorder()
			fixture.service.Logout(w, httptest.NewRequest(http.MethodPost, "/logout", nil).WithContext(fixture.ctx))
			if w.Code != http.StatusNoContent {
				actionDone <- fmt.Errorf("logout: %d", w.Code)
				return
			}
			actionDone <- nil
		}()
		select {
		case e = <-actionDone:
			t.Fatalf("%s passed the session lock: %v", action, e)
		case <-time.After(150 * time.Millisecond):
		}
		if e = block.Rollback(ctx); e != nil {
			t.Fatal(e)
		}
		got := <-done
		if got.err != nil {
			t.Fatalf("checkpoint: %v", got.err)
		}
		if e = <-actionDone; e != nil {
			t.Fatalf("%s: %v", action, e)
		}
		if _, e = service.AcceptProgramReview(fixture.ctx, input); !errors.Is(e, ErrNotFound) {
			t.Fatalf("%s allowed old session: %v", action, e)
		}
		return got.digest
	}
	for _, action := range []string{"logout", "rotation"} {
		t.Run("prior "+action, func(t *testing.T) {
			fixture := newCheckpointOIDCFixture(t, pool, "program-review-a", entityA, principalA, time.Now().Add(time.Hour))
			actor, _ := identity.FromContext(fixture.ctx)
			if action == "logout" {
				w := httptest.NewRecorder()
				fixture.service.Logout(w, httptest.NewRequest(http.MethodPost, "/logout", nil).WithContext(fixture.ctx))
				if w.Code != http.StatusNoContent {
					t.Fatalf("logout: %d", w.Code)
				}
			} else if _, e := fixture.service.SwitchScope(fixture.ctx, actor, entityA); e != nil {
				t.Fatal(e)
			}
			_, e := service.AcceptProgramReview(fixture.ctx, AcceptProgramReviewInput{TenantID: "program-review-a", ProgramID: program.Program.ID, PrincipalID: principalA, ExpectedProgramVersion: program.Program.Version, ExpectedProjectionVersion: firstDigest.CurrentProjectionVersion})
			if !errors.Is(e, ErrNotFound) {
				t.Fatalf("%s session accepted: %v", action, e)
			}
		})
	}
	invalidSession := newCheckpointOIDCFixture(t, pool, "program-review-a", entityA, principalA, time.Now().Add(time.Hour))
	invalidActor, _ := identity.FromContext(invalidSession.ctx)
	for _, mutate := range []func(*identity.Actor){func(a *identity.Actor) { a.AuthenticationMethod = "SSO" }, func(a *identity.Actor) { a.PrincipalID = principalB }, func(a *identity.Actor) { a.LegalEntityID = "missing-entity" }} {
		altered := invalidActor
		mutate(&altered)
		_, e := service.AcceptProgramReview(identity.WithActor(invalidSession.ctx, altered), AcceptProgramReviewInput{TenantID: "program-review-a", ProgramID: program.Program.ID, PrincipalID: principalA, ExpectedProgramVersion: program.Program.Version, ExpectedProjectionVersion: firstDigest.CurrentProjectionVersion})
		if e == nil {
			t.Fatal("mismatched session actor accepted")
		}
	}
	invalidProof, err := federation.VerifiedOIDCSession(invalidSession.ctx, invalidActor)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE web_sessions SET expiry=clock_timestamp()-interval '1 second' WHERE token=$1`, invalidProof.TokenHash); err != nil {
		t.Fatal(err)
	}
	if _, err = service.AcceptProgramReview(invalidSession.ctx, AcceptProgramReviewInput{TenantID: "program-review-a", ProgramID: program.Program.ID, PrincipalID: principalA, ExpectedProgramVersion: program.Program.Version, ExpectedProjectionVersion: firstDigest.CurrentProjectionVersion}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expired session accepted: %v", err)
	}
	routeBlock, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = routeBlock.Exec(ctx, `SELECT id FROM programs WHERE id=$1::uuid FOR UPDATE`, program.Program.ID); err != nil {
		t.Fatal(err)
	}
	routeResult := make(chan error, 1)
	go func() {
		_, e := service.AcceptProgramReview(ctx, AcceptProgramReviewInput{TenantID: "program-review-a", ProgramID: program.Program.ID, PrincipalID: principalA, ExpectedProgramVersion: program.Program.Version, ExpectedProjectionVersion: firstDigest.CurrentProjectionVersion})
		routeResult <- e
	}()
	routeDeadline := time.Now().Add(3 * time.Second)
	for {
		var waiting bool
		if err = pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND query LIKE '%FOR UPDATE OF p%')`).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		if time.Now().After(routeDeadline) {
			_ = routeBlock.Rollback(ctx)
			t.Fatal("checkpoint did not reach Program lock")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err = pool.Exec(ctx, `UPDATE responsibility_assignments SET valid_until=clock_timestamp() WHERE tenant_id=$1::uuid AND principal_id=$2::uuid`, tenantA, principalA); err != nil {
		t.Fatal(err)
	}
	if err = routeBlock.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	if e := <-routeResult; !errors.Is(e, ErrVersionConflict) {
		t.Fatalf("route revoked before decision: %v", e)
	}
	if _, err = pool.Exec(ctx, `UPDATE responsibility_assignments SET valid_until=NULL WHERE tenant_id=$1::uuid AND principal_id=$2::uuid`, tenantA, principalA); err != nil {
		t.Fatal(err)
	}
	visibilityBlock, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = visibilityBlock.Exec(ctx, `SELECT id FROM programs WHERE id=$1::uuid FOR UPDATE`, program.Program.ID); err != nil {
		t.Fatal(err)
	}
	visibilityResult := make(chan error, 1)
	go func() {
		_, e := service.AcceptProgramReview(ctx, AcceptProgramReviewInput{TenantID: "program-review-a", ProgramID: program.Program.ID, PrincipalID: principalA, ExpectedProgramVersion: program.Program.Version, ExpectedProjectionVersion: firstDigest.CurrentProjectionVersion})
		visibilityResult <- e
	}()
	visibilityDeadline := time.Now().Add(3 * time.Second)
	for {
		var waiting bool
		if err = pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND query LIKE '%FOR UPDATE OF p%')`).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		if time.Now().After(visibilityDeadline) {
			_ = visibilityBlock.Rollback(ctx)
			t.Fatal("checkpoint did not reach Program lock")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err = pool.Exec(ctx, `UPDATE matters SET scope='{"access":"INTERNAL"}'::jsonb WHERE id=$1::uuid`, restrictedMatter); err != nil {
		t.Fatal(err)
	}
	if err = visibilityBlock.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	if e := <-visibilityResult; !errors.Is(e, ErrVersionConflict) {
		t.Fatalf("changed reviewer visibility accepted stale projection: %v", e)
	}
	if _, err = pool.Exec(ctx, `UPDATE matters SET scope=jsonb_build_object('access','RESTRICTED','allowed_principal_ids',jsonb_build_array($2::text)) WHERE id=$1::uuid`, restrictedMatter, principalB); err != nil {
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
	for _, action := range []string{"logout", "rotation"} {
		replayed := orderedSessionAction(t, principalA, accepted.CurrentProjectionVersion, action)
		if replayed.Checkpoint == nil || replayed.Checkpoint.ID != accepted.Checkpoint.ID {
			t.Fatalf("%s replay changed receipt: %+v", action, replayed)
		}
	}
	// A matching historical receipt must not be replayed for a projection that
	// changed while the request waited for the Program lock.
	replayBlock, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = replayBlock.Exec(ctx, `SELECT id FROM programs WHERE id=$1::uuid FOR UPDATE`, program.Program.ID); err != nil {
		t.Fatal(err)
	}
	replayResult := make(chan error, 1)
	go func() {
		_, e := service.AcceptProgramReview(ctx, AcceptProgramReviewInput{TenantID: "program-review-a", ProgramID: program.Program.ID, PrincipalID: principalA, ExpectedProgramVersion: program.Program.Version, ExpectedProjectionVersion: accepted.CurrentProjectionVersion})
		replayResult <- e
	}()
	replayDeadline := time.Now().Add(3 * time.Second)
	for {
		var waiting bool
		if err = pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND query LIKE '%FOR UPDATE OF p%')`).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		if time.Now().After(replayDeadline) {
			_ = replayBlock.Rollback(ctx)
			t.Fatal("replay did not reach Program lock")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err = pool.Exec(ctx, `UPDATE matters SET scope='{"access":"INTERNAL"}'::jsonb WHERE id=$1::uuid`, restrictedMatter); err != nil {
		t.Fatal(err)
	}
	if err = replayBlock.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	if e := <-replayResult; !errors.Is(e, ErrVersionConflict) {
		t.Fatalf("changed reviewer visibility replayed stale projection: %v", e)
	}
	if _, err = pool.Exec(ctx, `UPDATE matters SET scope=jsonb_build_object('access','RESTRICTED','allowed_principal_ids',jsonb_build_array($2::text)) WHERE id=$1::uuid`, restrictedMatter, principalB); err != nil {
		t.Fatal(err)
	}
	const legacyPrincipal = "91111111-1111-7111-8111-111111111115"
	const legacyReceipt = "91111111-1111-7111-8111-111111111116"
	if _, err = pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name,status) VALUES($1::uuid,$2::uuid,'PERSON','Legacy reviewer','ACTIVE'); INSERT INTO program_review_checkpoints(id,tenant_id,program_id,principal_id,program_version,projection_version,accepted_at) VALUES($3::uuid,$2::uuid,$4::uuid,$1::uuid,$5,$6,$7)`, pgx.QueryExecModeSimpleProtocol, legacyPrincipal, tenantA, legacyReceipt, program.Program.ID, program.Program.Version, program.CurrentState.ProjectionVersion, now); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO responsibility_assignments(tenant_id,legal_entity_id,principal_id,responsibility,object_type,object_id,scope,priority,valid_from,policy_version) VALUES($1::uuid,$2::uuid,$3::uuid,'REVIEWER','PROGRAM',$4::uuid,jsonb_build_object('decision_type','program.review.accept'),10,$5,'review:v1')`, tenantA, entityA, legacyPrincipal, program.Program.ID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE responsibility_assignments SET valid_until=clock_timestamp() WHERE tenant_id=$1::uuid AND principal_id=$2::uuid`, tenantA, principalA); err != nil {
		t.Fatal(err)
	}
	legacyCtx := checkpointOIDCContext(t, pool, "program-review-a", entityA, legacyPrincipal, time.Now().Add(time.Hour))
	legacyDigest, err := service.ProgramReviewDigest(legacyCtx, "program-review-a", program.Program.ID, legacyPrincipal)
	if err != nil {
		t.Fatal(err)
	}
	if legacyDigest.State != "NO_BASELINE" || !legacyDigest.ReviewRequired || legacyDigest.Checkpoint != nil {
		t.Fatalf("legacy receipt claimed a known baseline: %+v", legacyDigest)
	}
	const unsignedReceipt = "91111111-1111-7111-8111-111111111117"
	if _, err = pool.Exec(ctx, `INSERT INTO program_review_checkpoints(id,tenant_id,program_id,principal_id,program_version,projection_version,accepted_at,visible_projection_version,visible_open_matter_count) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,$8,0)`, unsignedReceipt, tenantA, program.Program.ID, legacyPrincipal, program.Program.Version, program.CurrentState.ProjectionVersion, now, legacyDigest.CurrentProjectionVersion); err != nil {
		t.Fatal(err)
	}
	unsignedDigest, err := service.ProgramReviewDigest(legacyCtx, "program-review-a", program.Program.ID, legacyPrincipal)
	if err != nil || unsignedDigest.State != "NO_BASELINE" || unsignedDigest.Checkpoint != nil {
		t.Fatalf("unverified receipt claimed a baseline: %+v %v", unsignedDigest, err)
	}
	legacyAccepted := orderedSessionAction(t, legacyPrincipal, legacyDigest.CurrentProjectionVersion, "logout")
	if legacyAccepted.Checkpoint == nil || legacyAccepted.Checkpoint.ID == legacyReceipt || legacyAccepted.Checkpoint.ID == unsignedReceipt {
		t.Fatalf("unscoped legacy receipt replayed: %+v", legacyAccepted)
	}
	if _, err = pool.Exec(ctx, `UPDATE matters SET scope='{"access":"INTERNAL"}'::jsonb WHERE id=$1::uuid`, restrictedMatter); err != nil {
		t.Fatal(err)
	}
	legacyVisible, err := service.ProgramReviewDigest(legacyCtx, "program-review-a", program.Program.ID, legacyPrincipal)
	if err != nil {
		t.Fatal(err)
	}
	legacyRotated := orderedSessionAction(t, legacyPrincipal, legacyVisible.CurrentProjectionVersion, "rotation")
	if legacyRotated.Checkpoint == nil || legacyRotated.Checkpoint.ID == legacyAccepted.Checkpoint.ID {
		t.Fatalf("rotated session failed to record new visible state: %+v", legacyRotated)
	}
	if _, err = pool.Exec(ctx, `UPDATE matters SET scope=jsonb_build_object('access','RESTRICTED','allowed_principal_ids',jsonb_build_array($2::text)) WHERE id=$1::uuid`, restrictedMatter, principalB); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE responsibility_assignments SET valid_until=CASE WHEN principal_id=$2::uuid THEN NULL ELSE clock_timestamp() END WHERE tenant_id=$1::uuid`, tenantA, principalA); err != nil {
		t.Fatal(err)
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
	if _, err = pool.Exec(ctx, `UPDATE responsibility_assignments SET valid_until=clock_timestamp() WHERE tenant_id=$1::uuid AND principal_id=$2::uuid`, tenantA, principalA); err != nil {
		t.Fatal(err)
	}
	_, err = service.AcceptProgramReview(ctx, AcceptProgramReviewInput{TenantID: "program-review-a", ProgramID: program.Program.ID, PrincipalID: principalA, ExpectedProgramVersion: program.Program.Version, ExpectedProjectionVersion: accepted.CurrentProjectionVersion})
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("revoked route replayed receipt: %v", err)
	}
	if _, err = pool.Exec(ctx, `UPDATE responsibility_assignments SET valid_until=NULL WHERE tenant_id=$1::uuid AND principal_id=$2::uuid`, tenantA, principalA); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE programs SET scope='{"access":"RESTRICTED","allowed_principal_ids":[]}'::jsonb WHERE id=$1::uuid`, program.Program.ID); err != nil {
		t.Fatal(err)
	}
	_, err = service.AcceptProgramReview(ctx, AcceptProgramReviewInput{TenantID: "program-review-a", ProgramID: program.Program.ID, PrincipalID: principalA, ExpectedProgramVersion: program.Program.Version, ExpectedProjectionVersion: accepted.CurrentProjectionVersion})
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("restricted program replayed receipt: %v", err)
	}
	if _, err = pool.Exec(ctx, `UPDATE programs SET scope='{}'::jsonb WHERE id=$1::uuid`, program.Program.ID); err != nil {
		t.Fatal(err)
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
	var eventFingerprint, outboxFingerprint int64
	var eventMatterCount, outboxMatterCount int
	if err := pool.QueryRow(ctx, `SELECT (e.payload->>'visible_projection_version')::bigint, (o.payload->>'visible_projection_version')::bigint, (e.payload->>'visible_open_matter_count')::int, (o.payload->>'visible_open_matter_count')::int FROM continuity_events e JOIN outbox_events o ON o.aggregate_id=e.aggregate_id AND o.event_type=e.event_type WHERE e.aggregate_type='PROGRAM_REVIEW' AND e.aggregate_id=$1::uuid`, accepted.Checkpoint.ID).Scan(&eventFingerprint, &outboxFingerprint, &eventMatterCount, &outboxMatterCount); err != nil || eventFingerprint != accepted.CurrentProjectionVersion || outboxFingerprint != eventFingerprint || eventMatterCount != 0 || outboxMatterCount != 0 {
		t.Fatalf("visible receipt missing from audit: %d/%d %d/%d %v", eventFingerprint, outboxFingerprint, eventMatterCount, outboxMatterCount, err)
	}
	var receiptSource, eventSource, outboxSource string
	if err := pool.QueryRow(ctx, `SELECT c.authority_source_id::text,e.payload->>'authority_source_id',o.payload->>'authority_source_id' FROM program_review_checkpoints c JOIN continuity_events e ON e.aggregate_id=c.id AND e.event_type=$2 JOIN outbox_events o ON o.aggregate_id=c.id AND o.event_type=e.event_type WHERE c.id=$1::uuid`, accepted.Checkpoint.ID, EventProgramReviewAccepted).Scan(&receiptSource, &eventSource, &outboxSource); err != nil || receiptSource == "" || receiptSource != eventSource || receiptSource != outboxSource {
		t.Fatalf("authority receipt mismatch: %q %q %q %v", receiptSource, eventSource, outboxSource, err)
	}
	// A new visible state with the same canonical versions is a new review,
	// not a replay of the receipt for the previously visible state.
	if _, err = pool.Exec(ctx, `UPDATE matters SET scope='{"access":"INTERNAL"}'::jsonb WHERE id=$1::uuid`, restrictedMatter); err != nil {
		t.Fatal(err)
	}
	visibleDigest, err := service.ProgramReviewDigest(ctx, "program-review-a", program.Program.ID, principalA)
	if err != nil || visibleDigest.CurrentProjectionVersion == accepted.CurrentProjectionVersion {
		t.Fatalf("visible projection did not change: %+v %v", visibleDigest, err)
	}
	visibleInput := AcceptProgramReviewInput{TenantID: "program-review-a", ProgramID: program.Program.ID, PrincipalID: principalA, ExpectedProgramVersion: program.Program.Version, ExpectedProjectionVersion: visibleDigest.CurrentProjectionVersion}
	visibleAccepted, err := service.AcceptProgramReview(ctx, visibleInput)
	if err != nil || visibleAccepted.Checkpoint == nil || visibleAccepted.Checkpoint.ID == accepted.Checkpoint.ID {
		t.Fatalf("changed visibility reused old receipt: %+v %v", visibleAccepted, err)
	}
	visibleCurrent, err := service.ProgramReviewDigest(ctx, "program-review-a", program.Program.ID, principalA)
	if err != nil || visibleCurrent.State != "CURRENT" || visibleCurrent.ReviewRequired || visibleCurrent.OpenMatterDelta != 0 {
		t.Fatalf("accepted visible baseline was reconstructed incorrectly: %+v %v", visibleCurrent, err)
	}
	visibleReplay, err := service.AcceptProgramReview(ctx, visibleInput)
	if err != nil || visibleReplay.Checkpoint == nil || visibleReplay.Checkpoint.ID != visibleAccepted.Checkpoint.ID {
		t.Fatalf("visible receipt retry: %+v %v", visibleReplay, err)
	}
	if _, err = pool.Exec(ctx, `UPDATE matter_links SET retired_at=clock_timestamp(),retired_by=$2::uuid,retirement_reason='No longer applies' WHERE id=$1::uuid`, restrictedLink, principalA); err != nil {
		t.Fatal(err)
	}
	retiredDigest, err := service.ProgramReviewDigest(ctx, "program-review-a", program.Program.ID, principalA)
	if err != nil || retiredDigest.OpenMatterCount != 0 {
		t.Fatalf("retired link counted: %+v %v", retiredDigest, err)
	}
	if _, err = pool.Exec(ctx, `UPDATE matter_links SET retired_at=NULL,retired_by=NULL,retirement_reason='' WHERE id=$1::uuid`, restrictedLink); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO demo_record_archives(tenant_id,legal_entity_id,record_type,record_id,reason,source_manifest,archived_by,archived_at) VALUES($1::uuid,$2::uuid,'MATTER',$3::uuid,'Excluded sample','test-manifest',$4::uuid,clock_timestamp())`, tenantA, entityA, restrictedMatter, principalA); err != nil {
		t.Fatal(err)
	}
	archivedDigest, err := service.ProgramReviewDigest(ctx, "program-review-a", program.Program.ID, principalA)
	if err != nil || archivedDigest.OpenMatterCount != 0 {
		t.Fatalf("archived Matter counted: %+v %v", archivedDigest, err)
	}
	if _, err = pool.Exec(ctx, `UPDATE demo_record_archives SET restored_at=clock_timestamp(),restored_by=$3::uuid,restoration_reason='Restored sample' WHERE tenant_id=$1::uuid AND record_id=$2::uuid`, tenantA, restrictedMatter, principalA); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE matters SET scope=jsonb_build_object('access','RESTRICTED','allowed_principal_ids',jsonb_build_array($2::text)) WHERE id=$1::uuid`, restrictedMatter, principalB); err != nil {
		t.Fatal(err)
	}

	// Compare the actual checkpoint transaction to the existing authoritative
	// resolver for delegation, grant and segregation changes.
	const delegate = "91111111-1111-7111-8111-111111111119"
	if _, err = pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($1::uuid,$2::uuid,'PERSON','Delegated reviewer','ACTIVE',$5); INSERT INTO delegations(tenant_id,legal_entity_id,from_principal_id,to_principal_id,responsibility,scope,starts_at,ends_at,status,created_by,approved_by,approved_at,version) VALUES($2::uuid,$3::uuid,$4::uuid,$1::uuid,'REVIEWER',jsonb_build_object('legal_entity_id',$3::text,'decision_type','program.review.accept'),$5::timestamptz-interval '1 minute',$5::timestamptz+interval '1 hour','ACTIVE',$4::uuid,$6::uuid,$5,1)`, pgx.QueryExecModeSimpleProtocol, delegate, tenantA, entityA, principalA, now, legacyPrincipal); err != nil {
		t.Fatal(err)
	}
	delegateCtx := checkpointOIDCContext(t, pool, "program-review-a", entityA, delegate, time.Now().Add(time.Hour))
	delegateDigest, err := service.ProgramReviewDigest(delegateCtx, "program-review-a", program.Program.ID, delegate)
	if err != nil {
		t.Fatal(err)
	}
	delegateInput := AcceptProgramReviewInput{TenantID: "program-review-a", ProgramID: program.Program.ID, PrincipalID: delegate, ExpectedProgramVersion: program.Program.Version, ExpectedProjectionVersion: delegateDigest.CurrentProjectionVersion}
	router := authority.NewPostgresService(pool)
	routeInput := authority.ResolveInput{TenantID: tenantA, LegalEntityID: entityA, ObjectType: "PROGRAM", ObjectID: program.Program.ID, Responsibility: authority.ResponsibilityReviewer, DecisionType: "program.review.accept", Materiality: 3}
	resolution, err := router.Resolve(delegateCtx, routeInput)
	if err != nil || !resolution.AllowsPrincipal(delegate) {
		t.Fatalf("delegated reviewer route: %+v %v", resolution, err)
	}
	delegated, err := service.AcceptProgramReview(delegateCtx, delegateInput)
	if err != nil || delegated.Checkpoint == nil {
		t.Fatalf("delegated checkpoint: %+v %v", delegated, err)
	}
	assertDelegated := func(label string, allowed bool) {
		t.Helper()
		current, routeErr := router.Resolve(delegateCtx, routeInput)
		routeAllowed := routeErr == nil && current.AllowsPrincipal(delegate)
		replay, replayErr := service.AcceptProgramReview(delegateCtx, delegateInput)
		if routeAllowed != allowed || (replayErr == nil) != allowed || (allowed && (replay.Checkpoint == nil || replay.Checkpoint.ID != delegated.Checkpoint.ID)) || (!allowed && replay.Checkpoint != nil) {
			t.Fatalf("%s: route=%v (%v) receipt=%+v (%v)", label, routeAllowed, routeErr, replay, replayErr)
		}
	}
	assertDelegatedWriteRace := func(label string, mutate func() error) {
		t.Helper()
		if _, e := pool.Exec(ctx, `UPDATE matters SET scope='{"access":"INTERNAL"}'::jsonb WHERE id=$1::uuid`, restrictedMatter); e != nil {
			t.Fatal(e)
		}
		fresh, e := service.ProgramReviewDigest(delegateCtx, "program-review-a", program.Program.ID, delegate)
		if e != nil {
			t.Fatal(e)
		}
		attempt := delegateInput
		attempt.ExpectedProjectionVersion = fresh.CurrentProjectionVersion
		block, e := pool.Begin(ctx)
		if e != nil {
			t.Fatal(e)
		}
		defer block.Rollback(context.Background())
		if _, e = block.Exec(ctx, `SELECT id FROM programs WHERE id=$1::uuid FOR UPDATE`, program.Program.ID); e != nil {
			t.Fatal(e)
		}
		done := make(chan error, 1)
		go func() { _, err := service.AcceptProgramReview(delegateCtx, attempt); done <- err }()
		deadline := time.Now().Add(3 * time.Second)
		for {
			var waiting bool
			if e = pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND query LIKE '%FOR UPDATE OF p%')`).Scan(&waiting); e != nil {
				t.Fatal(e)
			}
			if waiting {
				break
			}
			if time.Now().After(deadline) {
				_ = block.Rollback(ctx)
				t.Fatalf("%s did not reach lock", label)
			}
			time.Sleep(10 * time.Millisecond)
		}
		if e = mutate(); e != nil {
			t.Fatal(e)
		}
		if e = block.Rollback(ctx); e != nil {
			t.Fatal(e)
		}
		if e = <-done; !errors.Is(e, ErrVersionConflict) {
			t.Fatalf("%s allowed stale authority: %v", label, e)
		}
		var count int
		if e = pool.QueryRow(ctx, `SELECT count(*) FROM program_review_checkpoints WHERE tenant_id=$1::uuid AND program_id=$2::uuid AND principal_id=$3::uuid`, tenantA, program.Program.ID, delegate).Scan(&count); e != nil || count != 1 {
			t.Fatalf("%s changed receipt count=%d err=%v", label, count, e)
		}
		if _, e = pool.Exec(ctx, `UPDATE matters SET scope=jsonb_build_object('access','RESTRICTED','allowed_principal_ids',jsonb_build_array($2::text)) WHERE id=$1::uuid`, restrictedMatter, principalB); e != nil {
			t.Fatal(e)
		}
	}
	assertDelegated("delegation", true)
	aliasCtx := checkpointOIDCContext(t, pool, tenantA, "ENTITY-A", delegate, time.Now().Add(time.Hour))
	aliasInput := delegateInput
	aliasInput.TenantID = tenantA
	aliasReplay, err := service.AcceptProgramReview(aliasCtx, aliasInput)
	if err != nil || aliasReplay.Checkpoint == nil || aliasReplay.Checkpoint.ID != delegated.Checkpoint.ID {
		t.Fatalf("tenant UUID/entity code replay: %+v %v", aliasReplay, err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO authority_grants(tenant_id,legal_entity_id,principal_id,decision_type,limits,valid_from,policy_version) VALUES($1::uuid,$2::uuid,$3::uuid,'program.review.accept','{"max_materiality":2}'::jsonb,$4,'grant:v1')`, tenantA, entityA, delegate, now.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	assertDelegated("grant denies materiality", false)
	if _, err = pool.Exec(ctx, `UPDATE authority_grants SET limits='{"max_materiality":4}'::jsonb WHERE tenant_id=$1::uuid AND principal_id=$2::uuid`, tenantA, delegate); err != nil {
		t.Fatal(err)
	}
	assertDelegated("grant permits materiality", true)
	assertDelegatedWriteRace("grant revoked during decision", func() error {
		_, e := pool.Exec(ctx, `UPDATE authority_grants SET limits='{"max_materiality":2}'::jsonb WHERE tenant_id=$1::uuid AND principal_id=$2::uuid`, tenantA, delegate)
		return e
	})
	if _, err = pool.Exec(ctx, `UPDATE authority_grants SET limits='{"max_materiality":4}'::jsonb WHERE tenant_id=$1::uuid AND principal_id=$2::uuid`, tenantA, delegate); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `DELETE FROM authority_grants WHERE tenant_id=$1::uuid AND principal_id=$2::uuid`, tenantA, delegate); err != nil {
		t.Fatal(err)
	}
	const conflictRole = "91111111-1111-7111-8111-111111111120"
	const conflictPosition = "91111111-1111-7111-8111-111111111121"
	const conflictBinding = "91111111-1111-7111-8111-111111111122"
	if _, err = pool.Exec(ctx, `INSERT INTO role_templates(id,tenant_id,code,name,valid_from) VALUES($1::uuid,$2::uuid,'CONFLICT','Conflicting role',$6); INSERT INTO org_positions(id,tenant_id,legal_entity_id,code,title,occupant_principal_id,valid_from) VALUES($3::uuid,$2::uuid,$4::uuid,'CONFLICT','Conflicting role',$5::uuid,$6); INSERT INTO position_role_bindings(id,tenant_id,position_id,role_template_id,priority,valid_from) VALUES($7::uuid,$2::uuid,$3::uuid,$1::uuid,100,$6)`, pgx.QueryExecModeSimpleProtocol, conflictRole, tenantA, conflictPosition, entityA, delegate, now.Add(-time.Minute), conflictBinding); err != nil {
		t.Fatal(err)
	}
	assertDelegatedWriteRace("segregation added during decision", func() error {
		_, e := pool.Exec(ctx, `INSERT INTO segregation_rules(tenant_id,code,responsibility,prohibited_role_code,status,valid_from) VALUES($1::uuid,'NO-CONFLICT','REVIEWER','CONFLICT','ACTIVE',$2)`, tenantA, now.Add(-time.Minute))
		return e
	})
	assertDelegated("segregation excludes reviewer", false)
	if _, err = pool.Exec(ctx, `DELETE FROM segregation_rules WHERE tenant_id=$1::uuid AND code='NO-CONFLICT'`, tenantA); err != nil {
		t.Fatal(err)
	}
	assertDelegated("segregation removed", true)
	if _, err = pool.Exec(ctx, `UPDATE principals SET status='INACTIVE' WHERE id=$1::uuid`, delegate); err != nil {
		t.Fatal(err)
	}
	assertDelegated("reviewer inactive", false)
	if _, err = pool.Exec(ctx, `UPDATE principals SET status='ACTIVE' WHERE id=$1::uuid`, delegate); err != nil {
		t.Fatal(err)
	}
	assertDelegated("reviewer reactivated", true)
	assertDelegatedWriteRace("delegation revoked during decision", func() error {
		_, e := pool.Exec(ctx, `UPDATE delegations SET status='REVOKED' WHERE tenant_id=$1::uuid AND to_principal_id=$2::uuid`, tenantA, delegate)
		return e
	})
	assertDelegated("delegation revoked", false)
	if _, err = pool.Exec(ctx, `UPDATE delegations SET status='ACTIVE' WHERE tenant_id=$1::uuid AND to_principal_id=$2::uuid; UPDATE matters SET scope='{"access":"INTERNAL"}'::jsonb WHERE id=$3::uuid`, pgx.QueryExecModeSimpleProtocol, tenantA, delegate, restrictedMatter); err != nil {
		t.Fatal(err)
	}
	var originalSource string
	if err = pool.QueryRow(ctx, `SELECT authority_source_id::text FROM program_review_checkpoints WHERE id=$1::uuid`, delegated.Checkpoint.ID).Scan(&originalSource); err != nil {
		t.Fatal(err)
	}
	changedDelegate, err := service.ProgramReviewDigest(delegateCtx, "program-review-a", program.Program.ID, delegate)
	if err != nil {
		t.Fatal(err)
	}
	changedInput := delegateInput
	changedInput.ExpectedProjectionVersion = changedDelegate.CurrentProjectionVersion
	replacementBlock, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = replacementBlock.Exec(ctx, `SELECT id FROM programs WHERE id=$1::uuid FOR UPDATE`, program.Program.ID); err != nil {
		t.Fatal(err)
	}
	type replacementResult struct {
		digest ProgramReviewDigest
		err    error
	}
	replacementDone := make(chan replacementResult, 1)
	go func() {
		d, e := service.AcceptProgramReview(delegateCtx, changedInput)
		replacementDone <- replacementResult{d, e}
	}()
	replacementDeadline := time.Now().Add(3 * time.Second)
	for {
		var waiting bool
		if err = pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND query LIKE '%FOR UPDATE OF p%')`).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		if time.Now().After(replacementDeadline) {
			_ = replacementBlock.Rollback(ctx)
			t.Fatal("replacement test did not reach lock")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err = pool.Exec(ctx, `UPDATE responsibility_assignments SET valid_until=clock_timestamp() WHERE tenant_id=$1::uuid AND principal_id=$2::uuid`, tenantA, principalA); err != nil {
		t.Fatal(err)
	}
	var replacementSource string
	if err = pool.QueryRow(ctx, `INSERT INTO responsibility_assignments(tenant_id,legal_entity_id,principal_id,responsibility,object_type,object_id,scope,priority,valid_from,policy_version) VALUES($1::uuid,$2::uuid,$3::uuid,'REVIEWER','PROGRAM',$4::uuid,jsonb_build_object('decision_type','program.review.accept'),10,$5,'review:v1') RETURNING id::text`, tenantA, entityA, principalA, program.Program.ID, now.Add(-time.Minute)).Scan(&replacementSource); err != nil {
		t.Fatal(err)
	}
	if err = replacementBlock.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	replacement := <-replacementDone
	if replacement.err != nil || replacement.digest.Checkpoint == nil {
		t.Fatalf("replacement source decision: %+v %v", replacement.digest, replacement.err)
	}
	var insertedSource, insertedEventSource string
	if err = pool.QueryRow(ctx, `SELECT c.authority_source_id::text,e.payload->>'authority_source_id' FROM program_review_checkpoints c JOIN continuity_events e ON e.aggregate_id=c.id AND e.event_type=$2 WHERE c.id=$1::uuid`, replacement.digest.Checkpoint.ID, EventProgramReviewAccepted).Scan(&insertedSource, &insertedEventSource); err != nil || insertedSource != replacementSource || insertedEventSource != replacementSource || replacementSource == originalSource {
		t.Fatalf("replacement route receipt: prior=%s selected=%s stored=%s event=%s err=%v", originalSource, replacementSource, insertedSource, insertedEventSource, err)
	}
	if _, err = pool.Exec(ctx, `UPDATE matters SET scope=jsonb_build_object('access','RESTRICTED','allowed_principal_ids',jsonb_build_array($2::text)) WHERE id=$1::uuid`, restrictedMatter, principalB); err != nil {
		t.Fatal(err)
	}
	assertDelegated("receipt replay under replacement route", true)
	// If a legal-entity code collides with another entity's UUID while the
	// command waits, neither entity may be silently selected for the write.
	aliasBlock, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = aliasBlock.Exec(ctx, `SELECT id FROM programs WHERE id=$1::uuid FOR UPDATE`, program.Program.ID); err != nil {
		t.Fatal(err)
	}
	aliasDone := make(chan error, 1)
	go func() {
		_, e := service.AcceptProgramReview(ctx, AcceptProgramReviewInput{TenantID: "program-review-a", ProgramID: program.Program.ID, PrincipalID: principalA, ExpectedProgramVersion: program.Program.Version, ExpectedProjectionVersion: accepted.CurrentProjectionVersion})
		aliasDone <- e
	}()
	aliasDeadline := time.Now().Add(3 * time.Second)
	for {
		var waiting bool
		if err = pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND query LIKE '%FOR UPDATE OF p%')`).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		if time.Now().After(aliasDeadline) {
			_ = aliasBlock.Rollback(ctx)
			t.Fatal("entity alias test did not reach lock")
		}
		time.Sleep(10 * time.Millisecond)
	}
	const collidingEntity = "91111111-1111-7111-8111-111111111123"
	if _, err = pool.Exec(ctx, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2::uuid,$3,'Conflicting identifier','NG',$4)`, collidingEntity, tenantA, entityA, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err = aliasBlock.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	if e := <-aliasDone; !errors.Is(e, ErrVersionConflict) {
		t.Fatalf("ambiguous entity allowed receipt: %v", e)
	}
	if _, err = pool.Exec(ctx, `UPDATE legal_entities SET valid_until=clock_timestamp() WHERE id=$1::uuid`, collidingEntity); err != nil {
		t.Fatal(err)
	}
	// Distinct policies may reuse a rule ID and version. Their selected source
	// must remain separate when priority changes during the row-lock wait.
	const policyReviewer = "91111111-1111-7111-8111-111111111124"
	const firstPolicy = "91111111-1111-7111-8111-111111111125"
	const secondPolicy = "91111111-1111-7111-8111-111111111126"
	if _, err = pool.Exec(ctx, `UPDATE responsibility_assignments SET valid_until=clock_timestamp() WHERE tenant_id=$1::uuid; INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($2::uuid,$1::uuid,'PERSON','Policy reviewer','ACTIVE',$7); INSERT INTO routing_policies(id,tenant_id,legal_entity_id,code,name,status) VALUES($3::uuid,$1::uuid,$6::uuid,'FIRST-POLICY','First policy','DRAFT'),($4::uuid,$1::uuid,$6::uuid,'SECOND-POLICY','Second policy','DRAFT'); INSERT INTO effective_authority_routes(tenant_id,source_policy_id,source_rule_id,policy_version,legal_entity_ref,object_type,object_id,responsibility,decision_type,priority,selector_kind,selector_ref,valid_from) VALUES($1::uuid,$3::uuid,'shared-rule','collision:v1',$6::text,'PROGRAM',$5::text,'REVIEWER','program.review.accept',100,'PRINCIPAL',$2::text,$7),($1::uuid,$4::uuid,'shared-rule','collision:v1',$6::text,'PROGRAM',$5::text,'REVIEWER','program.review.accept',10,'PRINCIPAL',$8::text,$7)`, pgx.QueryExecModeSimpleProtocol, tenantA, policyReviewer, firstPolicy, secondPolicy, program.Program.ID, entityA, now.Add(-time.Minute), delegate); err != nil {
		t.Fatal(err)
	}
	policyCtx := checkpointOIDCContext(t, pool, "program-review-a", entityA, policyReviewer, time.Now().Add(time.Hour))
	policyDigest, err := service.ProgramReviewDigest(policyCtx, "program-review-a", program.Program.ID, policyReviewer)
	if err != nil {
		t.Fatal(err)
	}
	policyInput := AcceptProgramReviewInput{TenantID: "program-review-a", ProgramID: program.Program.ID, PrincipalID: policyReviewer, ExpectedProgramVersion: program.Program.Version, ExpectedProjectionVersion: policyDigest.CurrentProjectionVersion}
	policyRoute, err := router.Resolve(policyCtx, routeInput)
	if err != nil || !policyRoute.AllowsPrincipal(policyReviewer) || policyRoute.SourceID != firstPolicy {
		t.Fatalf("first policy resolution: %+v %v", policyRoute, err)
	}
	policyReceipt, err := service.AcceptProgramReview(policyCtx, policyInput)
	if err != nil || policyReceipt.Checkpoint == nil {
		t.Fatalf("first policy checkpoint: %+v %v", policyReceipt, err)
	}
	var policyReceiptSource string
	if err = pool.QueryRow(ctx, `SELECT authority_source_id::text FROM program_review_checkpoints WHERE id=$1::uuid`, policyReceipt.Checkpoint.ID).Scan(&policyReceiptSource); err != nil || policyReceiptSource != firstPolicy {
		t.Fatalf("first policy source=%q err=%v", policyReceiptSource, err)
	}
	if denied, e := service.AcceptProgramReview(delegateCtx, delegateInput); e == nil || denied.Checkpoint != nil {
		t.Fatalf("lower-ranked same-rule policy disclosed receipt: %+v %v", denied, e)
	}
	if _, err = pool.Exec(ctx, `UPDATE matters SET scope='{"access":"INTERNAL"}'::jsonb WHERE id=$1::uuid; UPDATE effective_authority_routes SET selector_ref=$2 WHERE source_policy_id=$3::uuid`, pgx.QueryExecModeSimpleProtocol, restrictedMatter, policyReviewer, secondPolicy); err != nil {
		t.Fatal(err)
	}
	policyVisible, err := service.ProgramReviewDigest(policyCtx, "program-review-a", program.Program.ID, policyReviewer)
	if err != nil {
		t.Fatal(err)
	}
	policyInput.ExpectedProjectionVersion = policyVisible.CurrentProjectionVersion
	policyBlock, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = policyBlock.Exec(ctx, `SELECT id FROM programs WHERE id=$1::uuid FOR UPDATE`, program.Program.ID); err != nil {
		t.Fatal(err)
	}
	policyDone := make(chan replacementResult, 1)
	go func() {
		d, e := service.AcceptProgramReview(policyCtx, policyInput)
		policyDone <- replacementResult{d, e}
	}()
	policyDeadline := time.Now().Add(3 * time.Second)
	for {
		var waiting bool
		if err = pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND query LIKE '%FOR UPDATE OF p%')`).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		if time.Now().After(policyDeadline) {
			_ = policyBlock.Rollback(ctx)
			t.Fatal("policy replacement did not reach Program lock")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err = pool.Exec(ctx, `UPDATE effective_authority_routes SET priority=110 WHERE source_policy_id=$1::uuid`, secondPolicy); err != nil {
		t.Fatal(err)
	}
	if err = policyBlock.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	policyResult := <-policyDone
	if policyResult.err != nil || policyResult.digest.Checkpoint == nil {
		t.Fatalf("replacement policy checkpoint: %+v %v", policyResult.digest, policyResult.err)
	}
	var policyEventSource string
	if err = pool.QueryRow(ctx, `SELECT c.authority_source_id::text,e.payload->>'authority_source_id' FROM program_review_checkpoints c JOIN continuity_events e ON e.aggregate_id=c.id AND e.event_type=$2 WHERE c.id=$1::uuid`, policyResult.digest.Checkpoint.ID, EventProgramReviewAccepted).Scan(&policyReceiptSource, &policyEventSource); err != nil || policyReceiptSource != secondPolicy || policyEventSource != secondPolicy {
		t.Fatalf("same-rule replacement source=%q event=%q err=%v", policyReceiptSource, policyEventSource, err)
	}
	if _, err = pool.Exec(ctx, `UPDATE matters SET scope=jsonb_build_object('access','RESTRICTED','allowed_principal_ids',jsonb_build_array($2::text)) WHERE id=$1::uuid; DELETE FROM effective_authority_routes WHERE tenant_id=$3::uuid; UPDATE responsibility_assignments SET valid_until=NULL WHERE id=$4::uuid`, pgx.QueryExecModeSimpleProtocol, restrictedMatter, principalB, tenantA, replacementSource); err != nil {
		t.Fatal(err)
	}

	// Test the exact checkpoint count with a malformed link inside a rolled-back
	// transaction; the deferred database constraint must reject that link.
	corrupt, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer corrupt.Rollback(context.Background())
	if _, err = corrupt.Exec(ctx, `SET CONSTRAINTS matter_links_legal_entity_match DEFERRED`); err != nil {
		t.Fatal(err)
	}
	const foreignEntity = "91111111-1111-7111-8111-111111111127"
	const foreignMatter = "91111111-1111-7111-8111-111111111128"
	const foreignLink = "91111111-1111-7111-8111-111111111129"
	if _, err = corrupt.Exec(ctx, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2::uuid,'OTHER-CORRUPT','Other','NG',$6); INSERT INTO matters(id,tenant_id,legal_entity_id,reference,matter_type,status,priority,title,summary,scope,known_facts,missing_facts,contradictions,created_at,updated_at,version) VALUES($3::uuid,$2::uuid,$1::uuid,'CORRUPT-LINK','AUTHORITY_REQUEST','ASSESSMENT',4,'Other issue','Other issue','{"access":"INTERNAL"}','{}','[]','[]',$6,$6,1); INSERT INTO matter_links(id,tenant_id,matter_id,program_id,relationship,created_at) VALUES($4::uuid,$2::uuid,$3::uuid,$5::uuid,'AFFECTS',$6)`, pgx.QueryExecModeSimpleProtocol, foreignEntity, tenantA, foreignMatter, foreignLink, program.Program.ID, now); err != nil {
		t.Fatal(err)
	}
	var safeCount int
	if err = corrupt.QueryRow(ctx, `SELECT `+visibleProgramMatterCountSQL("p", "$2::text")+` FROM programs p WHERE p.id=$1::uuid`, program.Program.ID, principalA).Scan(&safeCount); err != nil || safeCount != 0 {
		t.Fatalf("cross-entity Matter affected checkpoint count=%d err=%v", safeCount, err)
	}
	if _, err = corrupt.Exec(ctx, `SET CONSTRAINTS matter_links_legal_entity_match IMMEDIATE`); err == nil {
		t.Fatal("cross-entity link constraint accepted malformed link")
	}
	if err = corrupt.Rollback(ctx); err != nil {
		t.Fatal(err)
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
