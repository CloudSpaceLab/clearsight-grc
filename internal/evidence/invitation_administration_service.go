package evidence

import (
	"context"
	"crypto/sha256"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

func (s *Service) authorizeRequesterAdministration(ctx context.Context, tenant, legalEntityID, requestID, actor string) (Request, error) {
	tenant = strings.TrimSpace(tenant)
	requestID = strings.TrimSpace(requestID)
	actor = strings.TrimSpace(actor)
	if tenant == "" || requestID == "" || actor == "" {
		return Request{}, ErrRecipientManagerRequired
	}
	request, err := s.GetRequest(ctx, tenant, requestID)
	if err != nil {
		return Request{}, err
	}
	if strings.TrimSpace(request.CreatedBy) == "" || request.CreatedBy != actor {
		return Request{}, ErrRecipientManagerRequired
	}
	if err := validateCurrentRequestScope(ctx, s.repo, request, legalEntityID); err != nil {
		return Request{}, err
	}
	checker, ok := s.repo.(SubjectAccessChecker)
	if !ok {
		return Request{}, ErrSubjectAccessDenied
	}
	allowed, err := checker.CanReadSubject(ctx, tenant, actor, request.SubjectType, request.SubjectID)
	if err != nil {
		return Request{}, err
	}
	if !allowed {
		return Request{}, ErrSubjectAccessDenied
	}
	return request, nil
}

func (s *Service) ListInvitationMetadata(ctx context.Context, input ManageInvitationsInput) ([]InvitationMetadata, error) {
	if _, err := s.authorizeRequesterAdministration(ctx, input.TenantID, input.LegalEntityID, input.RequestID, input.ActorPrincipalID); err != nil {
		return nil, err
	}
	store, err := invitationAdministrationPersistence(s.repo)
	if err != nil {
		return nil, err
	}
	return store.ListInvitationMetadata(ctx, input.TenantID, input.RequestID, 100)
}

func (s *Service) RevokeInvitationAsRequester(ctx context.Context, input RevokeInvitationAsRequesterInput) error {
	if strings.TrimSpace(input.InvitationID) == "" {
		return ErrNotFound
	}
	if _, err := s.authorizeRequesterAdministration(ctx, input.TenantID, input.LegalEntityID, input.RequestID, input.ActorPrincipalID); err != nil {
		return err
	}
	store, err := invitationAdministrationPersistence(s.repo)
	if err != nil {
		return err
	}
	return store.RevokeInvitationForRequester(ctx, input, s.now().UTC())
}

func (s *Service) RevokeSessionAsRequester(ctx context.Context, input RevokeSessionAsRequesterInput) error {
	if strings.TrimSpace(input.SessionID) == "" {
		return ErrNotFound
	}
	if _, err := s.authorizeRequesterAdministration(ctx, input.TenantID, input.LegalEntityID, input.RequestID, input.ActorPrincipalID); err != nil {
		return err
	}
	store, err := invitationAdministrationPersistence(s.repo)
	if err != nil {
		return err
	}
	return store.RevokeSessionForRequester(ctx, input, s.now().UTC())
}

func (s *Service) ReplaceInvitation(ctx context.Context, input ReplaceInvitationInput) (IssuedInvitation, error) {
	request, err := s.authorizeRequesterAdministration(ctx, input.TenantID, input.LegalEntityID, input.RequestID, input.ActorPrincipalID)
	if err != nil {
		return IssuedInvitation{}, err
	}
	now := s.now().UTC()
	if !requestOpenAt(request, now) {
		return IssuedInvitation{}, ErrRequestClosed
	}
	audience := normalizeAudience(input.Audience)
	if strings.TrimSpace(input.InvitationID) == "" || audience == "" || strings.TrimSpace(input.Purpose) == "" || !externalAudienceMatches(request, audience) {
		return IssuedInvitation{}, ErrRecipientMismatch
	}
	ttl := time.Duration(input.TTLMinutes) * time.Minute
	if ttl < 5*time.Minute || ttl > 30*24*time.Hour {
		return IssuedInvitation{}, ErrInvitationInvalid
	}
	token, tokenHash, err := tokenPair()
	if err != nil {
		return IssuedInvitation{}, err
	}
	invitationID, err := id.NewUUIDv7()
	if err != nil {
		return IssuedInvitation{}, err
	}
	audienceDigest := sha256.Sum256([]byte(audience))
	invitation := Invitation{
		ID: invitationID, TenantID: input.TenantID, RequestID: input.RequestID,
		TokenHash: tokenHash, AudienceHash: audienceDigest[:], AudienceHint: request.Recipient.AudienceHint,
		Purpose: strings.TrimSpace(input.Purpose), ExpiresAt: now.Add(ttl), MaxRedemptions: 1,
		CreatedBy: input.ActorPrincipalID, CreatedAt: now,
	}
	if invitation.ExpiresAt.After(request.Deadline) {
		invitation.ExpiresAt = request.Deadline
	}
	if !invitation.ExpiresAt.After(now) {
		return IssuedInvitation{}, ErrRequestClosed
	}
	store, err := invitationAdministrationPersistence(s.repo)
	if err != nil {
		return IssuedInvitation{}, err
	}
	if err := store.ReplaceInvitation(ctx, input, invitation, now); err != nil {
		return IssuedInvitation{}, err
	}
	return IssuedInvitation{InvitationID: invitation.ID, Token: token, AudienceHint: invitation.AudienceHint, ExpiresAt: invitation.ExpiresAt}, nil
}
