//go:build postgres

package evidence

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/federation"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5"
)

// responseReviewEligibilitySQL is shared by the write and receipt reads so
// both evaluate current actor, route, scope and visibility in one statement.
func responseReviewEligibilitySQL(extraJoin string) string {
	return `FROM selected s JOIN effective e USING(request_index)
JOIN capture_response_revisions r ON r.id=$8::uuid
JOIN capture_form_distributions d ON d.id=r.distribution_id AND d.tenant_id=r.tenant_id AND d.legal_entity_id=r.legal_entity_id
JOIN capture_submissions submission ON submission.id=r.submission_id AND submission.tenant_id=r.tenant_id
JOIN capture_requests req ON req.id=submission.request_id AND req.tenant_id=r.tenant_id AND req.legal_entity_id=r.legal_entity_id
JOIN principals actor ON actor.id=$6::uuid AND actor.tenant_id=r.tenant_id AND actor.kind='PERSON'
JOIN legal_entities scope ON scope.id=r.legal_entity_id AND scope.tenant_id=r.tenant_id AND scope.valid_from<=statement_timestamp() AND (scope.valid_until IS NULL OR statement_timestamp()<scope.valid_until)
` + extraJoin + `
WHERE current_setting('transaction_isolation')='read committed' AND s.request_index=0 AND NOT s.ambiguous AND e.principal_id=actor.id
 AND r.tenant_id=$9::uuid AND r.legal_entity_id=$10::uuid
 AND r.tenant_id=(SELECT CASE WHEN count(*)=1 THEN min(id::text)::uuid ELSE NULL END FROM tenants WHERE id::text=$11 OR slug=$11)
 AND ($12='*' OR r.legal_entity_id::text=$12)
 AND $13::timestamptz>statement_timestamp() AND $13::timestamptz>clock_timestamp()
 AND ($14::timestamptz IS NULL OR $14::timestamptz<=statement_timestamp()+interval '1 minute')
 AND COALESCE(req.origin_type,'') NOT IN ('THIRD_PARTY_WORK','THIRD_PARTY_ASSESSMENT')
 AND d.subject_type IN ('PROGRAM','MATTER','VENDOR_RELATIONSHIP') AND ` + documentRevisionScopeSQL() + `
 AND ((` + completedResponseSubjectVisibilitySQL("$6::uuid::text", "statement_timestamp()") + `) OR (` + vendorAssessmentCandidateSQL() + `))
 AND COALESCE(submission.submitted_by::text,'')<>$6::uuid::text
 AND (COALESCE(req.recipient_type,'')<>'INTERNAL_PRINCIPAL' OR COALESCE(req.recipient_principal_id::text,'')<>$6::uuid::text)
 AND COALESCE(r.submitted_recipient_principal_id::text,'')<>$6::uuid::text
 AND (r.submitted_recipient_id IS NULL OR EXISTS(SELECT 1 FROM capture_distribution_recipients recipient WHERE recipient.id=r.submitted_recipient_id AND recipient.tenant_id=r.tenant_id AND recipient.legal_entity_id=r.legal_entity_id AND recipient.distribution_id=r.distribution_id AND recipient.request_id=submission.request_id AND recipient.role='TO'))`
}

// insertCurrentResponseReview is called after locking the response,
// distribution and request. The decision and exact route receipt share one
// statement snapshot; event and outbox still belong to the caller's tx.
func insertCurrentResponseReview(ctx context.Context, tx pgx.Tx, actor identity.Actor, material assessmentMaterial, review ResponseReview, requestKey string, session federation.OIDCSessionProof) (ResponseReview, error) {
	if tx == nil || !strings.EqualFold(strings.TrimSpace(actor.Kind), "PERSON") || actor.PrincipalID != review.ReviewerID || material.Revision.ID != review.ResponseID || material.Summary.ID != material.Revision.ID || material.Revision.Revision != review.ResponseRevision {
		return ResponseReview{}, ErrAssessmentForbidden
	}
	request, err := authority.PostgresCurrentRouteRequest(authority.ResolveInput{TenantID: material.Summary.TenantID, LegalEntityID: material.Summary.LegalEntityID, ObjectType: "FORM_RESPONSE", ObjectID: material.Revision.ID, Responsibility: authority.ResponsibilityReviewer, DecisionType: "forms.response.review", Materiality: 3})
	if err != nil {
		return ResponseReview{}, ErrAssessmentForbidden
	}
	var issuedAt *time.Time
	if !actor.IssuedAt.IsZero() {
		issuedAt = &actor.IssuedAt
	}
	statement := authority.PostgresCurrentRouteCTEs() + `
INSERT INTO capture_response_reviews(id,tenant_id,legal_entity_id,response_revision_id,response_revision,version,decision,comment,reviewer_id,authority_route,reviewed_at,supersedes_id,authority_source_type,authority_source_id,authority_rule_id,authority_policy_version,request_key)
SELECT DISTINCT $2::uuid,r.tenant_id,r.legal_entity_id,r.id,r.revision,$3::bigint,$4,$5,$6::uuid,s.policy_version||':'||s.rule_id,statement_timestamp(),NULLIF($7,'')::uuid,
 CASE WHEN left(s.source_id,11)='assignment:' THEN 'ASSIGNMENT' ELSE 'POLICY' END,
 (CASE WHEN left(s.source_id,11)='assignment:' THEN substring(s.source_id FROM 12) ELSE s.source_id END)::uuid,s.rule_id,s.policy_version,NULLIF($16,'')::uuid
` + responseReviewEligibilitySQL("") + `
 AND r.is_current AND r.state IN ('FINAL','PROVISIONAL') AND d.status NOT IN ('REVOKED','SUPERSEDED')
 AND r.revision=$15 AND COALESCE((SELECT max(version) FROM capture_response_reviews prior WHERE prior.tenant_id=r.tenant_id AND prior.legal_entity_id=r.legal_entity_id AND prior.response_revision_id=r.id),0)=$3::bigint-1
 AND COALESCE((SELECT id::text FROM capture_response_reviews prior WHERE prior.tenant_id=r.tenant_id AND prior.legal_entity_id=r.legal_entity_id AND prior.response_revision_id=r.id ORDER BY version DESC LIMIT 1),'')=$7
 AND EXISTS(SELECT 1 FROM web_sessions staff_session WHERE staff_session.token=$17 AND staff_session.data=$18::bytea AND staff_session.expiry>statement_timestamp() AND staff_session.expiry>clock_timestamp())
RETURNING authority_route,reviewed_at,authority_source_type,authority_source_id::text,authority_rule_id,authority_policy_version`
	err = tx.QueryRow(ctx, statement, string(request), review.ID, review.Version, review.Decision, review.Comment, actor.PrincipalID, review.SupersedesID, material.Revision.ID, material.Summary.TenantID, material.Summary.LegalEntityID, actor.TenantID, actor.LegalEntityID, actor.ExpiresAt, issuedAt, material.Revision.Revision, requestKey, session.TokenHash, session.Data).Scan(&review.AuthorityRoute, &review.ReviewedAt, &review.AuthoritySourceType, &review.AuthoritySourceID, &review.AuthorityRuleID, &review.AuthorityPolicyVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return ResponseReview{}, ErrAssessmentForbidden
	}
	if err != nil {
		return ResponseReview{}, err
	}
	return review, nil
}

// replayResponseReview returns a historical decision only while the same
// verified actor remains an eligible reviewer with current subject visibility.
func replayResponseReview(ctx context.Context, tx pgx.Tx, actor identity.Actor, material assessmentMaterial, input RecordResponseReviewInput, session federation.OIDCSessionProof) (ResponseReview, error) {
	if tx == nil || !strings.EqualFold(strings.TrimSpace(actor.Kind), "PERSON") {
		return ResponseReview{}, ErrAssessmentForbidden
	}
	request, err := authority.PostgresCurrentRouteRequest(authority.ResolveInput{TenantID: material.Summary.TenantID, LegalEntityID: material.Summary.LegalEntityID, ObjectType: "FORM_RESPONSE", ObjectID: material.Revision.ID, Responsibility: authority.ResponsibilityReviewer, DecisionType: "forms.response.review", Materiality: 3})
	if err != nil {
		return ResponseReview{}, ErrAssessmentForbidden
	}
	var issuedAt *time.Time
	if !actor.IssuedAt.IsZero() {
		issuedAt = &actor.IssuedAt
	}
	statement := authority.PostgresCurrentRouteCTEs() + `
 SELECT DISTINCT prior.id::text,prior.response_revision_id::text,prior.response_revision,prior.version,prior.decision,prior.comment,prior.reviewer_id::text,prior.authority_route,prior.reviewed_at,COALESCE(prior.supersedes_id::text,''),COALESCE(prior.authority_source_type,''),COALESCE(prior.authority_source_id::text,''),COALESCE(prior.authority_rule_id,''),COALESCE(prior.authority_policy_version,''),
 (prior.version=$3::bigint+1 AND prior.decision=$4 AND prior.comment=$5) AS matches
 ` + responseReviewEligibilitySQL("JOIN capture_response_reviews prior ON prior.tenant_id=r.tenant_id AND prior.legal_entity_id=r.legal_entity_id AND prior.response_revision_id=r.id AND prior.reviewer_id=actor.id AND prior.request_key=$2::uuid") + `
 AND r.revision=$15::bigint AND $7::text='' AND EXISTS(SELECT 1 FROM web_sessions staff_session WHERE staff_session.token=$16 AND staff_session.data=$17::bytea AND staff_session.expiry>statement_timestamp() AND staff_session.expiry>clock_timestamp())
 `
	var review ResponseReview
	var matches bool
	err = tx.QueryRow(ctx, statement, string(request), input.RequestKey, input.ExpectedVersion, input.Decision, input.Comment, actor.PrincipalID, "", material.Revision.ID, material.Summary.TenantID, material.Summary.LegalEntityID, actor.TenantID, actor.LegalEntityID, actor.ExpiresAt, issuedAt, material.Revision.Revision, session.TokenHash, session.Data).Scan(&review.ID, &review.ResponseID, &review.ResponseRevision, &review.Version, &review.Decision, &review.Comment, &review.ReviewerID, &review.AuthorityRoute, &review.ReviewedAt, &review.SupersedesID, &review.AuthoritySourceType, &review.AuthoritySourceID, &review.AuthorityRuleID, &review.AuthorityPolicyVersion, &matches)
	if errors.Is(err, pgx.ErrNoRows) {
		return ResponseReview{}, ErrAssessmentForbidden
	}
	if err != nil {
		return ResponseReview{}, err
	}
	if !matches {
		return ResponseReview{}, ErrAssessmentConflict
	}
	return review, nil
}
