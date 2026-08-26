package documentimport

import (
	"context"
	"strings"
	"time"
)

func (r *MemoryRepository) ReviewProposalHandoff(_ context.Context, input HandoffReviewInput, now time.Time) (Document, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.items[input.DocumentID]
	if !ok || value.TenantID != strings.TrimSpace(input.TenantID) {
		return Document{}, ErrNotFound
	}
	if value.Version != input.ExpectedDocumentVersion {
		return Document{}, ErrVersionConflict
	}
	for index := range value.Proposals {
		proposal := &value.Proposals[index]
		if proposal.ID != input.ProposalID {
			continue
		}
		if proposal.Status != ProposalAccepted || proposal.Handoff == nil || proposal.Handoff.Status != HandoffAwaitingReview || proposal.Handoff.Version != input.ExpectedHandoffVersion {
			return Document{}, ErrInvalidHandoff
		}
		if proposal.Handoff.IntakePrincipalID == input.ActorID {
			return Document{}, ErrHandoffSegregation
		}
		handoff := proposal.Handoff
		handoff.ReviewerPrincipalID = input.ActorID
		handoff.ReviewNote = input.Note
		handoff.Route = nil
		switch input.Action {
		case HandoffReviewReturn:
			handoff.Status = HandoffReturned
		case HandoffReviewReject:
			handoff.Status = HandoffRejected
		case HandoffReviewSubmit:
			handoff.Status = HandoffAwaitingAuthorization
			handoff.DraftTitle = input.Title
			handoff.DraftStatement = input.Statement
			handoff.TargetType = input.TargetType
			handoff.TargetProgramID = input.TargetProgramID
			handoff.TargetProgramVersion = input.TargetProgramVersion
		default:
			return Document{}, ErrInvalidHandoff
		}
		handoff.Version++
		handoff.UpdatedAt = now.UTC()
		value.UpdatedAt = now.UTC()
		value.Version++
		r.items[value.ID] = cloneDocument(value)
		return cloneDocument(value), nil
	}
	return Document{}, ErrNotFound
}

func (r *MemoryRepository) AuthorizeProposalHandoff(_ context.Context, input HandoffAuthorizationInput, now time.Time) (Document, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.items[input.DocumentID]
	if !ok || value.TenantID != strings.TrimSpace(input.TenantID) {
		return Document{}, ErrNotFound
	}
	if value.Version != input.ExpectedDocumentVersion {
		return Document{}, ErrVersionConflict
	}
	for index := range value.Proposals {
		proposal := &value.Proposals[index]
		if proposal.ID != input.ProposalID {
			continue
		}
		if proposal.Status != ProposalAccepted || proposal.Handoff == nil || proposal.Handoff.Status != HandoffAwaitingAuthorization || proposal.Handoff.Version != input.ExpectedHandoffVersion {
			return Document{}, ErrInvalidHandoff
		}
		if proposal.Handoff.IntakePrincipalID == input.ActorID || proposal.Handoff.ReviewerPrincipalID == input.ActorID {
			return Document{}, ErrHandoffSegregation
		}
		handoff := proposal.Handoff
		handoff.AuthorizerPrincipalID = input.ActorID
		handoff.AuthorizationNote = input.Note
		handoff.Route = nil
		switch input.Action {
		case HandoffAuthorizeReturn:
			handoff.Status = HandoffReturned
		case HandoffAuthorizeReject:
			handoff.Status = HandoffRejected
		case HandoffAuthorizeApprove:
			handoff.Status = HandoffApproved
			handoff.ResultObjectType = input.ResultObjectType
			handoff.ResultObjectID = input.ResultObjectID
		default:
			return Document{}, ErrInvalidHandoff
		}
		handoff.Version++
		handoff.UpdatedAt = now.UTC()
		value.UpdatedAt = now.UTC()
		value.Version++
		r.items[value.ID] = cloneDocument(value)
		return cloneDocument(value), nil
	}
	return Document{}, ErrNotFound
}

var _ ProposalHandoffRepository = (*MemoryRepository)(nil)
