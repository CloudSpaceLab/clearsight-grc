//go:build postgres

package evidence

import (
	"context"
	"errors"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5"
)

const assessmentParticipantScope = `
 FROM capture_requests req JOIN tenants t ON t.id=req.tenant_id
 JOIN programs p ON p.tenant_id=req.tenant_id AND p.legal_entity_id=req.legal_entity_id AND p.id::text=req.subject_id
`
const assessmentHistoryScope = ` (t.id::text=$1 OR t.slug=$1) AND req.legal_entity_id::text=$2
 AND req.subject_type='PROGRAM' AND req.subject_id=$3 AND req.id::text=$4
 AND (p.owner_principal_id::text=$5 OR p.authority_principal_id::text=$5 OR (` + assessmentRespondent + `))`
const assessmentRespondent = `req.recipient_state='ASSIGNED' AND req.recipient_principal_id::text=$5`
const assessmentReviewer = `req.status='SUBMITTED' AND p.authority_principal_id::text=$5
 AND req.known_facts->>'reviewer'=$5 AND req.created_by::text<>$5`
const assessmentCanRead = `((` + assessmentRespondent + `) AND sub.submitted_by::text=$5)
 OR ((` + assessmentReviewer + `) AND sub.submitted_by::text<>$5)`

func (r *PostgresRepository) AssessmentHistory(ctx context.Context, actor identity.Actor, q AssessmentHistoryQuery) (AssessmentHistory, error) {
	value := AssessmentHistory{Revisions: []AssessmentRevisionSummary{}}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return value, err
	}
	defer tx.Rollback(ctx)
	args := []any{actor.TenantID, actor.LegalEntityID, q.ProgramID, q.RequestID, actor.PrincipalID}
	var correction AssessmentCorrectionWindow
	var correctionOpen bool
	err = tx.QueryRow(ctx, `SELECT
 COALESCE((`+assessmentReviewer+`) AND sub.submitted_by::text<>$5,false),
 COALESCE((SELECT max(v.version) FROM capture_submission_reviews v WHERE v.tenant_id=req.tenant_id AND v.request_id=req.id),0),
 COALESCE((`+assessmentRespondent+`) AND (`+assessmentCorrectionScope+`),false),
 COALESCE(sub.id::text,''),COALESCE(review.id::text,''),COALESCE(review.correction_due_at,req.deadline)
 `+assessmentParticipantScope+`
 LEFT JOIN capture_submissions sub ON sub.id=(`+assessmentLatestSubmission+`)
 `+assessmentCurrentReviewJoin+` WHERE `+assessmentHistoryScope, args...).Scan(&value.CanReview, &value.LatestReviewVersion,
		&correctionOpen, &correction.SubmissionID, &correction.ReviewID, &correction.DueAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return value, ErrNotFound
	}
	if err != nil {
		return value, err
	}
	if correctionOpen {
		value.Correction = &correction
	}
	if q.SubmissionID != "" {
		err = readAssessmentRevision(ctx, tx, &value, q, args)
	} else {
		err = readAssessmentRevisionPage(ctx, tx, &value, q.BeforeRevision, args)
	}
	if err != nil {
		return value, err
	}
	return value, tx.Commit(ctx)
}

const assessmentHistoryPageSize = 20

func readAssessmentRevisionPage(ctx context.Context, tx pgx.Tx, value *AssessmentHistory, before int64, args []any) error {
	rows, err := tx.Query(ctx, `SELECT sub.id::text,sub.assessment_revision,COALESCE(sub.correction_of::text,''),
 sub.submitted_at,COALESCE(sub.submitted_by::text,''),COALESCE(review.status,''),COALESCE(`+assessmentCanRead+`,false)
 `+assessmentParticipantScope+`
 JOIN capture_submissions sub ON sub.tenant_id=req.tenant_id AND sub.request_id=req.id AND sub.channel='INTERNAL' AND sub.session_id IS NULL
 `+assessmentCurrentReviewJoin+`
 WHERE `+assessmentHistoryScope+` AND ($6::bigint=0 OR sub.assessment_revision<$6::bigint)
 ORDER BY sub.assessment_revision DESC,sub.submitted_at DESC,sub.id DESC LIMIT $7`, append(args, before, assessmentHistoryPageSize+1)...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var item AssessmentRevisionSummary
		if err := rows.Scan(&item.ID, &item.Revision, &item.CorrectionOf, &item.SubmittedAt, &item.SubmittedBy, &item.ReviewStatus, &item.CanRead); err != nil {
			return err
		}
		if len(value.Revisions) == assessmentHistoryPageSize {
			value.NextBeforeRevision = value.Revisions[len(value.Revisions)-1].Revision
			break
		}
		value.Revisions = append(value.Revisions, item)
	}
	return rows.Err()
}

func readAssessmentRevision(ctx context.Context, tx pgx.Tx, value *AssessmentHistory, q AssessmentHistoryQuery, args []any) error {
	// A Program owner's history shows decisions. Answers additionally require
	// the exact submitter or the current, pinned independent reviewer.
	submission, err := scanSubmission(tx.QueryRow(ctx, `SELECT `+assessmentSubmissionColumns+assessmentParticipantScope+`
 JOIN capture_submissions sub ON sub.tenant_id=req.tenant_id AND sub.request_id=req.id AND sub.channel='INTERNAL' AND sub.session_id IS NULL
 WHERE `+assessmentHistoryScope+` AND sub.id::text=$6`, append(args, q.SubmissionID)...))
	if err != nil {
		return err
	}
	var canRead bool
	err = tx.QueryRow(ctx, `SELECT COALESCE(`+assessmentCanRead+`,false)`+assessmentParticipantScope+`
 JOIN capture_submissions sub ON sub.tenant_id=req.tenant_id AND sub.request_id=req.id AND sub.id::text=$6
 WHERE `+assessmentHistoryScope, append(args, q.SubmissionID)...).Scan(&canRead)
	if err != nil {
		return err
	}
	if !canRead {
		submission.Answers = nil
		submission.AnswerProvenance = nil
	}
	value.Submission = &submission
	rows, err := tx.Query(ctx, `SELECT `+submissionReviewColumns+` FROM capture_submission_reviews review
 WHERE review.tenant_id=$1::uuid AND review.request_id=$2::uuid AND review.submission_id=$3::uuid
 AND ($4::bigint=0 OR review.version<$4::bigint) ORDER BY review.version DESC LIMIT $5`,
		submission.TenantID, submission.RequestID, submission.ID, q.BeforeReview, assessmentHistoryPageSize+1)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		review, err := scanSubmissionReview(rows)
		if err != nil {
			return err
		}
		if len(value.Reviews) == assessmentHistoryPageSize {
			value.NextBeforeReview = value.Reviews[len(value.Reviews)-1].Version
			break
		}
		value.Reviews = append(value.Reviews, review)
	}
	return rows.Err()
}
