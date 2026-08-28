//go:build postgres

package evidence

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

func (store *PostgresDistributionStore) ActiveOTPChallenge(ctx context.Context, route AccessRoute, recipientID string, now time.Time) (otpChallengeSnapshot, error) {
	challenge, err := scanOTPChallenge(store.repo.pool.QueryRow(ctx, otpChallengeSelect+`
		WHERE c.route_id=$1::uuid AND c.tenant_id=$2::uuid AND c.legal_entity_id=$3::uuid AND c.distribution_id=$4::uuid
		  AND c.recipient_id=$5::uuid AND c.consumed_at IS NULL AND c.expires_at>$6
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
			WHERE route_id=$1::uuid AND recipient_id=$2::uuid AND consumed_at IS NULL AND expires_at>$3
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
	if otpFailedAttemptMutation(challenge, expectedAttempts, expectedResends, expectedDigest) {
		tag, err := store.repo.pool.Exec(ctx, `
			UPDATE capture_otp_challenges
			SET attempts=attempts+1
			WHERE id=$1::uuid AND route_id=$2::uuid AND recipient_id=$3::uuid
			  AND resends=$4 AND code_hash=$5 AND consumed_at IS NULL AND attempts<max_attempts`,
			challenge.ID, challenge.RouteID, challenge.RecipientID, expectedResends, expectedDigest)
		if err != nil || tag.RowsAffected() != 1 {
			return ErrAccessVerificationFailed
		}
		return nil
	}

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
