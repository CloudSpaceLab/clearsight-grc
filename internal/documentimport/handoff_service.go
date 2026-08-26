package documentimport

import (
	"context"
	"strings"
)

func (s *Service) ReviewProposalHandoff(ctx context.Context, input HandoffReviewInput) (Document, error) {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.DocumentID = strings.TrimSpace(input.DocumentID)
	input.ProposalID = strings.TrimSpace(input.ProposalID)
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.Title = strings.TrimSpace(input.Title)
	input.Statement = strings.TrimSpace(input.Statement)
	input.TargetProgramID = strings.TrimSpace(input.TargetProgramID)
	input.Note = strings.TrimSpace(input.Note)
	if input.TenantID == "" || input.DocumentID == "" || input.ProposalID == "" || input.ActorID == "" || input.ExpectedDocumentVersion < 1 || input.ExpectedHandoffVersion < 1 {
		return Document{}, ErrInvalidHandoff
	}
	switch input.Action {
	case HandoffReviewReturn, HandoffReviewReject:
		if input.Note == "" {
			return Document{}, ErrInvalidHandoff
		}
	case HandoffReviewSubmit:
		if input.Title == "" || input.Statement == "" || input.TargetProgramID == "" || input.TargetProgramVersion < 1 {
			return Document{}, ErrInvalidHandoff
		}
		if input.TargetType != ConversionRequirement && input.TargetType != ConversionControlObjective {
			return Document{}, ErrInvalidHandoff
		}
	default:
		return Document{}, ErrInvalidHandoff
	}
	repo, ok := s.repo.(ProposalHandoffRepository)
	if !ok {
		return Document{}, ErrInvalidHandoff
	}
	return repo.ReviewProposalHandoff(ctx, input, s.now().UTC())
}

func (s *Service) AuthorizeProposalHandoff(ctx context.Context, input HandoffAuthorizationInput) (Document, error) {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.DocumentID = strings.TrimSpace(input.DocumentID)
	input.ProposalID = strings.TrimSpace(input.ProposalID)
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.Note = strings.TrimSpace(input.Note)
	input.ResultObjectType = strings.TrimSpace(input.ResultObjectType)
	input.ResultObjectID = strings.TrimSpace(input.ResultObjectID)
	if input.TenantID == "" || input.DocumentID == "" || input.ProposalID == "" || input.ActorID == "" || input.ExpectedDocumentVersion < 1 || input.ExpectedHandoffVersion < 1 {
		return Document{}, ErrInvalidHandoff
	}
	switch input.Action {
	case HandoffAuthorizeReturn, HandoffAuthorizeReject:
		if input.Note == "" {
			return Document{}, ErrInvalidHandoff
		}
	case HandoffAuthorizeApprove:
		if input.Note == "" || input.ResultObjectType == "" || input.ResultObjectID == "" {
			return Document{}, ErrInvalidHandoff
		}
	default:
		return Document{}, ErrInvalidHandoff
	}
	repo, ok := s.repo.(ProposalHandoffRepository)
	if !ok {
		return Document{}, ErrInvalidHandoff
	}
	return repo.AuthorizeProposalHandoff(ctx, input, s.now().UTC())
}
