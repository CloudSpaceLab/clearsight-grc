package evidence

import (
	"context"
	"strings"
	"time"
)

func (r *MemoryRepository) DeclareWrongRecipient(_ context.Context, input DeclareWrongRecipientInput, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	request, ok := r.requests[input.RequestID]
	if !ok || request.TenantID != input.TenantID {
		return ErrNotFound
	}
	if !requestOpenAt(request, now) {
		return ErrRequestClosed
	}
	if input.ExpectedVersion <= 0 || request.Version != input.ExpectedVersion {
		return ErrVersionConflict
	}
	if request.Recipient.Type != RecipientInternalPrincipal || request.Recipient.PrincipalID != input.ActorPrincipalID || !recipientIsAssigned(request.Recipient) {
		return ErrRecipientMismatch
	}
	request.Recipient.State = RecipientStateReassignmentRequired
	request.Recipient.IssueReason = strings.TrimSpace(input.Reason)
	request.Version++
	request.UpdatedAt = now
	r.requests[request.ID] = request
	return nil
}

func (r *MemoryRepository) ReassignRecipient(_ context.Context, input ReassignRecipientInput, next Recipient, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	request, ok := r.requests[input.RequestID]
	if !ok || request.TenantID != input.TenantID {
		return ErrNotFound
	}
	if !requestOpenAt(request, now) {
		return ErrRequestClosed
	}
	if input.ExpectedVersion <= 0 || request.Version != input.ExpectedVersion {
		return ErrVersionConflict
	}
	if strings.TrimSpace(request.CreatedBy) == "" || request.CreatedBy != input.ActorPrincipalID {
		return ErrRecipientManagerRequired
	}
	if strings.TrimSpace(input.LegalEntityID) != "" && request.LegalEntityID != input.LegalEntityID {
		return ErrRecipientInvalid
	}
	if sameRecipient(request.Recipient, next) && recipientIsAssigned(request.Recipient) {
		return ErrRecipientInvalid
	}
	subjectType := strings.ToUpper(strings.TrimSpace(request.SubjectType))
	if subjectType == "PROGRAM" || subjectType == "MATTER" {
		subjectKey := subjectType + ":" + strings.TrimSpace(request.SubjectID)
		requester, exists := r.candidates[input.ActorPrincipalID]
		if !exists || !memoryRecipientEligibleForScope(requester, request.TenantID, request.LegalEntityID, subjectKey) {
			return ErrRecipientInvalid
		}
	}
	if next.Type == RecipientInternalPrincipal && (subjectType == "PROGRAM" || subjectType == "MATTER") {
		subjectKey := subjectType + ":" + strings.TrimSpace(request.SubjectID)
		candidate, exists := r.candidates[next.PrincipalID]
		if !exists || !memoryRecipientEligibleForScope(candidate, request.TenantID, request.LegalEntityID, subjectKey) {
			return ErrRecipientInvalid
		}
	}
	revision := request.Recipient.Revision + 1
	if revision < 1 {
		revision = 1
	}
	next.Revision = revision
	next.State = RecipientStateAssigned
	request.Recipient = cloneRecipient(next)
	request.Version++
	request.UpdatedAt = now
	r.requests[request.ID] = request
	for key, invitation := range r.invitations {
		if invitation.TenantID == input.TenantID && invitation.RequestID == input.RequestID && invitation.RevokedAt == nil {
			invitation.RevokedAt = pointerTime(now)
			r.invitations[key] = invitation
		}
	}
	for key, session := range r.sessions {
		if session.TenantID == input.TenantID && session.RequestID == input.RequestID && session.RevokedAt == nil {
			session.RevokedAt = pointerTime(now)
			r.sessions[key] = session
		}
	}
	return nil
}

var _ recipientLifecycleStore = (*MemoryRepository)(nil)
