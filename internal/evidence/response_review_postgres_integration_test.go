//go:build postgres && postgresintegration

package evidence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5"
)

func TestPostgresResponseReviewRejectsSessionExpiryDuringAuthorityCheck(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	tenant, entity, owner, reviewer, form := mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t)
	now := time.Now().UTC()
	prefix := "whole-review-expiry-" + tenant
	setupResponseWorkspaceFixture(t, ctx, pool, tenant, prefix, entity, owner, form, now)
	seedCompletedResponseRows(t, ctx, pool, tenant, entity, owner, form, prefix, 1, now)
	t.Cleanup(func() { cleanupResponseWorkspaceTenant(context.Background(), pool, tenant) })
	_, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($2::uuid,$1::uuid,'PERSON','Risk reviewer','ACTIVE',$3); UPDATE capture_submissions SET answers='{"score":{"text":"Report supplied"}}'::jsonb WHERE tenant_id=$1::uuid; UPDATE capture_requests req SET subject_id=d.subject_id::text FROM capture_form_distributions d WHERE req.distribution_id=d.id AND req.tenant_id=$1::uuid`, pgx.QueryExecModeSimpleProtocol, tenant, reviewer, now)
	if err != nil {
		t.Fatal(err)
	}
	var response string
	if err = pool.QueryRow(ctx, `SELECT id::text FROM capture_response_revisions WHERE tenant_id=$1::uuid`, tenant).Scan(&response); err != nil {
		t.Fatal(err)
	}
	var tick atomic.Int64
	tick.Store(now.UnixNano())
	started, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	service := NewDistributionService(NewPostgresDistributionStore(NewPostgresRepository(pool), nil)).WithResponseReviewAuthorizer(func(context.Context, identity.Actor, CompletedResponseSummary) (string, error) {
		once.Do(func() { close(started); <-release })
		return "policy:rule", nil
	})
	service.demoResponseReviewDecisions = true // Exercise the unexposed transaction in isolation.
	service.now = func() time.Time { return time.Unix(0, tick.Load()).UTC() }
	ctx, cancel := context.WithTimeout(identity.WithActor(ctx, identity.Actor{TenantID: tenant, LegalEntityID: entity, PrincipalID: reviewer, ExpiresAt: now.Add(time.Second)}), 15*time.Second)
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, e := service.RecordResponseReview(ctx, tenant, entity, response, RecordResponseReviewInput{Decision: "REVIEWED"})
		result <- e
	}()
	select {
	case <-started:
	case <-ctx.Done():
		close(release)
		t.Fatal(ctx.Err())
	}
	tick.Store(now.Add(2 * time.Second).UnixNano())
	close(release)
	if err := <-result; !errors.Is(err, ErrAssessmentForbidden) {
		t.Fatalf("expired session: %v", err)
	}
	var reviews, events, outbox int
	if err = pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM capture_response_reviews WHERE tenant_id=$1::uuid),(SELECT count(*) FROM capture_distribution_events WHERE tenant_id=$1::uuid AND event_type LIKE 'FORM_RESPONSE_REVIEWED_%'),(SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND event_type='FORM_RESPONSE_REVIEWED')`, tenant).Scan(&reviews, &events, &outbox); err != nil || reviews != 0 || events != 0 || outbox != 0 {
		t.Fatalf("review=%d events=%d outbox=%d err=%v", reviews, events, outbox, err)
	}
}

func TestPostgresResponseReviewRejectsSessionExpiredAtDatabaseDecision(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	tenant, entity, owner, reviewer, form := mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t)
	now := time.Now().UTC()
	prefix := "whole-review-db-expiry-" + tenant
	setupResponseWorkspaceFixture(t, ctx, pool, tenant, prefix, entity, owner, form, now)
	seedCompletedResponseRows(t, ctx, pool, tenant, entity, owner, form, prefix, 1, now)
	t.Cleanup(func() { cleanupResponseWorkspaceTenant(context.Background(), pool, tenant) })
	_, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($2::uuid,$1::uuid,'PERSON','Risk reviewer','ACTIVE',$3); UPDATE capture_submissions SET answers='{"score":{"text":"Report supplied"}}'::jsonb WHERE tenant_id=$1::uuid; UPDATE capture_requests req SET subject_id=d.subject_id::text FROM capture_form_distributions d WHERE req.distribution_id=d.id AND req.tenant_id=$1::uuid`, pgx.QueryExecModeSimpleProtocol, tenant, reviewer, now.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	var response string
	if err = pool.QueryRow(ctx, `SELECT id::text FROM capture_response_revisions WHERE tenant_id=$1::uuid`, tenant).Scan(&response); err != nil {
		t.Fatal(err)
	}
	service := NewDistributionService(NewPostgresDistributionStore(NewPostgresRepository(pool), nil)).WithResponseReviewAuthorizer(func(context.Context, identity.Actor, CompletedResponseSummary) (string, error) {
		return "policy:rule", nil
	})
	service.demoResponseReviewDecisions = true // Exercise the unexposed transaction in isolation.
	// Model an application clock behind the database clock: the service-level
	// check passes, but the session is already expired at the write boundary.
	service.now = func() time.Time { return now.Add(-time.Minute) }
	ctx = identity.WithActor(ctx, identity.Actor{TenantID: tenant, LegalEntityID: entity, PrincipalID: reviewer, ExpiresAt: now.Add(-time.Second)})
	if _, err = service.RecordResponseReview(ctx, tenant, entity, response, RecordResponseReviewInput{Decision: "REVIEWED"}); !errors.Is(err, ErrAssessmentForbidden) {
		t.Fatalf("database-expired session: %v", err)
	}
	var reviews, events, outbox int
	if err = pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM capture_response_reviews WHERE tenant_id=$1::uuid),(SELECT count(*) FROM capture_distribution_events WHERE tenant_id=$1::uuid AND event_type LIKE 'FORM_RESPONSE_REVIEWED_%'),(SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND event_type='FORM_RESPONSE_REVIEWED')`, tenant).Scan(&reviews, &events, &outbox); err != nil || reviews != 0 || events != 0 || outbox != 0 {
		t.Fatalf("review=%d events=%d outbox=%d err=%v", reviews, events, outbox, err)
	}
}

func TestPostgresResponseReviewAtomicHistory(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	tenant, entity, owner, reviewer, form := mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t)
	now := time.Now().UTC()
	prefix := "whole-review-" + tenant
	setupResponseWorkspaceFixture(t, ctx, pool, tenant, prefix, entity, owner, form, now)
	seedCompletedResponseRows(t, ctx, pool, tenant, entity, owner, form, prefix, 1, now)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `BEGIN; SET LOCAL session_replication_role='replica'; DELETE FROM capture_response_reviews WHERE tenant_id=$1::uuid; COMMIT`, pgx.QueryExecModeSimpleProtocol, tenant)
		cleanupResponseWorkspaceTenant(context.Background(), pool, tenant)
	})
	_, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($2::uuid,$1::uuid,'PERSON','Risk reviewer','ACTIVE',$3); UPDATE capture_submissions SET answers='{"score":{"text":"Report supplied"}}'::jsonb WHERE tenant_id=$1::uuid; UPDATE capture_requests req SET subject_id=d.subject_id::text FROM capture_form_distributions d WHERE req.distribution_id=d.id AND req.tenant_id=$1::uuid`, pgx.QueryExecModeSimpleProtocol, tenant, reviewer, now)
	if err != nil {
		t.Fatal(err)
	}
	var response string
	if err = pool.QueryRow(ctx, `SELECT id::text FROM capture_response_revisions WHERE tenant_id=$1::uuid`, tenant).Scan(&response); err != nil {
		t.Fatal(err)
	}
	allowed := true
	service := NewDistributionService(NewPostgresDistributionStore(NewPostgresRepository(pool), nil)).WithResponseReviewAuthorizer(func(context.Context, identity.Actor, CompletedResponseSummary) (string, error) {
		if !allowed {
			return "", ErrAssessmentForbidden
		}
		return "policy:rule", nil
	})
	service.ConfigureDemoResponseReviewDecisions(true)
	ctx = identity.WithActor(ctx, identity.Actor{TenantID: tenant, LegalEntityID: entity, PrincipalID: reviewer, ExpiresAt: now.Add(time.Hour)})
	input := RecordResponseReviewInput{Decision: "REVIEWED"}
	blocked, err := service.GetResponseReviews(ctx, tenant, entity, reviewer, response, 0, 10)
	if err != nil || blocked.MayReview {
		t.Fatalf("postgres demo opt-in enabled decisions: %+v %v", blocked, err)
	}
	if _, err = service.RecordResponseReview(ctx, tenant, entity, response, input); !errors.Is(err, ErrProgramReviewUnavailable) {
		t.Fatalf("postgres demo opt-in allowed write: %v", err)
	}
	// The following exercises the underlying transaction only in this test.
	service.demoResponseReviewDecisions = true
	first, err := service.RecordResponseReview(ctx, tenant, entity, response, input)
	if err != nil || first.Version != 1 {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	service.ConfigureDemoResponseReviewDecisions(false)
	lockedHistory, err := service.GetResponseReviews(ctx, tenant, entity, reviewer, response, 0, 10)
	if err != nil || lockedHistory.MayReview || len(lockedHistory.Items) != 1 {
		t.Fatalf("disabled decisions must preserve history: %+v %v", lockedHistory, err)
	}
	if _, err = service.RecordResponseReview(ctx, tenant, entity, response, RecordResponseReviewInput{ExpectedVersion: 1, Decision: "REVIEWED"}); !errors.Is(err, ErrProgramReviewUnavailable) {
		t.Fatalf("disabled decision must fail closed: %v", err)
	}
	service.demoResponseReviewDecisions = true // Exercise the unexposed transaction in isolation.
	if _, err = service.RecordResponseReview(ctx, tenant, entity, response, input); !errors.Is(err, ErrAssessmentConflict) {
		t.Fatalf("stale=%v", err)
	}
	input.ExpectedVersion = 1
	input.Decision = "CHANGES_REQUESTED"
	input.Comment = "Add the reporting period."
	second, err := service.RecordResponseReview(ctx, tenant, entity, response, input)
	if err != nil || second.SupersedesID != first.ID {
		t.Fatalf("second=%+v err=%v", second, err)
	}
	page, err := service.GetResponseReviews(ctx, tenant, entity, reviewer, response, 0, 1)
	if err != nil || page.Version != 2 || !page.MayReview || len(page.Items) != 1 || page.Items[0].ID != second.ID || page.NextBeforeVersion != 2 {
		t.Fatalf("history=%+v err=%v", page, err)
	}
	page, err = service.GetResponseReviews(ctx, tenant, entity, reviewer, response, page.NextBeforeVersion, 1)
	if err != nil || page.Version != 2 || len(page.Items) != 1 || page.Items[0].ID != first.ID || page.NextBeforeVersion != 0 {
		t.Fatalf("older history=%+v err=%v", page, err)
	}
	allowed = false
	input.ExpectedVersion = 2
	if _, err = service.RecordResponseReview(ctx, tenant, entity, response, input); !errors.Is(err, ErrAssessmentForbidden) {
		t.Fatalf("revoked=%v", err)
	}
	allowed = true
	trigger := "response_review_failure_" + strings.ReplaceAll(tenant, "-", "")
	_, err = pool.Exec(ctx, fmt.Sprintf(`CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.aggregate_id='%s'::uuid AND NEW.event_type='FORM_RESPONSE_REVIEWED' THEN RAISE EXCEPTION 'injected outbox failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER %s BEFORE INSERT ON outbox_events FOR EACH ROW EXECUTE FUNCTION %s()`, trigger, response, trigger, trigger), pgx.QueryExecModeSimpleProtocol)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), fmt.Sprintf(`DROP TRIGGER IF EXISTS %s ON outbox_events; DROP FUNCTION IF EXISTS %s()`, trigger, trigger), pgx.QueryExecModeSimpleProtocol)
	})
	if _, err = service.RecordResponseReview(ctx, tenant, entity, response, input); err == nil {
		t.Fatal("outbox failure committed a review")
	}
	var reviews, events, outbox int
	err = pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM capture_response_reviews WHERE tenant_id=$1::uuid),(SELECT count(*) FROM capture_distribution_events WHERE tenant_id=$1::uuid AND event_type LIKE 'FORM_RESPONSE_REVIEWED_%'),(SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND event_type='FORM_RESPONSE_REVIEWED')`, tenant).Scan(&reviews, &events, &outbox)
	if err != nil || reviews != 2 || events != 2 || outbox != 2 {
		t.Fatalf("counts %d %d %d: %v", reviews, events, outbox, err)
	}
	if _, err = pool.Exec(ctx, `UPDATE capture_response_reviews SET comment='changed' WHERE id=$1::uuid`, first.ID); err == nil {
		t.Fatal("review history was mutable")
	}
}
