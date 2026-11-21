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
	"github.com/jackc/pgx/v5/pgxpool"
)

func seedResponseReviewerRoute(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenant, entity, reviewer, response string, now time.Time) {
	t.Helper()
	program := mustResponseWorkspaceID(t)
	if _, err := pool.Exec(ctx, `INSERT INTO programs(id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,jurisdiction,scope,effective_from) VALUES($1::uuid,$2::uuid,$3::uuid,'REVIEW-TEST','Review test','COMPLIANCE','ACTIVE','Risk','NG','{"access":"INTERNAL"}'::jsonb,$4); UPDATE capture_form_distributions SET subject_type='PROGRAM',subject_id=$1::uuid WHERE tenant_id=$2::uuid; UPDATE capture_requests SET subject_type='PROGRAM',subject_id=$1::text WHERE tenant_id=$2::uuid`, pgx.QueryExecModeSimpleProtocol, program, tenant, entity, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO responsibility_assignments(tenant_id,legal_entity_id,principal_id,responsibility,object_type,object_id,decision_type,priority,valid_from,policy_version) VALUES($1::uuid,$2::uuid,$3::uuid,'REVIEWER','FORM_RESPONSE',$4::uuid,'forms.response.review',10,$5,'review:v1')`, tenant, entity, reviewer, response, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM responsibility_assignments WHERE tenant_id=$1::uuid`, tenant)
		_, _ = pool.Exec(context.Background(), `DELETE FROM programs WHERE id=$1::uuid`, program)
	})
}

func TestPostgresResponseReviewRechecksBeforeDecision(t *testing.T) {
	for _, scenario := range []struct {
		name, mutation string
		succeeds       bool
	}{
		{"revoked route", `UPDATE responsibility_assignments SET valid_until=clock_timestamp()-interval '1 second' WHERE tenant_id=$1::uuid AND principal_id=$2::uuid`, false},
		{"respondent conflict", `UPDATE capture_submissions SET submitted_by=$2::uuid WHERE tenant_id=$1::uuid`, false},
		{"subject access revoked", `UPDATE programs SET scope='{"access":"RESTRICTED","allowed_principal_ids":[]}'::jsonb WHERE tenant_id=$1::uuid AND $2::uuid IS NOT NULL`, false},
		{"entity expired", `UPDATE legal_entities SET valid_until=clock_timestamp()-interval '1 second' WHERE tenant_id=$1::uuid AND $2::uuid IS NOT NULL`, false},
		{"route replaced", `BEGIN; UPDATE responsibility_assignments SET valid_until=clock_timestamp()-interval '1 second' WHERE tenant_id=$1::uuid AND principal_id=$2::uuid; INSERT INTO responsibility_assignments(tenant_id,legal_entity_id,principal_id,responsibility,object_type,object_id,decision_type,priority,valid_from,policy_version) SELECT tenant_id,legal_entity_id,principal_id,responsibility,object_type,object_id,decision_type,priority,clock_timestamp()-interval '1 second','review:v2' FROM responsibility_assignments WHERE tenant_id=$1::uuid AND principal_id=$2::uuid; COMMIT`, true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			pool, ctx := distributionTestPool(t)
			tenant, entity, owner, reviewer, form := mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t)
			now := time.Now().UTC()
			prefix := "review-route-revoke-" + tenant
			setupResponseWorkspaceFixture(t, ctx, pool, tenant, prefix, entity, owner, form, now)
			seedCompletedResponseRows(t, ctx, pool, tenant, entity, owner, form, prefix, 1, now)
			t.Cleanup(func() {
				_, _ = pool.Exec(context.Background(), `BEGIN; SET LOCAL session_replication_role='replica'; DELETE FROM capture_response_reviews WHERE tenant_id=$1::uuid; COMMIT`, pgx.QueryExecModeSimpleProtocol, tenant)
				cleanupResponseWorkspaceTenant(context.Background(), pool, tenant)
			})
			if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($2::uuid,$1::uuid,'PERSON','Reviewer','ACTIVE',$3); UPDATE capture_submissions SET answers='{"score":{"text":"Report supplied"}}'::jsonb WHERE tenant_id=$1::uuid; UPDATE capture_requests req SET subject_id=d.subject_id::text FROM capture_form_distributions d WHERE req.distribution_id=d.id AND req.tenant_id=$1::uuid`, pgx.QueryExecModeSimpleProtocol, tenant, reviewer, now.Add(-time.Hour)); err != nil {
				t.Fatal(err)
			}
			var response string
			if err := pool.QueryRow(ctx, `SELECT id::text FROM capture_response_revisions WHERE tenant_id=$1::uuid`, tenant).Scan(&response); err != nil {
				t.Fatal(err)
			}
			seedResponseReviewerRoute(t, ctx, pool, tenant, entity, reviewer, response, now)
			started, release := make(chan struct{}), make(chan struct{})
			service := NewDistributionService(NewPostgresDistributionStore(NewPostgresRepository(pool), nil)).WithResponseReviewAuthorizer(func(context.Context, identity.Actor, CompletedResponseSummary) (string, error) {
				close(started)
				<-release
				return "review:v1:rule", nil
			})
			service.demoResponseReviewDecisions = true // Test the private transaction while the public route remains gated.
			actorCtx, cancel := context.WithTimeout(identity.WithActor(ctx, identity.Actor{TenantID: tenant, LegalEntityID: entity, PrincipalID: reviewer, Kind: "PERSON", ExpiresAt: now.Add(time.Hour)}), 15*time.Second)
			defer cancel()
			type outcome struct {
				review ResponseReview
				err    error
			}
			result := make(chan outcome, 1)
			go func() {
				review, err := service.RecordResponseReview(actorCtx, tenant, entity, response, RecordResponseReviewInput{Decision: "REVIEWED"})
				result <- outcome{review, err}
			}()
			select {
			case <-started:
			case result := <-result:
				t.Fatalf("decision stopped before authority barrier: %v", result.err)
			case <-actorCtx.Done():
				close(release)
				t.Fatal(actorCtx.Err())
			}
			if _, err := pool.Exec(ctx, scenario.mutation, pgx.QueryExecModeSimpleProtocol, tenant, reviewer); err != nil {
				close(release)
				t.Fatal(err)
			}
			close(release)
			out := <-result
			if scenario.succeeds {
				if out.err != nil || out.review.AuthorityPolicyVersion != "review:v2" {
					t.Fatalf("replacement route receipt: %+v %v", out.review, out.err)
				}
			} else if !errors.Is(out.err, ErrAssessmentForbidden) {
				t.Fatalf("%s recorded decision: %v", scenario.name, out.err)
			}
			var reviews, events, outbox int
			want := 0
			if scenario.succeeds {
				want = 1
			}
			if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM capture_response_reviews WHERE tenant_id=$1::uuid),(SELECT count(*) FROM capture_distribution_events WHERE tenant_id=$1::uuid AND event_type LIKE 'FORM_RESPONSE_REVIEWED_%'),(SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND event_type='FORM_RESPONSE_REVIEWED')`, tenant).Scan(&reviews, &events, &outbox); err != nil || reviews != want || events != want || outbox != want {
				t.Fatalf("denied review wrote records review=%d events=%d outbox=%d err=%v", reviews, events, outbox, err)
			}
		})
	}
}

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
	seedResponseReviewerRoute(t, ctx, pool, tenant, entity, reviewer, response, now)
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
	ctx, cancel := context.WithTimeout(identity.WithActor(ctx, identity.Actor{TenantID: tenant, LegalEntityID: entity, PrincipalID: reviewer, Kind: "PERSON", ExpiresAt: now.Add(time.Second)}), 15*time.Second)
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
	seedResponseReviewerRoute(t, ctx, pool, tenant, entity, reviewer, response, now)
	service := NewDistributionService(NewPostgresDistributionStore(NewPostgresRepository(pool), nil)).WithResponseReviewAuthorizer(func(context.Context, identity.Actor, CompletedResponseSummary) (string, error) {
		return "policy:rule", nil
	})
	service.demoResponseReviewDecisions = true // Exercise the unexposed transaction in isolation.
	// Model an application clock behind the database clock: the service-level
	// check passes, but the session is already expired at the write boundary.
	service.now = func() time.Time { return now.Add(-time.Minute) }
	ctx = identity.WithActor(ctx, identity.Actor{TenantID: tenant, LegalEntityID: entity, PrincipalID: reviewer, Kind: "PERSON", ExpiresAt: now.Add(-time.Second)})
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
	seedResponseReviewerRoute(t, ctx, pool, tenant, entity, reviewer, response, now)
	allowed := true
	service := NewDistributionService(NewPostgresDistributionStore(NewPostgresRepository(pool), nil)).WithResponseReviewAuthorizer(func(context.Context, identity.Actor, CompletedResponseSummary) (string, error) {
		if !allowed {
			return "", ErrAssessmentForbidden
		}
		return "policy:rule", nil
	})
	service.ConfigureDemoResponseReviewDecisions(true)
	ctx = identity.WithActor(ctx, identity.Actor{TenantID: tenant, LegalEntityID: entity, PrincipalID: reviewer, Kind: "PERSON", ExpiresAt: now.Add(time.Hour)})
	input := RecordResponseReviewInput{Decision: "REVIEWED"}
	blocked, err := service.GetResponseReviews(ctx, tenant, entity, reviewer, response, 0, 10)
	if err != nil || blocked.MayReview {
		t.Fatalf("postgres demo opt-in enabled decisions: %+v %v", blocked, err)
	}
	if _, err = service.RecordResponseReview(ctx, tenant, entity, response, input); !errors.Is(err, ErrProgramReviewUnavailable) {
		t.Fatalf("postgres demo opt-in allowed write: %v", err)
	}
	repeatable, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		t.Fatal(err)
	}
	probeID := mustResponseWorkspaceID(t)
	probe := ResponseReview{ID: probeID, ResponseID: response, ResponseRevision: 1, Version: 1, Decision: "REVIEWED", ReviewerID: reviewer}
	material := assessmentMaterial{Summary: CompletedResponseSummary{ID: response, TenantID: tenant, LegalEntityID: entity}, Revision: ResponseRevision{ID: response, Revision: 1}}
	actor, _ := identity.FromContext(ctx)
	if _, err = insertCurrentResponseReview(ctx, repeatable, actor, material, probe, ""); !errors.Is(err, ErrAssessmentForbidden) {
		_ = repeatable.Rollback(ctx)
		t.Fatalf("repeatable-read decision must fail closed: %v", err)
	}
	_ = repeatable.Rollback(ctx)
	// The following exercises the underlying transaction only in this test.
	service.demoResponseReviewDecisions = true
	first, err := service.RecordResponseReview(ctx, tenant, entity, response, input)
	if err != nil || first.Version != 1 {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	var expectedSource string
	if err = pool.QueryRow(ctx, `SELECT id::text FROM responsibility_assignments WHERE tenant_id=$1::uuid AND object_id=$2::uuid`, tenant, response).Scan(&expectedSource); err != nil {
		t.Fatal(err)
	}
	if first.AuthoritySourceType != "ASSIGNMENT" || first.AuthoritySourceID != expectedSource || first.AuthorityRuleID != "assignment:"+expectedSource || first.AuthorityPolicyVersion != "review:v1" {
		t.Fatalf("incorrect authority receipt: %+v", first)
	}
	var eventTime, outboxTime time.Time
	var eventSource, outboxSource string
	if err = pool.QueryRow(ctx, `SELECT e.occurred_at,o.occurred_at,e.payload->>'authority_source_id',o.payload->>'authority_source_id' FROM capture_distribution_events e JOIN outbox_events o ON o.tenant_id=e.tenant_id AND o.aggregate_id=$2::uuid AND o.event_type='FORM_RESPONSE_REVIEWED' WHERE e.tenant_id=$1::uuid AND e.event_type LIKE 'FORM_RESPONSE_REVIEWED_%'`, tenant, response).Scan(&eventTime, &outboxTime, &eventSource, &outboxSource); err != nil || !eventTime.Equal(first.ReviewedAt) || !outboxTime.Equal(first.ReviewedAt) || eventSource != expectedSource || outboxSource != expectedSource {
		t.Fatalf("event/outbox receipt mismatch: event=%v/%s outbox=%v/%s err=%v", eventTime, eventSource, outboxTime, outboxSource, err)
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
	eventTrigger := "response_event_failure_" + strings.ReplaceAll(tenant, "-", "")
	_, err = pool.Exec(ctx, fmt.Sprintf(`CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.tenant_id='%s'::uuid AND NEW.event_type LIKE 'FORM_RESPONSE_REVIEWED_%%' THEN RAISE EXCEPTION 'injected event failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER %s BEFORE INSERT ON capture_distribution_events FOR EACH ROW EXECUTE FUNCTION %s()`, eventTrigger, tenant, eventTrigger, eventTrigger), pgx.QueryExecModeSimpleProtocol)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), fmt.Sprintf(`DROP TRIGGER IF EXISTS %s ON capture_distribution_events; DROP FUNCTION IF EXISTS %s()`, eventTrigger, eventTrigger), pgx.QueryExecModeSimpleProtocol)
	})
	if _, err = service.RecordResponseReview(ctx, tenant, entity, response, input); err == nil {
		t.Fatal("event failure committed a review")
	}
	var eventFailureReviews int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM capture_response_reviews WHERE tenant_id=$1::uuid`, tenant).Scan(&eventFailureReviews); err != nil || eventFailureReviews != 2 {
		t.Fatalf("event failure left review count=%d err=%v", eventFailureReviews, err)
	}
	if _, err = pool.Exec(ctx, fmt.Sprintf(`DROP TRIGGER %s ON capture_distribution_events; DROP FUNCTION %s()`, eventTrigger, eventTrigger), pgx.QueryExecModeSimpleProtocol); err != nil {
		t.Fatal(err)
	}
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
	if _, err = pool.Exec(ctx, fmt.Sprintf(`DROP TRIGGER %s ON outbox_events; DROP FUNCTION %s()`, trigger, trigger), pgx.QueryExecModeSimpleProtocol); err != nil {
		t.Fatal(err)
	}
	decisions := make(chan error, 2)
	start := make(chan struct{})
	for i := 0; i < 2; i++ {
		go func() {
			<-start
			_, err := service.RecordResponseReview(ctx, tenant, entity, response, input)
			decisions <- err
		}()
	}
	close(start)
	var success, conflicts int
	for i := 0; i < 2; i++ {
		if err := <-decisions; err == nil {
			success++
		} else if errors.Is(err, ErrAssessmentConflict) {
			conflicts++
		} else {
			t.Fatalf("unexpected competing review error: %v", err)
		}
	}
	if success != 1 || conflicts != 1 {
		t.Fatalf("competing reviews: success=%d conflicts=%d", success, conflicts)
	}
	if err = pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM capture_response_reviews WHERE tenant_id=$1::uuid),(SELECT count(*) FROM capture_distribution_events WHERE tenant_id=$1::uuid AND event_type LIKE 'FORM_RESPONSE_REVIEWED_%'),(SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND event_type='FORM_RESPONSE_REVIEWED')`, tenant).Scan(&reviews, &events, &outbox); err != nil || reviews != 3 || events != 3 || outbox != 3 {
		t.Fatalf("competing review records %d %d %d: %v", reviews, events, outbox, err)
	}
	if _, err = pool.Exec(ctx, `UPDATE capture_response_reviews SET comment='changed' WHERE id=$1::uuid`, first.ID); err == nil {
		t.Fatal("review history was mutable")
	}
}

func TestPostgresResponseReviewAcceptsVerifiedTenantSlug(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	tenant, entity, owner, reviewer, form := mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t)
	now := time.Now().UTC()
	slug := "review-slug-" + tenant
	setupResponseWorkspaceFixture(t, ctx, pool, tenant, slug, entity, owner, form, now)
	seedCompletedResponseRows(t, ctx, pool, tenant, entity, owner, form, slug, 1, now)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `BEGIN; SET LOCAL session_replication_role='replica'; DELETE FROM capture_response_reviews WHERE tenant_id=$1::uuid; COMMIT`, pgx.QueryExecModeSimpleProtocol, tenant)
		cleanupResponseWorkspaceTenant(context.Background(), pool, tenant)
	})
	if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($2::uuid,$1::uuid,'PERSON','Reviewer','ACTIVE',$3); UPDATE capture_submissions SET answers='{"score":{"text":"Report supplied"}}'::jsonb WHERE tenant_id=$1::uuid`, pgx.QueryExecModeSimpleProtocol, tenant, reviewer, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	var response string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM capture_response_revisions WHERE tenant_id=$1::uuid`, tenant).Scan(&response); err != nil {
		t.Fatal(err)
	}
	seedResponseReviewerRoute(t, ctx, pool, tenant, entity, reviewer, response, now)
	service := NewDistributionService(NewPostgresDistributionStore(NewPostgresRepository(pool), nil)).WithResponseReviewAuthorizer(func(_ context.Context, actor identity.Actor, summary CompletedResponseSummary) (string, error) {
		if actor.TenantID != summary.TenantID {
			return "", errors.New("scope mismatch")
		}
		return "review:v1:rule", nil
	})
	service.demoResponseReviewDecisions = true
	// A UUID that is also another tenant's slug is ambiguous and must not select either tenant.
	otherTenant := mustResponseWorkspaceID(t)
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,'Ambiguous tenant')`, otherTenant, tenant); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, otherTenant) })
	ambiguousCtx := identity.WithActor(ctx, identity.Actor{TenantID: tenant, LegalEntityID: entity, PrincipalID: reviewer, Kind: "PERSON", ExpiresAt: now.Add(time.Hour)})
	if _, err := service.RecordResponseReview(ambiguousCtx, tenant, entity, response, RecordResponseReviewInput{Decision: "REVIEWED"}); !errors.Is(err, ErrAssessmentForbidden) {
		t.Fatalf("ambiguous tenant identity: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, otherTenant); err != nil {
		t.Fatal(err)
	}
	actorCtx := identity.WithActor(ctx, identity.Actor{TenantID: slug, LegalEntityID: entity, PrincipalID: reviewer, Kind: "PERSON", ExpiresAt: now.Add(time.Hour)})
	requestKey := mustResponseWorkspaceID(t)
	review, err := service.RecordResponseReview(actorCtx, slug, entity, response, RecordResponseReviewInput{Decision: "REVIEWED", RequestKey: requestKey})
	if err != nil || review.AuthoritySourceID == "" {
		t.Fatalf("verified tenant slug: review=%+v err=%v", review, err)
	}
	// A retry after a lost response returns the exact recorded decision.
	retry, err := service.RecordResponseReview(actorCtx, slug, entity, response, RecordResponseReviewInput{Decision: "REVIEWED", RequestKey: requestKey})
	if err != nil || retry.ID != review.ID {
		t.Fatalf("retry: %+v %v", retry, err)
	}
	if _, err := service.RecordResponseReview(actorCtx, slug, entity, response, RecordResponseReviewInput{Decision: "CHANGES_REQUESTED", Comment: "Changed", RequestKey: requestKey}); !errors.Is(err, ErrAssessmentConflict) {
		t.Fatalf("key reuse: %v", err)
	}
	history, err := service.GetResponseReviews(actorCtx, slug, entity, reviewer, response, 0, 10)
	if err != nil || history.Version != 1 || len(history.Items) != 1 || history.Items[0].ID != review.ID || history.Items[0].ReviewerID != reviewer {
		t.Fatalf("scoped recovery history: %+v %v", history, err)
	}
	// Competing requests with one key must return the same decision and emit once.
	nextKey := mustResponseWorkspaceID(t)
	type result struct {
		review ResponseReview
		err    error
	}
	results := make(chan result, 2)
	for i := 0; i < 2; i++ {
		go func() {
			v, e := service.RecordResponseReview(actorCtx, slug, entity, response, RecordResponseReviewInput{ExpectedVersion: 1, Decision: "REVIEWED", RequestKey: nextKey})
			results <- result{v, e}
		}()
	}
	first, second := <-results, <-results
	if first.err != nil || second.err != nil || first.review.ID != second.review.ID {
		t.Fatalf("concurrent same-key: %+v %+v", first, second)
	}
	var reviews, events, outbox int
	if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM capture_response_reviews WHERE tenant_id=$1::uuid),(SELECT count(*) FROM capture_distribution_events WHERE tenant_id=$1::uuid AND event_type LIKE 'FORM_RESPONSE_REVIEWED_%'),(SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND event_type='FORM_RESPONSE_REVIEWED')`, tenant).Scan(&reviews, &events, &outbox); err != nil || reviews != 2 || events != 2 || outbox != 2 {
		t.Fatalf("records %d %d %d: %v", reviews, events, outbox, err)
	}
	// Loss of subject visibility prevents recovery even when the preflight callback allows it.
	if _, err := pool.Exec(ctx, `UPDATE programs SET scope='{"access":"RESTRICTED","allowed_principal_ids":[]}'::jsonb WHERE tenant_id=$1::uuid`, tenant); err != nil {
		t.Fatal(err)
	}
	if _, err := service.RecordResponseReview(actorCtx, slug, entity, response, RecordResponseReviewInput{ExpectedVersion: 1, Decision: "REVIEWED", RequestKey: nextKey}); !errors.Is(err, ErrAssessmentForbidden) && !errors.Is(err, ErrNotFound) {
		t.Fatalf("restricted retry: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE programs SET scope='{"access":"INTERNAL"}'::jsonb WHERE tenant_id=$1::uuid`, tenant); err != nil {
		t.Fatal(err)
	}
	// An expired database-clock session cannot recover a receipt even if the service clock lags.
	service.now = func() time.Time { return now.Add(-time.Minute) }
	expiredCtx := identity.WithActor(ctx, identity.Actor{TenantID: slug, LegalEntityID: entity, PrincipalID: reviewer, Kind: "PERSON", ExpiresAt: now.Add(-time.Second)})
	if _, err := service.RecordResponseReview(expiredCtx, slug, entity, response, RecordResponseReviewInput{ExpectedVersion: 1, Decision: "REVIEWED", RequestKey: nextKey}); !errors.Is(err, ErrAssessmentForbidden) {
		t.Fatalf("expired retry: %v", err)
	}
	service.now = nil
	// A receipt is not disclosed once the current route has been revoked.
	if _, err := pool.Exec(ctx, `UPDATE responsibility_assignments SET valid_until=clock_timestamp()-interval '1 second' WHERE tenant_id=$1::uuid`, tenant); err != nil {
		t.Fatal(err)
	}
	if _, err := service.RecordResponseReview(actorCtx, slug, entity, response, RecordResponseReviewInput{ExpectedVersion: 1, Decision: "REVIEWED", RequestKey: nextKey}); !errors.Is(err, ErrAssessmentForbidden) {
		t.Fatalf("revoked retry: %v", err)
	}
}
