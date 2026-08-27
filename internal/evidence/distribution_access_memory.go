package evidence

import (
	"bytes"
	"context"
	"encoding/hex"
	"sync"
	"time"
)

type MemoryDistributionAccessStore struct {
	mu            sync.RWMutex
	distributions *MemoryDistributionStore
	routes        map[string]AccessRoute
	routeHashes   map[string]string
	challenges    map[string]OTPChallenge
	sessions      map[string]DistributionAccessSession
}

func NewMemoryDistributionAccessStore(distributions *MemoryDistributionStore) *MemoryDistributionAccessStore {
	return &MemoryDistributionAccessStore{
		distributions: distributions,
		routes:        map[string]AccessRoute{},
		routeHashes:   map[string]string{},
		challenges:    map[string]OTPChallenge{},
		sessions:      map[string]DistributionAccessSession{},
	}
}

func (store *MemoryDistributionAccessStore) GetDistribution(ctx context.Context, tenantID, legalEntityID, distributionID string) (DistributionBundle, error) {
	if store == nil || store.distributions == nil {
		return DistributionBundle{}, ErrNotFound
	}
	return store.distributions.GetDistribution(ctx, tenantID, legalEntityID, distributionID)
}

func (store *MemoryDistributionAccessStore) GetRequest(ctx context.Context, tenantID, requestID string) (Request, error) {
	if store == nil || store.distributions == nil || store.distributions.repo == nil {
		return Request{}, ErrNotFound
	}
	return store.distributions.repo.GetRequest(ctx, tenantID, requestID)
}

func (store *MemoryDistributionAccessStore) CreateAccessRoutes(_ context.Context, routes []AccessRoute) error {
	if store == nil || len(routes) == 0 {
		return ErrDistributionAccessUnavailable
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	for _, route := range routes {
		key := hex.EncodeToString(route.SelectorHash)
		if !validPersistedAccessRoute(route) || key == "" || store.routeHashes[key] != "" || store.activeRouteConflict(route) {
			return ErrDistributionAccessUnavailable
		}
	}
	for _, route := range routes {
		cloned := cloneAccessRoute(route)
		store.routes[route.ID] = cloned
		store.routeHashes[hex.EncodeToString(route.SelectorHash)] = route.ID
	}
	return nil
}

func (store *MemoryDistributionAccessStore) AccessRouteBySelectorHash(_ context.Context, selectorHash []byte) (AccessRoute, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	routeID := store.routeHashes[hex.EncodeToString(selectorHash)]
	route, ok := store.routes[routeID]
	if !ok {
		return AccessRoute{}, ErrDistributionAccessUnavailable
	}
	return cloneAccessRoute(route), nil
}

func (store *MemoryDistributionAccessStore) AccessRouteByID(_ context.Context, tenantID, legalEntityID, distributionID, routeID string) (AccessRoute, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	route, ok := store.routes[routeID]
	if !ok || route.TenantID != tenantID || route.LegalEntityID != legalEntityID || route.DistributionID != distributionID {
		return AccessRoute{}, ErrDistributionAccessUnavailable
	}
	return cloneAccessRoute(route), nil
}

func (store *MemoryDistributionAccessStore) ProtectedRecipientForAccess(_ context.Context, route AccessRoute, recipientID string) (DistributionRecipient, protectedRecipientAddress, error) {
	if store == nil || store.distributions == nil {
		return DistributionRecipient{}, protectedRecipientAddress{}, ErrAccessVerificationFailed
	}
	store.distributions.mu.RLock()
	defer store.distributions.mu.RUnlock()
	for _, recipient := range store.distributions.recipients[route.DistributionID] {
		if recipient.safe.ID == recipientID && recipient.safe.TenantID == route.TenantID && recipient.safe.LegalEntityID == route.LegalEntityID &&
			recipient.safe.DistributionID == route.DistributionID && len(eligibleAccessRecipients(route, []DistributionRecipient{recipient.safe})) == 1 {
			return recipient.safe, cloneProtectedRecipientAddress(recipient.protected), nil
		}
	}
	return DistributionRecipient{}, protectedRecipientAddress{}, ErrAccessVerificationFailed
}

func (store *MemoryDistributionAccessStore) ActiveOTPChallenge(_ context.Context, route AccessRoute, recipientID string, now time.Time) (otpChallengeSnapshot, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	var selected OTPChallenge
	for _, challenge := range store.challenges {
		if challenge.RouteID != route.ID || challenge.RecipientID != recipientID || challenge.ConsumedAt != nil ||
			challenge.Attempts >= challenge.MaxAttempts || !challenge.ExpiresAt.After(now) {
			continue
		}
		if selected.ID == "" || challenge.CreatedAt.After(selected.CreatedAt) {
			selected = challenge
		}
	}
	if selected.ID == "" {
		return otpChallengeSnapshot{}, nil
	}
	return otpChallengeSnapshot{Challenge: cloneOTPChallenge(selected), Found: true}, nil
}

func (store *MemoryDistributionAccessStore) OTPChallengeByID(_ context.Context, route AccessRoute, challengeID string, _ time.Time) (OTPChallenge, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	challenge, ok := store.challenges[challengeID]
	if !ok || challenge.RouteID != route.ID || challenge.TenantID != route.TenantID || challenge.LegalEntityID != route.LegalEntityID || challenge.DistributionID != route.DistributionID {
		return OTPChallenge{}, ErrAccessVerificationFailed
	}
	return cloneOTPChallenge(challenge), nil
}

func (store *MemoryDistributionAccessStore) CreateOTPChallenge(_ context.Context, challenge OTPChallenge) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	if _, exists := store.challenges[challenge.ID]; exists {
		return ErrAccessVerificationFailed
	}
	for _, existing := range store.challenges {
		if existing.RouteID == challenge.RouteID && existing.RecipientID == challenge.RecipientID && existing.ConsumedAt == nil &&
			existing.Attempts < existing.MaxAttempts && existing.ExpiresAt.After(challenge.CreatedAt) {
			return ErrAccessVerificationFailed
		}
	}
	store.challenges[challenge.ID] = cloneOTPChallenge(challenge)
	return nil
}

func (store *MemoryDistributionAccessStore) UpdateOTPChallenge(_ context.Context, challenge OTPChallenge, expectedAttempts, expectedResends int, expectedDigest []byte) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	current, ok := store.challenges[challenge.ID]
	if !ok || current.Resends != expectedResends || current.ConsumedAt != nil || !bytes.Equal(current.Digest, expectedDigest) ||
		current.RouteID != challenge.RouteID || current.RecipientID != challenge.RecipientID {
		return ErrAccessVerificationFailed
	}
	if otpFailedAttemptMutation(challenge, expectedAttempts, expectedResends, expectedDigest) {
		if current.Attempts >= current.MaxAttempts {
			return ErrAccessVerificationFailed
		}
		current.Attempts++
		store.challenges[challenge.ID] = cloneOTPChallenge(current)
		return nil
	}
	if current.Attempts != expectedAttempts {
		return ErrAccessVerificationFailed
	}
	store.challenges[challenge.ID] = cloneOTPChallenge(challenge)
	return nil
}

func (store *MemoryDistributionAccessStore) CommitAccessSession(_ context.Context, commit accessSessionCommit) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	current, ok := store.routes[commit.Route.ID]
	now := commit.Session.CreatedAt
	if !ok || current.RevokedAt != nil || current.Redemptions != commit.ExpectedRedemptions || !current.ExpiresAt.After(now) ||
		current.TenantID != commit.Session.TenantID || current.LegalEntityID != commit.Session.LegalEntityID || current.DistributionID != commit.Session.DistributionID ||
		commit.Recipient.ID != commit.Session.RecipientID || commit.Recipient.RequestID != commit.Session.RequestID ||
		len(eligibleAccessRecipients(current, []DistributionRecipient{commit.Recipient})) != 1 || !commit.Session.ExpiresAt.After(now) || commit.Session.ExpiresAt.After(current.ExpiresAt) ||
		!accessGrantAssuranceMatches(current.Policy, commit.Session.Assurance) || current.Redemptions >= current.MaxRedemptions {
		return ErrAccessVerificationFailed
	}
	if commit.Challenge != nil {
		challenge, exists := store.challenges[commit.Challenge.ID]
		if !exists || challenge.ConsumedAt != nil || challenge.Attempts != commit.ExpectedAttempts || challenge.Resends != commit.ExpectedResends ||
			!bytes.Equal(challenge.Digest, commit.ExpectedDigest) || challenge.RouteID != current.ID || challenge.RecipientID != commit.Recipient.ID || commit.Challenge.ConsumedAt == nil {
			return ErrAccessVerificationFailed
		}
		store.challenges[challenge.ID] = cloneOTPChallenge(*commit.Challenge)
	} else if current.Policy != AccessDirectMagicLink {
		return ErrAccessVerificationFailed
	}
	key := hex.EncodeToString(commit.Session.TokenHash)
	if key == "" || store.sessions[key].ID != "" {
		return ErrAccessVerificationFailed
	}
	current.Redemptions++
	store.routes[current.ID] = current
	store.sessions[key] = cloneDistributionAccessSession(commit.Session)
	return nil
}

func (store *MemoryDistributionAccessStore) RotateAccessRoute(_ context.Context, current, next AccessRoute, now time.Time) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	persisted, ok := store.routes[current.ID]
	nextHash := hex.EncodeToString(next.SelectorHash)
	if !ok || persisted.RevokedAt != nil || persisted.TenantID != next.TenantID || persisted.LegalEntityID != next.LegalEntityID || persisted.DistributionID != next.DistributionID ||
		nextHash == "" || store.routeHashes[nextHash] != "" || store.activeRouteConflictIgnoring(next, persisted.ID) {
		return ErrDistributionAccessUnavailable
	}
	revokedAt := now.UTC()
	persisted.RevokedAt = &revokedAt
	store.routes[persisted.ID] = persisted
	for key, session := range store.sessions {
		if session.RouteID == persisted.ID && session.RevokedAt == nil {
			session.RevokedAt = &revokedAt
			store.sessions[key] = session
		}
	}
	store.routes[next.ID] = cloneAccessRoute(next)
	store.routeHashes[nextHash] = next.ID
	return nil
}

func (store *MemoryDistributionAccessStore) RevokeAccessRoute(_ context.Context, route AccessRoute, now time.Time) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	persisted, ok := store.routes[route.ID]
	if !ok || persisted.TenantID != route.TenantID || persisted.LegalEntityID != route.LegalEntityID || persisted.DistributionID != route.DistributionID {
		return ErrDistributionAccessUnavailable
	}
	revokedAt := now.UTC()
	if persisted.RevokedAt == nil {
		persisted.RevokedAt = &revokedAt
		store.routes[persisted.ID] = persisted
	}
	for key, session := range store.sessions {
		if session.RouteID == persisted.ID && session.RevokedAt == nil {
			session.RevokedAt = &revokedAt
			store.sessions[key] = session
		}
	}
	return nil
}

func (store *MemoryDistributionAccessStore) DistributionSessionByTokenHash(_ context.Context, tokenHash []byte, now time.Time) (DistributionAccessSession, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	session, ok := store.sessions[hex.EncodeToString(tokenHash)]
	if !ok || session.RevokedAt != nil || !session.ExpiresAt.After(now) {
		return DistributionAccessSession{}, ErrSessionInvalid
	}
	route, ok := store.routes[session.RouteID]
	if !ok || route.RevokedAt != nil || !route.ExpiresAt.After(now) {
		return DistributionAccessSession{}, ErrSessionInvalid
	}
	return cloneDistributionAccessSession(session), nil
}

func (store *MemoryDistributionAccessStore) activeRouteConflict(candidate AccessRoute) bool {
	return store.activeRouteConflictIgnoring(candidate, "")
}

func (store *MemoryDistributionAccessStore) activeRouteConflictIgnoring(candidate AccessRoute, ignoredID string) bool {
	for _, existing := range store.routes {
		if existing.ID == ignoredID || existing.RevokedAt != nil || existing.TenantID != candidate.TenantID || existing.DistributionID != candidate.DistributionID {
			continue
		}
		if candidate.Policy == AccessSharedEmailOTP && existing.Policy == AccessSharedEmailOTP {
			return true
		}
		if candidate.Policy != AccessSharedEmailOTP && existing.RecipientID == candidate.RecipientID {
			return true
		}
	}
	return false
}

func validPersistedAccessRoute(route AccessRoute) bool {
	return route.ID != "" && len(route.SelectorHash) == 32 && accessRouteOpen(route, route.CreatedAt) && route.Redemptions == 0
}

func cloneAccessRoute(route AccessRoute) AccessRoute {
	route.SelectorHash = append([]byte(nil), route.SelectorHash...)
	if route.RevokedAt != nil {
		value := *route.RevokedAt
		route.RevokedAt = &value
	}
	return route
}

func cloneOTPChallenge(challenge OTPChallenge) OTPChallenge {
	challenge.Digest = append([]byte(nil), challenge.Digest...)
	if challenge.ConsumedAt != nil {
		value := *challenge.ConsumedAt
		challenge.ConsumedAt = &value
	}
	return challenge
}

func cloneProtectedRecipientAddress(value protectedRecipientAddress) protectedRecipientAddress {
	value.Hash = append([]byte(nil), value.Hash...)
	value.Ciphertext = append([]byte(nil), value.Ciphertext...)
	return value
}

func cloneDistributionAccessSession(session DistributionAccessSession) DistributionAccessSession {
	session.TokenHash = append([]byte(nil), session.TokenHash...)
	if session.RevokedAt != nil {
		value := *session.RevokedAt
		session.RevokedAt = &value
	}
	return session
}

var _ DistributionAccessStore = (*MemoryDistributionAccessStore)(nil)
