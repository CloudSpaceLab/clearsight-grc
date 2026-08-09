package evidence

import (
	"context"
	"strings"
)

func (s *Service) DeclareWrongRecipient(ctx context.Context, input DeclareWrongRecipientInput) (Request, error) {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.RequestID = strings.TrimSpace(input.RequestID)
	input.ActorPrincipalID = strings.TrimSpace(input.ActorPrincipalID)
	input.Reason = strings.TrimSpace(input.Reason)
	if input.TenantID == "" || input.RequestID == "" || input.ActorPrincipalID == "" || input.ExpectedVersion <= 0 || input.Reason == "" || len(input.Reason) > 500 {
		return Request{}, ErrRecipientInvalid
	}
	request, err := s.GetRequest(ctx, input.TenantID, input.RequestID)
	if err != nil {
		return Request{}, err
	}
	if !requestOpenAt(request, s.now().UTC()) {
		return Request{}, ErrRequestClosed
	}
	if request.Version != input.ExpectedVersion {
		return Request{}, ErrVersionConflict
	}
	if !RequestAssignedTo(request, input.ActorPrincipalID) {
		return Request{}, ErrRecipientMismatch
	}
	store, err := recipientLifecyclePersistence(s.repo)
	if err != nil {
		return Request{}, err
	}
	if err := store.DeclareWrongRecipient(ctx, input, s.now().UTC()); err != nil {
		return Request{}, err
	}
	return s.GetRequest(ctx, input.TenantID, input.RequestID)
}

func (s *Service) ReassignRecipient(ctx context.Context, input ReassignRecipientInput) (Request, error) {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.RequestID = strings.TrimSpace(input.RequestID)
	input.ActorPrincipalID = strings.TrimSpace(input.ActorPrincipalID)
	input.Reason = strings.TrimSpace(input.Reason)
	if input.TenantID == "" || input.RequestID == "" || input.ActorPrincipalID == "" || input.ExpectedVersion <= 0 || input.Reason == "" || len(input.Reason) > 500 {
		return Request{}, ErrRecipientInvalid
	}
	request, err := s.GetRequest(ctx, input.TenantID, input.RequestID)
	if err != nil {
		return Request{}, err
	}
	if !requestOpenAt(request, s.now().UTC()) {
		return Request{}, ErrRequestClosed
	}
	if request.Version != input.ExpectedVersion {
		return Request{}, ErrVersionConflict
	}
	if strings.TrimSpace(request.CreatedBy) == "" || request.CreatedBy != input.ActorPrincipalID {
		return Request{}, ErrRecipientManagerRequired
	}
	checker, ok := s.repo.(SubjectAccessChecker)
	if !ok {
		return Request{}, ErrRecipientInvalid
	}
	requesterCanRead, err := checker.CanReadSubject(ctx, input.TenantID, input.ActorPrincipalID, request.SubjectType, request.SubjectID)
	if err != nil {
		return Request{}, err
	}
	if !requesterCanRead {
		return Request{}, ErrRecipientMismatch
	}
	next, err := buildRecipient(ctx, s.repo, input.TenantID, request.AudienceType, input.Recipient)
	if err != nil {
		return Request{}, err
	}
	if next.Type == RecipientInternalPrincipal {
		canRead, accessErr := checker.CanReadSubject(ctx, input.TenantID, next.PrincipalID, request.SubjectType, request.SubjectID)
		if accessErr != nil {
			return Request{}, accessErr
		}
		if !canRead {
			return Request{}, ErrRecipientInvalid
		}
	}
	store, err := recipientLifecyclePersistence(s.repo)
	if err != nil {
		return Request{}, err
	}
	if err := store.ReassignRecipient(ctx, input, next, s.now().UTC()); err != nil {
		return Request{}, err
	}
	return s.GetRequest(ctx, input.TenantID, input.RequestID)
}
