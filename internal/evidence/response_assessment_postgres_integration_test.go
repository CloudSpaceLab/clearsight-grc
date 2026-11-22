//go:build postgres && postgresintegration

package evidence

import (
	"context"
	"errors"
	"fmt"
	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func setupAssessmentRiskRole(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenant, entity, reviewer string, now time.Time) {
	t.Helper()
	role, position, binding := mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t)
	_, err := pool.Exec(ctx, `INSERT INTO role_templates(id,tenant_id,code,name,description,responsibilities,valid_from) VALUES($4::uuid,$1::uuid,'RISK','Risk reviewer','',ARRAY['REVIEWER'],$7);
 INSERT INTO org_positions(id,tenant_id,legal_entity_id,code,title,occupant_principal_id,valid_from) VALUES($5::uuid,$1::uuid,$2::uuid,'RISK','Risk reviewer',$3::uuid,$7);
 INSERT INTO position_role_bindings(id,tenant_id,position_id,role_template_id,priority,valid_from) VALUES($6::uuid,$1::uuid,$5::uuid,$4::uuid,100,$7)`, pgx.QueryExecModeSimpleProtocol, tenant, entity, reviewer, role, position, binding, now.Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
}

func TestPostgresResponseAssessmentAtomicHistoryAndReplay(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	tenant := mustResponseWorkspaceID(t)
	entity := mustResponseWorkspaceID(t)
	actor := mustResponseWorkspaceID(t)
	reviewer := mustResponseWorkspaceID(t)
	form := mustResponseWorkspaceID(t)
	now := time.Now().UTC()
	prefix := "bank-assessment-" + tenant
	setupResponseWorkspaceFixture(t, ctx, pool, tenant, prefix, entity, actor, form, now)
	seedCompletedResponseRows(t, ctx, pool, tenant, entity, actor, form, prefix, 1, now)
	t.Cleanup(func() {
		tx, err := pool.Begin(context.Background())
		if err == nil {
			defer tx.Rollback(context.Background())
			_, err = tx.Exec(context.Background(), `SET LOCAL session_replication_role='replica'`)
			if err == nil {
				_, err = tx.Exec(context.Background(), `DELETE FROM capture_field_assessments WHERE tenant_id=$1::uuid; DELETE FROM capture_response_assessments WHERE tenant_id=$1::uuid`, pgx.QueryExecModeSimpleProtocol, tenant)
			}
			if err == nil {
				_ = tx.Commit(context.Background())
			}
		}
		_, _ = pool.Exec(context.Background(), `DELETE FROM position_role_bindings WHERE tenant_id=$1::uuid; DELETE FROM org_positions WHERE tenant_id=$1::uuid; DELETE FROM role_templates WHERE tenant_id=$1::uuid; DELETE FROM third_party_relationships WHERE tenant_id=$1::uuid; DELETE FROM third_parties WHERE tenant_id=$1::uuid`, pgx.QueryExecModeSimpleProtocol, tenant)
		cleanupResponseWorkspaceTenant(context.Background(), pool, tenant)
	})
	_, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($1::uuid,$2::uuid,'PERSON','Bank reviewer','ACTIVE',$3); UPDATE capture_requests SET fields='[{"id":"score","section_id":"general","label":"Evidence quality","type":"short_text","assessment":{"mode":"MANUAL","required":true,"weight":100,"reviewer_role":"RISK","rubric":[{"id":"poor","label":"Incomplete","points":80},{"id":"good","label":"Sufficient","points":10}]}}]'::jsonb WHERE tenant_id=$2::uuid; UPDATE capture_requests req SET subject_id=d.subject_id::text FROM capture_form_distributions d WHERE req.distribution_id=d.id AND req.tenant_id=$2::uuid; UPDATE capture_submissions SET answers='{"score":{"text":"Report supplied"}}'::jsonb WHERE tenant_id=$2::uuid`, pgx.QueryExecModeSimpleProtocol, reviewer, tenant, now)
	if err != nil {
		t.Fatal(err)
	}
	setupAssessmentRiskRole(t, ctx, pool, tenant, entity, reviewer, now)
	_, err = pool.Exec(ctx, `INSERT INTO third_parties(id,tenant_id,legal_name,status,created_at,updated_at,version) VALUES(md5($4||':vendor')::uuid,$1::uuid,'Sample vendor','ACTIVE',$5,$5,1);
 INSERT INTO third_party_relationships(id,tenant_id,legal_entity_id,vendor_id,service_name,business_owner_principal_id,criticality,privacy_role,status,created_at,updated_at,version) VALUES(md5($4||':relationship')::uuid,$1::uuid,$2::uuid,md5($4||':vendor')::uuid,'Sample service',$3::uuid,'IMPORTANT','PROCESSOR','ACTIVE',$5,$5,1);
 UPDATE capture_form_distributions SET subject_type='VENDOR_RELATIONSHIP',subject_id=md5($4||':relationship')::uuid WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid;
 UPDATE capture_requests SET subject_type='VENDOR_RELATIONSHIP',subject_id=(md5($4||':relationship')::uuid)::text WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid`, pgx.QueryExecModeSimpleProtocol, tenant, entity, reviewer, prefix, now)
	if err != nil {
		t.Fatal(err)
	}
	var response string
	if err = pool.QueryRow(ctx, `SELECT id::text FROM capture_response_revisions WHERE tenant_id=$1::uuid`, tenant).Scan(&response); err != nil {
		t.Fatal(err)
	}
	config := pool.Config()
	config.MaxConns = 4
	assessmentPool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer assessmentPool.Close()
	policy, policyVersion := mustResponseWorkspaceID(t), mustResponseWorkspaceID(t)
	definition := fmt.Sprintf(`{"rules":[{"id":"bank-review","legal_entity_id":"%s","object_type":"FORM_RESPONSE","object_id":"%s","responsibility":"REVIEWER","decision_type":"forms.response.assess","min_materiality":0,"priority":100,"selector":{"kind":"ROLE","ref":"RISK"}}]}`, entity, response)
	_, err = pool.Exec(ctx, `INSERT INTO routing_policies(id,tenant_id,legal_entity_id,code,name,status,current_version,approved_at,version) VALUES($1::uuid,$2::uuid,$3::uuid,'BANK-REVIEW','Bank review','DRAFT',1,$6,1); INSERT INTO routing_policy_versions(id,policy_id,legal_entity_id,version,definition,checksum,effective_from,approved_at) VALUES($4::uuid,$1::uuid,$3::uuid,1,$5::jsonb,'review-test',$6,$6); UPDATE routing_policies SET status='ACTIVE' WHERE id=$1::uuid`, pgx.QueryExecModeSimpleProtocol, policy, tenant, entity, policyVersion, definition, now.Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM routing_policies WHERE tenant_id=$1::uuid`, tenant)
	})
	router := authority.NewPostgresService(assessmentPool)
	service := NewDistributionService(NewPostgresDistributionStore(NewPostgresRepository(assessmentPool), nil)).WithAssessmentAuthorizer(func(ctx context.Context, a identity.Actor, r CompletedResponseSummary, f formcontract.Field) (string, error) {
		route, err := router.Resolve(ctx, authority.ResolveInput{TenantID: a.TenantID, LegalEntityID: r.LegalEntityID, ObjectType: "FORM_RESPONSE", ObjectID: r.ID, Responsibility: authority.ResponsibilityReviewer, DecisionType: "forms.response.assess", Materiality: 3})
		if err != nil {
			return "", err
		}
		if !route.AllowsPrincipalWithRole(a.PrincipalID, f.Assessment.ReviewerRole) {
			return "", ErrAssessmentForbidden
		}
		return route.PolicyVersion + ":" + route.RuleID, nil
	})
	service.assessmentDecisionsEnabled = true // Exercise the unexposed transaction in isolation.
	ctx = identity.WithActor(ctx, identity.Actor{Kind: "PERSON", TenantID: tenant, LegalEntityID: entity, PrincipalID: reviewer, ExpiresAt: now.Add(time.Hour)})
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	baseline, err := service.GetResponseAssessment(ctx, tenant, entity, reviewer, response)
	if err != nil || !baseline.MayReview {
		var visible bool
		var details string
		diag := pool.QueryRow(ctx, `SELECT (`+completedResponseVisibilitySQL(4, 5)+`), concat_ws('|',req.form_template_id::text,d.form_template_id::text,req.form_template_version::text,d.form_template_version::text,req.subject_type,d.subject_type,req.subject_id,d.subject_id::text) FROM capture_response_revisions r JOIN tenants t ON t.id=r.tenant_id JOIN capture_form_distributions d ON d.id=r.distribution_id AND d.tenant_id=r.tenant_id AND d.legal_entity_id=r.legal_entity_id `+completedResponseRequestJoinsSQL()+` WHERE (t.id::text=$1 OR t.slug=$1) AND r.legal_entity_id=$2::uuid AND r.id=$3::uuid`, tenant, entity, response, reviewer, time.Now()).Scan(&visible, &details)
		t.Fatalf("authorized assessment baseline: %+v %v; visibility=%v details=%s diagnostic=%v", baseline, err, visible, details, diag)
	}
	lock, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Rollback(ctx)
	if _, err = lock.Exec(ctx, `SELECT id FROM capture_response_revisions WHERE id=$1::uuid FOR UPDATE`, response); err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	var once sync.Once
	service.now = func() time.Time { once.Do(func() { close(started) }); return time.Now() }
	expires := time.Now().Add(600 * time.Millisecond)
	expiring := identity.WithActor(ctx, identity.Actor{Kind: "PERSON", TenantID: tenant, LegalEntityID: entity, PrincipalID: reviewer, ExpiresAt: expires})
	expiryResult := make(chan error, 1)
	go func() {
		_, writeErr := service.RecordResponseAssessment(expiring, tenant, entity, response, RecordResponseAssessmentInput{RequestKey: "00000000-0000-4000-8000-000000000001", Decisions: []FieldAssessmentInput{{FieldID: "score", OutcomeID: "poor", Rationale: "Report omits tested systems."}}})
		expiryResult <- writeErr
	}()
	<-started
	time.Sleep(time.Until(expires) + 30*time.Millisecond)
	if err = lock.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err = <-expiryResult; !errors.Is(err, ErrAssessmentForbidden) {
		t.Fatalf("expired session accepted after lock: %v", err)
	}
	service.now = time.Now
	var expiredWrites int
	if err = pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM capture_response_assessments WHERE tenant_id=$1::uuid)+(SELECT count(*) FROM capture_field_assessments WHERE tenant_id=$1::uuid)+(SELECT count(*) FROM capture_distribution_events WHERE tenant_id=$1::uuid AND event_type LIKE 'FORM_RESPONSE_ASSESSED_%')+(SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND event_type='FORM_RESPONSE_ASSESSED')`, tenant).Scan(&expiredWrites); err != nil || expiredWrites != 0 {
		t.Fatalf("expired assessment wrote %d records: %v", expiredWrites, err)
	}
	originalAuthorizer := service.assessmentAuthorizer
	authorized, proceed := make(chan struct{}), make(chan struct{})
	var authOnce sync.Once
	service.WithAssessmentAuthorizer(func(c context.Context, a identity.Actor, r CompletedResponseSummary, f formcontract.Field) (string, error) {
		route, routeErr := originalAuthorizer(c, a, r, f)
		if routeErr == nil {
			authOnce.Do(func() { close(authorized); <-proceed })
		}
		return route, routeErr
	})
	revokedResult := make(chan error, 1)
	go func() {
		_, e := service.RecordResponseAssessment(ctx, tenant, entity, response, RecordResponseAssessmentInput{RequestKey: "00000000-0000-4000-8000-000000000001", Decisions: []FieldAssessmentInput{{FieldID: "score", OutcomeID: "poor", Rationale: "Report omits tested systems."}}})
		revokedResult <- e
	}()
	<-authorized
	if _, err = pool.Exec(ctx, `UPDATE routing_policies SET status='RETIRED' WHERE id=$1::uuid`, policy); err != nil {
		close(proceed)
		t.Fatal(err)
	}
	close(proceed)
	if err = <-revokedResult; !errors.Is(err, ErrAssessmentForbidden) {
		t.Fatalf("revoked route accepted before assessment insert: %v", err)
	}
	if err = pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM capture_response_assessments WHERE tenant_id=$1::uuid)+(SELECT count(*) FROM capture_field_assessments WHERE tenant_id=$1::uuid)+(SELECT count(*) FROM capture_distribution_events WHERE tenant_id=$1::uuid AND event_type LIKE 'FORM_RESPONSE_ASSESSED_%')+(SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND event_type='FORM_RESPONSE_ASSESSED')`, tenant).Scan(&expiredWrites); err != nil || expiredWrites != 0 {
		t.Fatalf("revoked assessment wrote %d records: %v", expiredWrites, err)
	}
	if _, err = pool.Exec(ctx, `UPDATE routing_policies SET status='ACTIVE' WHERE id=$1::uuid`, policy); err != nil {
		t.Fatal(err)
	}
	service.WithAssessmentAuthorizer(originalAuthorizer)
	// A replacement policy may reuse the rule ID/version while dropping the
	// field's required reviewer role. The selected source must be rechecked.
	replacement, replacementVersion := mustResponseWorkspaceID(t), mustResponseWorkspaceID(t)
	replacementDefinition := fmt.Sprintf(`{"rules":[{"id":"bank-review","legal_entity_id":"%s","object_type":"FORM_RESPONSE","object_id":"%s","responsibility":"REVIEWER","decision_type":"forms.response.assess","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"%s"}}]}`, entity, response, reviewer)
	_, err = pool.Exec(ctx, `INSERT INTO routing_policies(id,tenant_id,legal_entity_id,code,name,status,current_version,approved_at,version) VALUES($1::uuid,$2::uuid,$3::uuid,'REPLACEMENT','Replacement','DRAFT',1,$6,1); INSERT INTO routing_policy_versions(id,policy_id,legal_entity_id,version,definition,checksum,effective_from,approved_at) VALUES($4::uuid,$1::uuid,$3::uuid,1,$5::jsonb,'replacement',$6,$6)`, pgx.QueryExecModeSimpleProtocol, replacement, tenant, entity, replacementVersion, replacementDefinition, now.Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	checked, continueWrite := make(chan struct{}), make(chan struct{})
	var collisionOnce sync.Once
	var beforeRoute string
	service.WithAssessmentAuthorizer(func(c context.Context, a identity.Actor, r CompletedResponseSummary, f formcontract.Field) (string, error) {
		route, e := originalAuthorizer(c, a, r, f)
		if e == nil {
			collisionOnce.Do(func() { beforeRoute = route; close(checked); <-continueWrite })
		}
		return route, e
	})
	collisionResult := make(chan error, 1)
	go func() {
		_, e := service.RecordResponseAssessment(ctx, tenant, entity, response, RecordResponseAssessmentInput{RequestKey: "00000000-0000-4000-8000-000000000001", Decisions: []FieldAssessmentInput{{FieldID: "score", OutcomeID: "poor", Rationale: "Report omits tested systems."}}})
		collisionResult <- e
	}()
	<-checked
	_, err = pool.Exec(ctx, `UPDATE routing_policies SET status='RETIRED',code='OLD-REVIEW' WHERE id=$1::uuid; UPDATE routing_policies SET status='ACTIVE',code='BANK-REVIEW' WHERE id=$2::uuid`, pgx.QueryExecModeSimpleProtocol, policy, replacement)
	if err == nil {
		newRoute, routeErr := authority.NewPostgresService(pool).Resolve(context.Background(), authority.ResolveInput{TenantID: tenant, LegalEntityID: entity, ObjectType: "FORM_RESPONSE", ObjectID: response, Responsibility: authority.ResponsibilityReviewer, DecisionType: "forms.response.assess", Materiality: 3})
		if routeErr != nil || beforeRoute != newRoute.PolicyVersion+":"+newRoute.RuleID || !newRoute.AllowsPrincipal(reviewer) || newRoute.AllowsPrincipalWithRole(reviewer, "RISK") {
			err = fmt.Errorf("invalid collision fixture: old=%s replacement=%+v err=%v", beforeRoute, newRoute, routeErr)
		}
	}
	close(continueWrite)
	if err != nil {
		t.Fatal(err)
	}
	if err = <-collisionResult; !errors.Is(err, ErrAssessmentForbidden) {
		t.Fatalf("replacement without reviewer role recorded assessment: %v", err)
	}
	if err = pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM capture_response_assessments WHERE tenant_id=$1::uuid)+(SELECT count(*) FROM capture_field_assessments WHERE tenant_id=$1::uuid)+(SELECT count(*) FROM capture_distribution_events WHERE tenant_id=$1::uuid AND event_type LIKE 'FORM_RESPONSE_ASSESSED_%')+(SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND event_type='FORM_RESPONSE_ASSESSED')`, tenant).Scan(&expiredWrites); err != nil || expiredWrites != 0 {
		t.Fatalf("role collision wrote %d records: %v", expiredWrites, err)
	}
	_, err = pool.Exec(ctx, `UPDATE routing_policies SET status='RETIRED',code='REPLACEMENT' WHERE id=$2::uuid; UPDATE routing_policies SET status='ACTIVE',code='BANK-REVIEW' WHERE id=$1::uuid`, pgx.QueryExecModeSimpleProtocol, policy, replacement)
	if err != nil {
		t.Fatal(err)
	}
	service.WithAssessmentAuthorizer(originalAuthorizer)
	// A recipient change must not race a review after its conflict check.
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		_, writeErr := service.store.(responseAssessmentStore).WriteResponseAssessment(ctx, tenant, entity, reviewer, response, RecordResponseAssessmentInput{}, func(context.Context, assessmentMaterial) bool { return true }, func(context.Context, assessmentMaterial) (assessmentSnapshot, []FieldAssessmentDecision, error) {
			close(entered)
			<-release
			return assessmentSnapshot{}, nil, ErrAssessmentForbidden
		})
		done <- writeErr
	}()
	select {
	case <-entered:
	case err := <-done:
		t.Fatalf("assessment lock failed: %v", err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	changeCtx, changeCancel := context.WithTimeout(ctx, 250*time.Millisecond)
	_, changeErr := pool.Exec(changeCtx, `UPDATE capture_requests SET title=title WHERE tenant_id=$1::uuid`, tenant)
	changeCancel()
	close(release)
	if err := <-done; !errors.Is(err, ErrAssessmentForbidden) {
		t.Fatalf("assessment rollback: %v", err)
	}
	if !errors.Is(changeErr, context.DeadlineExceeded) {
		t.Fatalf("request change did not wait on the assessment lock: %v", changeErr)
	}
	key := mustResponseWorkspaceID(t)
	input := RecordResponseAssessmentInput{RequestKey: key, Decisions: []FieldAssessmentInput{{FieldID: "score", OutcomeID: "poor", Rationale: "Report does not identify tested systems."}}}
	originalInput := input
	originalInput.Decisions = append([]FieldAssessmentInput(nil), input.Decisions...)
	type assessmentAttempt struct {
		value ResponseAssessment
		err   error
	}
	startAttempts := make(chan struct{})
	attempts := make(chan assessmentAttempt, 2)
	for i := 0; i < 2; i++ {
		go func() {
			<-startAttempts
			value, e := service.RecordResponseAssessment(ctx, tenant, entity, response, input)
			attempts <- assessmentAttempt{value, e}
		}()
	}
	close(startAttempts)
	first, second := <-attempts, <-attempts
	if first.err != nil || second.err != nil || first.value.Version != 1 || second.value.Version != 1 || first.value.Fields[0].Decision.ID != second.value.Fields[0].Decision.ID {
		t.Fatalf("concurrent keyed attempts: %+v / %+v", first, second)
	}
	v := first.value
	down, err := os.ReadFile("../../migrations/000155_response_assessment_request_key.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	rollbackConn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, rollbackErr := rollbackConn.Exec(ctx, string(down))
	_, _ = rollbackConn.Exec(ctx, "ROLLBACK")
	rollbackConn.Release()
	if rollbackErr == nil || !strings.Contains(rollbackErr.Error(), "assessment request receipts exist") {
		t.Fatalf("populated receipt rollback: %v", rollbackErr)
	}
	if v.Version != 1 || v.State != "ASSESSED" {
		t.Fatalf("assessment %+v", v)
	}
	var sourceID, sourceType, sourceRule, sourceVersion string
	var decisionTime, assessmentTime, eventTime, outboxTime time.Time
	err = pool.QueryRow(ctx, `SELECT decision->>'authority_source_id', decision->>'authority_source_type', decision->>'authority_rule_id', decision->>'authority_policy_version', f.assessed_at, a.created_at, e.occurred_at, o.occurred_at FROM capture_field_assessments f JOIN capture_response_assessments a ON a.tenant_id=f.tenant_id AND a.response_revision_id=f.response_revision_id AND a.version=f.assessment_version JOIN capture_distribution_events e ON e.tenant_id=f.tenant_id AND e.event_type='FORM_RESPONSE_ASSESSED_'||f.response_revision_id::text||'_'||f.assessment_version::text JOIN outbox_events o ON o.tenant_id=f.tenant_id AND o.aggregate_id=f.response_revision_id AND o.event_type='FORM_RESPONSE_ASSESSED' WHERE f.tenant_id=$1::uuid`, tenant).Scan(&sourceID, &sourceType, &sourceRule, &sourceVersion, &decisionTime, &assessmentTime, &eventTime, &outboxTime)
	if err != nil || sourceID != policy || sourceType != "POLICY" || sourceRule != "bank-review" || sourceVersion == "" || !decisionTime.Equal(assessmentTime) || !decisionTime.Equal(eventTime) || !decisionTime.Equal(outboxTime) {
		t.Fatalf("assessment receipt: source=%s/%s/%s/%s times=%v/%v/%v/%v err=%v", sourceType, sourceID, sourceRule, sourceVersion, decisionTime, assessmentTime, eventTime, outboxTime, err)
	}
	material, err := readResponseAssessmentMaterial(ctx, pool, tenant, entity, reviewer, response, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	replayTx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		t.Fatal(err)
	}
	replayActor, _ := identity.FromContext(ctx)
	replayRequest := assessmentReplayRequest{Key: key, ExpectedVersion: 0, Decisions: []FieldAssessmentInput{{FieldID: "score", OutcomeID: "poor", Rationale: "Report does not identify tested systems."}}}
	replayed, err := lookupAssessmentReplay(ctx, replayTx, replayActor, material, replayRequest)
	if err != nil || replayed.Version != 1 || replayed.Decisions["score"].ID != v.Fields[0].Decision.ID {
		t.Fatalf("authorized historical receipt: %+v %v", replayed, err)
	}
	changed := replayRequest
	changed.Decisions = []FieldAssessmentInput{{FieldID: "score", OutcomeID: "poor", Rationale: "Changed rationale."}}
	if _, err = lookupAssessmentReplay(ctx, replayTx, replayActor, material, changed); !errors.Is(err, ErrAssessmentConflict) {
		t.Fatalf("changed payload: %v", err)
	}
	changedVersion := replayRequest
	changedVersion.ExpectedVersion++
	if _, err = lookupAssessmentReplay(ctx, replayTx, replayActor, material, changedVersion); !errors.Is(err, ErrAssessmentConflict) {
		t.Fatalf("changed expected version: %v", err)
	}
	wrongResponse := material
	wrongResponse.Summary.ID = mustResponseWorkspaceID(t)
	wrongResponse.Revision.ID = wrongResponse.Summary.ID
	if _, err = lookupAssessmentReplay(ctx, replayTx, replayActor, wrongResponse, replayRequest); !errors.Is(err, ErrAssessmentForbidden) {
		t.Fatalf("cross-response receipt access: %v", err)
	}
	expiredActor := replayActor
	expiredActor.ExpiresAt = time.Now().Add(-time.Second)
	if _, err = lookupAssessmentReplay(ctx, replayTx, expiredActor, material, replayRequest); !errors.Is(err, ErrAssessmentForbidden) {
		t.Fatalf("expired receipt access: %v", err)
	}
	otherActor := replayActor
	otherActor.PrincipalID = actor
	if _, err = lookupAssessmentReplay(ctx, replayTx, otherActor, material, replayRequest); !errors.Is(err, ErrAssessmentForbidden) {
		t.Fatalf("cross-actor receipt access: %v", err)
	}
	_ = replayTx.Rollback(ctx)
	repeatable, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = lookupAssessmentReplay(ctx, repeatable, replayActor, material, replayRequest); !errors.Is(err, ErrAssessmentForbidden) {
		t.Fatalf("repeatable read receipt access: %v", err)
	}
	_ = repeatable.Rollback(ctx)
	if _, err = pool.Exec(ctx, `UPDATE capture_requests SET fields=jsonb_set(fields,'{0,assessment,reviewer_role}', '"OTHER"'::jsonb) WHERE tenant_id=$1::uuid`, tenant); err != nil {
		t.Fatal(err)
	}
	changedRoleTx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = lookupAssessmentReplay(ctx, changedRoleTx, replayActor, material, replayRequest); !errors.Is(err, ErrAssessmentForbidden) {
		t.Fatalf("changed field role exposed receipt: %v", err)
	}
	_ = changedRoleTx.Rollback(ctx)
	if _, err = pool.Exec(ctx, `UPDATE capture_requests SET fields=jsonb_set(fields,'{0,assessment,reviewer_role}', '"RISK"'::jsonb) WHERE tenant_id=$1::uuid`, tenant); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE routing_policies SET status='RETIRED' WHERE id=$1::uuid`, policy); err != nil {
		t.Fatal(err)
	}
	revokedTx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = lookupAssessmentReplay(ctx, revokedTx, replayActor, material, replayRequest); !errors.Is(err, ErrAssessmentForbidden) {
		t.Fatalf("revoked receipt access: %v", err)
	}
	_ = revokedTx.Rollback(ctx)
	if _, err = service.RecordResponseAssessment(ctx, tenant, entity, response, originalInput); !errors.Is(err, ErrAssessmentForbidden) {
		t.Fatalf("revoked command replay: %v", err)
	}
	if _, err = pool.Exec(ctx, `UPDATE routing_policies SET status='ACTIVE' WHERE id=$1::uuid`, policy); err != nil {
		t.Fatal(err)
	}
	var snapshotSource string
	if err = pool.QueryRow(ctx, `SELECT decisions->'score'->>'authority_source_id' FROM capture_response_assessments WHERE tenant_id=$1::uuid AND version=1`, tenant).Scan(&snapshotSource); err != nil || snapshotSource != policy || v.Fields[0].Decision == nil || v.Fields[0].Decision.AuthoritySourceID != policy || !v.Fields[0].Decision.AssessedAt.Equal(decisionTime) || v.AssessedScore == nil || !v.AssessedScore.CalculatedAt.Equal(decisionTime) {
		t.Fatalf("snapshot or response receipt differs: source=%q response=%+v err=%v", snapshotSource, v, err)
	}
	if replay, replayErr := service.RecordResponseAssessment(ctx, tenant, entity, response, input); replayErr != nil || replay.Version != v.Version || replay.Fields[0].Decision.ID != v.Fields[0].Decision.ID {
		t.Fatalf("replay=%+v err=%v", replay, replayErr)
	}
	changedInput := originalInput
	changedInput.Decisions = []FieldAssessmentInput{{FieldID: "score", OutcomeID: "good", Rationale: "Changed rationale."}}
	if _, err = service.RecordResponseAssessment(ctx, tenant, entity, response, changedInput); !errors.Is(err, ErrAssessmentConflict) {
		t.Fatalf("changed command replay: %v", err)
	}
	input.RequestKey = mustResponseWorkspaceID(t)
	var decisions, outbox int
	if err = pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM capture_field_assessments WHERE tenant_id=$1::uuid),(SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND event_type='FORM_RESPONSE_ASSESSED')`, tenant).Scan(&decisions, &outbox); err != nil || decisions != 1 || outbox != 1 {
		t.Fatalf("counts %d %d %v", decisions, outbox, err)
	}
	input.ExpectedVersion = 1
	input.Decisions[0].OutcomeID = "good"
	v2, err := service.RecordResponseAssessment(ctx, tenant, entity, response, input)
	if err != nil {
		t.Fatal(err)
	}
	if v2.Fields[0].Decision.SupersedesID != v.Fields[0].Decision.ID {
		t.Fatal("correction lost immutable predecessor")
	}
	oldResult, err := service.RecordResponseAssessment(ctx, tenant, entity, response, originalInput)
	if err != nil || oldResult.Version != 1 || oldResult.Fields[0].Decision.ID != v.Fields[0].Decision.ID {
		t.Fatalf("historical command receipt: %+v %v", oldResult, err)
	}
	historyTx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		t.Fatal(err)
	}
	older, err := lookupAssessmentReplay(ctx, historyTx, replayActor, material, replayRequest)
	if err != nil || older.Version != 1 || older.Decisions["score"].ID != v.Fields[0].Decision.ID {
		t.Fatalf("later version changed receipt: %+v %v", older, err)
	}
	_ = historyTx.Rollback(ctx)
	trigger := "bank_assessment_outbox_fail_" + strings.ReplaceAll(tenant, "-", "")
	_, err = pool.Exec(ctx, fmt.Sprintf(`CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.aggregate_id='%s'::uuid AND NEW.event_type='FORM_RESPONSE_ASSESSED' THEN RAISE EXCEPTION 'injected outbox failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER %s BEFORE INSERT ON outbox_events FOR EACH ROW EXECUTE FUNCTION %s()`, trigger, response, trigger, trigger), pgx.QueryExecModeSimpleProtocol)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), fmt.Sprintf(`DROP TRIGGER IF EXISTS %s ON outbox_events; DROP FUNCTION IF EXISTS %s()`, trigger, trigger), pgx.QueryExecModeSimpleProtocol)
	})
	input.ExpectedVersion = 2
	input.RequestKey = mustResponseWorkspaceID(t)
	if _, err = service.RecordResponseAssessment(ctx, tenant, entity, response, input); err == nil {
		t.Fatal("outbox failure committed an assessment")
	}
	var versions int
	if err = pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM capture_response_assessments WHERE tenant_id=$1::uuid),(SELECT count(*) FROM capture_field_assessments WHERE tenant_id=$1::uuid)`, tenant).Scan(&versions, &decisions); err != nil || versions != 2 || decisions != 2 {
		t.Fatalf("failed transaction left rows: versions=%d decisions=%d err=%v", versions, decisions, err)
	}
	_, err = pool.Exec(ctx, fmt.Sprintf(`DROP TRIGGER %s ON outbox_events; DROP FUNCTION %s()`, trigger, trigger), pgx.QueryExecModeSimpleProtocol)
	if err != nil {
		t.Fatal(err)
	}
	if retried, retryErr := service.RecordResponseAssessment(ctx, tenant, entity, response, input); retryErr != nil || retried.Version != 3 {
		t.Fatalf("retry after rolled-back write: %+v %v", retried, retryErr)
	}
	_, err = pool.Exec(ctx, `UPDATE capture_field_assessments SET decision='{}' WHERE tenant_id=$1::uuid`, tenant)
	if err == nil {
		t.Fatal("historical assessment was mutable")
	}
	service.WithAssessmentAuthorizer(func(context.Context, identity.Actor, CompletedResponseSummary, formcontract.Field) (string, error) {
		return "", errors.New("revoked")
	})
	input.ExpectedVersion = 3
	if _, err = service.RecordResponseAssessment(ctx, tenant, entity, response, input); !errors.Is(err, ErrAssessmentForbidden) {
		t.Fatal(err)
	}
}

func TestPostgresResponseAssessmentVendorReviewerDocuments(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	tenant, entity, owner, reviewer, form, vendor, relationship := mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t)
	now := time.Now().UTC()
	setupResponseWorkspaceFixture(t, ctx, pool, tenant, "review-docs-"+tenant, entity, owner, form, now)
	seedCompletedResponseRows(t, ctx, pool, tenant, entity, owner, form, "review-docs-"+tenant, 1, now)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `BEGIN; SET LOCAL session_replication_role='replica'; DELETE FROM capture_field_assessments WHERE tenant_id=$1::uuid; DELETE FROM capture_response_assessments WHERE tenant_id=$1::uuid; COMMIT`, pgx.QueryExecModeSimpleProtocol, tenant)
		_, _ = pool.Exec(context.Background(), `DELETE FROM position_role_bindings WHERE tenant_id=$1::uuid; DELETE FROM org_positions WHERE tenant_id=$1::uuid; DELETE FROM role_templates WHERE tenant_id=$1::uuid; DELETE FROM third_party_relationships WHERE tenant_id=$1::uuid; DELETE FROM third_parties WHERE tenant_id=$1::uuid`, pgx.QueryExecModeSimpleProtocol, tenant)
		cleanupResponseWorkspaceTenant(context.Background(), pool, tenant)
	})
	_, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($4::uuid,$1::uuid,'PERSON','Bank reviewer','ACTIVE',$8);
 INSERT INTO third_parties(id,tenant_id,legal_name,status,created_at,updated_at,version) VALUES($6::uuid,$1::uuid,'Sample review supplier','ACTIVE',$8,$8,1);
 INSERT INTO third_party_relationships(id,tenant_id,legal_entity_id,vendor_id,service_name,business_owner_principal_id,criticality,privacy_role,status,created_at,updated_at,version) VALUES($7::uuid,$1::uuid,$2::uuid,$6::uuid,'Processing',$3::uuid,'IMPORTANT','PROCESSOR','PROPOSED',$8,$8,1);
 UPDATE capture_form_distributions SET subject_type='VENDOR_RELATIONSHIP',subject_id=$7::uuid WHERE tenant_id=$1::uuid;
 UPDATE capture_requests SET form_template_id=$5::uuid,subject_type='VENDOR_RELATIONSHIP',subject_id=$7::uuid::text, fields='[{"id":"file","label":"Security report","type":"file","assessment":{"mode":"MANUAL","required":true,"weight":100,"reviewer_role":"RISK","rubric":[{"id":"poor","label":"Incomplete","points":80}]}}]' WHERE tenant_id=$1::uuid;
 UPDATE capture_submissions SET answers=jsonb_build_object('file',jsonb_build_object('artifact_ids',jsonb_build_array(md5('artifact:'||id::text)::uuid::text))) WHERE tenant_id=$1::uuid;
 INSERT INTO capture_artifacts(id,tenant_id,request_id,submission_id,file_name,media_type,size_bytes,sha256,storage_key,status,created_by,created_at) SELECT md5('artifact:'||id::text)::uuid,tenant_id,request_id,id,'report.pdf','application/pdf',4,repeat('a',64),'private-key/'||id::text,'STORED_UNSCANNED',$3::uuid,submitted_at FROM capture_submissions WHERE tenant_id=$1::uuid`, pgx.QueryExecModeSimpleProtocol, tenant, entity, owner, reviewer, form, vendor, relationship, now)
	if err != nil {
		t.Fatal(err)
	}
	var response string
	if err = pool.QueryRow(ctx, `SELECT id::text FROM capture_response_revisions WHERE tenant_id=$1::uuid`, tenant).Scan(&response); err != nil {
		t.Fatal(err)
	}
	policy, policyVersion := mustResponseWorkspaceID(t), mustResponseWorkspaceID(t)
	setupAssessmentRiskRole(t, ctx, pool, tenant, entity, reviewer, now)
	definition := fmt.Sprintf(`{"rules":[{"id":"review","legal_entity_id":"%s","object_type":"FORM_RESPONSE","object_id":"%s","responsibility":"REVIEWER","decision_type":"forms.response.assess","min_materiality":0,"priority":100,"selector":{"kind":"ROLE","ref":"RISK"}}]}`, entity, response)
	_, err = pool.Exec(ctx, `INSERT INTO routing_policies(id,tenant_id,legal_entity_id,code,name,status,current_version,approved_at,version) VALUES($1::uuid,$2::uuid,$3::uuid,'REVIEW','Review','DRAFT',1,$6,1); INSERT INTO routing_policy_versions(id,policy_id,legal_entity_id,version,definition,checksum,effective_from,approved_at) VALUES($4::uuid,$1::uuid,$3::uuid,1,$5::jsonb,'review-test',$6,$6); UPDATE routing_policies SET status='ACTIVE' WHERE id=$1::uuid`, pgx.QueryExecModeSimpleProtocol, policy, tenant, entity, policyVersion, definition, now.Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM routing_policies WHERE tenant_id=$1::uuid`, tenant)
	})
	router := authority.NewPostgresService(pool)
	service := NewDistributionService(NewPostgresDistributionStore(NewPostgresRepository(pool), nil)).WithAssessmentAuthorizer(func(c context.Context, a identity.Actor, r CompletedResponseSummary, f formcontract.Field) (string, error) {
		route, e := router.Resolve(c, authority.ResolveInput{TenantID: a.TenantID, LegalEntityID: r.LegalEntityID, ObjectType: "FORM_RESPONSE", ObjectID: r.ID, Responsibility: authority.ResponsibilityReviewer, DecisionType: "forms.response.assess", Materiality: 3})
		if e != nil || !route.AllowsPrincipalWithRole(a.PrincipalID, f.Assessment.ReviewerRole) {
			return "", ErrAssessmentForbidden
		}
		return route.PolicyVersion + ":" + route.RuleID, nil
	})
	service.assessmentDecisionsEnabled = true // Exercise the unexposed transaction in isolation.
	ctx = identity.WithActor(ctx, identity.Actor{Kind: "PERSON", TenantID: tenant, LegalEntityID: entity, PrincipalID: reviewer, ExpiresAt: now.Add(time.Hour)})
	assessment, err := service.GetResponseAssessment(ctx, tenant, entity, reviewer, response)
	if err != nil || !assessment.MayReview {
		t.Fatalf("reviewer assessment %+v %v", assessment, err)
	}
	q := DocumentQuery{TenantID: tenant, LegalEntityID: entity, PrincipalID: reviewer, ResponseRevisionID: response, Limit: 10}
	page, err := service.ListDocuments(ctx, q)
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("reviewer inventory %+v %v", page, err)
	}
	for _, view := range []DocumentView{DocumentViewAwaitingReview, DocumentViewReviewHistory, DocumentViewNeedsRefresh} {
		q.View = view
		filtered, filterErr := service.ListDocuments(ctx, q)
		if filterErr != nil || len(filtered.Items) != 0 {
			t.Fatalf("unknown review in %s: %+v %v", view, filtered, filterErr)
		}
	}
	if _, err = pool.Exec(ctx, `UPDATE capture_submissions SET answers=jsonb_set(answers,'{file,document}', '{"expires_on":"2020-01-01"}'::jsonb) WHERE tenant_id=$1::uuid`, tenant); err != nil {
		t.Fatal(err)
	}
	q.View = DocumentViewNeedsRefresh
	filtered, filterErr := service.ListDocuments(ctx, q)
	if filterErr != nil || len(filtered.Items) != 1 {
		t.Fatalf("expired document: %+v %v", filtered, filterErr)
	}
	if _, err = pool.Exec(ctx, `UPDATE capture_submissions SET answers=jsonb_set(answers,'{file,document}', '{"expires_on":"2020-02-30"}'::jsonb) WHERE tenant_id=$1::uuid`, tenant); err != nil {
		t.Fatal(err)
	}
	filtered, filterErr = service.ListDocuments(ctx, q)
	if filterErr != nil || len(filtered.Items) != 0 {
		t.Fatalf("invalid expiry: %+v %v", filtered, filterErr)
	}
	q.View = ""
	q.SubmissionID = page.Items[0].SubmissionID
	q.FieldID = page.Items[0].FieldID
	q.ArtifactID = page.Items[0].ArtifactID
	page, err = service.ListDocuments(ctx, q)
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("reviewer exact protected occurrence %+v %v", page, err)
	}
	q.ResponseRevisionID = ""
	page, err = service.ListDocuments(ctx, q)
	if err != nil || len(page.Items) != 0 {
		t.Fatalf("unscoped reviewer inventory %+v %v", page, err)
	}
	q.ResponseRevisionID = response
	updated, err := service.RecordResponseAssessment(ctx, tenant, entity, response, RecordResponseAssessmentInput{RequestKey: "00000000-0000-4000-8000-000000000001", Decisions: []FieldAssessmentInput{{FieldID: "file", OutcomeID: "poor", Rationale: "Report omits tested systems."}}})
	if err != nil || updated.Version != 1 || !updated.MayReview {
		t.Fatalf("non-owner reviewer cannot record: %+v %v", updated, err)
	}
	for _, status := range []string{"REVOKED", "SUPERSEDED"} {
		if _, err = pool.Exec(ctx, `UPDATE capture_form_distributions SET status=$2 WHERE tenant_id=$1::uuid`, tenant, status); err != nil {
			t.Fatal(err)
		}
		retired, err := service.GetResponseAssessment(ctx, tenant, entity, reviewer, response)
		if err != nil || retired.Current || retired.MayReview {
			t.Fatalf("retired review %+v %v", retired, err)
		}
		_, err = service.RecordResponseAssessment(ctx, tenant, entity, response, RecordResponseAssessmentInput{RequestKey: "00000000-0000-4000-8000-000000000001", ExpectedVersion: 1, Decisions: []FieldAssessmentInput{{FieldID: "file", OutcomeID: "poor", Rationale: "Must not save."}}})
		if !errors.Is(err, ErrAssessmentConflict) {
			t.Fatalf("retired distribution write: %v", err)
		}
	}
	service.WithAssessmentAuthorizer(nil)
	page, err = service.ListDocuments(ctx, q)
	if err != nil || len(page.Items) != 0 {
		t.Fatalf("revoked reviewer inventory %+v %v", page, err)
	}
}
