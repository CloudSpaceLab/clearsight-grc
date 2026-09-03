//go:build postgres

package evidence

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

func (store *PostgresDistributionStore) CommitAccessSession(ctx context.Context, commit accessSessionCommit) error {
	tx, err := store.repo.pool.Begin(ctx)
	if err != nil {
		return ErrAccessVerificationFailed
	}
	defer tx.Rollback(ctx)
	persisted, err := scanAccessRoute(tx.QueryRow(ctx, accessRouteSelect+` WHERE ar.id=$1::uuid FOR UPDATE`, commit.Route.ID))
	if err != nil || persisted.TenantID != commit.Session.TenantID || persisted.LegalEntityID != commit.Session.LegalEntityID || persisted.DistributionID != commit.Session.DistributionID ||
		persisted.RevokedAt != nil || !persisted.ExpiresAt.After(commit.Session.CreatedAt) ||
		commit.Recipient.ID != commit.Session.RecipientID || commit.Recipient.RequestID != commit.Session.RequestID || len(eligibleAccessRecipients(persisted, []DistributionRecipient{commit.Recipient})) != 1 ||
		!commit.Session.ExpiresAt.After(commit.Session.CreatedAt) || commit.Session.ExpiresAt.After(persisted.ExpiresAt) || !accessGrantAssuranceMatches(persisted.Policy, commit.Session.Assurance) {
		return ErrAccessVerificationFailed
	}
	if commit.Challenge != nil {
		if commit.Challenge.ConsumedAt == nil || persisted.Policy == AccessDirectMagicLink {
			return ErrAccessVerificationFailed
		}
		tag, updateErr := tx.Exec(ctx, `
			UPDATE capture_otp_challenges
			SET consumed_at=$2
			WHERE id=$1::uuid AND route_id=$3::uuid AND recipient_id=$4::uuid
			  AND attempts=$5 AND resends=$6 AND code_hash=$7 AND consumed_at IS NULL`,
			commit.Challenge.ID, *commit.Challenge.ConsumedAt, persisted.ID, commit.Recipient.ID,
			commit.ExpectedAttempts, commit.ExpectedResends, commit.ExpectedDigest)
		if updateErr != nil || tag.RowsAffected() != 1 {
			return ErrAccessVerificationFailed
		}
	} else if persisted.Policy != AccessDirectMagicLink {
		return ErrAccessVerificationFailed
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO capture_distribution_sessions(
			id,tenant_id,legal_entity_id,distribution_id,recipient_id,request_id,route_id,token_hash,
			audience_hint,assurance,expires_at,created_at
		) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6::uuid,$7::uuid,$8,$9,$10,$11,$12)`,
		commit.Session.ID, commit.Session.TenantID, commit.Session.LegalEntityID, commit.Session.DistributionID,
		commit.Session.RecipientID, commit.Session.RequestID, commit.Session.RouteID, commit.Session.TokenHash,
		commit.Session.AudienceHint, commit.Session.Assurance, commit.Session.ExpiresAt, commit.Session.CreatedAt); err != nil {
		return ErrAccessVerificationFailed
	}
	if err := tx.Commit(ctx); err != nil {
		return ErrAccessVerificationFailed
	}
	return nil
}

func (store *PostgresDistributionStore) RotateAccessRoute(ctx context.Context, current, next AccessRoute, now time.Time) error {
	tx, err := store.repo.pool.Begin(ctx)
	if err != nil {
		return ErrDistributionAccessUnavailable
	}
	defer tx.Rollback(ctx)
	persisted, err := scanAccessRoute(tx.QueryRow(ctx, accessRouteSelect+` WHERE ar.id=$1::uuid FOR UPDATE`, current.ID))
	if err != nil || persisted.RevokedAt != nil || persisted.TenantID != next.TenantID || persisted.LegalEntityID != next.LegalEntityID || persisted.DistributionID != next.DistributionID || !validPersistedPostgresAccessRoute(next) {
		return ErrDistributionAccessUnavailable
	}
	if _, err := tx.Exec(ctx, `UPDATE capture_access_routes SET revoked_at=$2 WHERE id=$1::uuid AND revoked_at IS NULL`, persisted.ID, now); err != nil {
		return ErrDistributionAccessUnavailable
	}
	if _, err := tx.Exec(ctx, `UPDATE capture_distribution_sessions SET revoked_at=COALESCE(revoked_at,$2) WHERE route_id=$1::uuid`, persisted.ID, now); err != nil {
		return ErrDistributionAccessUnavailable
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO capture_access_routes(
			id,tenant_id,legal_entity_id,distribution_id,recipient_id,access_policy,selector_hash,audience_hint,
			expires_at,created_by,created_at
		) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,NULLIF($5,'')::uuid,$6,$7,$8,$9,$10::uuid,$11)`,
		next.ID, next.TenantID, next.LegalEntityID, next.DistributionID, next.RecipientID, next.Policy,
		next.SelectorHash, next.AudienceHint, next.ExpiresAt, next.CreatedBy, next.CreatedAt); err != nil {
		return ErrDistributionAccessUnavailable
	}
	if err := tx.Commit(ctx); err != nil {
		return ErrDistributionAccessUnavailable
	}
	return nil
}

func (store *PostgresDistributionStore) RevokeAccessRoute(ctx context.Context, route AccessRoute, now time.Time) error {
	tx, err := store.repo.pool.Begin(ctx)
	if err != nil {
		return ErrDistributionAccessUnavailable
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `
		UPDATE capture_access_routes SET revoked_at=COALESCE(revoked_at,$5)
		WHERE id=$1::uuid AND tenant_id=$2::uuid AND legal_entity_id=$3::uuid AND distribution_id=$4::uuid`,
		route.ID, route.TenantID, route.LegalEntityID, route.DistributionID, now)
	if err != nil || tag.RowsAffected() != 1 {
		return ErrDistributionAccessUnavailable
	}
	if _, err := tx.Exec(ctx, `UPDATE capture_distribution_sessions SET revoked_at=COALESCE(revoked_at,$2) WHERE route_id=$1::uuid`, route.ID, now); err != nil {
		return ErrDistributionAccessUnavailable
	}
	if err := tx.Commit(ctx); err != nil {
		return ErrDistributionAccessUnavailable
	}
	return nil
}

func (store *PostgresDistributionStore) DistributionSessionByTokenHash(ctx context.Context, tokenHash []byte, now time.Time) (DistributionAccessSession, error) {
	var session DistributionAccessSession
	var revoked sql.NullTime
	err := store.repo.pool.QueryRow(ctx, `
		SELECT s.id::text,s.tenant_id::text,s.legal_entity_id::text,s.distribution_id::text,s.recipient_id::text,
		       s.request_id::text,s.route_id::text,s.audience_hint,s.assurance,s.token_hash,s.expires_at,s.revoked_at,s.created_at
		FROM capture_distribution_sessions s
		JOIN capture_access_routes ar ON ar.id=s.route_id AND ar.tenant_id=s.tenant_id AND ar.legal_entity_id=s.legal_entity_id AND ar.distribution_id=s.distribution_id
		WHERE s.token_hash=$1 AND s.expires_at>$2 AND ar.revoked_at IS NULL AND ar.expires_at>$2`, tokenHash, now).Scan(
		&session.ID, &session.TenantID, &session.LegalEntityID, &session.DistributionID, &session.RecipientID,
		&session.RequestID, &session.RouteID, &session.AudienceHint, &session.Assurance, &session.TokenHash,
		&session.ExpiresAt, &revoked, &session.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) || err != nil || revoked.Valid {
		return DistributionAccessSession{}, ErrSessionInvalid
	}
	return session, nil
}
