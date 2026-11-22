//go:build postgres && postgresintegration

package evidence

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5"
)

func TestPostgresProgramReviewPageScopeAndDecisionEligibility(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	tenant, entity, owner, reviewer, other, form, program := mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t)
	now := time.Now().UTC()
	prefix := "program-review-page-" + tenant
	setupResponseWorkspaceFixture(t, ctx, pool, tenant, prefix, entity, owner, form, now)
	seedCompletedResponseRows(t, ctx, pool, tenant, entity, owner, form, prefix, 4, now)
	// Review and event rows are immutable. This test uses a unique tenant in an
	// isolated test database and retains its complete audit history.
	_, err := pool.Exec(ctx, `
		INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($4::uuid,$1::uuid,'PERSON','Reviewer','ACTIVE',$8),($5::uuid,$1::uuid,'PERSON','Other','ACTIVE',$8);
		INSERT INTO programs(id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,jurisdiction,scope,effective_from)
		VALUES($7::uuid,$1::uuid,$2::uuid,'P','Program','COMPLIANCE','ACTIVE','Risk','NG',jsonb_build_object('access','RESTRICTED','allowed_principal_ids',jsonb_build_array($3::text,$4::text)),$8);
		UPDATE capture_form_distributions SET subject_type='PROGRAM',subject_id=$7::uuid WHERE tenant_id=$1::uuid;
		UPDATE capture_requests SET subject_type='PROGRAM',subject_id=$7::text WHERE tenant_id=$1::uuid;
		UPDATE capture_submissions SET answers='{"score":{"text":"3"}}'::jsonb WHERE tenant_id=$1::uuid;
		INSERT INTO responsibility_assignments(tenant_id,legal_entity_id,principal_id,responsibility,object_type,object_id,decision_type,scope,priority,valid_from,policy_version)
		SELECT $1::uuid,$2::uuid,$4::uuid,'REVIEWER','FORM_RESPONSE',md5($6||':response:'||g)::uuid,'forms.response.review','{}',10,$8,'route-v1' FROM unnest(ARRAY[1,3]) g`, pgx.QueryExecModeSimpleProtocol, tenant, entity, owner, reviewer, other, prefix, program, now.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	store := NewPostgresDistributionStore(NewPostgresRepository(pool), nil)
	actor := identity.Actor{TenantID: tenant, LegalEntityID: entity, PrincipalID: reviewer, Kind: "PERSON", ExpiresAt: now.Add(time.Hour)}
	query := ProgramReviewQuery{TenantID: tenant, LegalEntityID: entity, Limit: 1}
	first, err := store.listPendingProgramReviews(ctx, actor, query)
	if err != nil || len(first.Items) != 1 || first.NextCursor == "" {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	if _, _, err := store.GetCompletedResponse(ctx, tenant, entity, reviewer, first.Items[0].ID); err != nil {
		t.Fatalf("list/detail mismatch: %v", err)
	}
	query.Cursor = first.NextCursor
	_, err = pool.Exec(ctx, `UPDATE responsibility_assignments SET valid_until=$2 WHERE tenant_id=$1::uuid`, tenant, now.Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	revoked, err := store.listPendingProgramReviews(ctx, actor, query)
	if err != nil || len(revoked.Items) != 0 || revoked.NextCursor != "" {
		t.Fatalf("revoked second page=%+v err=%v", revoked, err)
	}
	_, err = pool.Exec(ctx, `UPDATE responsibility_assignments SET valid_until=NULL WHERE tenant_id=$1::uuid`, tenant)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.listPendingProgramReviews(ctx, actor, query)
	if err != nil || len(second.Items) != 1 || second.NextCursor != "" || second.Items[0].ID == first.Items[0].ID {
		t.Fatalf("second=%+v err=%v", second, err)
	}
	query.Cursor = ""
	_, err = pool.Exec(ctx, `UPDATE programs SET scope=jsonb_build_object('access','RESTRICTED','allowed_principal_ids',jsonb_build_array($2::text)) WHERE id=$1::uuid`, program, owner)
	if err != nil {
		t.Fatal(err)
	}
	restricted, err := store.listPendingProgramReviews(ctx, actor, query)
	if err != nil || len(restricted.Items) != 0 {
		t.Fatalf("restricted pending page=%+v err=%v", restricted, err)
	}
	_, err = pool.Exec(ctx, `UPDATE programs SET scope=jsonb_build_object('access','RESTRICTED','allowed_principal_ids',jsonb_build_array($2::text,$3::text)) WHERE id=$1::uuid`, program, owner, reviewer)
	if err != nil {
		t.Fatal(err)
	}
	// A discovered revision must be actionable through the existing exact review
	// service, not merely readable in the listing.
	resolver := authority.NewPostgresService(pool)
	service := NewDistributionService(store).WithResponseReviewAuthorizer(func(ctx context.Context, a identity.Actor, summary CompletedResponseSummary) (string, error) {
		route, routeErr := resolver.Resolve(ctx, authority.ResolveInput{TenantID: a.TenantID, LegalEntityID: summary.LegalEntityID, ObjectType: "FORM_RESPONSE", ObjectID: summary.ID, Responsibility: authority.ResponsibilityReviewer, DecisionType: "forms.response.review", Materiality: 3})
		if routeErr != nil || !route.AllowsPrincipal(a.PrincipalID) {
			return "", ErrAssessmentForbidden
		}
		return route.PolicyVersion + ":" + route.RuleID, nil
	})
	service.demoResponseReviewDecisions = true // Exercise the unexposed transaction in isolation.
	actorCtx := assessmentOIDCContext(t, pool, tenant, entity, reviewer, time.Now().Add(time.Hour))
	answers, err := service.GetCompletedResponseAnswers(actorCtx, tenant, entity, reviewer, first.Items[0].ID)
	if err != nil || len(answers.Answers) != 1 {
		t.Fatalf("response answers=%+v err=%v", answers, err)
	}
	documents, err := service.ListDocuments(actorCtx, DocumentQuery{TenantID: tenant, LegalEntityID: entity, PrincipalID: reviewer, ResponseRevisionID: first.Items[0].ID, Limit: 100})
	if err != nil || len(documents.Items) != 0 {
		t.Fatalf("response documents=%+v err=%v", documents, err)
	}
	history, err := service.GetResponseReviews(actorCtx, tenant, entity, reviewer, first.Items[0].ID, 0, 20)
	if err != nil || !history.MayReview {
		t.Fatalf("review history=%+v err=%v", history, err)
	}
	_, err = service.RecordResponseReview(actorCtx, tenant, entity, first.Items[0].ID, RecordResponseReviewInput{Decision: "REVIEWED", ExpectedVersion: history.Version, RequestKey: mustResponseWorkspaceID(t)})
	if err != nil {
		t.Fatalf("record review=%v", err)
	}
	query.Cursor = ""
	afterReview, err := store.listPendingProgramReviews(ctx, actor, query)
	if err != nil || len(afterReview.Items) != 1 || afterReview.Items[0].ID != second.Items[0].ID || afterReview.NextCursor != "" {
		t.Fatalf("after review=%+v err=%v", afterReview, err)
	}
	// A submitter, including a named internal recipient, cannot review their own response.
	_, err = pool.Exec(ctx, `UPDATE capture_submissions SET submitted_by=$2::uuid WHERE id=(SELECT submission_id FROM capture_response_revisions WHERE id=$1::uuid)`, second.Items[0].ID, reviewer)
	if err != nil {
		t.Fatal(err)
	}
	self, err := store.listPendingProgramReviews(ctx, actor, query)
	if err != nil || len(self.Items) != 0 {
		t.Fatalf("submitter page=%+v err=%v", self, err)
	}
	_, err = pool.Exec(ctx, `UPDATE capture_submissions SET submitted_by=$2::uuid WHERE id=(SELECT submission_id FROM capture_response_revisions WHERE id=$1::uuid)`, second.Items[0].ID, owner)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `UPDATE capture_requests SET recipient_type='INTERNAL_PRINCIPAL',recipient_principal_id=$2::uuid,recipient_state='ASSIGNED',recipient_revision=1 WHERE id=(SELECT submission.request_id FROM capture_response_revisions revision JOIN capture_submissions submission ON submission.id=revision.submission_id WHERE revision.id=$1::uuid)`, second.Items[0].ID, reviewer)
	if err != nil {
		t.Fatal(err)
	}
	self, err = store.listPendingProgramReviews(ctx, actor, query)
	if err != nil || len(self.Items) != 0 {
		t.Fatalf("recipient page=%+v err=%v", self, err)
	}
	_, err = pool.Exec(ctx, `UPDATE capture_requests SET recipient_type=NULL,recipient_principal_id=NULL,recipient_state='LEGACY_UNASSIGNED',recipient_revision=0 WHERE id=(SELECT submission.request_id FROM capture_response_revisions revision JOIN capture_submissions submission ON submission.id=revision.submission_id WHERE revision.id=$1::uuid); UPDATE capture_response_revisions SET state='PROVISIONAL' WHERE id=$1::uuid`, pgx.QueryExecModeSimpleProtocol, second.Items[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	provisional, err := store.listPendingProgramReviews(ctx, actor, query)
	if err != nil || len(provisional.Items) != 1 || provisional.Items[0].State != ResponseRevisionProvisional {
		t.Fatalf("provisional=%+v err=%v", provisional, err)
	}
	provisionalHistory, err := service.GetResponseReviews(actorCtx, tenant, entity, reviewer, second.Items[0].ID, 0, 20)
	if err != nil || !provisionalHistory.MayReview {
		t.Fatalf("provisional history=%+v err=%v", provisionalHistory, err)
	}
	if _, err = service.RecordResponseReview(actorCtx, tenant, entity, second.Items[0].ID, RecordResponseReviewInput{Decision: "CHANGES_REQUESTED", Comment: "Please update the response.", ExpectedVersion: provisionalHistory.Version, RequestKey: mustResponseWorkspaceID(t)}); err != nil {
		t.Fatalf("provisional review=%v", err)
	}
	query.Cursor = ""
	actor.PrincipalID = other
	denied, err := store.listPendingProgramReviews(ctx, actor, query)
	if err != nil || len(denied.Items) != 0 || denied.NextCursor != "" {
		t.Fatalf("other=%+v err=%v", denied, err)
	}
	actor.PrincipalID = reviewer
	_, err = pool.Exec(ctx, `UPDATE programs SET scope=jsonb_build_object('access','RESTRICTED','allowed_principal_ids',jsonb_build_array($2::text)) WHERE id=$1::uuid`, program, owner)
	if err != nil {
		t.Fatal(err)
	}
	denied, err = store.listPendingProgramReviews(ctx, actor, query)
	if err != nil || len(denied.Items) != 0 {
		t.Fatalf("restricted=%+v err=%v", denied, err)
	}
	if _, _, err := store.GetCompletedResponse(ctx, tenant, entity, reviewer, first.Items[0].ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("restricted detail err=%v", err)
	}
	actor.ExpiresAt = now.Add(-time.Minute)
	denied, err = store.listPendingProgramReviews(ctx, actor, query)
	if err != nil || len(denied.Items) != 0 {
		t.Fatalf("expired=%+v err=%v", denied, err)
	}
}
