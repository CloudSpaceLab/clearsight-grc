//go:build postgres

package evidence

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

type assessmentRowQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

const assessmentLatestSubmission = `SELECT latest.id FROM capture_submissions latest
 WHERE latest.tenant_id=req.tenant_id AND latest.request_id=req.id
 ORDER BY latest.assessment_revision DESC,latest.submitted_at DESC,latest.id DESC LIMIT 1`

const assessmentCorrectionScope = `req.status='SUBMITTED' AND req.subject_type='PROGRAM'
 AND req.audience_type='INTERNAL' AND req.form_template_id IS NOT NULL AND req.distribution_id IS NULL
 AND req.recipient_state='ASSIGNED' AND sub.channel='INTERNAL' AND sub.session_id IS NULL
 AND sub.submitted_by=req.recipient_principal_id AND review.status='FOLLOW_UP_REQUIRED'
 AND COALESCE(review.correction_due_at,req.deadline)>clock_timestamp()
 AND review.reviewed_by::text=req.known_facts->>'reviewer'
 AND review.reviewed_by<>req.created_by AND review.reviewed_by<>req.recipient_principal_id
 AND EXISTS(SELECT 1 FROM principals p WHERE p.tenant_id=req.tenant_id AND p.id=req.recipient_principal_id
 AND p.kind='PERSON' AND p.status='ACTIVE' AND p.valid_from<=clock_timestamp()
 AND (p.valid_until IS NULL OR p.valid_until>clock_timestamp()))`

const assessmentCurrentReviewJoin = `
 LEFT JOIN LATERAL (SELECT r.* FROM capture_submission_reviews r
 WHERE r.tenant_id=req.tenant_id AND r.request_id=req.id AND r.submission_id=sub.id
 ORDER BY r.version DESC LIMIT 1) review ON true`

func (r *PostgresRepository) AssessmentCorrectionBase(ctx context.Context, req Request, principal string, proof AssessmentCorrection) (Submission, error) {
	return readAssessmentCorrectionBase(ctx, r.pool, req.TenantID, req.ID, principal, proof)
}

// Writers call this after locking the request. A concurrent submission or
// review cannot change the base while a draft, upload or submission commits.
func readAssessmentCorrectionBase(ctx context.Context, q assessmentRowQuerier, tenant, requestID, principal string, proof AssessmentCorrection) (Submission, error) {
	value, err := scanSubmission(q.QueryRow(ctx, `SELECT `+assessmentSubmissionColumns+`
 FROM capture_requests req JOIN tenants t ON t.id=req.tenant_id
 JOIN capture_submissions sub ON sub.id=(`+assessmentLatestSubmission+`)
 `+assessmentCurrentReviewJoin+`
 WHERE (t.id::text=$1 OR t.slug=$1) AND req.id::text=$2 AND req.recipient_principal_id::text=$3
 AND sub.id::text=$4 AND review.id::text=$5 AND `+assessmentCorrectionScope,
		tenant, requestID, principal, proof.SubmissionID, proof.ReviewID))
	if errors.Is(err, ErrNotFound) {
		return Submission{}, ErrRequestClosed
	}
	return value, err
}
