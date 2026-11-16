//go:build postgres

package evidence

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) GetInternalDraft(ctx context.Context, tenant, request, principal string) (ResponseDraft, error) {
	value, err := scanResponseDraft(r.pool.QueryRow(ctx, `SELECT d.id::text,d.tenant_id::text,d.request_id::text,'',d.answers,d.presentation_mode,d.version,d.created_at,d.updated_at
 FROM capture_response_drafts d JOIN tenants t ON t.id=d.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND d.request_id=$2::uuid AND d.principal_id=$3::uuid`, tenant, request, principal))
	if errors.Is(err, pgx.ErrNoRows) {
		return value, ErrNotFound
	}
	return value, err
}
func (r *PostgresRepository) SaveInternalDraft(ctx context.Context, record InternalDraftRecord) (ResponseDraft, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ResponseDraft{}, err
	}
	defer tx.Rollback(ctx)
	var version int64
	err = tx.QueryRow(ctx, `SELECT req.version FROM capture_requests req JOIN tenants t ON t.id=req.tenant_id
 JOIN principals p ON p.tenant_id=req.tenant_id AND p.id=req.recipient_principal_id
 WHERE (t.id::text=$1 OR t.slug=$1) AND req.id=$2::uuid AND req.recipient_principal_id=$3::uuid AND req.recipient_state='ASSIGNED'
 AND req.status IN ('READY','IN_PROGRESS') AND req.deadline>clock_timestamp() AND p.kind='PERSON' AND p.status='ACTIVE'
 AND p.valid_from<=clock_timestamp() AND (p.valid_until IS NULL OR p.valid_until>clock_timestamp()) FOR UPDATE OF req`, record.Draft.TenantID, record.Draft.RequestID, record.PrincipalID).Scan(&version)
	if errors.Is(err, pgx.ErrNoRows) {
		return ResponseDraft{}, ErrRequestClosed
	}
	if err != nil {
		return ResponseDraft{}, err
	}
	if version != record.RequestVersion {
		return ResponseDraft{}, ErrVersionConflict
	}
	data, err := json.Marshal(record.Draft.Answers)
	if err != nil {
		return ResponseDraft{}, err
	}
	var row pgx.Row
	if record.ExpectedVersion == 0 {
		row = tx.QueryRow(ctx, `INSERT INTO capture_response_drafts(id,tenant_id,request_id,principal_id,answers,presentation_mode,version,created_at,updated_at)
  VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4::uuid,$5::jsonb,$6,1,$7,$7)
  ON CONFLICT (tenant_id,request_id,principal_id) WHERE principal_id IS NOT NULL DO NOTHING
  RETURNING id::text,tenant_id::text,request_id::text,'',answers,presentation_mode,version,created_at,updated_at`, record.Draft.ID, record.Draft.TenantID, record.Draft.RequestID, record.PrincipalID, string(data), record.Draft.PresentationMode, record.Draft.UpdatedAt)
	} else {
		row = tx.QueryRow(ctx, `UPDATE capture_response_drafts d SET answers=$5::jsonb,presentation_mode=$6,version=d.version+1,updated_at=$7
  FROM tenants t WHERE t.id=d.tenant_id AND (t.id::text=$1 OR t.slug=$1) AND d.request_id=$2::uuid AND d.principal_id=$3::uuid AND d.version=$4
  RETURNING d.id::text,d.tenant_id::text,d.request_id::text,'',d.answers,d.presentation_mode,d.version,d.created_at,d.updated_at`, record.Draft.TenantID, record.Draft.RequestID, record.PrincipalID, record.ExpectedVersion, string(data), record.Draft.PresentationMode, record.Draft.UpdatedAt)
	}
	value, err := scanResponseDraft(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return ResponseDraft{}, ErrVersionConflict
	}
	if err != nil {
		return ResponseDraft{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at)
 VALUES($1::uuid,'EVIDENCE_REQUEST',$2::uuid,'InternalResponseDraftSaved',jsonb_build_object('draft_version',$3::bigint,'saved_by',$4::text),$5,$5,$5)`, value.TenantID, value.RequestID, value.Version, record.PrincipalID, record.Draft.UpdatedAt)
	if err != nil {
		return ResponseDraft{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return ResponseDraft{}, err
	}
	return value, nil
}

const submissionReviewColumns = `review.id::text,review.request_id::text,review.submission_id::text,review.status,review.rationale,review.groups,review.reviewed_by::text,review.version,review.created_at`

func scanSubmissionReview(row scanner) (SubmissionReview, error) {
	var value SubmissionReview
	var groups []byte
	err := row.Scan(&value.ID, &value.RequestID, &value.SubmissionID, &value.Status, &value.Rationale, &groups, &value.ReviewedBy, &value.Version, &value.CreatedAt)
	if err == nil {
		err = json.Unmarshal(groups, &value.Groups)
	}
	return value, err
}
func (r *PostgresRepository) ListSubmissionReviews(ctx context.Context, tenant, entity, program string) ([]SubmissionReview, error) {
	actor, err := identity.Require(ctx)
	if err != nil {
		return nil, ErrRecipientMismatch
	}
	rows, err := r.pool.Query(ctx, `SELECT `+submissionReviewColumns+` FROM capture_submission_reviews review JOIN tenants t ON t.id=review.tenant_id JOIN capture_requests req ON req.tenant_id=review.tenant_id AND req.id=review.request_id
 WHERE (t.id::text=$1 OR t.slug=$1) AND review.legal_entity_id=$2::uuid AND req.subject_type='PROGRAM' AND req.subject_id=$3
 AND EXISTS(SELECT 1 FROM programs p WHERE p.id::text=req.subject_id AND p.tenant_id=req.tenant_id AND p.legal_entity_id=req.legal_entity_id AND (p.owner_principal_id::text=$4 OR p.authority_principal_id::text=$4 OR (req.recipient_principal_id::text=$4 AND req.recipient_state='ASSIGNED'))) ORDER BY review.created_at DESC,review.id DESC LIMIT 500`, tenant, entity, program, actor.PrincipalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SubmissionReview{}
	for rows.Next() {
		v, e := scanSubmissionReview(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *PostgresRepository) RecordSubmissionReview(ctx context.Context, req Request, sub Submission, value SubmissionReview, expected int64) (SubmissionReview, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return value, err
	}
	defer tx.Rollback(ctx)
	var version int64
	err = tx.QueryRow(ctx, `SELECT req.version FROM capture_requests req JOIN programs p ON p.tenant_id=req.tenant_id AND p.legal_entity_id=req.legal_entity_id AND p.id::text=req.subject_id
 WHERE req.tenant_id=$1::uuid AND req.id=$2::uuid AND req.status='SUBMITTED' AND req.subject_type='PROGRAM'
 AND req.known_facts->>'reviewer'=$3 AND req.created_by::text<>$3 AND p.authority_principal_id::text=$3
 AND EXISTS(SELECT 1 FROM capture_submissions submission WHERE submission.tenant_id=req.tenant_id AND submission.request_id=req.id AND submission.id=$4::uuid AND submission.submitted_by::text<>$3)
 FOR UPDATE OF req FOR SHARE OF p`, req.TenantID, req.ID, value.ReviewedBy, sub.ID).Scan(&version)
	if errors.Is(err, pgx.ErrNoRows) {
		return value, ErrRecipientMismatch
	}
	if err != nil {
		return value, err
	}
	if version != req.Version {
		return value, ErrVersionConflict
	}
	var current int64
	if err = tx.QueryRow(ctx, `SELECT COALESCE(max(version),0) FROM capture_submission_reviews WHERE tenant_id=$1::uuid AND request_id=$2::uuid`, req.TenantID, req.ID).Scan(&current); err != nil {
		return value, err
	}
	if current != expected {
		return value, ErrVersionConflict
	}
	groups, err := json.Marshal(value.Groups)
	if err != nil {
		return value, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO capture_submission_reviews(id,tenant_id,legal_entity_id,request_id,submission_id,reviewed_by,status,rationale,groups,version,created_at)
 VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6::uuid,$7,$8,$9::jsonb,$10,$11)`, value.ID, req.TenantID, req.LegalEntityID, req.ID, sub.ID, value.ReviewedBy, value.Status, value.Rationale, string(groups), value.Version, value.CreatedAt)
	if err != nil {
		return value, err
	}
	payload, _ := json.Marshal(value)
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at) VALUES($1::uuid,'EVIDENCE_REQUEST',$2::uuid,'ProgramAssessmentReviewed',$3::jsonb,$4,$4,$4)`, req.TenantID, req.ID, string(payload), value.CreatedAt)
	if err != nil {
		return value, err
	}
	if err = tx.Commit(ctx); err != nil {
		return value, err
	}
	return value, nil
}
