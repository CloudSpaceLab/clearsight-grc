package evidence

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

// DistributionAccessService owns the new distribution access ceremony while
// the existing Service continues to own legacy one-recipient invitations.
type DistributionAccessService struct {
	store      DistributionAccessStore
	revealer   recipientAddressRevealer
	delivery   OTPDelivery
	engine     *AccessPolicyEngine
	otp        *OTPService
	now        func() time.Time
	sessionTTL time.Duration
}

func NewDistributionAccessService(store DistributionAccessStore, revealer recipientAddressRevealer, delivery OTPDelivery, hmacKey [32]byte, sessionTTL time.Duration) (*DistributionAccessService, error) {
	if store == nil || revealer == nil || !securityKeyConfigured(hmacKey) || sessionTTL <= 0 {
		return nil, ErrDistributionAccessUnavailable
	}
	service := &DistributionAccessService{
		store: store, revealer: revealer, delivery: delivery,
		engine: NewAccessPolicyEngine(hmacKey), otp: NewOTPService(hmacKey),
		now: time.Now, sessionTTL: sessionTTL,
	}
	service.engine.now = func() time.Time { return service.currentTime() }
	service.otp.now = func() time.Time { return service.currentTime() }
	return service, nil
}

func (service *DistributionAccessService) IssueDistributionAccessRoutes(ctx context.Context, tenantID, legalEntityID, distributionID, createdBy string) ([]IssuedAccessRoute, error) {
	if service == nil || strings.TrimSpace(createdBy) == "" {
		return nil, ErrDistributionAccessUnavailable
	}
	bundle, err := service.store.GetDistribution(ctx, tenantID, legalEntityID, distributionID)
	if err != nil || !distributionMayIssueAccess(bundle.Distribution, service.currentTime()) {
		return nil, ErrDistributionAccessUnavailable
	}
	external := externalTORecipients(bundle.Recipients)
	if len(external) == 0 {
		return []IssuedAccessRoute{}, nil
	}

	routes := make([]AccessRoute, 0, len(external))
	issued := make([]IssuedAccessRoute, 0, len(external))
	appendRoute := func(recipientID, hint string) error {
		route, secret, routeErr := service.engine.IssueRoute(AccessRouteInput{
			TenantID: bundle.Distribution.TenantID, LegalEntityID: bundle.Distribution.LegalEntityID,
			DistributionID: bundle.Distribution.ID, RecipientID: recipientID,
			Policy: bundle.Distribution.AccessPolicy, AudienceHint: hint,
			RouteExpiresAt: bundle.Distribution.RouteExpiresAt, Deadline: bundle.Distribution.Deadline,
			CreatedBy: createdBy,
		})
		if routeErr != nil {
			return routeErr
		}
		routes = append(routes, route)
		issued = append(issued, secret)
		return nil
	}

	if bundle.Distribution.AccessPolicy == AccessSharedEmailOTP {
		if err := appendRoute("", ""); err != nil {
			return nil, ErrDistributionAccessUnavailable
		}
	} else {
		for _, recipient := range external {
			if err := appendRoute(recipient.ID, recipient.AudienceHint); err != nil {
				return nil, ErrDistributionAccessUnavailable
			}
		}
	}
	if err := service.store.CreateAccessRoutes(ctx, routes); err != nil {
		return nil, ErrDistributionAccessUnavailable
	}
	return issued, nil
}

func (service *DistributionAccessService) StartDistributionAccess(ctx context.Context, routeSelector string) (AccessStart, error) {
	route, bundle, selector, err := service.resolvePublicRoute(ctx, routeSelector)
	if err != nil {
		return AccessStart{}, ErrDistributionAccessUnavailable
	}
	start, err := service.engine.Start(route, selector, bundle.Recipients, service.currentTime())
	if err != nil {
		return AccessStart{}, ErrDistributionAccessUnavailable
	}
	return start, nil
}

func (service *DistributionAccessService) SendOTP(ctx context.Context, routeSelector, recipientSelector string) (OTPSendReceipt, error) {
	if service == nil || service.delivery == nil {
		return OTPSendReceipt{}, ErrAccessVerificationFailed
	}
	route, bundle, selector, err := service.resolvePublicRoute(ctx, routeSelector)
	if err != nil {
		return OTPSendReceipt{}, ErrAccessVerificationFailed
	}
	recipient, err := service.engine.ResolveOTPRecipient(route, selector, recipientSelector, bundle.Recipients, service.currentTime())
	if err != nil {
		return OTPSendReceipt{}, ErrAccessVerificationFailed
	}
	safe, protected, err := service.store.ProtectedRecipientForAccess(ctx, route, recipient.ID)
	if err != nil || safe.ID != recipient.ID {
		return OTPSendReceipt{}, ErrAccessVerificationFailed
	}
	address, err := service.revealer.RevealRecipientAddress(ctx, route.TenantID, route.DistributionID, recipient.ID, protected)
	if err != nil {
		return OTPSendReceipt{}, ErrAccessVerificationFailed
	}

	now := service.currentTime()
	snapshot, err := service.store.ActiveOTPChallenge(ctx, route, recipient.ID, now)
	if err != nil {
		return OTPSendReceipt{}, ErrAccessVerificationFailed
	}
	var issued IssuedOTP
	if snapshot.Found {
		challenge := snapshot.Challenge
		expectedAttempts, expectedResends := challenge.Attempts, challenge.Resends
		expectedDigest := append([]byte(nil), challenge.Digest...)
		issued, err = service.otp.Resend(route, &challenge, now)
		if err == nil {
			err = service.store.UpdateOTPChallenge(ctx, challenge, expectedAttempts, expectedResends, expectedDigest)
		}
	} else {
		challenge, generated, issueErr := service.otp.Issue(route, recipient, now)
		if issueErr != nil {
			return OTPSendReceipt{}, ErrAccessVerificationFailed
		}
		issued = generated
		err = service.store.CreateOTPChallenge(ctx, challenge)
	}
	if err != nil {
		return OTPSendReceipt{}, ErrAccessVerificationFailed
	}
	if err := service.delivery.DeliverDistributionOTP(ctx, DistributionOTPDelivery{
		Address: address, Code: issued.Code, ChallengeID: issued.ChallengeID,
		DistributionID: route.DistributionID, ExpiresAt: issued.ExpiresAt,
	}); err != nil {
		return OTPSendReceipt{}, ErrAccessVerificationFailed
	}
	return OTPSendReceipt{ChallengeID: issued.ChallengeID, Hint: recipient.AudienceHint, ExpiresAt: issued.ExpiresAt}, nil
}

func (service *DistributionAccessService) VerifyOTP(ctx context.Context, routeSelector, challengeID, code string) (RedeemedDistributionSession, error) {
	route, bundle, selector, err := service.resolvePublicRoute(ctx, routeSelector)
	if err != nil || strings.TrimSpace(challengeID) == "" {
		service.consumeInvalidOTP(code)
		return RedeemedDistributionSession{}, ErrAccessVerificationFailed
	}
	if _, err := service.engine.Start(route, selector, bundle.Recipients, service.currentTime()); err != nil {
		service.consumeInvalidOTP(code)
		return RedeemedDistributionSession{}, ErrAccessVerificationFailed
	}
	challenge, err := service.store.OTPChallengeByID(ctx, route, strings.TrimSpace(challengeID), service.currentTime())
	if err != nil {
		service.consumeInvalidOTP(code)
		return RedeemedDistributionSession{}, ErrAccessVerificationFailed
	}
	recipient, _, err := service.store.ProtectedRecipientForAccess(ctx, route, challenge.RecipientID)
	if err != nil {
		service.consumeInvalidOTP(code)
		return RedeemedDistributionSession{}, ErrAccessVerificationFailed
	}

	expectedAttempts, expectedResends := challenge.Attempts, challenge.Resends
	expectedDigest := append([]byte(nil), challenge.Digest...)
	verification, verifyErr := service.otp.Verify(route, &challenge, code, service.currentTime())
	if verifyErr != nil {
		if challenge.Attempts != expectedAttempts {
			_ = service.store.UpdateOTPChallenge(ctx, challenge, expectedAttempts, expectedResends, expectedDigest)
		}
		return RedeemedDistributionSession{}, ErrAccessVerificationFailed
	}

	grant, err := service.engine.RedeemVerifiedRoute(&route, verification, service.currentTime().Add(service.sessionTTL), service.currentTime())
	if err != nil {
		return RedeemedDistributionSession{}, ErrAccessVerificationFailed
	}
	return service.commitSession(ctx, route, recipient, grant, &challenge, expectedAttempts, expectedResends, expectedDigest)
}

func (service *DistributionAccessService) RedeemDirectRoute(ctx context.Context, routeSelector string) (RedeemedDistributionSession, error) {
	route, bundle, selector, err := service.resolvePublicRoute(ctx, routeSelector)
	if err != nil || route.Policy != AccessDirectMagicLink {
		return RedeemedDistributionSession{}, ErrDistributionAccessUnavailable
	}
	grant, err := service.engine.RedeemDirectRoute(&route, selector, bundle.Recipients, service.currentTime().Add(service.sessionTTL), service.currentTime())
	if err != nil {
		return RedeemedDistributionSession{}, ErrDistributionAccessUnavailable
	}
	var recipient DistributionRecipient
	for _, candidate := range bundle.Recipients {
		if candidate.ID == grant.RecipientID {
			recipient = candidate
			break
		}
	}
	if recipient.ID == "" {
		return RedeemedDistributionSession{}, ErrDistributionAccessUnavailable
	}
	result, err := service.commitSession(ctx, route, recipient, grant, nil, 0, 0, nil)
	if err != nil {
		return RedeemedDistributionSession{}, ErrDistributionAccessUnavailable
	}
	return result, nil
}

func (service *DistributionAccessService) RotateDistributionAccessRoute(ctx context.Context, tenantID, legalEntityID, distributionID, routeID, createdBy string) (IssuedAccessRoute, error) {
	if service == nil || strings.TrimSpace(createdBy) == "" {
		return IssuedAccessRoute{}, ErrDistributionAccessUnavailable
	}
	current, err := service.store.AccessRouteByID(ctx, tenantID, legalEntityID, distributionID, routeID)
	if err != nil {
		return IssuedAccessRoute{}, ErrDistributionAccessUnavailable
	}
	bundle, err := service.store.GetDistribution(ctx, tenantID, legalEntityID, distributionID)
	if err != nil || !distributionMayIssueAccess(bundle.Distribution, service.currentTime()) || current.Policy != bundle.Distribution.AccessPolicy {
		return IssuedAccessRoute{}, ErrDistributionAccessUnavailable
	}
	input := AccessRouteInput{
		TenantID: current.TenantID, LegalEntityID: current.LegalEntityID, DistributionID: current.DistributionID,
		RecipientID: current.RecipientID, Policy: current.Policy, AudienceHint: current.AudienceHint,
		RouteExpiresAt: bundle.Distribution.RouteExpiresAt, Deadline: bundle.Distribution.Deadline,
		CreatedBy: createdBy,
	}
	next, issued, err := service.engine.RotateRoute(&current, input, service.currentTime())
	if err != nil {
		return IssuedAccessRoute{}, ErrDistributionAccessUnavailable
	}
	if err := service.store.RotateAccessRoute(ctx, current, next, service.currentTime()); err != nil {
		return IssuedAccessRoute{}, ErrDistributionAccessUnavailable
	}
	return issued, nil
}

func (service *DistributionAccessService) RevokeDistributionAccessRoute(ctx context.Context, tenantID, legalEntityID, distributionID, routeID string) error {
	if service == nil {
		return ErrDistributionAccessUnavailable
	}
	route, err := service.store.AccessRouteByID(ctx, tenantID, legalEntityID, distributionID, routeID)
	if err != nil {
		return ErrDistributionAccessUnavailable
	}
	if err := RevokeAccessRoute(&route, service.currentTime()); err != nil {
		return ErrDistributionAccessUnavailable
	}
	if err := service.store.RevokeAccessRoute(ctx, route, service.currentTime()); err != nil {
		return ErrDistributionAccessUnavailable
	}
	return nil
}

func (service *DistributionAccessService) SessionRequest(ctx context.Context, sessionToken string) (DistributionAccessSession, Request, error) {
	if service == nil || strings.TrimSpace(sessionToken) == "" {
		return DistributionAccessSession{}, Request{}, ErrSessionInvalid
	}
	now := service.currentTime()
	session, err := service.store.DistributionSessionByTokenHash(ctx, hashToken(sessionToken), now)
	if err != nil {
		return DistributionAccessSession{}, Request{}, ErrSessionInvalid
	}
	bundle, err := service.store.GetDistribution(ctx, session.TenantID, session.LegalEntityID, session.DistributionID)
	if err != nil || !distributionOpenForAccess(bundle.Distribution, now) {
		return DistributionAccessSession{}, Request{}, ErrSessionInvalid
	}
	route, err := service.store.AccessRouteByID(ctx, session.TenantID, session.LegalEntityID, session.DistributionID, session.RouteID)
	if err != nil || !AccessGrantUsable(route, AccessGrant{
		RouteID: route.ID, TenantID: session.TenantID, DistributionID: session.DistributionID,
		RecipientID: session.RecipientID, Assurance: session.Assurance, ExpiresAt: session.ExpiresAt,
	}, now) {
		return DistributionAccessSession{}, Request{}, ErrSessionInvalid
	}
	eligible := eligibleAccessRecipients(route, bundle.Recipients)
	bound := false
	for _, recipient := range eligible {
		if recipient.ID == session.RecipientID && recipient.RequestID == session.RequestID && recipient.AudienceHint == session.AudienceHint {
			bound = true
			break
		}
	}
	if !bound {
		return DistributionAccessSession{}, Request{}, ErrSessionInvalid
	}
	request, err := service.store.GetRequest(ctx, session.TenantID, session.RequestID)
	if err != nil || !requestOpenAt(request, now) || !externalRecipientRequest(request) || request.Recipient.AudienceHint != session.AudienceHint {
		return DistributionAccessSession{}, Request{}, ErrSessionInvalid
	}
	return session, RespondentRequest(request), nil
}

func (service *DistributionAccessService) resolvePublicRoute(ctx context.Context, routeSelector string) (AccessRoute, DistributionBundle, string, error) {
	if service == nil {
		return AccessRoute{}, DistributionBundle{}, "", ErrDistributionAccessUnavailable
	}
	selector := strings.TrimSpace(routeSelector)
	if selector == "" {
		return AccessRoute{}, DistributionBundle{}, "", ErrDistributionAccessUnavailable
	}
	digest := sha256.Sum256([]byte(selector))
	route, err := service.store.AccessRouteBySelectorHash(ctx, digest[:])
	if err != nil {
		return AccessRoute{}, DistributionBundle{}, "", ErrDistributionAccessUnavailable
	}
	bundle, err := service.store.GetDistribution(ctx, route.TenantID, route.LegalEntityID, route.DistributionID)
	if err != nil || !distributionOpenForAccess(bundle.Distribution, service.currentTime()) || bundle.Distribution.AccessPolicy != route.Policy {
		return AccessRoute{}, DistributionBundle{}, "", ErrDistributionAccessUnavailable
	}
	return route, bundle, selector, nil
}

func (service *DistributionAccessService) commitSession(ctx context.Context, route AccessRoute, recipient DistributionRecipient, grant AccessGrant, challenge *OTPChallenge, expectedAttempts, expectedResends int, expectedDigest []byte) (RedeemedDistributionSession, error) {
	token, tokenHash, err := tokenPair()
	if err != nil {
		return RedeemedDistributionSession{}, ErrAccessVerificationFailed
	}
	sessionID, err := id.NewUUIDv7()
	if err != nil {
		return RedeemedDistributionSession{}, ErrAccessVerificationFailed
	}
	session := DistributionAccessSession{
		ID: sessionID, TenantID: route.TenantID, LegalEntityID: route.LegalEntityID,
		DistributionID: route.DistributionID, RecipientID: recipient.ID, RequestID: recipient.RequestID,
		RouteID: route.ID, AudienceHint: recipient.AudienceHint, Assurance: grant.Assurance,
		TokenHash: tokenHash, ExpiresAt: grant.ExpiresAt, CreatedAt: service.currentTime(),
	}
	if err := service.store.CommitAccessSession(ctx, accessSessionCommit{
		Route: route, Recipient: recipient, Session: session, Challenge: challenge,
		ExpectedAttempts: expectedAttempts, ExpectedResends: expectedResends,
		ExpectedDigest: append([]byte(nil), expectedDigest...),
	}); err != nil {
		return RedeemedDistributionSession{}, ErrAccessVerificationFailed
	}
	return RedeemedDistributionSession{
		SessionID: session.ID, SessionToken: token, DistributionID: session.DistributionID,
		RequestID: session.RequestID, AudienceHint: session.AudienceHint,
		Assurance: session.Assurance, ExpiresAt: session.ExpiresAt,
	}, nil
}

func (service *DistributionAccessService) consumeInvalidOTP(code string) {
	if service != nil && service.otp != nil {
		service.otp.consumeDummyOTP(nil, code)
	}
}

func (service *DistributionAccessService) currentTime() time.Time {
	if service != nil && service.now != nil {
		return service.now().UTC()
	}
	return time.Now().UTC()
}

func distributionMayIssueAccess(distribution FormDistribution, now time.Time) bool {
	if !validAccessPolicy(distribution.AccessPolicy) || distribution.ID == "" || distribution.TenantID == "" || distribution.LegalEntityID == "" ||
		distribution.RouteExpiresAt.IsZero() || distribution.Deadline.IsZero() || !distribution.RouteExpiresAt.After(now) || !distribution.Deadline.After(now) {
		return false
	}
	switch distribution.Status {
	case DistributionDraft, DistributionReady, DistributionOpen:
		return true
	default:
		return false
	}
}

func distributionOpenForAccess(distribution FormDistribution, now time.Time) bool {
	return distribution.Status == DistributionOpen && distribution.Deadline.After(now) && distribution.RouteExpiresAt.After(now)
}

func externalTORecipients(recipients []DistributionRecipient) []DistributionRecipient {
	values := make([]DistributionRecipient, 0, len(recipients))
	for _, recipient := range recipients {
		if recipient.Role == RecipientTo && recipient.Type == RecipientExternalAudience && recipient.RequestID != "" &&
			recipient.State != DistributionRecipientRevoked && recipient.State != DistributionRecipientCompleted {
			values = append(values, recipient)
		}
	}
	return values
}

func isAccessPublicError(err error) bool {
	return errors.Is(err, ErrDistributionAccessUnavailable) || errors.Is(err, ErrAccessVerificationFailed) || errors.Is(err, ErrSessionInvalid)
}

var _ = fmt.Sprintf
