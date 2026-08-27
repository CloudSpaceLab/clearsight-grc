//go:build postgres

package evidence

import "database/sql"

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
