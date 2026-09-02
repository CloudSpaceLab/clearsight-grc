package thirdparty

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

const (
	AddressVerificationRequestOrigin = "THIRD_PARTY_ADDRESS_VERIFICATION"
	addressVerificationConsumerName  = "third-party-address-verification-setup"
	addressSubmissionConsumerName    = "third-party-address-verification-submission"
	addressVerificationFormCode      = "VENDOR-ADDRESS-VERIFICATION"
	addressVerificationActionOrigin  = "thirdparty-address-verification"
)

type addressVerificationMatterService interface {
	CreateMatter(context.Context, continuity.CreateMatterInput) (continuity.MatterAggregate, error)
	MatterByTriggerKey(context.Context, string, string) (continuity.MatterAggregate, error)
	GetMatter(context.Context, string, string) (continuity.MatterAggregate, error)
	ListMatters(context.Context, string, string, int) ([]continuity.MatterAggregate, error)
	AddAction(context.Context, continuity.AddActionInput) (continuity.MatterAggregate, error)
	TransitionAction(context.Context, continuity.TransitionActionInput) (continuity.MatterAggregate, error)
	AddVerificationContract(context.Context, continuity.AddVerificationContractInput) (continuity.MatterAggregate, error)
}

type addressVerificationEvidence interface {
	CreateRequest(context.Context, evidence.CreateRequestInput) (evidence.Request, error)
	GetRequestByOrigin(context.Context, string, evidence.RequestOrigin) (evidence.Request, error)
}

type addressVerificationFormReader interface {
	ListReusableFormRevisions(context.Context, string, string, int) ([]monitoring.FormTemplate, error)
}

type AddressVerificationProvisioner struct {
	inbox       AssessmentSubmissionInbox
	requests    AssessmentSubmissionRequestReader
	assessments AssessmentRepository
	matters     addressVerificationMatterService
	evidence    addressVerificationEvidence
	forms       addressVerificationFormReader
	now         func() time.Time
}

func NewAddressVerificationProvisioner(inbox AssessmentSubmissionInbox, requests AssessmentSubmissionRequestReader, assessments AssessmentRepository, matters addressVerificationMatterService, evidenceService addressVerificationEvidence, forms addressVerificationFormReader) *AddressVerificationProvisioner {
	return &AddressVerificationProvisioner{inbox: inbox, requests: requests, assessments: assessments, matters: matters, evidence: evidenceService, forms: forms, now: time.Now}
}

func (p *AddressVerificationProvisioner) Publish(ctx context.Context, event workflowruntime.OutboxEvent) error {
	if event.AggregateType != assessmentSubmissionAggregateType || event.EventType != assessmentSubmissionEventType {
		return nil
	}
	if p == nil || p.inbox == nil || p.requests == nil || p.assessments == nil || p.matters == nil || p.evidence == nil || p.forms == nil {
		return errors.New("address verification setup is not configured")
	}
	processed, err := p.inbox.InboxProcessed(ctx, event.TenantID, addressVerificationConsumerName, event.ID)
	if err != nil || processed {
		return err
	}
	request, err := p.requests.GetRequest(ctx, event.TenantID, event.AggregateID)
	if err != nil {
		return fmt.Errorf("read vendor registration request: %w", err)
	}
	origin := request.Origin
	if origin.Type != AssessmentRequestOrigin || origin.Version != 1 || strings.TrimSpace(origin.ID) == "" {
		return nil
	}
	scope := Scope{TenantID: event.TenantID, LegalEntityID: request.LegalEntityID}
	assessment, err := p.assessments.GetAssessment(ctx, scope, origin.ID)
	if err != nil {
		return err
	}
	links, err := p.assessments.ListAssessmentRequestLinks(ctx, scope, assessment.ID)
	if err != nil {
		return err
	}
	linked := false
	for _, link := range links {
		if link.RequestID == request.ID && link.OriginType == origin.Type && link.OriginID == origin.ID && int64(link.OriginSequence) == origin.Version {
			linked = true
			break
		}
	}
	if !linked {
		return errors.New("vendor registration request is not linked to the assessment")
	}
	if assessment.ReviewKind != AssessmentReviewOnboarding || (assessment.Status != AssessmentSubmitted && assessment.Status != AssessmentUnderReview && assessment.Status != AssessmentCompleted) {
		return nil
	}
	relationship, err := p.assessments.GetRelationship(ctx, scope, assessment.RelationshipID)
	if err != nil {
		return err
	}
	matterContext := continuity.WithTrustedSystemEntityScope(ctx, scope.TenantID, scope.LegalEntityID)
	triggerKey := "thirdparty-address-verification:" + assessment.ID
	matter, err := p.matters.MatterByTriggerKey(matterContext, scope.TenantID, triggerKey)
	if errors.Is(err, continuity.ErrNotFound) {
		matter, err = p.matters.CreateMatter(matterContext, addressVerificationMatterInput(assessment, relationship, triggerKey))
	}
	if err != nil {
		return fmt.Errorf("create address verification issue: %w", err)
	}
	action, found := addressVerificationAction(matter.Actions)
	if !found {
		matter, err = p.matters.AddAction(matterContext, continuity.AddActionInput{
			TenantID: scope.TenantID, MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
			Title: "Verify the vendor registered address", Description: "Confirm the observed address and provide the source and evidence used.",
			OwnerPrincipalID: relationship.Relationship.BusinessOwnerPrincipalID, DueAt: &assessment.ReviewDueAt,
			ActorID: relationship.Relationship.BusinessOwnerPrincipalID, OriginKey: addressVerificationActionOrigin,
		})
		if err != nil {
			return fmt.Errorf("create address verification action: %w", err)
		}
		action, found = addressVerificationAction(matter.Actions)
	}
	if !found {
		return errors.New("address verification action is unavailable")
	}
	if action.Status == continuity.ActionPlanned {
		matter, err = p.matters.TransitionAction(matterContext, continuity.TransitionActionInput{
			TenantID: scope.TenantID, MatterID: matter.Matter.ID, ActionID: action.ID, ExpectedVersion: matter.Matter.Version,
			To: continuity.ActionInProgress, ActorID: action.OwnerPrincipalID,
		})
		if err != nil {
			return fmt.Errorf("start address verification action: %w", err)
		}
		action, _ = addressVerificationAction(matter.Actions)
	}
	if !addressVerificationContractExists(matter.VerificationContracts, action.ID) {
		matter, err = p.matters.AddVerificationContract(matterContext, continuity.AddVerificationContractInput{
			TenantID: scope.TenantID, MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, ActionID: action.ID,
			ExpectedOutcome: "The observed vendor address matches the current registered address.",
			Baseline:        json.RawMessage(`{"registered_address_source":"vendor relationship"}`),
			Scope:           json.RawMessage(`{"result":"address_result","evidence":"address_proof"}`),
			Threshold:       json.RawMessage(`{"address_result":"YES"}`), FailureResponse: "BLOCK_CLOSE",
			ActorID: relationship.Relationship.BusinessOwnerPrincipalID,
		})
		if err != nil {
			return fmt.Errorf("create address outcome check: %w", err)
		}
		action, _ = addressVerificationAction(matter.Actions)
	}
	form, err := p.activeAddressVerificationForm(ctx, scope)
	if err != nil {
		return err
	}
	requestOrigin := evidence.RequestOrigin{Type: AddressVerificationRequestOrigin, ID: action.ID, Version: action.Version}
	_, requestErr := p.evidence.GetRequestByOrigin(ctx, scope.TenantID, requestOrigin)
	if errors.Is(requestErr, evidence.ErrNotFound) {
		_, requestErr = p.evidence.CreateRequest(evidence.WithRequestOriginAuthority(ctx, AddressVerificationRequestOrigin), addressVerificationRequestInput(assessment, relationship, matter, action, form, requestOrigin))
	}
	if requestErr != nil {
		return fmt.Errorf("create address verification request: %w", requestErr)
	}
	_, err = p.inbox.RecordInbox(ctx, event.TenantID, addressVerificationConsumerName, event.ID, event.OccurredAt.UTC())
	return err
}

func (p *AddressVerificationProvisioner) activeAddressVerificationForm(ctx context.Context, scope Scope) (monitoring.FormTemplate, error) {
	forms, err := p.forms.ListReusableFormRevisions(ctx, scope.TenantID, scope.LegalEntityID, 200)
	if err != nil {
		return monitoring.FormTemplate{}, err
	}
	var selected monitoring.FormTemplate
	for _, form := range forms {
		if form.Code != addressVerificationFormCode || form.Status != monitoring.LifecycleActive || !form.IsCurrent {
			continue
		}
		if selected.ID != "" {
			return monitoring.FormTemplate{}, errors.New("more than one current address verification form is active")
		}
		selected = form
	}
	if selected.ID == "" {
		return monitoring.FormTemplate{}, errors.New("the current address verification form is unavailable")
	}
	return selected, nil
}

func addressVerificationMatterInput(assessment Assessment, relationship Aggregate, triggerKey string) continuity.CreateMatterInput {
	owner := relationship.Relationship.BusinessOwnerPrincipalID
	scope, _ := json.Marshal(map[string]any{
		"access": continuity.MatterAccessRestricted, "allowed_principal_ids": uniqueAssessmentPrincipals(owner, assessment.StartedByPrincipalID),
		"assessment_id": assessment.ID, "relationship_id": relationship.Relationship.ID, "vendor_id": relationship.Vendor.ID,
	})
	known, _ := json.Marshal(map[string]any{
		"vendor_legal_name": relationship.Vendor.LegalName, "registered_address": relationship.Vendor.RegisteredAddress,
		"service_name": relationship.Relationship.ServiceName,
	})
	return continuity.CreateMatterInput{
		TenantID: assessment.TenantID, LegalEntityID: assessment.LegalEntityID, Type: continuity.MatterVendorReview, Priority: 4,
		Title:   "Verify " + strings.TrimSpace(relationship.Vendor.LegalName) + " registered address",
		Summary: "Confirm the vendor registered address and independently review the evidence before relationship activation.",
		Scope:   scope, SourceType: "THIRD_PARTY_ASSESSMENT", SourceID: assessment.ID,
		TriggerType: "VENDOR_REGISTRATION_SUBMITTED", TriggerID: assessment.ID, TriggerKey: triggerKey,
		KnownFacts: known, MissingFacts: json.RawMessage(`["address verification result","address verification evidence"]`), Contradictions: json.RawMessage(`[]`),
		OwnerPrincipalID: owner, RequiredAuthority: "REVIEWER", DueAt: &assessment.ReviewDueAt, ActorID: owner,
	}
}

func addressVerificationRequestInput(assessment Assessment, relationship Aggregate, matter continuity.MatterAggregate, action continuity.Action, form monitoring.FormTemplate, origin evidence.RequestOrigin) evidence.CreateRequestInput {
	fields := make([]evidence.Field, len(form.Fields))
	for index, field := range form.Fields {
		fields[index] = evidence.Field{ID: field.ID, SectionID: field.SectionID, Label: field.Label, Type: string(field.Type), Required: field.Required, Description: field.Description, Options: append([]string(nil), field.Options...), AcceptedFormats: append([]string(nil), field.AcceptedFormats...), Attestation: field.Attestation, Constraints: field.Constraints, Condition: field.Condition, Scoring: field.Scoring}
	}
	return evidence.CreateRequestInput{
		TenantID: assessment.TenantID, LegalEntityID: assessment.LegalEntityID, SubjectType: "MATTER", SubjectID: matter.Matter.ID,
		Title: form.Name, Purpose: "Verify the vendor registered address before relationship activation.",
		WhyYou: "You are assigned to confirm the observed address and provide the evidence used.", Sensitivity: "CONFIDENTIAL", AudienceType: "INTERNAL",
		Recipient:        evidence.RecipientInput{Type: evidence.RecipientInternalPrincipal, PrincipalID: action.OwnerPrincipalID},
		EstimatedMinutes: estimateAssessmentMinutes(len(fields)), Deadline: assessment.ReviewDueAt,
		KnownFacts:   map[string]string{"Vendor": relationship.Vendor.LegalName, "Registered address": relationship.Vendor.RegisteredAddress, "Service": relationship.Relationship.ServiceName},
		Presentation: form.Presentation, ScoringMode: form.ScoringMode, ScoreProfile: form.ScoreProfile,
		Sections: form.Sections, Fields: fields, FormTemplateID: form.ID, FormTemplateVersion: form.Version,
		Origin: origin, CreatedBy: relationship.Relationship.BusinessOwnerPrincipalID,
	}
}

func addressVerificationAction(actions []continuity.Action) (continuity.Action, bool) {
	for _, action := range actions {
		if action.OriginKey == addressVerificationActionOrigin {
			return action, true
		}
	}
	return continuity.Action{}, false
}

func addressVerificationContractExists(contracts []continuity.VerificationContract, actionID string) bool {
	for _, contract := range contracts {
		if contract.ActionID == actionID && contract.Status == continuity.VerificationActive {
			return true
		}
	}
	return false
}

type AddressVerificationSubmissionConsumer struct {
	inbox    AssessmentSubmissionInbox
	requests AssessmentSubmissionRequestReader
	matters  addressVerificationMatterService
}

func NewAddressVerificationSubmissionConsumer(inbox AssessmentSubmissionInbox, requests AssessmentSubmissionRequestReader, matters addressVerificationMatterService) *AddressVerificationSubmissionConsumer {
	return &AddressVerificationSubmissionConsumer{inbox: inbox, requests: requests, matters: matters}
}

func (c *AddressVerificationSubmissionConsumer) Publish(ctx context.Context, event workflowruntime.OutboxEvent) error {
	if event.AggregateType != assessmentSubmissionAggregateType || event.EventType != assessmentSubmissionEventType {
		return nil
	}
	if c == nil || c.inbox == nil || c.requests == nil || c.matters == nil {
		return errors.New("address verification submission handling is not configured")
	}
	processed, err := c.inbox.InboxProcessed(ctx, event.TenantID, addressSubmissionConsumerName, event.ID)
	if err != nil || processed {
		return err
	}
	request, err := c.requests.GetRequest(ctx, event.TenantID, event.AggregateID)
	if err != nil {
		return err
	}
	if request.Origin.Type != AddressVerificationRequestOrigin {
		return nil
	}
	if request.SubjectType != "MATTER" || request.SubjectID == "" || request.Origin.ID == "" || request.Status != evidence.RequestSubmitted {
		return errors.New("address verification submission does not match a current request")
	}
	matterContext := continuity.WithTrustedSystemEntityScope(ctx, request.TenantID, request.LegalEntityID)
	matter, err := c.matters.GetMatter(matterContext, request.TenantID, request.SubjectID)
	if err != nil {
		return err
	}
	var action *continuity.Action
	for index := range matter.Actions {
		if matter.Actions[index].ID == request.Origin.ID {
			action = &matter.Actions[index]
			break
		}
	}
	if action == nil || action.OriginKey != addressVerificationActionOrigin {
		return errors.New("address verification action is unavailable")
	}
	if action.Status != continuity.ActionImplemented {
		if action.Status != continuity.ActionInProgress || action.Version != request.Origin.Version {
			return continuity.ErrVersionConflict
		}
		matter, err = c.matters.TransitionAction(matterContext, continuity.TransitionActionInput{
			TenantID: request.TenantID, MatterID: matter.Matter.ID, ActionID: action.ID, ExpectedVersion: matter.Matter.Version,
			To: continuity.ActionImplemented, ActorID: request.Recipient.PrincipalID,
		})
		if err != nil {
			return err
		}
	}
	_, err = c.inbox.RecordInbox(ctx, event.TenantID, addressSubmissionConsumerName, event.ID, event.OccurredAt.UTC())
	return err
}

var _ workflowruntime.Publisher = (*AddressVerificationProvisioner)(nil)
var _ workflowruntime.Publisher = (*AddressVerificationSubmissionConsumer)(nil)
