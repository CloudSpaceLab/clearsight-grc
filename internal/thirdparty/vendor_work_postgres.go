//go:build postgres

package thirdparty

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
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
	table, targetColumn, tableErr := relationshipLinkTable(value.TargetType)
	if tableErr != nil {
		return VendorWorkRequest{}, tableErr
	}
	if value.TargetType == LinkTargetProgram {
		programLink = value.RelationshipLinkID
	} else if value.TargetType == LinkTargetMatter {
		matterLink = value.RelationshipLinkID
	} else {
		return VendorWorkRequest{}, ErrInvalid
	}
	var linkState RelationshipLinkState
	linkQuery := fmt.Sprintf(`SELECT state FROM %s WHERE id=$1::uuid AND tenant_id=$2::uuid AND legal_entity_id=$3::uuid AND relationship_id=$4::uuid AND %s=$5::uuid FOR UPDATE`, table, targetColumn)
	if err := tx.QueryRow(ctx, linkQuery, value.RelationshipLinkID, tenantID, value.LegalEntityID, value.RelationshipID, value.TargetID).Scan(&linkState); errors.Is(err, pgx.ErrNoRows) {
		return VendorWorkRequest{}, ErrNotFound
	} else if err != nil {
		return VendorWorkRequest{}, fmt.Errorf("lock vendor relationship link: %w", err)
	}
	if linkState != RelationshipLinkActive {
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
	eventID, err := appendVendorWorkEvent(ctx, tx, tenantID, value, value.OwnerPrincipalID, "VendorWorkPrepared")
	if err != nil {
		return VendorWorkRequest{}, err
	}
	if err := r.commitThirdPartyEvents(ctx, tx, vendorWorkCommitProof(eventID, value, "VendorWorkPrepared")); err != nil {
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
	eventID, err := appendVendorWorkEvent(ctx, tx, tenantID, current, "", "VendorWorkCaptureAttached")
	if err != nil {
		return VendorWorkRequest{}, err
	}
	if err := r.commitThirdPartyEvents(ctx, tx, vendorWorkCommitProof(eventID, current, "VendorWorkCaptureAttached")); err != nil {
		return VendorWorkRequest{}, err
	}
	return current, nil
}

func (r *PostgresRepository) ReserveVendorWorkInvitation(ctx context.Context, scope Scope, id string, expected int64, invitationID string, now time.Time) (VendorWorkRequest, error) {
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
	if invitationID == "" || current.CurrentRequestID == "" || (current.State != VendorWorkPreparing && current.State != VendorWorkAwaitingVendor && current.State != VendorWorkChangesRequested) {
		return VendorWorkRequest{}, ErrInvalidAssessmentTransition
	}
	tenantID, err := resolveTenant(ctx, tx, scope.TenantID)
	if err != nil {
		return VendorWorkRequest{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE third_party_work_invitation_reservations SET state='SUPERSEDED',resolved_at=$4 WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND work_request_id=$3::uuid AND state='PENDING'`, tenantID, scope.LegalEntityID, id, now.UTC()); err != nil {
		return VendorWorkRequest{}, fmt.Errorf("supersede vendor work invitation reservation: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO third_party_work_invitation_reservations(invitation_id,tenant_id,legal_entity_id,work_request_id,request_id,capture_sequence,state,created_at) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6,'PENDING',$7)`, invitationID, tenantID, scope.LegalEntityID, id, current.CurrentRequestID, current.CurrentCaptureSequence, now.UTC()); err != nil {
		if isUniqueViolation(err) {
			return VendorWorkRequest{}, ErrVersionConflict
		}
		return VendorWorkRequest{}, fmt.Errorf("reserve vendor work invitation: %w", err)
	}
	current.PendingInvitationID, current.PendingInvitationRequestID = invitationID, current.CurrentRequestID
	current.DeliveryState = VendorWorkDeliveryRetryRequired
	current.Recovery = "Complete secure-link setup for this vendor request."
	current.Version++
	current.UpdatedAt = now.UTC()
	if _, err := tx.Exec(ctx, `UPDATE third_party_work_requests SET delivery_state=$4,recovery=$5,version=$6,updated_at=$7 WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND version=$8`, tenantID, scope.LegalEntityID, id, current.DeliveryState, current.Recovery, current.Version, current.UpdatedAt, expected); err != nil {
		return VendorWorkRequest{}, fmt.Errorf("record vendor work invitation reservation: %w", err)
	}
	eventID, err := appendVendorWorkEvent(ctx, tx, tenantID, current, "", "VendorWorkInvitationReserved")
	if err != nil {
		return VendorWorkRequest{}, err
	}
	if err := r.commitThirdPartyEvents(ctx, tx, vendorWorkCommitProof(eventID, current, "VendorWorkInvitationReserved")); err != nil {
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
		var reservedRequestID string
		if err := tx.QueryRow(ctx, `SELECT request_id::text FROM third_party_work_invitation_reservations WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND work_request_id=$3::uuid AND invitation_id=$4::uuid AND state='PENDING' FOR UPDATE`, tenantID, scope.LegalEntityID, id, invitationID).Scan(&reservedRequestID); errors.Is(err, pgx.ErrNoRows) {
			return VendorWorkRequest{}, ErrVersionConflict
		} else if err != nil {
			return VendorWorkRequest{}, fmt.Errorf("lock vendor work invitation reservation: %w", err)
		}
		if reservedRequestID != current.CurrentRequestID {
			return VendorWorkRequest{}, ErrVersionConflict
		}
		current.CurrentInvitationID = invitationID
		if current.State == VendorWorkPreparing {
			current.State = VendorWorkAwaitingVendor
		}
		_, err = tx.Exec(ctx, `UPDATE third_party_work_capture_links SET invitation_id=$4::uuid WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND work_request_id=$3::uuid AND request_id=(SELECT current_request_id FROM third_party_work_requests WHERE id=$3::uuid)`, tenantID, scope.LegalEntityID, id, invitationID)
		if err != nil {
			return VendorWorkRequest{}, fmt.Errorf("attach vendor work invitation: %w", err)
		}
		if _, err := tx.Exec(ctx, `UPDATE third_party_work_invitation_reservations SET state='FINALIZED',resolved_at=$5 WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND work_request_id=$3::uuid AND invitation_id=$4::uuid AND state='PENDING'`, tenantID, scope.LegalEntityID, id, invitationID, now.UTC()); err != nil {
			return VendorWorkRequest{}, fmt.Errorf("finalize vendor work invitation reservation: %w", err)
		}
		current.PendingInvitationID, current.PendingInvitationRequestID = "", ""
	}
	current.DeliveryState, current.Recovery, current.Version, current.UpdatedAt = delivery, recovery, current.Version+1, now.UTC()
	_, err = tx.Exec(ctx, `UPDATE third_party_work_requests SET state=$4,current_invitation_id=NULLIF($5,'')::uuid,delivery_state=$6,recovery=$7,version=$8,updated_at=$9
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND version=$10`, tenantID, scope.LegalEntityID, id, current.State, current.CurrentInvitationID, delivery, recovery, current.Version, current.UpdatedAt, expected)
	if err != nil {
		return VendorWorkRequest{}, fmt.Errorf("record vendor work delivery: %w", err)
	}
	eventType := "VendorWorkSent"
	if invitationID != "" {
		eventType = "VendorWorkInvitationReady"
	}
	if delivery == VendorWorkDeliveryRetryRequired {
		eventType = "VendorWorkDeliveryRetryRequired"
	}
	eventID, err := appendVendorWorkEvent(ctx, tx, tenantID, current, "", eventType)
	if err != nil {
		return VendorWorkRequest{}, err
	}
	if err := r.commitThirdPartyEvents(ctx, tx, vendorWorkCommitProof(eventID, current, eventType)); err != nil {
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
	eventID, err := appendVendorWorkEvent(ctx, tx, tenantID, current, "", "VendorWorkPreparationRetryRequired")
	if err != nil {
		return VendorWorkRequest{}, err
	}
	if err := r.commitThirdPartyEvents(ctx, tx, vendorWorkCommitProof(eventID, current, "VendorWorkPreparationRetryRequired")); err != nil {
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
	eventID, err := appendVendorWorkEvent(ctx, tx, tenantID, current, "", "VendorWorkResponseReceived")
	if err != nil {
		return VendorWorkRequest{}, err
	}
	if err := r.commitThirdPartyEvents(ctx, tx, vendorWorkCommitProof(eventID, current, "VendorWorkResponseReceived")); err != nil {
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
	eventID, err := appendVendorWorkEvent(ctx, tx, tenantID, current, actor, eventType)
	if err != nil {
		return VendorWorkRequest{}, err
	}
	if err := r.commitThirdPartyEvents(ctx, tx, vendorWorkCommitProof(eventID, current, eventType)); err != nil {
		return VendorWorkRequest{}, err
	}
	return current, nil
}

func (r *PostgresRepository) RecordVendorWorkChanges(ctx context.Context, scope Scope, id string, expected int64, link VendorWorkCaptureLink, actor, message string, dueAt, now time.Time) (VendorWorkRequest, error) {
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
	if current.State != VendorWorkUnderReview || link.Sequence != current.CurrentCaptureSequence+1 || !dueAt.After(now) {
		return VendorWorkRequest{}, ErrInvalidAssessmentTransition
	}
	tenantID, err := resolveTenant(ctx, tx, scope.TenantID)
	if err != nil {
		return VendorWorkRequest{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO third_party_work_capture_links(id,tenant_id,legal_entity_id,work_request_id,request_id,sequence,purpose,origin_type,origin_id,origin_version,created_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6,'CLARIFICATION','THIRD_PARTY_WORK',$4::uuid,$6,$7)`, link.ID, tenantID, scope.LegalEntityID, id, link.RequestID, link.Sequence, link.CreatedAt)
	if isUniqueViolation(err) {
		return VendorWorkRequest{}, ErrVersionConflict
	}
	if err != nil {
		return VendorWorkRequest{}, fmt.Errorf("attach vendor work clarification: %w", err)
	}
	current.CurrentRequestID, current.CurrentInvitationID, current.CurrentCaptureSequence, current.SubmissionID = link.RequestID, "", link.Sequence, ""
	current.State, current.DeliveryState, current.Recovery = VendorWorkChangesRequested, VendorWorkDeliveryNotSent, ""
	current.ReviewerPrincipalID, current.ReviewRationale, current.DueAt = actor, message, dueAt.UTC()
	current.Version, current.UpdatedAt = current.Version+1, now.UTC()
	_, err = tx.Exec(ctx, `UPDATE third_party_work_requests SET current_request_id=$4::uuid,current_invitation_id=NULL,current_capture_sequence=$5,submission_id=NULL,state='CHANGES_REQUESTED',delivery_state='NOT_SENT',recovery='',reviewer_principal_id=$6::uuid,review_rationale=$7,due_at=$8,version=$9,updated_at=$10
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND version=$11`, tenantID, scope.LegalEntityID, id, link.RequestID, link.Sequence, actor, message, current.DueAt, current.Version, current.UpdatedAt, expected)
	if err != nil {
		return VendorWorkRequest{}, fmt.Errorf("record vendor work clarification: %w", err)
	}
	eventID, err := appendVendorWorkEvent(ctx, tx, tenantID, current, actor, "VendorWorkChangesRequested")
	if err != nil {
		return VendorWorkRequest{}, err
	}
	if err := r.commitThirdPartyEvents(ctx, tx, vendorWorkCommitProof(eventID, current, "VendorWorkChangesRequested")); err != nil {
		return VendorWorkRequest{}, err
	}
	return current, nil
}

func (r *PostgresRepository) ListVendorWork(ctx context.Context, scope Scope, input VendorWorkListInput) (VendorWorkPage, error) {
	actor, err := identity.Require(ctx)
	if err != nil {
		return VendorWorkPage{}, ErrNotFound
	}
	at := time.Now().UTC()
	if err := actor.Valid(at); err != nil || actor.TenantID != scope.TenantID || actor.LegalEntityID != scope.LegalEntityID || (strings.TrimSpace(input.VisiblePrincipalID) != "" && strings.TrimSpace(input.VisiblePrincipalID) != actor.PrincipalID) {
		return VendorWorkPage{}, ErrNotFound
	}
	query, args, err := postgresVendorWorkListQuery(scope, input, actor.PrincipalID, at)
	if err != nil {
		return VendorWorkPage{}, err
	}
	rows, err := r.pool.Query(ctx, query, args...)
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

func postgresVendorWorkListQuery(scope Scope, input VendorWorkListInput, actorID string, at time.Time) (string, []any, error) {
	args := []any{scope.TenantID, scope.LegalEntityID, input.RelationshipID, string(input.TargetType), input.TargetID}
	cursorClause := ""
	if input.Cursor != "" {
		at, id, err := decodeCursor(input.Cursor)
		if err != nil {
			return "", nil, ErrInvalid
		}
		args = append(args, at, id)
		cursorClause = " AND (w.updated_at,w.id)<($6,$7::uuid)"
	}
	actorParam := len(args) + 1
	args = append(args, actorID)
	atParam := len(args) + 1
	args = append(args, at.UTC())
	args = append(args, input.Limit+1)
	visibility := postgresVendorWorkVisibilitySQL(actorParam, atParam)
	query := vendorWorkSelect + ` JOIN third_party_relationships relationship
		ON relationship.id=w.relationship_id AND relationship.tenant_id=w.tenant_id AND relationship.legal_entity_id=w.legal_entity_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND w.legal_entity_id::text=$2
		  AND ($3='' OR w.relationship_id::text=$3) AND ($4='' OR w.target_type=$4) AND ($5='' OR w.target_id::text=$5)
		  AND (w.target_type<>'MATTER' OR EXISTS (SELECT 1 FROM matters m WHERE m.tenant_id=w.tenant_id AND m.id=w.target_id AND ` + matterVisibilitySQL("m", actorParam) + `))
		  AND (` + fmt.Sprintf("w.owner_principal_id=$%d::uuid OR w.reviewer_principal_id=$%d::uuid OR relationship.business_owner_principal_id=$%d::uuid OR ", actorParam, actorParam, actorParam) + visibility + `)` + cursorClause + `
		ORDER BY w.updated_at DESC,w.id DESC LIMIT $` + fmt.Sprint(len(args))
	return query, args, nil
}

// postgresVendorWorkVisibilitySQL mirrors authority.postgresService for the
// exact vendor-work reviewer route. Keeping the complete route rank,
// delegation, grant and segregation boundaries in the WHERE predicate makes
// keyset pagination operate only on records visible to the current actor.
func postgresVendorWorkVisibilitySQL(actorParam, atParam int) string {
	return fmt.Sprintf(`EXISTS (
		WITH RECURSIVE route_defs AS (
			SELECT ear.source_rule_id AS rule_id,ear.policy_version,ear.priority,
				(CASE WHEN ear.legal_entity_ref<>'*' THEN 8 ELSE 0 END + CASE WHEN ear.object_type<>'*' THEN 4 ELSE 0 END + CASE WHEN ear.object_id<>'*' THEN 2 ELSE 0 END + CASE WHEN ear.decision_type<>'' THEN 1 ELSE 0 END) AS specificity,
				ear.selector_kind,ear.selector_ref
			FROM effective_authority_routes ear
			WHERE ear.tenant_id=w.tenant_id
			  AND (ear.legal_entity_ref='*' OR ear.legal_entity_ref=w.legal_entity_id::text OR ear.legal_entity_ref=(SELECT le.code FROM legal_entities le WHERE le.id=w.legal_entity_id AND le.tenant_id=w.tenant_id))
			  AND (ear.object_type='*' OR upper(ear.object_type)='VENDOR_RELATIONSHIP')
			  AND (ear.object_id='*' OR ear.object_id=w.relationship_id::text)
			  AND ear.responsibility='REVIEWER'
			  AND (ear.decision_type='' OR upper(ear.decision_type)='THIRDPARTY.WORK.REVIEW')
			  AND ear.min_materiality<=3 AND ear.valid_from<=$%[2]d AND (ear.valid_until IS NULL OR $%[2]d<ear.valid_until)
			UNION ALL
			SELECT 'assignment:'||ra.id::text,ra.policy_version,ra.priority,
				(CASE WHEN ra.legal_entity_id IS NOT NULL THEN 8 ELSE 0 END + 4 + CASE WHEN ra.object_id IS NOT NULL THEN 2 ELSE 0 END + CASE WHEN COALESCE(ra.decision_type,'')<>'' THEN 1 ELSE 0 END),
				CASE WHEN ra.principal_id IS NOT NULL THEN 'PRINCIPAL_ID' WHEN ra.position_id IS NOT NULL THEN 'POSITION_ID' ELSE 'ROLE_ID' END,
				COALESCE(ra.principal_id::text,ra.position_id::text,ra.role_template_id::text)
			FROM responsibility_assignments ra
			WHERE ra.tenant_id=w.tenant_id AND (ra.legal_entity_id IS NULL OR ra.legal_entity_id=w.legal_entity_id)
			  AND upper(ra.object_type)='VENDOR_RELATIONSHIP' AND (ra.object_id IS NULL OR ra.object_id=w.relationship_id)
			  AND ra.responsibility='REVIEWER' AND (COALESCE(ra.decision_type,'')='' OR upper(ra.decision_type)='THIRDPARTY.WORK.REVIEW')
			  AND ra.valid_from<=$%[2]d AND (ra.valid_until IS NULL OR $%[2]d<ra.valid_until)
		), resolved AS (
			SELECT rd.rule_id,rd.policy_version,rd.priority,rd.specificity,p.principal_id
			FROM route_defs rd
			JOIN LATERAL (
				SELECT p.id AS principal_id FROM principals p
				WHERE rd.selector_kind IN ('PRINCIPAL','TEAM','QUEUE','COMMITTEE') AND p.tenant_id=w.tenant_id
				  AND (p.id::text=rd.selector_ref OR p.external_ref=rd.selector_ref) AND p.status='ACTIVE'
				  AND p.valid_from<=$%[2]d AND (p.valid_until IS NULL OR $%[2]d<p.valid_until)
				  AND (rd.selector_kind='PRINCIPAL' OR p.kind=rd.selector_kind)
				UNION ALL
				SELECT p.id FROM principals p WHERE rd.selector_kind='PRINCIPAL_ID' AND p.id::text=rd.selector_ref AND p.tenant_id=w.tenant_id
				  AND p.status='ACTIVE' AND p.valid_from<=$%[2]d AND (p.valid_until IS NULL OR $%[2]d<p.valid_until)
				UNION ALL
				SELECT p.id FROM org_positions op JOIN principals p ON p.id=op.occupant_principal_id
				WHERE rd.selector_kind IN ('POSITION','POSITION_ID') AND op.tenant_id=w.tenant_id
				  AND ((rd.selector_kind='POSITION' AND (op.code=rd.selector_ref OR op.id::text=rd.selector_ref)) OR (rd.selector_kind='POSITION_ID' AND op.id::text=rd.selector_ref))
				  AND (op.legal_entity_id IS NULL OR op.legal_entity_id=w.legal_entity_id)
				  AND op.valid_from<=$%[2]d AND (op.valid_until IS NULL OR $%[2]d<op.valid_until)
				  AND p.status='ACTIVE' AND p.valid_from<=$%[2]d AND (p.valid_until IS NULL OR $%[2]d<p.valid_until)
				UNION ALL
				SELECT p.id FROM role_templates rt JOIN position_role_bindings prb ON prb.role_template_id=rt.id
				JOIN org_positions op ON op.id=prb.position_id JOIN principals p ON p.id=op.occupant_principal_id
				WHERE rd.selector_kind IN ('ROLE','ROLE_ID') AND rt.tenant_id=w.tenant_id
				  AND ((rd.selector_kind='ROLE' AND (rt.code=rd.selector_ref OR rt.id::text=rd.selector_ref)) OR (rd.selector_kind='ROLE_ID' AND rt.id::text=rd.selector_ref))
				  AND (op.legal_entity_id IS NULL OR op.legal_entity_id=w.legal_entity_id)
				  AND rt.valid_from<=$%[2]d AND (rt.valid_until IS NULL OR $%[2]d<rt.valid_until)
				  AND prb.valid_from<=$%[2]d AND (prb.valid_until IS NULL OR $%[2]d<prb.valid_until)
				  AND op.valid_from<=$%[2]d AND (op.valid_until IS NULL OR $%[2]d<op.valid_until)
				  AND p.status='ACTIVE' AND p.valid_from<=$%[2]d AND (p.valid_until IS NULL OR $%[2]d<p.valid_until)
			) p ON true
		), route_groups AS (
			SELECT rule_id,policy_version,priority,specificity,array_agg(DISTINCT principal_id ORDER BY principal_id) AS candidates
			FROM resolved GROUP BY rule_id,policy_version,priority,specificity
		), top_rank AS (
			SELECT priority,specificity FROM route_groups ORDER BY priority DESC,specificity DESC,rule_id LIMIT 1
		), top_groups AS (
			SELECT g.* FROM route_groups g JOIN top_rank r USING(priority,specificity)
		), unambiguous AS (
			SELECT count(*)>0 AND count(DISTINCT candidates::text)=1 AS allowed FROM top_groups
		), seeds AS (
			SELECT DISTINCT candidate.principal_id FROM top_groups g CROSS JOIN LATERAL unnest(g.candidates) candidate(principal_id)
		), chain(origin_id,principal_id,path,depth) AS (
			SELECT principal_id,principal_id,ARRAY[principal_id],0 FROM seeds
			UNION ALL
			SELECT c.origin_id,d.to_principal_id,c.path||d.to_principal_id,c.depth+1 FROM chain c JOIN delegations d ON d.from_principal_id=c.principal_id
			WHERE d.tenant_id=w.tenant_id AND d.responsibility='REVIEWER' AND d.status='ACTIVE'
			  AND d.starts_at<=$%[2]d AND $%[2]d<d.ends_at AND c.depth<8 AND NOT d.to_principal_id=ANY(c.path)
			  AND (NOT (d.scope?'legal_entity_id') OR d.scope->>'legal_entity_id' IN ('*',w.legal_entity_id::text))
			  AND (NOT (d.scope?'object_type') OR upper(d.scope->>'object_type') IN ('*','VENDOR_RELATIONSHIP'))
			  AND (NOT (d.scope?'object_id') OR d.scope->>'object_id' IN ('*',w.relationship_id::text))
			  AND (NOT (d.scope?'decision_type') OR upper(d.scope->>'decision_type') IN ('*','THIRDPARTY.WORK.REVIEW'))
		), effective AS (
			SELECT DISTINCT c.origin_id,c.principal_id FROM chain c JOIN principals p ON p.id=c.principal_id
			WHERE p.status='ACTIVE' AND p.valid_from<=$%[2]d AND (p.valid_until IS NULL OR $%[2]d<p.valid_until)
		), relevant_grants AS (
			SELECT ag.* FROM authority_grants ag WHERE ag.tenant_id=w.tenant_id AND (ag.legal_entity_id IS NULL OR ag.legal_entity_id=w.legal_entity_id)
			  AND (ag.decision_type='*' OR upper(ag.decision_type)='THIRDPARTY.WORK.REVIEW')
			  AND ag.valid_from<=$%[2]d AND (ag.valid_until IS NULL OR $%[2]d<ag.valid_until)
		), granted AS (
			SELECT ag.principal_id AS principal_id FROM relevant_grants ag WHERE ag.principal_id IS NOT NULL
			  AND COALESCE(NULLIF(ag.limits->>'min_materiality','')::integer,0)<=3 AND COALESCE(NULLIF(ag.limits->>'max_materiality','')::integer,5)>=3
			UNION
			SELECT op.occupant_principal_id FROM relevant_grants ag JOIN org_positions op ON op.id=ag.position_id
			WHERE ag.position_id IS NOT NULL AND COALESCE(NULLIF(ag.limits->>'min_materiality','')::integer,0)<=3 AND COALESCE(NULLIF(ag.limits->>'max_materiality','')::integer,5)>=3
			  AND op.valid_from<=$%[2]d AND (op.valid_until IS NULL OR $%[2]d<op.valid_until)
			UNION
			SELECT op.occupant_principal_id FROM relevant_grants ag JOIN position_role_bindings prb ON prb.role_template_id=ag.role_template_id
			JOIN org_positions op ON op.id=prb.position_id WHERE ag.role_template_id IS NOT NULL
			  AND COALESCE(NULLIF(ag.limits->>'min_materiality','')::integer,0)<=3 AND COALESCE(NULLIF(ag.limits->>'max_materiality','')::integer,5)>=3
			  AND prb.valid_from<=$%[2]d AND (prb.valid_until IS NULL OR $%[2]d<prb.valid_until)
			  AND op.valid_from<=$%[2]d AND (op.valid_until IS NULL OR $%[2]d<op.valid_until)
		), eligible AS (
			SELECT e.* FROM effective e WHERE NOT EXISTS(SELECT 1 FROM relevant_grants)
			  OR e.principal_id IN (SELECT principal_id FROM granted) OR e.origin_id IN (SELECT principal_id FROM granted)
		)
		SELECT 1 FROM eligible e CROSS JOIN unambiguous u
		WHERE u.allowed AND e.principal_id=$%[1]d::uuid
		  AND NOT EXISTS (
			SELECT 1 FROM org_positions op JOIN position_role_bindings prb ON prb.position_id=op.id
			JOIN role_templates rt ON rt.id=prb.role_template_id JOIN segregation_rules sr ON sr.tenant_id=op.tenant_id AND sr.prohibited_role_code=rt.code
			WHERE op.tenant_id=w.tenant_id AND op.occupant_principal_id=e.principal_id AND sr.responsibility='REVIEWER' AND sr.status='ACTIVE'
			  AND sr.valid_from<=$%[2]d AND (sr.valid_until IS NULL OR $%[2]d<sr.valid_until)
			  AND op.valid_from<=$%[2]d AND (op.valid_until IS NULL OR $%[2]d<op.valid_until)
			  AND prb.valid_from<=$%[2]d AND (prb.valid_until IS NULL OR $%[2]d<prb.valid_until)
		  )
	)`, actorParam, atParam)
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

const vendorWorkSelect = `SELECT w.id::text,t.slug,w.legal_entity_id::text,w.relationship_id::text,COALESCE(w.program_link_id,w.matter_link_id)::text,w.target_type,w.target_id::text,w.purpose,w.instructions,w.owner_principal_id::text,COALESCE(w.reviewer_principal_id::text,''),w.form_template_id::text,w.form_template_version,w.presentation,COALESCE(w.current_request_id::text,''),COALESCE(w.current_invitation_id::text,''),COALESCE((SELECT reservation.invitation_id::text FROM third_party_work_invitation_reservations reservation WHERE reservation.tenant_id=w.tenant_id AND reservation.legal_entity_id=w.legal_entity_id AND reservation.work_request_id=w.id AND reservation.state='PENDING' ORDER BY reservation.created_at DESC LIMIT 1),''),COALESCE((SELECT reservation.request_id::text FROM third_party_work_invitation_reservations reservation WHERE reservation.tenant_id=w.tenant_id AND reservation.legal_entity_id=w.legal_entity_id AND reservation.work_request_id=w.id AND reservation.state='PENDING' ORDER BY reservation.created_at DESC LIMIT 1),''),w.current_capture_sequence,COALESCE(w.submission_id::text,''),w.state,w.delivery_state,w.recovery,w.review_rationale,w.cancellation_reason,w.due_at,w.version,w.created_at,w.updated_at,w.response_received_at,w.review_started_at,w.accepted_at,w.cancelled_at FROM third_party_work_requests w JOIN tenants t ON t.id=w.tenant_id`

func scanVendorWork(row rowScanner) (VendorWorkRequest, error) {
	var value VendorWorkRequest
	err := row.Scan(&value.ID, &value.TenantID, &value.LegalEntityID, &value.RelationshipID, &value.RelationshipLinkID, &value.TargetType, &value.TargetID, &value.Purpose, &value.Instructions, &value.OwnerPrincipalID, &value.ReviewerPrincipalID, &value.FormTemplateID, &value.FormTemplateVersion, &value.Presentation, &value.CurrentRequestID, &value.CurrentInvitationID, &value.PendingInvitationID, &value.PendingInvitationRequestID, &value.CurrentCaptureSequence, &value.SubmissionID, &value.State, &value.DeliveryState, &value.Recovery, &value.ReviewRationale, &value.CancellationReason, &value.DueAt, &value.Version, &value.CreatedAt, &value.UpdatedAt, &value.ResponseReceivedAt, &value.ReviewStartedAt, &value.AcceptedAt, &value.CancelledAt)
	return value, err
}

func appendVendorWorkEvent(ctx context.Context, tx pgx.Tx, tenantID string, value VendorWorkRequest, actor, eventType string) (string, error) {
	var eventID string
	err := tx.QueryRow(ctx, `INSERT INTO third_party_work_events(tenant_id,legal_entity_id,work_request_id,work_version,actor_principal_id,event_type,payload,occurred_at) VALUES($1::uuid,$2::uuid,$3::uuid,$4,NULLIF($5,'')::uuid,$6,jsonb_build_object('state',$7::text,'delivery_state',$8::text,'request_id',$9::text,'submission_id',$10::text),$11) RETURNING id::text`, tenantID, value.LegalEntityID, value.ID, value.Version, actor, eventType, value.State, value.DeliveryState, value.CurrentRequestID, value.SubmissionID, value.UpdatedAt).Scan(&eventID)
	if err != nil {
		return "", fmt.Errorf("append vendor work event: %w", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at) VALUES($1::uuid,'VENDOR_WORK_REQUEST',$2::uuid,$3,jsonb_build_object('version',$4::bigint,'state',$5::text,'delivery_state',$6::text),$7,$7)`, tenantID, value.ID, eventType, value.Version, value.State, value.DeliveryState, value.UpdatedAt)
	if err != nil {
		return "", fmt.Errorf("append vendor work outbox event: %w", err)
	}
	return eventID, nil
}

var _ VendorWorkRepository = (*PostgresRepository)(nil)
