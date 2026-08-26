//go:build postgres

package thirdparty

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) CreateVendorWork(ctx context.Context, value VendorWorkRequest) (VendorWorkRequest, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return VendorWorkRequest{}, fmt.Errorf("begin vendor work preparation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tenantID, err := resolveTenant(ctx, tx, value.TenantID)
	if err != nil {
		return VendorWorkRequest{}, err
	}
	var programLink, matterLink any
	if value.TargetType == LinkTargetProgram {
		programLink = value.RelationshipLinkID
	} else if value.TargetType == LinkTargetMatter {
		matterLink = value.RelationshipLinkID
	} else {
		return VendorWorkRequest{}, ErrInvalid
	}
	_, err = tx.Exec(ctx, `INSERT INTO third_party_work_requests(
		id,tenant_id,legal_entity_id,relationship_id,program_link_id,matter_link_id,target_type,target_id,purpose,instructions,
		owner_principal_id,form_template_id,form_template_version,presentation,state,delivery_state,due_at,version,created_at,updated_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6::uuid,$7,$8::uuid,$9,$10,$11::uuid,$12::uuid,$13,$14,'PREPARING','NOT_SENT',$15,1,$16,$16)`,
		value.ID, tenantID, value.LegalEntityID, value.RelationshipID, programLink, matterLink, value.TargetType, value.TargetID, value.Purpose, value.Instructions,
		value.OwnerPrincipalID, value.FormTemplateID, value.FormTemplateVersion, value.Presentation, value.DueAt, value.CreatedAt)
	if isUniqueViolation(err) {
		return VendorWorkRequest{}, ErrVersionConflict
	}
	if err != nil {
		return VendorWorkRequest{}, fmt.Errorf("store vendor work: %w", err)
	}
	if err := appendVendorWorkEvent(ctx, tx, tenantID, value, value.OwnerPrincipalID, "VendorWorkPrepared"); err != nil {
		return VendorWorkRequest{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return VendorWorkRequest{}, fmt.Errorf("commit vendor work preparation: %w", err)
	}
	return value, nil
}

func (r *PostgresRepository) GetVendorWork(ctx context.Context, scope Scope, id string) (VendorWorkRequest, error) {
	value, err := scanVendorWork(r.pool.QueryRow(ctx, vendorWorkSelect+` WHERE (t.id::text=$1 OR t.slug=$1) AND w.legal_entity_id::text=$2 AND w.id::text=$3`, scope.TenantID, scope.LegalEntityID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return VendorWorkRequest{}, ErrNotFound
	}
	if err != nil {
		return VendorWorkRequest{}, fmt.Errorf("get vendor work: %w", err)
	}
	return value, nil
}

func (r *PostgresRepository) FindActiveVendorWork(ctx context.Context, scope Scope, linkID string) (VendorWorkRequest, error) {
	value, err := scanVendorWork(r.pool.QueryRow(ctx, vendorWorkSelect+` WHERE (t.id::text=$1 OR t.slug=$1) AND w.legal_entity_id::text=$2 AND COALESCE(w.program_link_id,w.matter_link_id)::text=$3 AND w.state NOT IN ('ACCEPTED','CANCELLED')`, scope.TenantID, scope.LegalEntityID, linkID))
	if errors.Is(err, pgx.ErrNoRows) {
		return VendorWorkRequest{}, ErrNotFound
	}
	return value, err
}

func (r *PostgresRepository) AttachVendorWorkCapture(ctx context.Context, scope Scope, id string, expected int64, link VendorWorkCaptureLink, now time.Time) (VendorWorkRequest, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return VendorWorkRequest{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	current, err := getVendorWorkLocked(ctx, tx, scope, id)
	if err != nil {
		return VendorWorkRequest{}, err
	}
	if current.Version != expected {
		return VendorWorkRequest{}, ErrVersionConflict
	}
	if current.State != VendorWorkPreparing && current.State != VendorWorkUnderReview {
		return VendorWorkRequest{}, ErrInvalidAssessmentTransition
	}
	tenantID, err := resolveTenant(ctx, tx, scope.TenantID)
	if err != nil {
		return VendorWorkRequest{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO third_party_work_capture_links(id,tenant_id,legal_entity_id,work_request_id,request_id,sequence,purpose,origin_type,origin_id,origin_version,created_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6,$7,'THIRD_PARTY_WORK',$4::uuid,$6,$8)`, link.ID, tenantID, scope.LegalEntityID, id, link.RequestID, link.Sequence, link.Purpose, link.CreatedAt)
	if isUniqueViolation(err) {
		return VendorWorkRequest{}, ErrVersionConflict
	}
	if err != nil {
		return VendorWorkRequest{}, fmt.Errorf("attach vendor work capture: %w", err)
	}
	current.CurrentRequestID, current.CurrentInvitationID, current.CurrentCaptureSequence, current.SubmissionID = link.RequestID, "", link.Sequence, ""
	current.DeliveryState, current.Recovery, current.Version, current.UpdatedAt = VendorWorkDeliveryNotSent, "", current.Version+1, now.UTC()
	_, err = tx.Exec(ctx, `UPDATE third_party_work_requests SET current_request_id=$4::uuid,current_invitation_id=NULL,current_capture_sequence=$5,submission_id=NULL,delivery_state='NOT_SENT',recovery='',version=$6,updated_at=$7
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND version=$8`, tenantID, scope.LegalEntityID, id, link.RequestID, link.Sequence, current.Version, current.UpdatedAt, expected)
	if err != nil {
		return VendorWorkRequest{}, fmt.Errorf("update vendor work capture: %w", err)
	}
	if err := appendVendorWorkEvent(ctx, tx, tenantID, current, "", "VendorWorkCaptureAttached"); err != nil {
		return VendorWorkRequest{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return VendorWorkRequest{}, err
	}
	return current, nil
}

func (r *PostgresRepository) MarkVendorWorkSent(ctx context.Context, scope Scope, id string, expected int64, invitationID string, delivery VendorWorkDeliveryState, recovery string, now time.Time) (VendorWorkRequest, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return VendorWorkRequest{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	current, err := getVendorWorkLocked(ctx, tx, scope, id)
	if err != nil {
		return VendorWorkRequest{}, err
	}
	if current.Version != expected {
		return VendorWorkRequest{}, ErrVersionConflict
	}
	if current.State != VendorWorkPreparing && current.State != VendorWorkAwaitingVendor && current.State != VendorWorkChangesRequested {
		return VendorWorkRequest{}, ErrInvalidAssessmentTransition
	}
	tenantID, err := resolveTenant(ctx, tx, scope.TenantID)
	if err != nil {
		return VendorWorkRequest{}, err
	}
	if invitationID != "" {
		current.CurrentInvitationID = invitationID
		if current.State == VendorWorkPreparing {
			current.State = VendorWorkAwaitingVendor
		}
		_, err = tx.Exec(ctx, `UPDATE third_party_work_capture_links SET invitation_id=$4::uuid WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND work_request_id=$3::uuid AND request_id=(SELECT current_request_id FROM third_party_work_requests WHERE id=$3::uuid)`, tenantID, scope.LegalEntityID, id, invitationID)
		if err != nil {
			return VendorWorkRequest{}, fmt.Errorf("attach vendor work invitation: %w", err)
		}
	}
	current.DeliveryState, current.Recovery, current.Version, current.UpdatedAt = delivery, recovery, current.Version+1, now.UTC()
	_, err = tx.Exec(ctx, `UPDATE third_party_work_requests SET state=$4,current_invitation_id=NULLIF($5,'')::uuid,delivery_state=$6,recovery=$7,version=$8,updated_at=$9
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND version=$10`, tenantID, scope.LegalEntityID, id, current.State, current.CurrentInvitationID, delivery, recovery, current.Version, current.UpdatedAt, expected)
	if err != nil {
		return VendorWorkRequest{}, fmt.Errorf("record vendor work delivery: %w", err)
	}
	eventType := "VendorWorkSent"
	if delivery == VendorWorkDeliveryRetryRequired {
		eventType = "VendorWorkDeliveryRetryRequired"
		_, err = tx.Exec(ctx, `INSERT INTO third_party_work_jobs(tenant_id,legal_entity_id,work_request_id,job_type,dedupe_key,state,available_at,created_at,updated_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,'DELIVERY_RETRY',$4,'READY',$5,$5,$5) ON CONFLICT (tenant_id,dedupe_key) DO UPDATE SET state='READY',available_at=EXCLUDED.available_at,updated_at=EXCLUDED.updated_at`, tenantID, scope.LegalEntityID, id, fmt.Sprintf("vendor-work-delivery:%s:%d", id, current.CurrentCaptureSequence), now.UTC())
	}
	if err != nil {
		return VendorWorkRequest{}, fmt.Errorf("schedule vendor work delivery retry: %w", err)
	}
	if err := appendVendorWorkEvent(ctx, tx, tenantID, current, "", eventType); err != nil {
		return VendorWorkRequest{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return VendorWorkRequest{}, err
	}
	return current, nil
}

func (r *PostgresRepository) MarkVendorWorkPreparationRequired(ctx context.Context, scope Scope, id string, expected int64, recovery string, now time.Time) (VendorWorkRequest, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return VendorWorkRequest{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	current, err := getVendorWorkLocked(ctx, tx, scope, id)
	if err != nil {
		return VendorWorkRequest{}, err
	}
	if current.Version != expected {
		return VendorWorkRequest{}, ErrVersionConflict
	}
	if current.State != VendorWorkPreparing || current.CurrentRequestID != "" {
		return VendorWorkRequest{}, ErrInvalidAssessmentTransition
	}
	tenantID, err := resolveTenant(ctx, tx, scope.TenantID)
	if err != nil {
		return VendorWorkRequest{}, err
	}
	current.DeliveryState, current.Recovery, current.Version, current.UpdatedAt = VendorWorkDeliveryRetryRequired, recovery, current.Version+1, now.UTC()
	_, err = tx.Exec(ctx, `UPDATE third_party_work_requests SET delivery_state='RETRY_REQUIRED',recovery=$4,version=$5,updated_at=$6 WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND version=$7`, tenantID, scope.LegalEntityID, id, recovery, current.Version, current.UpdatedAt, expected)
	if err != nil {
		return VendorWorkRequest{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO third_party_work_jobs(tenant_id,legal_entity_id,work_request_id,job_type,dedupe_key,state,available_at,created_at,updated_at) VALUES($1::uuid,$2::uuid,$3::uuid,'DELIVERY_RETRY',$4,'READY',$5,$5,$5) ON CONFLICT (tenant_id,dedupe_key) DO UPDATE SET state='READY',available_at=EXCLUDED.available_at,updated_at=EXCLUDED.updated_at`, tenantID, scope.LegalEntityID, id, "vendor-work-prepare:"+id, now.UTC())
	if err != nil {
		return VendorWorkRequest{}, err
	}
	if err := appendVendorWorkEvent(ctx, tx, tenantID, current, "", "VendorWorkPreparationRetryRequired"); err != nil {
		return VendorWorkRequest{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return VendorWorkRequest{}, err
	}
	return current, nil
}

func (r *PostgresRepository) RecordVendorWorkSubmission(ctx context.Context, input VendorWorkSubmissionInput, now time.Time) (VendorWorkRequest, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return VendorWorkRequest{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tenantID, err := resolveTenant(ctx, tx, input.TenantID)
	if err != nil {
		return VendorWorkRequest{}, err
	}
	var existing bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM third_party_work_reactions WHERE tenant_id=$1::uuid AND reaction_kind='SUBMITTED' AND causation_id=$2)`, tenantID, input.CausationID).Scan(&existing); err != nil {
		return VendorWorkRequest{}, err
	}
	current, err := scanVendorWork(tx.QueryRow(ctx, vendorWorkSelect+` WHERE w.tenant_id=$1::uuid AND w.id::text=$2 FOR UPDATE OF w`, tenantID, input.WorkRequestID))
	if errors.Is(err, pgx.ErrNoRows) {
		return VendorWorkRequest{}, ErrNotFound
	}
	if err != nil {
		return VendorWorkRequest{}, err
	}
	if existing {
		return current, nil
	}
	if current.CurrentRequestID != input.RequestID {
		return VendorWorkRequest{}, ErrNotFound
	}
	if current.State != VendorWorkAwaitingVendor && current.State != VendorWorkChangesRequested {
		return VendorWorkRequest{}, ErrInvalidAssessmentTransition
	}
	current.State, current.SubmissionID, current.DeliveryState, current.Recovery = VendorWorkResponseReceived, input.SubmissionID, VendorWorkDeliveryDelivered, ""
	current.ResponseReceivedAt, current.UpdatedAt, current.Version = timePointer(now.UTC()), now.UTC(), current.Version+1
	_, err = tx.Exec(ctx, `UPDATE third_party_work_requests SET state='RESPONSE_RECEIVED',submission_id=$3::uuid,delivery_state='DELIVERED',recovery='',response_received_at=$4,updated_at=$4,version=$5 WHERE tenant_id=$1::uuid AND id=$2::uuid`, tenantID, current.ID, input.SubmissionID, now.UTC(), current.Version)
	if err != nil {
		return VendorWorkRequest{}, fmt.Errorf("record vendor response: %w", err)
	}
	_, err = tx.Exec(ctx, `UPDATE third_party_work_capture_links SET submission_id=$3::uuid WHERE tenant_id=$1::uuid AND request_id=$2::uuid`, tenantID, input.RequestID, input.SubmissionID)
	if err != nil {
		return VendorWorkRequest{}, err
	}
	snapshot, _ := json.Marshal(current)
	_, err = tx.Exec(ctx, `INSERT INTO third_party_work_reactions(tenant_id,legal_entity_id,work_request_id,reaction_kind,causation_id,request_id,submission_id,resulting_version,result_snapshot,applied_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,'SUBMITTED',$4,$5::uuid,$6::uuid,$7,$8::jsonb,$9)`, tenantID, current.LegalEntityID, current.ID, input.CausationID, input.RequestID, input.SubmissionID, current.Version, snapshot, now.UTC())
	if err != nil {
		return VendorWorkRequest{}, err
	}
	if err := appendVendorWorkEvent(ctx, tx, tenantID, current, "", "VendorWorkResponseReceived"); err != nil {
		return VendorWorkRequest{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return VendorWorkRequest{}, err
	}
	return current, nil
}

func (r *PostgresRepository) TransitionVendorWork(ctx context.Context, scope Scope, id string, expected int64, target VendorWorkState, actor, detail string, now time.Time) (VendorWorkRequest, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return VendorWorkRequest{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	current, err := getVendorWorkLocked(ctx, tx, scope, id)
	if err != nil {
		return VendorWorkRequest{}, err
	}
	if current.Version != expected {
		return VendorWorkRequest{}, ErrVersionConflict
	}
	eventType := ""
	switch target {
	case VendorWorkUnderReview:
		if current.State != VendorWorkResponseReceived {
			return VendorWorkRequest{}, ErrInvalidAssessmentTransition
		}
		current.ReviewerPrincipalID, current.ReviewStartedAt, eventType = actor, timePointer(now.UTC()), "VendorWorkReviewStarted"
	case VendorWorkChangesRequested:
		if current.State != VendorWorkUnderReview {
			return VendorWorkRequest{}, ErrInvalidAssessmentTransition
		}
		current.ReviewRationale, eventType = detail, "VendorWorkChangesRequested"
	case VendorWorkAccepted:
		if current.State != VendorWorkUnderReview {
			return VendorWorkRequest{}, ErrInvalidAssessmentTransition
		}
		current.ReviewerPrincipalID, current.ReviewRationale, current.AcceptedAt, eventType = actor, detail, timePointer(now.UTC()), "VendorWorkAccepted"
	case VendorWorkCancelled:
		if current.State == VendorWorkAccepted || current.State == VendorWorkCancelled {
			return VendorWorkRequest{}, ErrInvalidAssessmentTransition
		}
		current.CancellationReason, current.CancelledAt, eventType = detail, timePointer(now.UTC()), "VendorWorkCancelled"
	default:
		return VendorWorkRequest{}, ErrInvalid
	}
	current.State, current.Version, current.UpdatedAt = target, current.Version+1, now.UTC()
	tenantID, err := resolveTenant(ctx, tx, scope.TenantID)
	if err != nil {
		return VendorWorkRequest{}, err
	}
	_, err = tx.Exec(ctx, `UPDATE third_party_work_requests SET state=$4,reviewer_principal_id=NULLIF($5,'')::uuid,review_rationale=$6,cancellation_reason=$7,response_received_at=$8,review_started_at=$9,accepted_at=$10,cancelled_at=$11,version=$12,updated_at=$13 WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND version=$14`,
		tenantID, scope.LegalEntityID, id, current.State, current.ReviewerPrincipalID, current.ReviewRationale, current.CancellationReason, current.ResponseReceivedAt, current.ReviewStartedAt, current.AcceptedAt, current.CancelledAt, current.Version, current.UpdatedAt, expected)
	if err != nil {
		return VendorWorkRequest{}, fmt.Errorf("transition vendor work: %w", err)
	}
	if err := appendVendorWorkEvent(ctx, tx, tenantID, current, actor, eventType); err != nil {
		return VendorWorkRequest{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return VendorWorkRequest{}, err
	}
	return current, nil
}

func (r *PostgresRepository) ListVendorWork(ctx context.Context, scope Scope, input VendorWorkListInput) (VendorWorkPage, error) {
	args := []any{scope.TenantID, scope.LegalEntityID, input.RelationshipID, string(input.TargetType), input.TargetID}
	cursorClause := ""
	if input.Cursor != "" {
		at, id, err := decodeCursor(input.Cursor)
		if err != nil {
			return VendorWorkPage{}, ErrInvalid
		}
		args = append(args, at, id)
		cursorClause = " AND (w.updated_at,w.id)<($6,$7::uuid)"
	}
	args = append(args, input.Limit+1)
	rows, err := r.pool.Query(ctx, vendorWorkSelect+` WHERE (t.id::text=$1 OR t.slug=$1) AND w.legal_entity_id::text=$2 AND ($3='' OR w.relationship_id::text=$3) AND ($4='' OR w.target_type=$4) AND ($5='' OR w.target_id::text=$5)`+cursorClause+` ORDER BY w.updated_at DESC,w.id DESC LIMIT $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return VendorWorkPage{}, fmt.Errorf("list vendor work: %w", err)
	}
	defer rows.Close()
	items := make([]VendorWorkRequest, 0, input.Limit+1)
	for rows.Next() {
		value, scanErr := scanVendorWork(rows)
		if scanErr != nil {
			return VendorWorkPage{}, scanErr
		}
		items = append(items, value)
	}
	if err := rows.Err(); err != nil {
		return VendorWorkPage{}, err
	}
	page := VendorWorkPage{Items: items}
	if len(items) > input.Limit {
		page.Items = items[:input.Limit]
		last := page.Items[len(page.Items)-1]
		page.NextCursor = encodeCursor(last.UpdatedAt, last.ID)
	}
	return page, nil
}

func (r *PostgresRepository) ResolveVendorWorkCapture(ctx context.Context, tenant string, origin evidence.RequestOrigin, requestID string) (VendorWorkSubmissionTarget, error) {
	var value VendorWorkSubmissionTarget
	err := r.pool.QueryRow(ctx, `SELECT t.slug,w.legal_entity_id::text,w.id::text,w.version,l.request_id::text FROM third_party_work_capture_links l JOIN third_party_work_requests w ON w.id=l.work_request_id AND w.tenant_id=l.tenant_id JOIN tenants t ON t.id=w.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND l.origin_type='THIRD_PARTY_WORK' AND l.origin_id::text=$2 AND l.origin_version=$3 AND l.request_id::text=$4`, tenant, origin.ID, origin.Version, requestID).Scan(&value.TenantID, &value.LegalEntityID, &value.WorkRequestID, &value.WorkVersion, &value.RequestID)
	if errors.Is(err, pgx.ErrNoRows) {
		return VendorWorkSubmissionTarget{}, ErrNotFound
	}
	return value, err
}

func (r *PostgresRepository) HasActiveVendorWork(ctx context.Context, scope Scope, linkID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM third_party_work_requests w JOIN tenants t ON t.id=w.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND w.legal_entity_id::text=$2 AND COALESCE(w.program_link_id,w.matter_link_id)::text=$3 AND w.state NOT IN ('ACCEPTED','CANCELLED'))`, scope.TenantID, scope.LegalEntityID, linkID).Scan(&exists)
	return exists, err
}

func getVendorWorkLocked(ctx context.Context, tx pgx.Tx, scope Scope, id string) (VendorWorkRequest, error) {
	value, err := scanVendorWork(tx.QueryRow(ctx, vendorWorkSelect+` WHERE (t.id::text=$1 OR t.slug=$1) AND w.legal_entity_id::text=$2 AND w.id::text=$3 FOR UPDATE OF w`, scope.TenantID, scope.LegalEntityID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return VendorWorkRequest{}, ErrNotFound
	}
	return value, err
}

const vendorWorkSelect = `SELECT w.id::text,t.slug,w.legal_entity_id::text,w.relationship_id::text,COALESCE(w.program_link_id,w.matter_link_id)::text,w.target_type,w.target_id::text,w.purpose,w.instructions,w.owner_principal_id::text,COALESCE(w.reviewer_principal_id::text,''),w.form_template_id::text,w.form_template_version,w.presentation,COALESCE(w.current_request_id::text,''),COALESCE(w.current_invitation_id::text,''),w.current_capture_sequence,COALESCE(w.submission_id::text,''),w.state,w.delivery_state,w.recovery,w.review_rationale,w.cancellation_reason,w.due_at,w.version,w.created_at,w.updated_at,w.response_received_at,w.review_started_at,w.accepted_at,w.cancelled_at FROM third_party_work_requests w JOIN tenants t ON t.id=w.tenant_id`

func scanVendorWork(row rowScanner) (VendorWorkRequest, error) {
	var value VendorWorkRequest
	err := row.Scan(&value.ID, &value.TenantID, &value.LegalEntityID, &value.RelationshipID, &value.RelationshipLinkID, &value.TargetType, &value.TargetID, &value.Purpose, &value.Instructions, &value.OwnerPrincipalID, &value.ReviewerPrincipalID, &value.FormTemplateID, &value.FormTemplateVersion, &value.Presentation, &value.CurrentRequestID, &value.CurrentInvitationID, &value.CurrentCaptureSequence, &value.SubmissionID, &value.State, &value.DeliveryState, &value.Recovery, &value.ReviewRationale, &value.CancellationReason, &value.DueAt, &value.Version, &value.CreatedAt, &value.UpdatedAt, &value.ResponseReceivedAt, &value.ReviewStartedAt, &value.AcceptedAt, &value.CancelledAt)
	return value, err
}

func appendVendorWorkEvent(ctx context.Context, tx pgx.Tx, tenantID string, value VendorWorkRequest, actor, eventType string) error {
	_, err := tx.Exec(ctx, `INSERT INTO third_party_work_events(tenant_id,legal_entity_id,work_request_id,work_version,actor_principal_id,event_type,payload,occurred_at) VALUES($1::uuid,$2::uuid,$3::uuid,$4,NULLIF($5,'')::uuid,$6,jsonb_build_object('state',$7::text,'delivery_state',$8::text,'request_id',$9::text,'submission_id',$10::text),$11)`, tenantID, value.LegalEntityID, value.ID, value.Version, actor, eventType, value.State, value.DeliveryState, value.CurrentRequestID, value.SubmissionID, value.UpdatedAt)
	if err != nil {
		return fmt.Errorf("append vendor work event: %w", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at) VALUES($1::uuid,'VENDOR_WORK_REQUEST',$2::uuid,$3,jsonb_build_object('version',$4::bigint,'state',$5::text,'delivery_state',$6::text),$7,$7)`, tenantID, value.ID, eventType, value.Version, value.State, value.DeliveryState, value.UpdatedAt)
	if err != nil {
		return fmt.Errorf("append vendor work outbox event: %w", err)
	}
	return nil
}

var _ VendorWorkRepository = (*PostgresRepository)(nil)
