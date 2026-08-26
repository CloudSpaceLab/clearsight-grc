package thirdparty

import (
	"context"
	"errors"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
)

const (
	AssessmentSendRequestCommand = "thirdparty.assessment.send_request"

	SendRequestReadyInvitationNotIssued = "REQUEST_READY_INVITATION_NOT_ISSUED"
	SendRequestLinkCreatedEmailNotSent  = "LINK_CREATED_EMAIL_NOT_SENT"
	SendRequestDelivered                = "DELIVERED"
)

var ErrCapturePublicBaseURLInvalid = errors.New("capture public base URL is invalid")

type SendAssessmentRequestInput struct {
	ExpectedVersion      int64     `json:"expected_version"`
	Audience             string    `json:"audience"`
	Deadline             time.Time `json:"deadline"`
	InvitationTTLMinutes int       `json:"invitation_ttl_minutes"`
}

type SendRequestOutcome struct {
	Assessment Assessment                          `json:"assessment"`
	Request    evidence.Request                    `json:"request"`
	Invitation *evidence.IssuedInvitation          `json:"invitation,omitempty"`
	Delivery   *evidence.InvitationDeliveryReceipt `json:"delivery,omitempty"`
	State      string                              `json:"state"`
	Recovery   string                              `json:"recovery,omitempty"`
	CaptureURL string                              `json:"capture_url,omitempty"`
}

type assessmentRequestEvidence interface {
	CreateRequest(context.Context, evidence.CreateRequestInput) (evidence.Request, error)
	GetRequestByOrigin(context.Context, string, evidence.RequestOrigin) (evidence.Request, error)
	ReassignRecipient(context.Context, evidence.ReassignRecipientInput) (evidence.Request, error)
	IssueInvitation(context.Context, evidence.IssueInvitationInput) (evidence.IssuedInvitation, error)
	RevokeRequestCapabilities(context.Context, string, string) error
	RevokeInvitation(context.Context, string, string) error
}

type assessmentFormReader interface {
	FormRevision(context.Context, string, string, int64) (monitoring.FormTemplate, error)
}

type AssessmentRequestService struct {
	assessments *AssessmentService
	repo        AssessmentRepository
	evidence    assessmentRequestEvidence
	forms       assessmentFormReader
	delivery    *evidence.InvitationDeliveryService
	captureBase *url.URL
}

func NewAssessmentRequestService(assessments *AssessmentService, repo AssessmentRepository, evidenceService assessmentRequestEvidence, forms assessmentFormReader, delivery *evidence.InvitationDeliveryService, capturePublicBaseURL, environment string) (*AssessmentRequestService, error) {
	if assessments == nil || repo == nil || evidenceService == nil || forms == nil {
		return nil, ErrInvalid
	}
	base, err := parseCapturePublicBaseURL(capturePublicBaseURL, environment)
	if err != nil {
		return nil, err
	}
	return &AssessmentRequestService{assessments: assessments, repo: repo, evidence: evidenceService, forms: forms, delivery: delivery, captureBase: base}, nil
}

func (s *AssessmentRequestService) SendRequest(ctx context.Context, _ Actor, assessmentID string, input SendAssessmentRequestInput) (SendRequestOutcome, error) {
	assessmentID = strings.TrimSpace(assessmentID)
	audience, err := normalizeAssessmentAudience(input.Audience)
	if err != nil || !validAssessmentIdentifier(assessmentID) || input.ExpectedVersion < 1 || input.Deadline.IsZero() || input.InvitationTTLMinutes < 5 || input.InvitationTTLMinutes > 30*24*60 {
		return SendRequestOutcome{}, ErrInvalid
	}
	verified, err := s.assessments.authorize(ctx, assessmentID, assessmentObjectType, AssessmentSendRequestCommand, authority.ResponsibilityOwner)
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
	if assessment.Status != AssessmentReadyToSend || !validAssessmentIdentifier(assessment.ReviewMatterID) {
		return SendRequestOutcome{}, ErrInvalidAssessmentTransition
	}
	deadline := input.Deadline.UTC()
	if !deadline.After(s.assessments.now().UTC()) || deadline.After(assessment.ReviewDueAt) {
		return SendRequestOutcome{}, ErrInvalid
	}

	form, err := s.forms.FormRevision(ctx, scope.TenantID, assessment.FormTemplateID, assessment.FormTemplateVersion)
	if err != nil {
		return SendRequestOutcome{}, err
	}
	if form.ID != assessment.FormTemplateID || form.Version != assessment.FormTemplateVersion || form.Status != monitoring.LifecycleActive || !form.IsCurrent {
		return SendRequestOutcome{}, monitoring.ErrInactive
	}
	relationship, err := s.repo.GetRelationship(ctx, scope, assessment.RelationshipID)
	if err != nil {
		return SendRequestOutcome{}, err
	}
	if relationship.Relationship.Status != RelationshipProposed && relationship.Relationship.Status != RelationshipUnderReview {
		return SendRequestOutcome{}, ErrInvalidAssessmentTransition
	}

	origin := evidence.RequestOrigin{Type: AssessmentRequestOrigin, ID: assessment.ID, Version: 1}
	request, err := s.evidence.GetRequestByOrigin(ctx, scope.TenantID, origin)
	if errors.Is(err, evidence.ErrNotFound) {
		request, err = s.evidence.CreateRequest(evidence.WithRequestOriginAuthority(ctx, AssessmentRequestOrigin), assessmentEvidenceRequestInput(verified, assessment, relationship, form, origin, audience, deadline))
	}
	if err != nil {
		return SendRequestOutcome{}, err
	}
	if request.Origin != origin || request.SubjectType != "VENDOR_RELATIONSHIP" || request.SubjectID != assessment.RelationshipID || request.FormTemplateID != assessment.FormTemplateID || request.FormTemplateVersion != assessment.FormTemplateVersion {
		return SendRequestOutcome{}, ErrInvalid
	}
	if !evidence.ExternalAudienceMatches(request, audience) {
		request, err = s.evidence.ReassignRecipient(ctx, evidence.ReassignRecipientInput{
			TenantID: scope.TenantID, RequestID: request.ID, ActorPrincipalID: verified.PrincipalID,
			Recipient: evidence.RecipientInput{Type: evidence.RecipientExternalAudience, Audience: audience},
			Reason:    "Vendor request recipient corrected before collection.", ExpectedVersion: request.Version,
		})
		if err != nil {
			return SendRequestOutcome{}, err
		}
	}

	preparedLink, preparedAssessment, err := s.repo.PrepareAssessmentRequest(ctx, PrepareAssessmentRequestRecord{
		Scope: scope, AssessmentID: assessment.ID, ExpectedVersion: assessment.Version, ActorPrincipalID: verified.PrincipalID, RequestID: request.ID,
		Purpose: AssessmentRequestInitial, OriginType: AssessmentRequestOrigin, OriginID: assessment.ID, OriginSequence: 1,
		PreparedAt: s.assessments.now().UTC(),
	})
	if err != nil {
		return SendRequestOutcome{}, err
	}
	if preparedLink.InvitationID != "" {
		return SendRequestOutcome{Assessment: preparedAssessment, Request: request, State: SendRequestDelivered}, nil
	}
	if s.captureBase == nil {
		return preparedOutcome(preparedAssessment, request, "Set the secure capture address, then issue the invitation."), nil
	}
	issued, err := s.evidence.IssueInvitation(ctx, evidence.IssueInvitationInput{
		TenantID: scope.TenantID, RequestID: request.ID, Audience: audience,
		Purpose: "Complete the vendor due-diligence request.", TTLMinutes: input.InvitationTTLMinutes, CreatedBy: verified.PrincipalID,
	})
	if err != nil {
		return preparedOutcome(preparedAssessment, request, "Retry invitation creation for this request."), nil
	}
	linkURL := captureInvitationURL(s.captureBase, issued.Token)
	issuedOutcome := issued
	issuedOutcome.Token = ""
	finalized, err := s.assessments.RecordRequestIssued(ctx, verified, assessment.ID, RecordRequestIssuedInput{
		ExpectedVersion: preparedAssessment.Version, RequestID: request.ID, Purpose: AssessmentRequestInitial,
		OriginType: AssessmentRequestOrigin, OriginID: assessment.ID, OriginSequence: 1, InvitationID: issued.InvitationID,
	})
	if err != nil {
		if revokeErr := s.evidence.RevokeInvitation(ctx, scope.TenantID, issued.InvitationID); revokeErr != nil {
			return preparedOutcome(preparedAssessment, request, "Retry invitation creation. Any prior secure access will be revoked before a replacement is issued."), nil
		}
		return preparedOutcome(preparedAssessment, request, "Retry invitation creation for this request."), nil
	}
	outcome := SendRequestOutcome{Assessment: finalized.Assessment, Request: request, Invitation: &issuedOutcome}
	receipt, deliveryErr := s.delivery.Deliver(ctx, evidence.InvitationDeliveryRequest{RecipientAddress: audience, InvitationLink: linkURL})
	outcome.Delivery = &receipt
	if deliveryErr != nil || receipt.Status != evidence.InvitationDelivered {
		outcome.State = SendRequestLinkCreatedEmailNotSent
		outcome.CaptureURL = linkURL
		outcome.Recovery = "Copy the secure link or retry email delivery."
		return outcome, nil
	}
	outcome.State = SendRequestDelivered
	return outcome, nil
}

func preparedOutcome(assessment Assessment, request evidence.Request, recovery string) SendRequestOutcome {
	return SendRequestOutcome{Assessment: assessment, Request: request, State: SendRequestReadyInvitationNotIssued, Recovery: recovery}
}

func assessmentEvidenceRequestInput(actor Actor, assessment Assessment, aggregate Aggregate, form monitoring.FormTemplate, origin evidence.RequestOrigin, audience string, deadline time.Time) evidence.CreateRequestInput {
	fields := make([]evidence.Field, len(form.Fields))
	for index, field := range form.Fields {
		fields[index] = evidence.Field{
			ID: field.ID, SectionID: field.SectionID, Label: field.Label, Type: string(field.Type), Required: field.Required,
			Description: field.Description, Options: append([]string(nil), field.Options...), AcceptedFormats: append([]string(nil), field.AcceptedFormats...),
			Attestation: field.Attestation, Constraints: field.Constraints, Condition: field.Condition, Scoring: field.Scoring,
		}
	}
	facts := map[string]string{
		"vendor_legal_name":      aggregate.Vendor.LegalName,
		"vendor_trading_name":    aggregate.Vendor.TradingName,
		"registration_reference": aggregate.Vendor.RegistrationRef,
		"jurisdiction":           aggregate.Vendor.Jurisdiction,
		"service_name":           aggregate.Relationship.ServiceName,
		"criticality":            string(aggregate.Relationship.Criticality),
		"privacy_role":           string(aggregate.Relationship.PrivacyRole),
	}
	return evidence.CreateRequestInput{
		TenantID: actor.TenantID, SubjectType: "VENDOR_RELATIONSHIP", SubjectID: assessment.RelationshipID,
		Title: form.Name, Purpose: form.Purpose, WhyYou: "Provide the information required for the bank's review of this service.",
		Sensitivity: "CONFIDENTIAL", AudienceType: "VENDOR",
		Recipient:        evidence.RecipientInput{Type: evidence.RecipientExternalAudience, Audience: audience},
		EstimatedMinutes: estimateAssessmentMinutes(len(fields)), Deadline: deadline, KnownFacts: facts,
		Presentation: form.Presentation, Sections: form.Sections, Fields: fields,
		FormTemplateID: form.ID, FormTemplateVersion: form.Version, Origin: origin, CreatedBy: actor.PrincipalID,
	}
}

func estimateAssessmentMinutes(fieldCount int) int {
	minutes := (fieldCount + 2) / 3
	if minutes < 5 {
		return 5
	}
	if minutes > 60 {
		return 60
	}
	return minutes
}

func normalizeAssessmentAudience(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Address != value || len(value) > 254 {
		return "", ErrInvalid
	}
	return value, nil
}

func parseCapturePublicBaseURL(value, environment string) (*url.URL, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := url.Parse(value)
	if err != nil || !parsed.IsAbs() || parsed.User != nil || parsed.Hostname() == "" || parsed.Fragment != "" {
		return nil, ErrCapturePublicBaseURLInvalid
	}
	local := strings.EqualFold(parsed.Hostname(), "localhost") || parsed.Hostname() == "127.0.0.1" || parsed.Hostname() == "::1"
	if parsed.Scheme != "https" && !(strings.EqualFold(environment, "development") && parsed.Scheme == "http" && local) {
		return nil, ErrCapturePublicBaseURLInvalid
	}
	copy := *parsed
	return &copy, nil
}

func captureInvitationURL(base *url.URL, token string) string {
	copy := *base
	values := copy.Query()
	values.Set("capture_invite", token)
	copy.RawQuery = values.Encode()
	return copy.String()
}

var _ assessmentRequestEvidence = (*evidence.Service)(nil)
