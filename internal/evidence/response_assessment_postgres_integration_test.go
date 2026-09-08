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
	"strings"
	"testing"
	"time"
)

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
		cleanupResponseWorkspaceTenant(context.Background(), pool, tenant)
	})
	_, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($1::uuid,$2::uuid,'PERSON','Bank reviewer','ACTIVE',$3); UPDATE capture_requests SET fields='[{"id":"score","section_id":"general","label":"Evidence quality","type":"short_text","assessment":{"mode":"MANUAL","required":true,"weight":100,"reviewer_role":"RISK","rubric":[{"id":"poor","label":"Incomplete","points":80},{"id":"good","label":"Sufficient","points":10}]}}]'::jsonb WHERE tenant_id=$2::uuid; UPDATE capture_requests req SET subject_id=d.subject_id::text FROM capture_form_distributions d WHERE req.distribution_id=d.id AND req.tenant_id=$2::uuid; UPDATE capture_submissions SET answers='{"score":{"text":"Report supplied"}}'::jsonb WHERE tenant_id=$2::uuid`, pgx.QueryExecModeSimpleProtocol, reviewer, tenant, now)
	if err != nil {
		t.Fatal(err)
	}
	var response string
	if err = pool.QueryRow(ctx, `SELECT id::text FROM capture_response_revisions WHERE tenant_id=$1::uuid`, tenant).Scan(&response); err != nil {
		t.Fatal(err)
	}
	config := pool.Config()
	config.MaxConns = 1
	assessmentPool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer assessmentPool.Close()
	policy, policyVersion := mustResponseWorkspaceID(t), mustResponseWorkspaceID(t)
	definition := fmt.Sprintf(`{"rules":[{"id":"bank-review","legal_entity_id":"%s","object_type":"FORM_RESPONSE","object_id":"%s","responsibility":"REVIEWER","decision_type":"forms.response.assess","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"%s"}}]}`, entity, response, reviewer)
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
		if !route.AllowsPrincipal(a.PrincipalID) {
			return "", ErrAssessmentForbidden
		}
		return route.PolicyVersion + ":" + route.RuleID, nil
	})
	ctx = identity.WithActor(ctx, identity.Actor{TenantID: tenant, LegalEntityID: entity, PrincipalID: reviewer, ExpiresAt: now.Add(time.Hour)})
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	input := RecordResponseAssessmentInput{Decisions: []FieldAssessmentInput{{FieldID: "score", OutcomeID: "poor", Rationale: "Report does not identify tested systems."}}}
	v, err := service.RecordResponseAssessment(ctx, tenant, entity, response, input)
	if err != nil {
		t.Fatal(err)
	}
	if v.Version != 1 || v.State != "ASSESSED" {
		t.Fatalf("assessment %+v", v)
	}
	if _, err = service.RecordResponseAssessment(ctx, tenant, entity, response, input); !errors.Is(err, ErrAssessmentConflict) {
		t.Fatalf("replay=%v", err)
	}
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
	trigger := "bank_assessment_outbox_fail_" + strings.ReplaceAll(tenant, "-", "")
	_, err = pool.Exec(ctx, fmt.Sprintf(`CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.aggregate_id='%s'::uuid AND NEW.event_type='FORM_RESPONSE_ASSESSED' THEN RAISE EXCEPTION 'injected outbox failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER %s BEFORE INSERT ON outbox_events FOR EACH ROW EXECUTE FUNCTION %s()`, trigger, response, trigger, trigger), pgx.QueryExecModeSimpleProtocol)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), fmt.Sprintf(`DROP TRIGGER IF EXISTS %s ON outbox_events; DROP FUNCTION IF EXISTS %s()`, trigger, trigger), pgx.QueryExecModeSimpleProtocol)
	})
	input.ExpectedVersion = 2
	if _, err = service.RecordResponseAssessment(ctx, tenant, entity, response, input); err == nil {
		t.Fatal("outbox failure committed an assessment")
	}
	var versions int
	if err = pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM capture_response_assessments WHERE tenant_id=$1::uuid),(SELECT count(*) FROM capture_field_assessments WHERE tenant_id=$1::uuid)`, tenant).Scan(&versions, &decisions); err != nil || versions != 2 || decisions != 2 {
		t.Fatalf("failed transaction left rows: versions=%d decisions=%d err=%v", versions, decisions, err)
	}
	_, err = pool.Exec(ctx, `UPDATE capture_field_assessments SET decision='{}' WHERE tenant_id=$1::uuid`, tenant)
	if err == nil {
		t.Fatal("historical assessment was mutable")
	}
	service.WithAssessmentAuthorizer(func(context.Context, identity.Actor, CompletedResponseSummary, formcontract.Field) (string, error) {
		return "", errors.New("revoked")
	})
	input.ExpectedVersion = 2
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
		_, _ = pool.Exec(context.Background(), `DELETE FROM third_party_relationships WHERE tenant_id=$1::uuid; DELETE FROM third_parties WHERE tenant_id=$1::uuid`, pgx.QueryExecModeSimpleProtocol, tenant)
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
	service := NewDistributionService(NewPostgresDistributionStore(NewPostgresRepository(pool), nil)).WithAssessmentAuthorizer(func(context.Context, identity.Actor, CompletedResponseSummary, formcontract.Field) (string, error) {
		return "current-reviewer-route", nil
	})
	ctx = identity.WithActor(ctx, identity.Actor{TenantID: tenant, LegalEntityID: entity, PrincipalID: reviewer, ExpiresAt: now.Add(time.Hour)})
	assessment, err := service.GetResponseAssessment(ctx, tenant, entity, reviewer, response)
	if err != nil || !assessment.MayReview {
		t.Fatalf("reviewer assessment %+v %v", assessment, err)
	}
	q := DocumentQuery{TenantID: tenant, LegalEntityID: entity, PrincipalID: reviewer, ResponseRevisionID: response, Limit: 10}
	page, err := service.ListDocuments(ctx, q)
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("reviewer inventory %+v %v", page, err)
	}
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
	updated, err := service.RecordResponseAssessment(ctx, tenant, entity, response, RecordResponseAssessmentInput{Decisions: []FieldAssessmentInput{{FieldID: "file", OutcomeID: "poor", Rationale: "Report omits tested systems."}}})
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
		_, err = service.RecordResponseAssessment(ctx, tenant, entity, response, RecordResponseAssessmentInput{ExpectedVersion: 1, Decisions: []FieldAssessmentInput{{FieldID: "file", OutcomeID: "poor", Rationale: "Must not save."}}})
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
