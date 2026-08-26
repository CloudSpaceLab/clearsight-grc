package thirdparty

import (
	"context"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

const AssessmentReissueRequestCommand = "thirdparty.assessment.reissue_request"

type ReissueAssessmentRequestInput struct {
	ExpectedVersion      int64  `json:"expected_version"`
	Audience             string `json:"audience"`
	InvitationTTLMinutes int    `json:"invitation_ttl_minutes"`
}

func (s *AssessmentRequestService) ReissueRequest(ctx context.Context, _ Actor, assessmentID string, input ReissueAssessmentRequestInput) (SendRequestOutcome, error) {
	assessmentID = strings.TrimSpace(assessmentID)
	audience, err := normalizeAssessmentAudience(input.Audience)
	if err != nil || !validAssessmentIdentifier(assessmentID) || input.ExpectedVersion < 1 || input.InvitationTTLMinutes < 5 || input.InvitationTTLMinutes > 30*24*60 {
		return SendRequestOutcome{}, ErrInvalid
	}
	verified, err := s.assessments.authorize(ctx, assessmentID, assessmentObjectType, AssessmentReissueRequestCommand, authority.ResponsibilityOwner)
	if err != nil {
		return SendRequestOutcome{}, err
	}
	scope := scopeFrom(verified)
	assessment, err := s.repo.GetAssessment(ctx, scope, assessmentID)
	if err != nil {
		return SendRequestOutcome{}, err
	}
	if assessment.Version != input.ExpectedVersion {
		return SendRequestOutcome{}, ErrVersionConflict
	}
	if assessment.Status != AssessmentCollecting || !validAssessmentIdentifier(assessment.CurrentRequestID) {
		return SendRequestOutcome{}, ErrInvalidAssessmentTransition
	}
	relationship, err := s.repo.GetRelationship(ctx, scope, assessment.RelationshipID)
	if err != nil {
		return SendRequestOutcome{}, err
	}
	if !assessmentKindAllowedForRelationship(assessment.ReviewKind, relationship.Relationship.Status) {
		return SendRequestOutcome{}, ErrInvalidAssessmentTransition
	}
	link, err := s.repo.GetCurrentAssessmentRequestLink(ctx, scope, assessment.ID)
	if err != nil {
		return SendRequestOutcome{}, err
	}
	if link.RequestID != assessment.CurrentRequestID || link.AssessmentID != assessment.ID || link.OriginType != AssessmentRequestOrigin || link.OriginID != assessment.ID || link.OriginSequence < 1 {
		return SendRequestOutcome{}, ErrInvalid
	}
	request, err := s.evidence.GetRequestByOrigin(ctx, scope.TenantID, evidence.RequestOrigin{Type: link.OriginType, ID: link.OriginID, Version: int64(link.OriginSequence)})
	if err != nil {
		return SendRequestOutcome{}, err
	}
	if request.ID != link.RequestID || request.TenantID != scope.TenantID || request.SubjectType != "VENDOR_RELATIONSHIP" || request.SubjectID != assessment.RelationshipID || request.FormTemplateID != assessment.FormTemplateID || request.FormTemplateVersion != assessment.FormTemplateVersion {
		return SendRequestOutcome{}, ErrInvalid
	}
	if !evidence.ExternalAudienceMatches(request, audience) {
		return SendRequestOutcome{}, ErrInvalid
	}
	preparedLink, preparedAssessment, err := s.repo.PrepareRequestReissue(ctx, PrepareRequestReissueRecord{
		Scope: scope, AssessmentID: assessment.ID, ExpectedVersion: assessment.Version, ActorPrincipalID: verified.PrincipalID,
		RequestID: request.ID, ExpectedInvitationID: link.InvitationID, PreparedAt: s.assessments.now().UTC(),
	})
	if err != nil {
		return SendRequestOutcome{}, err
	}
	if preparedLink.InvitationID != "" {
		return SendRequestOutcome{}, ErrInvalid
	}
	if err := s.evidence.RevokeRequestCapabilities(ctx, scope.TenantID, request.ID); err != nil {
		return SendRequestOutcome{}, err
	}
	if s.captureBase == nil {
		return SendRequestOutcome{Assessment: preparedAssessment, Request: request, State: SendRequestReadyInvitationNotIssued, Recovery: "Set the secure capture address, then issue a replacement invitation."}, nil
	}
	issued, err := s.evidence.IssueInvitation(ctx, evidence.IssueInvitationInput{
		TenantID: scope.TenantID, RequestID: request.ID, Audience: audience,
		Purpose: "Complete the vendor due-diligence request.", TTLMinutes: input.InvitationTTLMinutes, CreatedBy: verified.PrincipalID,
	})
	if err != nil {
		return SendRequestOutcome{Assessment: preparedAssessment, Request: request, State: SendRequestReadyInvitationNotIssued, Recovery: "Retry replacement invitation creation for this request."}, nil
	}
	linkURL := captureInvitationURL(s.captureBase, issued.Token)
	issuedOutcome := issued
	issuedOutcome.Token = ""
	_, updated, err := s.repo.FinalizeRequestReissue(ctx, FinalizeRequestReissueRecord{
		Scope: scope, AssessmentID: assessment.ID, ExpectedVersion: preparedAssessment.Version, ActorPrincipalID: verified.PrincipalID,
		RequestID: request.ID, InvitationID: issued.InvitationID, ReissuedAt: s.assessments.now().UTC(),
	})
	if err != nil {
		_ = s.evidence.RevokeInvitation(ctx, scope.TenantID, issued.InvitationID)
		return SendRequestOutcome{}, err
	}
	outcome := SendRequestOutcome{Assessment: updated, Request: request, Invitation: &issuedOutcome}
	receipt, deliveryErr := s.delivery.Deliver(ctx, evidence.InvitationDeliveryRequest{RecipientAddress: audience, InvitationLink: linkURL})
	outcome.Delivery = &receipt
	if deliveryErr != nil || receipt.Status != evidence.InvitationDelivered {
		outcome.State = SendRequestLinkCreatedEmailNotSent
		outcome.CaptureURL = linkURL
		outcome.Recovery = "Copy the replacement secure link or retry email delivery."
		return outcome, nil
	}
	outcome.State = SendRequestDelivered
	return outcome, nil
}
