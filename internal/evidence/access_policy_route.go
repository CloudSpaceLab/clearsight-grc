package evidence

import (
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

func (engine *AccessPolicyEngine) IssueRoute(input AccessRouteInput) (AccessRoute, IssuedAccessRoute, error) {
	if engine == nil || !engine.configured || !validAccessRouteInput(input) {
		return AccessRoute{}, IssuedAccessRoute{}, ErrDistributionAccessUnavailable
	}
	now := engine.currentTime()
	expiresAt := clampAccessExpiry(input.RouteExpiresAt, input.Deadline)
	if !expiresAt.After(now) {
		return AccessRoute{}, IssuedAccessRoute{}, ErrDistributionAccessUnavailable
	}
	routeID, err := id.NewUUIDv7()
	if err != nil {
		return AccessRoute{}, IssuedAccessRoute{}, ErrDistributionAccessUnavailable
	}
	selector, selectorHash, err := randomAccessSelector(engine.random)
	if err != nil {
		return AccessRoute{}, IssuedAccessRoute{}, ErrDistributionAccessUnavailable
	}
	maxRedemptions := input.MaxRedemptions
	if maxRedemptions == 0 {
		maxRedemptions = 1
		if input.Policy == AccessSharedEmailOTP {
			maxRedemptions = sharedRouteRedemptions
		}
	}
	if maxRedemptions < 1 || maxRedemptions > sharedRouteRedemptions ||
		(input.Policy != AccessSharedEmailOTP && maxRedemptions != 1) {
		return AccessRoute{}, IssuedAccessRoute{}, ErrDistributionAccessUnavailable
	}
	route := AccessRoute{
		ID: routeID, TenantID: strings.TrimSpace(input.TenantID), LegalEntityID: strings.TrimSpace(input.LegalEntityID),
		DistributionID: strings.TrimSpace(input.DistributionID), RecipientID: strings.TrimSpace(input.RecipientID),
		Policy: input.Policy, SelectorHash: selectorHash, AudienceHint: strings.TrimSpace(input.AudienceHint),
		ExpiresAt: expiresAt, MaxRedemptions: maxRedemptions, CreatedBy: strings.TrimSpace(input.CreatedBy), CreatedAt: now,
	}
	issued := IssuedAccessRoute{RouteID: route.ID, Selector: selector, Policy: route.Policy, ExpiresAt: route.ExpiresAt}
	return route, issued, nil
}

func (engine *AccessPolicyEngine) Start(route AccessRoute, selector string, recipients []DistributionRecipient, now time.Time) (AccessStart, error) {
	if engine == nil || !engine.configured || !accessRouteUsable(route, selector, normalizeAccessNow(engine, now)) {
		return AccessStart{}, ErrDistributionAccessUnavailable
	}
	start := AccessStart{Policy: route.Policy, ExpiresAt: route.ExpiresAt}
	switch route.Policy {
	case AccessDirectMagicLink:
		return start, nil
	case AccessDirectEmailOTP, AccessSharedEmailOTP:
		eligible := eligibleAccessRecipients(route, recipients)
		if len(eligible) == 0 {
			return AccessStart{}, ErrDistributionAccessUnavailable
		}
		start.Recipients = make([]MaskedRecipient, 0, len(eligible))
		for _, recipient := range eligible {
			start.Recipients = append(start.Recipients, MaskedRecipient{
				SelectorID: engine.recipientSelector(route.ID, recipient.ID),
				Hint:       recipient.AudienceHint, ContactLabel: recipient.ContactLabel,
			})
		}
		return start, nil
	default:
		return AccessStart{}, ErrDistributionAccessUnavailable
	}
}

func (engine *AccessPolicyEngine) ResolveOTPRecipient(route AccessRoute, routeSelector, recipientSelector string, recipients []DistributionRecipient, now time.Time) (DistributionRecipient, error) {
	if engine == nil || !engine.configured || (route.Policy != AccessDirectEmailOTP && route.Policy != AccessSharedEmailOTP) ||
		!accessRouteUsable(route, routeSelector, normalizeAccessNow(engine, now)) || strings.TrimSpace(recipientSelector) == "" {
		engine.consumeDummySelector(route.ID, recipientSelector)
		return DistributionRecipient{}, ErrAccessVerificationFailed
	}
	for _, recipient := range eligibleAccessRecipients(route, recipients) {
		expected := engine.recipientSelector(route.ID, recipient.ID)
		if constantTimeStringEqual(expected, recipientSelector) {
			return recipient, nil
		}
	}
	engine.consumeDummySelector(route.ID, recipientSelector)
	return DistributionRecipient{}, ErrAccessVerificationFailed
}

func (engine *AccessPolicyEngine) RedeemDirectRoute(route *AccessRoute, selector string, recipients []DistributionRecipient, requestedSessionExpiry, now time.Time) (AccessGrant, error) {
	if engine == nil || !engine.configured || route == nil || route.Policy != AccessDirectMagicLink {
		return AccessGrant{}, ErrDistributionAccessUnavailable
	}
	now = normalizeAccessNow(engine, now)
	if !accessRouteUsable(*route, selector, now) {
		return AccessGrant{}, ErrDistributionAccessUnavailable
	}
	eligible := eligibleAccessRecipients(*route, recipients)
	if len(eligible) != 1 {
		return AccessGrant{}, ErrDistributionAccessUnavailable
	}
	return redeemAccessRoute(route, eligible[0].ID, AssuranceLinkPossession, requestedSessionExpiry, now)
}

func (engine *AccessPolicyEngine) RedeemVerifiedRoute(route *AccessRoute, verification OTPVerification, requestedSessionExpiry, now time.Time) (AccessGrant, error) {
	if engine == nil || !engine.configured || route == nil || (route.Policy != AccessDirectEmailOTP && route.Policy != AccessSharedEmailOTP) ||
		strings.TrimSpace(verification.challengeID) == "" || strings.TrimSpace(verification.recipientID) == "" {
		return AccessGrant{}, ErrAccessVerificationFailed
	}
	now = normalizeAccessNow(engine, now)
	if !accessRouteActive(*route, now) || verification.routeID != route.ID || verification.distributionID != route.DistributionID ||
		(route.Policy == AccessDirectEmailOTP && route.RecipientID != verification.recipientID) {
		return AccessGrant{}, ErrAccessVerificationFailed
	}
	grant, err := redeemAccessRoute(route, verification.recipientID, AssuranceEmailVerified, requestedSessionExpiry, now)
	if err != nil {
		return AccessGrant{}, ErrAccessVerificationFailed
	}
	return grant, nil
}

func (engine *AccessPolicyEngine) RotateRoute(current *AccessRoute, input AccessRouteInput, now time.Time) (AccessRoute, IssuedAccessRoute, error) {
	if engine == nil || !engine.configured || current == nil || current.RevokedAt != nil {
		return AccessRoute{}, IssuedAccessRoute{}, ErrDistributionAccessUnavailable
	}
	if input.TenantID != current.TenantID || input.LegalEntityID != current.LegalEntityID || input.DistributionID != current.DistributionID {
		return AccessRoute{}, IssuedAccessRoute{}, ErrDistributionAccessUnavailable
	}
	revokedAt := normalizeAccessNow(engine, now)
	if revokedAt.Before(current.CreatedAt) {
		return AccessRoute{}, IssuedAccessRoute{}, ErrDistributionAccessUnavailable
	}
	next, issued, err := engine.IssueRoute(input)
	if err != nil {
		return AccessRoute{}, IssuedAccessRoute{}, err
	}
	current.RevokedAt = &revokedAt
	return next, issued, nil
}

func RevokeAccessRoute(route *AccessRoute, now time.Time) error {
	if route == nil || now.IsZero() || now.Before(route.CreatedAt) {
		return ErrDistributionAccessUnavailable
	}
	if route.RevokedAt == nil {
		revokedAt := now.UTC()
		route.RevokedAt = &revokedAt
	}
	return nil
}

func AccessGrantUsable(route AccessRoute, grant AccessGrant, now time.Time) bool {
	return !now.IsZero() && accessRouteOpen(route, now.UTC()) && grant.RouteID == route.ID &&
		grant.TenantID == route.TenantID && grant.DistributionID == route.DistributionID &&
		strings.TrimSpace(grant.RecipientID) != "" && grant.ExpiresAt.After(now.UTC()) &&
		accessGrantAssuranceMatches(route.Policy, grant.Assurance)
}

func accessGrantAssuranceMatches(policy AccessPolicy, assurance AccessAssurance) bool {
	switch policy {
	case AccessDirectMagicLink:
		return assurance == AssuranceLinkPossession
	case AccessDirectEmailOTP, AccessSharedEmailOTP:
		return assurance == AssuranceEmailVerified
	default:
		return false
	}
}

func redeemAccessRoute(route *AccessRoute, recipientID string, assurance AccessAssurance, requestedSessionExpiry, now time.Time) (AccessGrant, error) {
	if route == nil || route.Redemptions >= route.MaxRedemptions || !accessRouteActive(*route, now) {
		return AccessGrant{}, ErrDistributionAccessUnavailable
	}
	expiresAt := requestedSessionExpiry.UTC()
	if requestedSessionExpiry.IsZero() || expiresAt.After(route.ExpiresAt) {
		expiresAt = route.ExpiresAt
	}
	if !expiresAt.After(now) {
		return AccessGrant{}, ErrDistributionAccessUnavailable
	}
	route.Redemptions++
	return AccessGrant{
		RouteID: route.ID, TenantID: route.TenantID, DistributionID: route.DistributionID,
		RecipientID: recipientID, Assurance: assurance, ExpiresAt: expiresAt,
	}, nil
}

func validAccessRouteInput(input AccessRouteInput) bool {
	if strings.TrimSpace(input.TenantID) == "" || strings.TrimSpace(input.LegalEntityID) == "" || strings.TrimSpace(input.DistributionID) == "" || strings.TrimSpace(input.CreatedBy) == "" || input.RouteExpiresAt.IsZero() || input.Deadline.IsZero() {
		return false
	}
	switch input.Policy {
	case AccessSharedEmailOTP:
		return strings.TrimSpace(input.RecipientID) == ""
	case AccessDirectMagicLink, AccessDirectEmailOTP:
		return strings.TrimSpace(input.RecipientID) != ""
	default:
		return false
	}
}

func accessRouteActive(route AccessRoute, now time.Time) bool {
	return accessRouteOpen(route, now) && route.MaxRedemptions > 0 && route.Redemptions < route.MaxRedemptions
}

func accessRouteOpen(route AccessRoute, now time.Time) bool {
	return route.ID != "" && route.TenantID != "" && route.LegalEntityID != "" && route.DistributionID != "" &&
		validAccessRouteBinding(route) && route.RevokedAt == nil && !route.CreatedAt.IsZero() &&
		!now.Before(route.CreatedAt) && route.ExpiresAt.After(route.CreatedAt) && route.ExpiresAt.After(now)
}

func validAccessRouteBinding(route AccessRoute) bool {
	switch route.Policy {
	case AccessSharedEmailOTP:
		return strings.TrimSpace(route.RecipientID) == "" && route.MaxRedemptions >= 1 && route.MaxRedemptions <= sharedRouteRedemptions
	case AccessDirectMagicLink, AccessDirectEmailOTP:
		return strings.TrimSpace(route.RecipientID) != "" && route.MaxRedemptions == 1
	default:
		return false
	}
}

func validAccessPolicy(policy AccessPolicy) bool {
	return policy == AccessDirectMagicLink || policy == AccessDirectEmailOTP || policy == AccessSharedEmailOTP
}

func clampAccessExpiry(routeExpiresAt, deadline time.Time) time.Time {
	routeExpiresAt = routeExpiresAt.UTC()
	deadline = deadline.UTC()
	if routeExpiresAt.IsZero() || (!deadline.IsZero() && deadline.Before(routeExpiresAt)) {
		return deadline
	}
	return routeExpiresAt
}

func normalizeAccessNow(engine *AccessPolicyEngine, now time.Time) time.Time {
	if now.IsZero() {
		return engine.currentTime()
	}
	return now.UTC()
}

func (engine *AccessPolicyEngine) currentTime() time.Time {
	if engine != nil && engine.now != nil {
		return engine.now().UTC()
	}
	return time.Now().UTC()
}
