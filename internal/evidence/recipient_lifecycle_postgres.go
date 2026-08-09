//go:build postgres

package evidence

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type lockedRecipientRequest struct {
	status      RequestStatus
	deadline    time.Time
	createdBy   string
	version     int64
	recipient   Recipient
}

func (r *PostgresRepository) lockRecipientRequest(ctx context.Context, tx pgx.Tx, tenant, requestID string) (lockedRecipientRequest, error) {
	var value lockedRecipientRequest
	var recipientType, principalID, hint, state, issueReason string
	var audienceHash []byte
	err := tx.QueryRow(ctx, `
		SELECT er.status,er.deadline,COALESCE(er.created_by::text,''),er.version,
		       COALESCE(er.recipient_type,''),COALESCE(er.recipient_principal_id::text,''),
		       COALESCE(er.recipient_audience_hash,''::bytea),er.recipient_hint,
		       er.recipient_state,er.recipient_revision,er.recipient_issue_reason
		FROM capture_requests er
		JOIN tenants t ON t.id=er.tenant_id
		WHERE er.id=$1::uuid AND (t.id::text=$2 OR t.slug=$2)
		FOR UPDATE`, requestID, tenant).Scan(
		&value.status, &value.deadline, &value.createdBy, &value.version,
		&recipientType, &principalID, &audienceHash, &hint,
		&state, &value.recipient.Revision, &issueReason,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return lockedRecipientRequest{}, ErrNotFound
	}
	if err != nil {
		return lockedRecipientRequest{}, err
	}
	value.recipient.Type = RecipientType(recipientType)
	value.recipient.PrincipalID = principalID
	value.recipient.AudienceHash = append([]byte(nil), audienceHash...)
	value.recipient.AudienceHint = hint
	value.recipient.State = RecipientState(state)
	value.recipient.IssueReason = issueReason
	return value, nil
}

func (r *PostgresRepository) DeclareWrongRecipient(ctx context.Context, input DeclareWrongRecipientInput, now time.Time) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	current, err := r.lockRecipientRequest(ctx, tx, input.TenantID, input.RequestID)
	if err != nil {
		return err
	}
	if current.status != RequestReady && current.status != RequestInProgress || !now.Before(current.deadline) {
		return ErrRequestClosed
	}
	if input.ExpectedVersion <= 0 || current.version != input.ExpectedVersion {
		return ErrVersionConflict
	}
	if current.recipient.State != RecipientStateAssigned || current.recipient.Type != RecipientInternalPrincipal || current.recipient.PrincipalID != input.ActorPrincipalID {
		return ErrRecipientMismatch
	}

	reason := strings.TrimSpace(input.Reason)
	if _, err := tx.Exec(ctx, `
		UPDATE capture_requests
		SET recipient_state='REASSIGNMENT_REQUIRED',recipient_issue_reason=$3,version=version+1,updated_at=$4
		WHERE id=$1::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2)`,
		input.RequestID, input.TenantID, reason, now); err != nil {
		return fmt.Errorf("mark evidence request wrong recipient: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO capture_recipient_history(
			tenant_id,request_id,event_type,from_recipient_type,from_principal_id,from_audience_hash,from_audience_hint,
			actor_principal_id,reason,recipient_revision,request_version,occurred_at
		) VALUES(
			(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2::uuid,'WRONG_RECIPIENT',$3,NULLIF($4,'')::uuid,$5,$6,
			$7::uuid,$8,$9,$10,$11
		)`, input.TenantID, input.RequestID, current.recipient.Type, current.recipient.PrincipalID,
		nullableAudienceHash(current.recipient), current.recipient.AudienceHint, input.ActorPrincipalID, reason,
		current.recipient.Revision, current.version+1, now); err != nil {
		return fmt.Errorf("record wrong recipient history: %w", err)
	}
	return tx.Commit(ctx)
}

func (r *PostgresRepository) ReassignRecipient(ctx context.Context, input ReassignRecipientInput, next Recipient, now time.Time) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	current, err := r.lockRecipientRequest(ctx, tx, input.TenantID, input.RequestID)
	if err != nil {
		return err
	}
	if current.status != RequestReady && current.status != RequestInProgress || !now.Before(current.deadline) {
		return ErrRequestClosed
	}
	if input.ExpectedVersion <= 0 || current.version != input.ExpectedVersion {
		return ErrVersionConflict
	}
	if strings.TrimSpace(current.createdBy) == "" || current.createdBy != input.ActorPrincipalID {
		return ErrRecipientManagerRequired
	}
	if current.recipient.Type == "" || current.recipient.State == RecipientStateLegacyUnassigned {
		return ErrRecipientInvalid
	}
	if sameRecipient(current.recipient, next) && current.recipient.State == RecipientStateAssigned {
		return ErrRecipientInvalid
	}

	nextRevision := current.recipient.Revision + 1
	if nextRevision < 1 {
		nextRevision = 1
	}
	next.Revision = nextRevision
	next.State = RecipientStateAssigned
	reason := strings.TrimSpace(input.Reason)

	if _, err := tx.Exec(ctx, `
		UPDATE capture_requests
		SET recipient_type=$3,
		    recipient_principal_id=NULLIF($4,'')::uuid,
		    recipient_audience_hash=$5,
		    recipient_hint=$6,
		    recipient_state='ASSIGNED',
		    recipient_revision=$7,
		    recipient_issue_reason='',
		    version=version+1,
		    updated_at=$8
		WHERE id=$1::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2)`,
		input.RequestID, input.TenantID, next.Type, next.PrincipalID, nullableAudienceHash(next), next.AudienceHint, nextRevision, now); err != nil {
		return fmt.Errorf("reassign evidence request recipient: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE capture_invitations
		SET revoked_at=COALESCE(revoked_at,$3)
		WHERE request_id=$1::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2) AND revoked_at IS NULL`,
		input.RequestID, input.TenantID, now); err != nil {
		return fmt.Errorf("revoke superseded invitations: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE capture_sessions
		SET revoked_at=COALESCE(revoked_at,$3)
		WHERE request_id=$1::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2) AND revoked_at IS NULL`,
		input.RequestID, input.TenantID, now); err != nil {
		return fmt.Errorf("revoke superseded sessions: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO capture_recipient_history(
			tenant_id,request_id,event_type,
			from_recipient_type,from_principal_id,from_audience_hash,from_audience_hint,
			to_recipient_type,to_principal_id,to_audience_hash,to_audience_hint,
			actor_principal_id,reason,recipient_revision,request_version,occurred_at
		) VALUES(
			(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2::uuid,'REASSIGNED',
			$3,NULLIF($4,'')::uuid,$5,$6,
			$7,NULLIF($8,'')::uuid,$9,$10,
			$11::uuid,$12,$13,$14,$15
		)`, input.TenantID, input.RequestID,
		current.recipient.Type, current.recipient.PrincipalID, nullableAudienceHash(current.recipient), current.recipient.AudienceHint,
		next.Type, next.PrincipalID, nullableAudienceHash(next), next.AudienceHint,
		input.ActorPrincipalID, reason, nextRevision, current.version+1, now); err != nil {
		return fmt.Errorf("record recipient reassignment history: %w", err)
	}
	return tx.Commit(ctx)
}

func sameRecipient(left, right Recipient) bool {
	if left.Type != right.Type {
		return false
	}
	switch left.Type {
	case RecipientInternalPrincipal:
		return left.PrincipalID == right.PrincipalID
	case RecipientExternalAudience:
		return bytes.Equal(left.AudienceHash, right.AudienceHash)
	default:
		return false
	}
}

var _ recipientLifecycleStore = (*PostgresRepository)(nil)
