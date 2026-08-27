//go:build postgres

package evidence

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func (store *PostgresDistributionStore) GetRequest(ctx context.Context, tenantID, requestID string) (Request, error) {
	if store == nil || store.repo == nil {
		return Request{}, ErrNotFound
	}
	return store.repo.GetRequest(ctx, tenantID, requestID)
}

func (store *PostgresDistributionStore) CreateAccessRoutes(ctx context.Context, routes []AccessRoute) error {
	if store == nil || store.repo == nil || store.repo.pool == nil || len(routes) == 0 {
		return ErrDistributionAccessUnavailable
	}
	tx, err := store.repo.pool.Begin(ctx)
	if err != nil {
		return ErrDistributionAccessUnavailable
	}
	defer tx.Rollback(ctx)
	for _, route := range routes {
		if !validPersistedPostgresAccessRoute(route) {
			return ErrDistributionAccessUnavailable
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO capture_access_routes(
				id,tenant_id,legal_entity_id,distribution_id,recipient_id,access_policy,selector_hash,
				audience_hint,expires_at,max_redemptions,redemptions,created_by,created_at
			) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,NULLIF($5,'')::uuid,$6,$7,$8,$9,$10,$11,$12::uuid,$13)`,
			route.ID, route.TenantID, route.LegalEntityID, route.DistributionID, route.RecipientID,
			route.Policy, route.SelectorHash, route.AudienceHint, route.ExpiresAt, route.MaxRedemptions,
			route.Redemptions, route.CreatedBy, route.CreatedAt); err != nil {
			return ErrDistributionAccessUnavailable
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return ErrDistributionAccessUnavailable
	}
	return nil
}

func (store *PostgresDistributionStore) AccessRouteBySelectorHash(ctx context.Context, selectorHash []byte) (AccessRoute, error) {
	if store == nil || store.repo == nil || store.repo.pool == nil || len(selectorHash) != 32 {
		return AccessRoute{}, ErrDistributionAccessUnavailable
	}
	route, err := scanAccessRoute(store.repo.pool.QueryRow(ctx, accessRouteSelect+` WHERE ar.selector_hash=$1`, selectorHash))
	if errors.Is(err, pgx.ErrNoRows) || err != nil {
		return AccessRoute{}, ErrDistributionAccessUnavailable
	}
	return route, nil
}

func (store *PostgresDistributionStore) AccessRouteByID(ctx context.Context, tenantID, legalEntityID, distributionID, routeID string) (AccessRoute, error) {
	if store == nil || store.repo == nil || store.repo.pool == nil {
		return AccessRoute{}, ErrDistributionAccessUnavailable
	}
	route, err := scanAccessRoute(store.repo.pool.QueryRow(ctx, accessRouteSelect+`
		WHERE ar.id=$1::uuid AND ar.tenant_id=$2::uuid AND ar.legal_entity_id=$3::uuid AND ar.distribution_id=$4::uuid`,
		routeID, tenantID, legalEntityID, distributionID))
	if errors.Is(err, pgx.ErrNoRows) || err != nil {
		return AccessRoute{}, ErrDistributionAccessUnavailable
	}
	return route, nil
}

func (store *PostgresDistributionStore) ProtectedRecipientForAccess(ctx context.Context, route AccessRoute, recipientID string) (DistributionRecipient, protectedRecipientAddress, error) {
	if store == nil || store.repo == nil || store.repo.pool == nil {
		return DistributionRecipient{}, protectedRecipientAddress{}, ErrAccessVerificationFailed
	}
	var recipient DistributionRecipient
	var protected protectedRecipientAddress
	err := store.repo.pool.QueryRow(ctx, `
		SELECT r.id::text,r.distribution_id::text,r.tenant_id::text,r.legal_entity_id::text,r.role,r.recipient_type,
		       COALESCE(r.principal_id::text,''),COALESCE(r.request_id::text,''),r.audience_hint,r.contact_label,r.state,r.version,r.created_at,r.updated_at,
		       r.address_hash,r.address_ciphertext,r.address_key_id
		FROM capture_distribution_recipients r
		WHERE r.id=$1::uuid AND r.tenant_id=$2::uuid AND r.legal_entity_id=$3::uuid AND r.distribution_id=$4::uuid`,
		recipientID, route.TenantID, route.LegalEntityID, route.DistributionID).Scan(
		&recipient.ID, &recipient.DistributionID, &recipient.TenantID, &recipient.LegalEntityID,
		&recipient.Role, &recipient.Type, &recipient.PrincipalID, &recipient.RequestID,
		&recipient.AudienceHint, &recipient.ContactLabel, &recipient.State, &recipient.Version,
		&recipient.CreatedAt, &recipient.UpdatedAt, &protected.Hash, &protected.Ciphertext, &protected.KeyID,
	)
	if errors.Is(err, pgx.ErrNoRows) || err != nil || len(eligibleAccessRecipients(route, []DistributionRecipient{recipient})) != 1 || len(protected.Hash) != 32 || len(protected.Ciphertext) == 0 || protected.KeyID == "" {
		return DistributionRecipient{}, protectedRecipientAddress{}, ErrAccessVerificationFailed
	}
	return recipient, protected, nil
}

func (store *PostgresDistributionStore) ActiveOTPChallenge(ctx context.Context, route AccessRoute, recipientID string, now time.Time) (otpChallengeSnapshot, error) {
	challenge, err := scanOTPChallenge(store.repo.pool.QueryRow(ctx, otpChallengeSelect+`
		WHERE c.route_id=$1::uuid AND c.tenant_id=$2::uuid AND c.legal_entity_id=$3::uuid AND c.distribution_id=$4::uuid
		  AND c.recipient_id=$5::uuid AND c.consumed_at IS NULL AND c.attempts<c.max_attempts AND c.expires_at>$6
		ORDER BY c.created_at DESC,c.id DESC LIMIT 1`, route.ID, route.TenantID, route.LegalEntityID, route.DistributionID, recipientID, now))
	if errors.Is(err, pgx.ErrNoRows) {
		return otpChallengeSnapshot{}, nil
	}
	if err != nil {
		return otpChallengeSnapshot{}, ErrAccessVerificationFailed
	}
	return otpChallengeSnapshot{Challenge: challenge, Found: true}, nil
}

func (store *PostgresDistributionStore) OTPChallengeByID(ctx context.Context, route AccessRoute, challengeID string, _ time.Time) (OTPChallenge, error) {
	challenge, err := scanOTPChallenge(store.repo.pool.QueryRow(ctx, otpChallengeSelect+`
		WHERE c.id=$1::uuid AND c.route_id=$2::uuid AND c.tenant_id=$3::uuid AND c.legal_entity_id=$4::uuid AND c.distribution_id=$5::uuid`,
		challengeID, route.ID, route.TenantID, route.LegalEntityID, route.DistributionID))
	if errors.Is(err, pgx.ErrNoRows) || err != nil {
		return OTPChallenge{}, ErrAccessVerificationFailed
	}
	return challenge, nil
}

func (store *PostgresDistributionStore) CreateOTPChallenge(ctx context.Context, challenge OTPChallenge) error {
	tx, err := store.repo.pool.Begin(ctx)
	if err != nil {
		return ErrAccessVerificationFailed
	}
	defer tx.Rollback(ctx)
	var lockedRouteID string
	if err := tx.QueryRow(ctx, `
		SELECT id::text FROM capture_access_routes
		WHERE id=$1::uuid AND tenant_id=$2::uuid AND legal_entity_id=$3::uuid AND distribution_id=$4::uuid
		FOR UPDATE`, challenge.RouteID, challenge.TenantID, challenge.LegalEntityID, challenge.DistributionID).Scan(&lockedRouteID); err != nil {
		return ErrAccessVerificationFailed
	}
	var active bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM capture_otp_challenges
			WHERE route_id=$1::uuid AND recipient_id=$2::uuid AND consumed_at IS NULL
			  AND attempts<max_attempts AND expires_at>$3
		)`, challenge.RouteID, challenge.RecipientID, challenge.CreatedAt).Scan(&active); err != nil || active {
		return ErrAccessVerificationFailed
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO capture_otp_challenges(
			id,tenant_id,legal_entity_id,distribution_id,route_id,recipient_id,code_hash,
			attempts,max_attempts,resends,max_resends,expires_at,created_at
		) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6::uuid,$7,$8,$9,$10,$11,$12,$13)`,
		challenge.ID, challenge.TenantID, challenge.LegalEntityID, challenge.DistributionID,
		challenge.RouteID, challenge.RecipientID, challenge.Digest, challenge.Attempts, challenge.MaxAttempts,
		challenge.Resends, challenge.MaxResends, challenge.ExpiresAt, challenge.CreatedAt); err != nil {
		return ErrAccessVerificationFailed
	}
	if err := tx.Commit(ctx); err != nil {
		return ErrAccessVerificationFailed
	}
	return nil
}

func (store *PostgresDistributionStore) UpdateOTPChallenge(ctx context.Context, challenge OTPChallenge, expectedAttempts, expectedResends int, expectedDigest []byte) error {
	tag, err := store.repo.pool.Exec(ctx, `
		UPDATE capture_otp_challenges
		SET code_hash=$2,attempts=$3,resends=$4,expires_at=$5
		WHERE id=$1::uuid AND route_id=$6::uuid AND recipient_id=$7::uuid
		  AND attempts=$8 AND resends=$9 AND code_hash=$10 AND consumed_at IS NULL`,
		challenge.ID, challenge.Digest, challenge.Attempts, challenge.Resends, challenge.ExpiresAt,
		challenge.RouteID, challenge.RecipientID, expectedAttempts, expectedResends, expectedDigest)
	if err != nil || tag.RowsAffected() != 1 {
		return ErrAccessVerificationFailed
	}
	return nil
}

func (store *PostgresDistributionStore) CommitAccessSession(ctx context.Context, commit accessSessionCommit) error {
	tx, err := store.repo.pool.Begin(ctx)
	if err != nil {
		return ErrAccessVerificationFailed
	}
	defer tx.Rollback(ctx)
	persisted, err := scanAccessRoute(tx.QueryRow(ctx, accessRouteSelect+` WHERE ar.id=$1::uuid FOR UPDATE`, commit.Route.ID))
	if err != nil || persisted.TenantID != commit.Session.TenantID || persisted.LegalEntityID != commit.Session.LegalEntityID || persisted.DistributionID != commit.Session.DistributionID ||
		persisted.RevokedAt != nil || persisted.Redemptions != commit.ExpectedRedemptions || persisted.Redemptions >= persisted.MaxRedemptions || !persisted.ExpiresAt.After(commit.Session.CreatedAt) ||
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
	if tag, err := tx.Exec(ctx, `
		UPDATE capture_access_routes SET redemptions=redemptions+1
		WHERE id=$1::uuid AND redemptions=$2 AND revoked_at IS NULL AND redemptions<max_redemptions`,
		persisted.ID, commit.ExpectedRedemptions); err != nil || tag.RowsAffected() != 1 {
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
			expires_at,max_redemptions,redemptions,created_by,created_at
		) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,NULLIF($5,'')::uuid,$6,$7,$8,$9,$10,0,$11::uuid,$12)`,
		next.ID, next.TenantID, next.LegalEntityID, next.DistributionID, next.RecipientID, next.Policy,
		next.SelectorHash, next.AudienceHint, next.ExpiresAt, next.MaxRedemptions, next.CreatedBy, next.CreatedAt); err != nil {
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

const accessRouteSelect = `
	SELECT ar.id::text,ar.tenant_id::text,ar.legal_entity_id::text,ar.distribution_id::text,
	       COALESCE(ar.recipient_id::text,''),ar.access_policy,ar.selector_hash,ar.audience_hint,
	       ar.expires_at,ar.max_redemptions,ar.redemptions,ar.revoked_at,ar.created_by::text,ar.created_at
	FROM capture_access_routes ar`

func scanAccessRoute(row scanner) (AccessRoute, error) {
	var route AccessRoute
	var revoked sql.NullTime
	if err := row.Scan(&route.ID, &route.TenantID, &route.LegalEntityID, &route.DistributionID, &route.RecipientID,
		&route.Policy, &route.SelectorHash, &route.AudienceHint, &route.ExpiresAt, &route.MaxRedemptions,
		&route.Redemptions, &revoked, &route.CreatedBy, &route.CreatedAt); err != nil {
		return AccessRoute{}, err
	}
	if revoked.Valid {
		value := revoked.Time.UTC()
		route.RevokedAt = &value
	}
	return route, nil
}

const otpChallengeSelect = `
	SELECT c.id::text,c.tenant_id::text,c.legal_entity_id::text,c.distribution_id::text,c.route_id::text,
	       COALESCE(c.recipient_id::text,''),c.code_hash,c.expires_at,c.attempts,c.max_attempts,c.resends,c.max_resends,c.consumed_at,c.created_at
	FROM capture_otp_challenges c`

func scanOTPChallenge(row scanner) (OTPChallenge, error) {
	var challenge OTPChallenge
	var consumed sql.NullTime
	if err := row.Scan(&challenge.ID, &challenge.TenantID, &challenge.LegalEntityID, &challenge.DistributionID,
		&challenge.RouteID, &challenge.RecipientID, &challenge.Digest, &challenge.ExpiresAt,
		&challenge.Attempts, &challenge.MaxAttempts, &challenge.Resends, &challenge.MaxResends,
		&consumed, &challenge.CreatedAt); err != nil {
		return OTPChallenge{}, err
	}
	if consumed.Valid {
		value := consumed.Time.UTC()
		challenge.ConsumedAt = &value
	}
	return challenge, nil
}

func validPersistedPostgresAccessRoute(route AccessRoute) bool {
	return route.ID != "" && len(route.SelectorHash) == 32 && route.Redemptions == 0 && accessRouteOpen(route, route.CreatedAt)
}

var (
	_ DistributionAccessStore = (*PostgresDistributionStore)(nil)
	_ = bytes.Equal
	_ = fmt.Sprintf
)
