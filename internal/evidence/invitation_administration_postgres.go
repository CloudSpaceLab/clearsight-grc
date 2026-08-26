//go:build postgres

package evidence

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) ListInvitationMetadata(ctx context.Context, tenant, requestID string, limit int) ([]InvitationMetadata, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT ei.id::text,ei.request_id::text,ei.audience_hint,ei.purpose,ei.expires_at,ei.max_redemptions,ei.redemptions,ei.revoked_at,ei.created_at
		FROM capture_invitations ei
		JOIN capture_requests er ON er.tenant_id=ei.tenant_id AND er.id=ei.request_id
		JOIN tenants t ON t.id=ei.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND er.id=$2::uuid
		ORDER BY ei.created_at DESC,ei.id DESC LIMIT $3`, tenant, requestID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]InvitationMetadata, 0, limit)
	for rows.Next() {
		var value InvitationMetadata
		var revoked sql.NullTime
		if err := rows.Scan(&value.ID, &value.RequestID, &value.AudienceHint, &value.Purpose, &value.ExpiresAt, &value.MaxRedemptions, &value.Redemptions, &revoked, &value.CreatedAt); err != nil {
			return nil, err
		}
		if revoked.Valid {
			value.RevokedAt = pointerTime(revoked.Time)
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func lockRequester(ctx context.Context, tx pgx.Tx, tenant, requestID, actor string) (RequestStatus, time.Time, error) {
	var status RequestStatus
	var deadline time.Time
	var createdBy string
	err := tx.QueryRow(ctx, `
		SELECT er.status,er.deadline,COALESCE(er.created_by::text,'')
		FROM capture_requests er JOIN tenants t ON t.id=er.tenant_id
		WHERE er.id=$1::uuid AND (t.id::text=$2 OR t.slug=$2) FOR UPDATE`, requestID, tenant).Scan(&status, &deadline, &createdBy)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", time.Time{}, ErrNotFound
	}
	if err != nil {
		return "", time.Time{}, err
	}
	if createdBy == "" || createdBy != actor {
		return "", time.Time{}, ErrRecipientManagerRequired
	}
	return status, deadline, nil
}

func (r *PostgresRepository) RevokeInvitationForRequester(ctx context.Context, input RevokeInvitationAsRequesterInput, now time.Time) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, _, err := lockRequester(ctx, tx, input.TenantID, input.RequestID, input.ActorPrincipalID); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
		UPDATE capture_invitations SET revoked_at=COALESCE(revoked_at,$4)
		WHERE id=$1::uuid AND request_id=$2::uuid
		  AND tenant_id=(SELECT id FROM tenants WHERE id::text=$3 OR slug=$3)`, input.InvitationID, input.RequestID, input.TenantID, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	if _, err := tx.Exec(ctx, `
		UPDATE capture_sessions SET revoked_at=COALESCE(revoked_at,$4)
		WHERE invitation_id=$1::uuid AND request_id=$2::uuid
		  AND tenant_id=(SELECT id FROM tenants WHERE id::text=$3 OR slug=$3)`, input.InvitationID, input.RequestID, input.TenantID, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PostgresRepository) RevokeSessionForRequester(ctx context.Context, input RevokeSessionAsRequesterInput, now time.Time) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, _, err := lockRequester(ctx, tx, input.TenantID, input.RequestID, input.ActorPrincipalID); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
		UPDATE capture_sessions SET revoked_at=COALESCE(revoked_at,$4)
		WHERE id=$1::uuid AND request_id=$2::uuid
		  AND tenant_id=(SELECT id FROM tenants WHERE id::text=$3 OR slug=$3)`, input.SessionID, input.RequestID, input.TenantID, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return tx.Commit(ctx)
}

func (r *PostgresRepository) ReplaceInvitation(ctx context.Context, input ReplaceInvitationInput, replacement Invitation, now time.Time) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	status, deadline, err := lockRequester(ctx, tx, input.TenantID, input.RequestID, input.ActorPrincipalID)
	if err != nil {
		return err
	}
	if (status != RequestReady && status != RequestInProgress) || !now.Before(deadline) || !replacement.ExpiresAt.After(now) || replacement.ExpiresAt.After(deadline) {
		return ErrRequestClosed
	}
	tag, err := tx.Exec(ctx, `
		UPDATE capture_invitations SET revoked_at=COALESCE(revoked_at,$4)
		WHERE id=$1::uuid AND request_id=$2::uuid
		  AND tenant_id=(SELECT id FROM tenants WHERE id::text=$3 OR slug=$3)`, input.InvitationID, input.RequestID, input.TenantID, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	if _, err := tx.Exec(ctx, `
		UPDATE capture_sessions SET revoked_at=COALESCE(revoked_at,$4)
		WHERE invitation_id=$1::uuid AND request_id=$2::uuid
		  AND tenant_id=(SELECT id FROM tenants WHERE id::text=$3 OR slug=$3)`, input.InvitationID, input.RequestID, input.TenantID, now); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO capture_invitations(id,tenant_id,request_id,token_hash,audience_hash,audience_hint,purpose,expires_at,max_redemptions,created_by,created_at)
		VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4,$5,$6,$7,$8,$9,$10::uuid,$11)`,
		replacement.ID, replacement.TenantID, replacement.RequestID, replacement.TokenHash, replacement.AudienceHash,
		replacement.AudienceHint, replacement.Purpose, replacement.ExpiresAt, replacement.MaxRedemptions, replacement.CreatedBy, replacement.CreatedAt); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

var _ invitationAdministrationStore = (*PostgresRepository)(nil)
