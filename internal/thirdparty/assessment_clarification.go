package thirdparty

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
)

const AssessmentClarificationCommand = "thirdparty.assessment.clarification.request"

type RequestAssessmentClarificationInput struct {
	ExpectedVersion      int64     `json:"expected_version"`
	RequestFields        []string  `json:"request_fields"`
	Message              string    `json:"message"`
	Audience             string    `json:"audience"`
	Deadline             time.Time `json:"deadline"`
	InvitationTTLMinutes int       `json:"invitation_ttl_minutes"`
}

type AssessmentClarificationOutcome struct {
	Assessment Assessment                          `json:"assessment"`
	Invitation *evidence.IssuedInvitation          `json:"invitation,omitempty"`
	Delivery   *evidence.InvitationDeliveryReceipt `json:"delivery,omitempty"`
	State      string                              `json:"state"`
	Recovery   string                              `json:"recovery,omitempty"`
	CaptureURL string                              `json:"capture_url,omitempty"`
}

func (s *AssessmentRequestService) RequestClarification(ctx context.Context, _ Actor, assessmentID string, input RequestAssessmentClarificationInput) (AssessmentClarificationOutcome, error) {
	assessmentID = strings.TrimSpace(assessmentID)
	input.Message = strings.TrimSpace(input.Message)
	audience, audienceErr := normalizeAssessmentAudience(input.Audience)
	if audienceErr != nil || !validAssessmentIdentifier(assessmentID) || input.ExpectedVersion < 1 || len(input.RequestFields) < 1 || len(input.RequestFields) > formcontract.MaxFields || input.Message == "" || len(input.Message) > 2000 || input.Deadline.IsZero() || input.InvitationTTLMinutes < 5 || input.InvitationTTLMinutes > 30*24*60 {
		return AssessmentClarificationOutcome{}, ErrInvalid
	}
	requested := make(map[string]struct{}, len(input.RequestFields))
	for _, fieldID := range input.RequestFields {
		fieldID = strings.TrimSpace(fieldID)
		if !validAssessmentIdentifier(fieldID) {
			return AssessmentClarificationOutcome{}, ErrInvalid
		}
		if _, duplicate := requested[fieldID]; duplicate {
			return AssessmentClarificationOutcome{}, ErrInvalid
		}
		requested[fieldID] = struct{}{}
	}
	verified, err := s.assessments.authorize(ctx, assessmentID, assessmentObjectType, AssessmentClarificationCommand, authority.ResponsibilityReviewer)
	if err != nil {
		return AssessmentClarificationOutcome{}, err
	}
	scope := scopeFrom(verified)
	assessment, err := s.repo.GetAssessment(ctx, scope, assessmentID)
	if err != nil {
		return AssessmentClarificationOutcome{}, err
	}
	if assessment.Version != input.ExpectedVersion {
		return AssessmentClarificationOutcome{}, ErrVersionConflict
	}
	if assessment.Status != AssessmentUnderReview || !validAssessmentIdentifiers(assessment.CurrentRequestID, assessment.SubmissionID) {
		return AssessmentClarificationOutcome{}, ErrInvalidAssessmentTransition
	}
	now, deadline := s.assessments.now().UTC(), input.Deadline.UTC()
	if !deadline.After(now) || deadline.After(assessment.ReviewDueAt) {
		return AssessmentClarificationOutcome{}, ErrInvalid
	}
	relationship, err := s.repo.GetRelationship(ctx, scope, assessment.RelationshipID)
	if err != nil {
		return AssessmentClarificationOutcome{}, err
	}
	if !assessmentKindAllowedForRelationship(assessment.ReviewKind, relationship.Relationship.Status) {
		return AssessmentClarificationOutcome{}, ErrInvalidAssessmentTransition
	}
	form, err := s.forms.AssessmentFormRevision(ctx, scope.TenantID, scope.LegalEntityID, assessment.FormTemplateID, assessment.FormTemplateVersion)
	if err != nil {
		return AssessmentClarificationOutcome{}, err
	}
	if form.ID != assessment.FormTemplateID || form.Version != assessment.FormTemplateVersion || form.Status != monitoring.LifecycleActive || !form.IsCurrent {
		return AssessmentClarificationOutcome{}, monitoring.ErrInactive
	}
	fields, sections, err := clarificationForm(form, requested)
	if err != nil {
		return AssessmentClarificationOutcome{}, err
	}
	links, err := s.repo.ListAssessmentRequestLinks(ctx, scope, assessment.ID)
	if err != nil || len(links) == 0 || len(links) >= formcontract.MaxFields {
		if err != nil {
			return AssessmentClarificationOutcome{}, err
		}
		return AssessmentClarificationOutcome{}, ErrInvalid
	}
	last := links[len(links)-1]
	if last.RequestID != assessment.CurrentRequestID || last.Sequence != len(links) || last.OriginType != AssessmentRequestOrigin || last.OriginID != assessment.ID || last.OriginSequence != last.Sequence {
		return AssessmentClarificationOutcome{}, ErrInvalid
	}
	sequence := last.Sequence + 1
	recoverPrepared := last.Purpose == AssessmentRequestClarification && last.InvitationID == ""
	if recoverPrepared {
		sequence = last.Sequence
	}
	origin := evidence.RequestOrigin{Type: AssessmentRequestOrigin, ID: assessment.ID, Version: int64(sequence)}
	request, err := s.evidence.GetRequestByOrigin(ctx, scope.TenantID, origin)
	if errors.Is(err, evidence.ErrNotFound) && !recoverPrepared {
		request, err = s.evidence.CreateRequest(evidence.WithRequestOriginAuthority(ctx, AssessmentRequestOrigin), clarificationEvidenceRequestInput(verified, assessment, relationship, form, fields, sections, origin, audience, input.Message, deadline))
	}
	if err != nil {
		return AssessmentClarificationOutcome{}, err
	}
	if !sameClarificationRequest(request, assessment, origin, audience, input.Message, deadline, fields) {
		return AssessmentClarificationOutcome{}, ErrVersionConflict
	}
	preparedLink, preparedAssessment, err := s.repo.PrepareAssessmentRequest(ctx, PrepareAssessmentRequestRecord{
		Scope: scope, AssessmentID: assessment.ID, ExpectedVersion: assessment.Version, ActorPrincipalID: verified.PrincipalID,
		RequestID: request.ID, Purpose: AssessmentRequestClarification, OriginType: origin.Type, OriginID: origin.ID, OriginSequence: sequence, PreparedAt: now,
	})
	if err != nil {
		return AssessmentClarificationOutcome{}, err
	}
	if preparedLink.InvitationID != "" {
		return AssessmentClarificationOutcome{Assessment: preparedAssessment, State: SendRequestDelivered}, nil
	}
	if s.captureBase == nil {
		return clarificationPreparedOutcome(preparedAssessment, "Set the secure capture address, then issue the clarification invitation."), nil
	}
	issued, err := s.evidence.IssueInvitation(ctx, evidence.IssueInvitationInput{TenantID: scope.TenantID, RequestID: request.ID, Audience: audience, Purpose: "Complete the requested vendor clarification.", TTLMinutes: input.InvitationTTLMinutes, CreatedBy: verified.PrincipalID})
	if err != nil {
		return clarificationPreparedOutcome(preparedAssessment, "Retry invitation creation for this clarification request."), nil
	}
	linkURL := captureInvitationURL(s.captureBase, issued.Token)
	publicInvitation := issued
	publicInvitation.Token = ""
	finalized, err := s.assessments.RecordRequestIssued(ctx, verified, assessment.ID, RecordRequestIssuedInput{ExpectedVersion: preparedAssessment.Version, RequestID: request.ID, Purpose: AssessmentRequestClarification, OriginType: origin.Type, OriginID: origin.ID, OriginSequence: sequence, InvitationID: issued.InvitationID})
	if err != nil {
		_ = s.evidence.RevokeInvitation(ctx, scope.TenantID, issued.InvitationID)
		return clarificationPreparedOutcome(preparedAssessment, "Retry invitation creation for this clarification request."), nil
	}
	outcome := AssessmentClarificationOutcome{Assessment: finalized.Assessment, Invitation: &publicInvitation}
	receipt, deliveryErr := s.delivery.Deliver(ctx, evidence.InvitationDeliveryRequest{RecipientAddress: audience, InvitationLink: linkURL})
	outcome.Delivery = &receipt
	if deliveryErr != nil || receipt.Status != evidence.InvitationDelivered {
		outcome.State, outcome.CaptureURL, outcome.Recovery = SendRequestLinkCreatedEmailNotSent, linkURL, "Copy the secure link or retry email delivery."
		return outcome, nil
	}
	outcome.State = SendRequestDelivered
	return outcome, nil
}

func clarificationPreparedOutcome(assessment Assessment, recovery string) AssessmentClarificationOutcome {
	return AssessmentClarificationOutcome{Assessment: assessment, State: SendRequestReadyInvitationNotIssued, Recovery: recovery}
}

func clarificationForm(form monitoring.FormTemplate, requested map[string]struct{}) ([]evidence.Field, []formcontract.Section, error) {
	fields := make([]evidence.Field, 0, len(requested))
	sectionsUsed := map[string]struct{}{}
	for _, field := range form.Fields {
		if _, ok := requested[field.ID]; !ok {
			continue
		}
		fields = append(fields, evidence.Field{ID: field.ID, SectionID: field.SectionID, Label: field.Label, Type: string(field.Type), Required: true, Description: field.Description, Options: append([]string(nil), field.Options...), AcceptedFormats: append([]string(nil), field.AcceptedFormats...), Attestation: field.Attestation, Constraints: field.Constraints})
		sectionsUsed[field.SectionID] = struct{}{}
		delete(requested, field.ID)
	}
	if len(requested) != 0 {
		return nil, nil, ErrInvalid
	}
	sections := make([]formcontract.Section, 0, len(sectionsUsed))
	for _, section := range form.Sections {
		if _, ok := sectionsUsed[section.ID]; ok {
			section.Condition = nil
			sections = append(sections, section)
		}
	}
	return fields, sections, nil
}

func clarificationEvidenceRequestInput(actor Actor, assessment Assessment, aggregate Aggregate, form monitoring.FormTemplate, fields []evidence.Field, sections []formcontract.Section, origin evidence.RequestOrigin, audience, message string, deadline time.Time) evidence.CreateRequestInput {
	input := assessmentEvidenceRequestInput(actor, assessment, aggregate, form, origin, audience, deadline)
	input.Title = "Clarify the vendor due-diligence response"
	input.Purpose = message
	input.WhyYou = "Update the selected response fields so the bank can continue its review."
	input.Fields, input.Sections = fields, sections
	input.EstimatedMinutes = estimateAssessmentMinutes(len(fields))
	return input
}

func sameClarificationRequest(request evidence.Request, assessment Assessment, origin evidence.RequestOrigin, audience, message string, deadline time.Time, fields []evidence.Field) bool {
	if request.TenantID != assessment.TenantID || request.SubjectType != "VENDOR_RELATIONSHIP" || request.SubjectID != assessment.RelationshipID || request.FormTemplateID != assessment.FormTemplateID || request.FormTemplateVersion != assessment.FormTemplateVersion || request.Origin != origin || request.Purpose != message || !request.Deadline.Equal(deadline) || !evidence.ExternalAudienceMatches(request, audience) || len(request.Fields) != len(fields) {
		return false
	}
	for index := range fields {
		if request.Fields[index].ID != fields[index].ID {
			return false
		}
	}
	return true
}
