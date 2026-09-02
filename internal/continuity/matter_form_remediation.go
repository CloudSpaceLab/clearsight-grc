package continuity

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

const (
	MatterFormRemediationOrigin = "MATTER_FORM_REMEDIATION"
	MatterFormBindingActive     = "ACTIVE"
)

var (
	ErrMatterFormBindingInvalid   = errors.New("matter form remediation binding is invalid")
	ErrMatterFormResponseRejected = errors.New("matter form response cannot be applied")
)

type MatterFormFieldMapping struct {
	FieldID     string `json:"field_id"`
	MissingItem string `json:"missing_item"`
	FactKey     string `json:"fact_key"`
}

type MatterFormRemediationBinding struct {
	ID                     string                   `json:"id"`
	TenantID               string                   `json:"tenant_id"`
	LegalEntityID          string                   `json:"legal_entity_id"`
	ProgramID              string                   `json:"program_id"`
	MatterID               string                   `json:"matter_id"`
	SubjectType            string                   `json:"subject_type"`
	SubjectID              string                   `json:"subject_id"`
	MatterVersionAtBinding int64                    `json:"matter_version_at_binding"`
	FormTemplateID         string                   `json:"form_template_id"`
	FormTemplateVersion    int64                    `json:"form_template_version"`
	Mappings               []MatterFormFieldMapping `json:"mappings"`
	ActionID               string                   `json:"action_id,omitempty"`
	VerificationContractID string                   `json:"verification_contract_id"`
	MinimumScore           *float64                 `json:"minimum_score,omitempty"`
	MaximumAdverseScore    *float64                 `json:"maximum_adverse_score,omitempty"`
	Purpose                string                   `json:"purpose"`
	AudienceClass          string                   `json:"audience_class"`
	ResponderClass         string                   `json:"responder_class"`
	Status                 string                   `json:"status"`
	EffectiveFrom          time.Time                `json:"effective_from"`
	CreatedBy              string                   `json:"created_by"`
	CreatedAt              time.Time                `json:"created_at"`
	Version                int64                    `json:"version"`
}

type CreateMatterFormBindingInput struct {
	LegalEntityID          string                   `json:"legal_entity_id,omitempty"`
	ExpectedMatterVersion  int64                    `json:"expected_matter_version"`
	ProgramID              string                   `json:"program_id"`
	FormTemplateID         string                   `json:"form_template_id"`
	FormTemplateVersion    int64                    `json:"form_template_version"`
	Mappings               []MatterFormFieldMapping `json:"mappings"`
	ActionID               string                   `json:"action_id,omitempty"`
	VerificationContractID string                   `json:"verification_contract_id"`
	MinimumScore           *float64                 `json:"minimum_score,omitempty"`
	MaximumAdverseScore    *float64                 `json:"maximum_adverse_score,omitempty"`
}

type SendMatterFormInput struct {
	BindingVersion int64                               `json:"binding_version"`
	Recipient      evidence.DistributionRecipientInput `json:"recipient"`
	Deadline       time.Time                           `json:"deadline"`
	RouteExpiresAt time.Time                           `json:"route_expires_at"`
}

type MatterFormApplication struct {
	ID                     string    `json:"id"`
	TenantID               string    `json:"tenant_id"`
	LegalEntityID          string    `json:"legal_entity_id"`
	BindingID              string    `json:"binding_id"`
	BindingVersion         int64     `json:"binding_version"`
	MatterID               string    `json:"matter_id"`
	MatterVersion          int64     `json:"matter_version"`
	DistributionID         string    `json:"distribution_id"`
	ResponseRevisionID     string    `json:"response_revision_id"`
	ResponseRevision       int64     `json:"response_revision"`
	SubmissionID           string    `json:"submission_id"`
	VerificationContractID string    `json:"verification_contract_id"`
	AppliedFieldIDs        []string  `json:"applied_field_ids"`
	AppliedBy              string    `json:"applied_by"`
	AppliedAt              time.Time `json:"applied_at"`
}

type matterFormResponseAppliedEvent struct {
	Matter      Matter                `json:"matter"`
	Application MatterFormApplication `json:"application"`
	Rationale   string                `json:"rationale"`
}

type ApplyMatterFormResponseInput struct {
	BindingVersion        int64  `json:"binding_version"`
	ExpectedMatterVersion int64  `json:"expected_matter_version"`
	ResponseRevisionID    string `json:"response_revision_id"`
	Rationale             string `json:"rationale"`
}

type MatterFormRemediationState struct {
	Binding      MatterFormRemediationBinding       `json:"binding"`
	Request      *evidence.Request                  `json:"request,omitempty"`
	Distribution *evidence.FormDistribution         `json:"distribution,omitempty"`
	Response     *evidence.CompletedResponseSummary `json:"response,omitempty"`
	Application  *MatterFormApplication             `json:"application,omitempty"`
	NextAction   string                             `json:"next_action"`
}

type MatterFormRemediationRepository interface {
	CreateMatterFormBinding(context.Context, MatterFormRemediationBinding) (MatterFormRemediationBinding, error)
	GetMatterFormBinding(context.Context, string, string, string) (MatterFormRemediationBinding, error)
	ListMatterFormBindings(context.Context, string, string, int) ([]MatterFormRemediationBinding, error)
	GetMatterFormApplication(context.Context, string, string, string) (MatterFormApplication, error)
	ApplyMatterFormApplication(context.Context, MatterFormApplicationCommand) (MatterAggregate, MatterFormApplication, error)
}

type MatterFormApplicationCommand struct {
	Binding               MatterFormRemediationBinding
	Aggregate             MatterAggregate
	ExpectedMatterVersion int64
	DistributionID        string
	ResponseRevisionID    string
	ResponseRevision      int64
	SubmissionID          string
	Answers               map[string]formcontract.AnswerValue
	ActorID               string
	Rationale             string
	AppliedAt             time.Time
	ApplicationID         string
	EventID               string
}

type matterFormReader interface {
	GetLibraryForm(context.Context, string, int64) (monitoring.FormTemplate, error)
}

type matterFormRequestReader interface {
	GetRequestByOrigin(context.Context, string, evidence.RequestOrigin) (evidence.Request, error)
	GetSubmission(context.Context, string, string) (evidence.Submission, error)
	GetArtifact(context.Context, string, string, string) (evidence.Artifact, error)
}

type MatterFormRemediationService struct {
	repo          MatterFormRemediationRepository
	matters       *Service
	forms         matterFormReader
	requests      matterFormRequestReader
	distributions *evidence.DistributionService
	dispatcher    *evidence.WorkflowDistributionDispatcher
	guard         *commandauth.Guard
	now           func() time.Time
}

func NewMatterFormRemediationService(repo MatterFormRemediationRepository, matters *Service, forms matterFormReader, requests matterFormRequestReader, distributions *evidence.DistributionService, dispatcher *evidence.WorkflowDistributionDispatcher, guard *commandauth.Guard) *MatterFormRemediationService {
	return &MatterFormRemediationService{repo: repo, matters: matters, forms: forms, requests: requests, distributions: distributions, dispatcher: dispatcher, guard: guard, now: time.Now}
}

func (s *MatterFormRemediationService) CreateBinding(ctx context.Context, matterID string, input CreateMatterFormBindingInput) (MatterFormRemediationBinding, error) {
	actor, matter, form, err := s.bindingContext(ctx, matterID, input.ExpectedMatterVersion, input.FormTemplateID, input.FormTemplateVersion, authority.ResponsibilityOwner, "matter.form-remediation.bind", 3)
	if err != nil {
		return MatterFormRemediationBinding{}, err
	}
	if strings.TrimSpace(input.LegalEntityID) != "" && strings.TrimSpace(input.LegalEntityID) != matter.Matter.LegalEntityID {
		return MatterFormRemediationBinding{}, ErrNotFound
	}
	if !matterLinkedToProgram(matter, input.ProgramID) || matter.Matter.Status == MatterClosed || matter.Matter.Status == MatterCancelled {
		return MatterFormRemediationBinding{}, ErrMatterFormBindingInvalid
	}
	if form.Status != monitoring.LifecycleActive || !form.IsCurrent || !containsFoldString(form.ApprovedUses, "MATTER_REMEDIATION") {
		return MatterFormRemediationBinding{}, ErrMatterFormBindingInvalid
	}
	if err := validateMatterFormMappings(matter, form, input); err != nil {
		return MatterFormRemediationBinding{}, err
	}
	bindingID, err := id.NewUUIDv7()
	if err != nil {
		return MatterFormRemediationBinding{}, err
	}
	now := s.currentTime()
	binding := MatterFormRemediationBinding{
		ID: bindingID, TenantID: actor.TenantID, LegalEntityID: matter.Matter.LegalEntityID, ProgramID: strings.TrimSpace(input.ProgramID), MatterID: matter.Matter.ID,
		SubjectType: "MATTER", SubjectID: matter.Matter.ID,
		MatterVersionAtBinding: matter.Matter.Version, FormTemplateID: form.ID, FormTemplateVersion: form.Version,
		Mappings: normalizeMatterFormMappings(input.Mappings), ActionID: strings.TrimSpace(input.ActionID), VerificationContractID: strings.TrimSpace(input.VerificationContractID),
		MinimumScore: cloneFloat(input.MinimumScore), MaximumAdverseScore: cloneFloat(input.MaximumAdverseScore),
		Purpose: "Supply the mapped information for independent review of this issue.", AudienceClass: "EXTERNAL", ResponderClass: "ISSUE_EVIDENCE_CONTACT",
		Status: MatterFormBindingActive, EffectiveFrom: now, CreatedBy: actor.PrincipalID, CreatedAt: now, Version: 1,
	}
	return s.repo.CreateMatterFormBinding(ctx, binding)
}

func (s *MatterFormRemediationService) List(ctx context.Context, matterID string, limit int) ([]MatterFormRemediationState, error) {
	actor, err := identity.Require(ctx)
	if err != nil {
		return nil, err
	}
	if limit < 1 || limit > 100 {
		return nil, ErrMatterFormBindingInvalid
	}
	matter, err := s.matters.GetMatter(ctx, actor.TenantID, strings.TrimSpace(matterID))
	if err != nil {
		return nil, err
	}
	if matter.Matter.LegalEntityID != actor.LegalEntityID || !MatterAggregateVisibleTo(matter, actor.PrincipalID) {
		return nil, ErrNotFound
	}
	bindings, err := s.repo.ListMatterFormBindings(ctx, actor.TenantID, matter.Matter.ID, limit)
	if err != nil {
		return nil, err
	}
	states := make([]MatterFormRemediationState, 0, len(bindings))
	for _, binding := range bindings {
		request, requestErr := s.requests.GetRequestByOrigin(ctx, actor.TenantID, evidence.RequestOrigin{Type: MatterFormRemediationOrigin, ID: binding.ID, Version: binding.Version})
		if errors.Is(requestErr, evidence.ErrNotFound) {
			states = append(states, MatterFormRemediationState{Binding: binding, NextAction: "Send form"})
			continue
		}
		if requestErr != nil {
			return nil, requestErr
		}
		state, stateErr := s.stateForRequest(ctx, actor.PrincipalID, binding, request)
		if stateErr != nil {
			return nil, stateErr
		}
		states = append(states, state)
	}
	return states, nil
}

func (s *MatterFormRemediationService) Send(ctx context.Context, matterID, bindingID string, input SendMatterFormInput) (MatterFormRemediationState, error) {
	actor, err := identity.Require(ctx)
	if err != nil {
		return MatterFormRemediationState{}, err
	}
	binding, err := s.repo.GetMatterFormBinding(ctx, actor.TenantID, strings.TrimSpace(matterID), strings.TrimSpace(bindingID))
	if err != nil {
		return MatterFormRemediationState{}, err
	}
	if binding.Version != input.BindingVersion {
		return MatterFormRemediationState{}, ErrVersionConflict
	}
	if binding.Status != MatterFormBindingActive || binding.EffectiveFrom.After(s.currentTime()) || binding.SubjectType != "MATTER" || binding.SubjectID != binding.MatterID || binding.AudienceClass != "EXTERNAL" || input.Recipient.Type != evidence.RecipientExternalAudience {
		return MatterFormRemediationState{}, ErrMatterFormBindingInvalid
	}
	matter, form, err := s.currentBindingDependencies(ctx, actor, binding, authority.ResponsibilityOwner, "matter.form-remediation.send", 3)
	if err != nil {
		return MatterFormRemediationState{}, err
	}
	origin := evidence.RequestOrigin{Type: MatterFormRemediationOrigin, ID: binding.ID, Version: binding.Version}
	if existing, readErr := s.requests.GetRequestByOrigin(ctx, actor.TenantID, origin); readErr == nil {
		return s.stateForRequest(ctx, actor.PrincipalID, binding, existing)
	} else if !errors.Is(readErr, evidence.ErrNotFound) {
		return MatterFormRemediationState{}, readErr
	}
	if !input.Deadline.After(s.currentTime()) || input.RouteExpiresAt.After(input.Deadline) || !input.RouteExpiresAt.After(s.currentTime()) {
		return MatterFormRemediationState{}, ErrMatterFormBindingInvalid
	}
	recipient := input.Recipient
	if recipient.Role == "" {
		recipient.Role = evidence.RecipientTo
	}
	requestInput := matterRemediationRequestInput(binding, matter, form, recipient, actor.PrincipalID, input.Deadline)
	workflowCtx := evidence.WithRequestOriginAuthority(ctx, MatterFormRemediationOrigin)
	var request evidence.Request
	if recipient.Type == evidence.RecipientExternalAudience {
		if s.dispatcher == nil {
			return MatterFormRemediationState{}, evidence.ErrDistributionAccessUnavailable
		}
		result, dispatchErr := s.dispatcher.Dispatch(workflowCtx, evidence.WorkflowDistributionDispatchInput{Request: requestInput, AccessPolicy: evidence.AccessDirectEmailOTP, RouteExpiresAt: input.RouteExpiresAt, AudienceHint: recipient.AudienceHint})
		if dispatchErr != nil {
			return MatterFormRemediationState{}, dispatchErr
		}
		request = result.Request
	} else {
		if s.distributions == nil {
			return MatterFormRemediationState{}, evidence.ErrDistributionInvalid
		}
		bundle, createErr := s.distributions.Create(workflowCtx, evidence.CreateDistributionInput{
			TenantID: binding.TenantID, LegalEntityID: binding.LegalEntityID, FormTemplateID: binding.FormTemplateID, FormTemplateVersion: binding.FormTemplateVersion,
			SubjectType: "MATTER", SubjectID: binding.MatterID, Title: requestInput.Title, Purpose: requestInput.Purpose, AccessPolicy: evidence.AccessDirectMagicLink,
			EstimatedMinutes: requestInput.EstimatedMinutes, Deadline: input.Deadline, RouteExpiresAt: input.RouteExpiresAt, CreatedBy: actor.PrincipalID,
			Recipients: []evidence.DistributionRecipientInput{recipient}, RequestInput: &requestInput,
		})
		if createErr != nil {
			return MatterFormRemediationState{}, createErr
		}
		for _, candidate := range bundle.Recipients {
			if candidate.Role == evidence.RecipientTo {
				request, err = s.requests.GetRequestByOrigin(ctx, actor.TenantID, origin)
				break
			}
		}
		if err != nil {
			return MatterFormRemediationState{}, err
		}
	}
	return s.stateForRequest(ctx, actor.PrincipalID, binding, request)
}

func (s *MatterFormRemediationService) Apply(ctx context.Context, matterID, bindingID string, input ApplyMatterFormResponseInput) (MatterAggregate, MatterFormApplication, error) {
	actor, err := identity.Require(ctx)
	if err != nil {
		return MatterAggregate{}, MatterFormApplication{}, err
	}
	if len(strings.TrimSpace(input.Rationale)) < 20 {
		return MatterAggregate{}, MatterFormApplication{}, ErrMatterFormResponseRejected
	}
	binding, err := s.repo.GetMatterFormBinding(ctx, actor.TenantID, strings.TrimSpace(matterID), strings.TrimSpace(bindingID))
	if err != nil {
		return MatterAggregate{}, MatterFormApplication{}, err
	}
	if binding.Version != input.BindingVersion {
		return MatterAggregate{}, MatterFormApplication{}, ErrVersionConflict
	}
	matter, _, err := s.currentBindingDependencies(ctx, actor, binding, authority.ResponsibilityReviewer, "matter.form-remediation.apply", 4)
	if err != nil {
		return MatterAggregate{}, MatterFormApplication{}, err
	}
	if existing, existingErr := s.repo.GetMatterFormApplication(ctx, actor.TenantID, binding.ID, strings.TrimSpace(input.ResponseRevisionID)); existingErr == nil {
		return matter, existing, nil
	} else if !errors.Is(existingErr, ErrNotFound) {
		return MatterAggregate{}, MatterFormApplication{}, existingErr
	}
	if matter.Matter.Version != input.ExpectedMatterVersion {
		return MatterAggregate{}, MatterFormApplication{}, ErrVersionConflict
	}
	if s.distributions == nil {
		return MatterAggregate{}, MatterFormApplication{}, evidence.ErrDistributionInvalid
	}
	summary, revision, err := s.distributions.GetCompletedResponse(ctx, actor.TenantID, actor.LegalEntityID, actor.PrincipalID, strings.TrimSpace(input.ResponseRevisionID))
	if err != nil {
		return MatterAggregate{}, MatterFormApplication{}, err
	}
	if summary.SubjectType != "MATTER" || summary.SubjectID != binding.MatterID || summary.FormTemplateID != binding.FormTemplateID || summary.FormTemplateVersion != binding.FormTemplateVersion || revision.State != evidence.ResponseRevisionFinal || !revision.Current {
		return MatterAggregate{}, MatterFormApplication{}, ErrMatterFormResponseRejected
	}
	if !scoreAcceptable(binding, revision.Score) {
		return MatterAggregate{}, MatterFormApplication{}, ErrMatterFormResponseRejected
	}
	submission, err := s.requests.GetSubmission(ctx, actor.TenantID, revision.SubmissionID)
	if err != nil {
		return MatterAggregate{}, MatterFormApplication{}, err
	}
	request, err := s.requests.GetRequestByOrigin(ctx, actor.TenantID, evidence.RequestOrigin{Type: MatterFormRemediationOrigin, ID: binding.ID, Version: binding.Version})
	if err != nil {
		return MatterAggregate{}, MatterFormApplication{}, err
	}
	if submission.ID != revision.SubmissionID || submission.RequestID != request.ID {
		return MatterAggregate{}, MatterFormApplication{}, ErrMatterFormResponseRejected
	}
	for _, mapping := range binding.Mappings {
		if answer, ok := submission.Answers[mapping.FieldID]; !ok || !answer.Answered() {
			return MatterAggregate{}, MatterFormApplication{}, ErrMatterFormResponseRejected
		}
	}
	if err := mappedMatterFormArtifactsAvailable(ctx, s.requests, request, binding, submission.Answers); err != nil {
		return MatterAggregate{}, MatterFormApplication{}, err
	}
	applicationID, err := id.NewUUIDv7()
	if err != nil {
		return MatterAggregate{}, MatterFormApplication{}, err
	}
	eventID, err := id.NewUUIDv7()
	if err != nil {
		return MatterAggregate{}, MatterFormApplication{}, err
	}
	return s.repo.ApplyMatterFormApplication(ctx, MatterFormApplicationCommand{Binding: binding, Aggregate: matter, ExpectedMatterVersion: matter.Matter.Version, DistributionID: summary.DistributionID, ResponseRevisionID: revision.ID, ResponseRevision: revision.Revision, SubmissionID: revision.SubmissionID, Answers: submission.Answers, ActorID: actor.PrincipalID, Rationale: strings.TrimSpace(input.Rationale), AppliedAt: s.currentTime(), ApplicationID: applicationID, EventID: eventID})
}

func (s *MatterFormRemediationService) bindingContext(ctx context.Context, matterID string, expected int64, formID string, formVersion int64, responsibility authority.Responsibility, decision string, materiality int) (identity.Actor, MatterAggregate, monitoring.FormTemplate, error) {
	actor, err := identity.Require(ctx)
	if err != nil {
		return identity.Actor{}, MatterAggregate{}, monitoring.FormTemplate{}, err
	}
	if s == nil || s.repo == nil || s.matters == nil || s.forms == nil || s.requests == nil || s.guard == nil {
		return identity.Actor{}, MatterAggregate{}, monitoring.FormTemplate{}, commandauth.ErrGuardUnavailable
	}
	matter, err := s.matters.GetMatter(ctx, actor.TenantID, strings.TrimSpace(matterID))
	if err != nil {
		return identity.Actor{}, MatterAggregate{}, monitoring.FormTemplate{}, err
	}
	if matter.Matter.Version != expected {
		return identity.Actor{}, MatterAggregate{}, monitoring.FormTemplate{}, ErrVersionConflict
	}
	if actor.LegalEntityID != matter.Matter.LegalEntityID || !MatterAggregateVisibleTo(matter, actor.PrincipalID) {
		return identity.Actor{}, MatterAggregate{}, monitoring.FormTemplate{}, ErrNotFound
	}
	if _, err = s.guard.Authorize(ctx, commandauth.Request{TenantID: actor.TenantID, LegalEntityID: matter.Matter.LegalEntityID, ObjectType: "MATTER", ObjectID: matter.Matter.ID, Responsibility: responsibility, DecisionType: decision, Materiality: materiality}); err != nil {
		return identity.Actor{}, MatterAggregate{}, monitoring.FormTemplate{}, err
	}
	form, err := s.forms.GetLibraryForm(ctx, strings.TrimSpace(formID), formVersion)
	if err != nil {
		return identity.Actor{}, MatterAggregate{}, monitoring.FormTemplate{}, err
	}
	return actor, matter, form, nil
}

func (s *MatterFormRemediationService) currentBindingDependencies(ctx context.Context, actor identity.Actor, binding MatterFormRemediationBinding, responsibility authority.Responsibility, decision string, materiality int) (MatterAggregate, monitoring.FormTemplate, error) {
	matter, err := s.matters.GetMatter(ctx, actor.TenantID, binding.MatterID)
	if err != nil {
		return MatterAggregate{}, monitoring.FormTemplate{}, err
	}
	if matter.Matter.LegalEntityID != binding.LegalEntityID || matter.Matter.Status == MatterClosed || matter.Matter.Status == MatterCancelled || !matterLinkedToProgram(matter, binding.ProgramID) || binding.Status != MatterFormBindingActive || binding.EffectiveFrom.After(s.currentTime()) || binding.SubjectType != "MATTER" || binding.SubjectID != binding.MatterID {
		return MatterAggregate{}, monitoring.FormTemplate{}, ErrMatterFormBindingInvalid
	}
	if _, err := s.guard.Authorize(ctx, commandauth.Request{TenantID: actor.TenantID, LegalEntityID: binding.LegalEntityID, ObjectType: "MATTER", ObjectID: binding.MatterID, Responsibility: responsibility, DecisionType: decision, Materiality: materiality}); err != nil {
		return MatterAggregate{}, monitoring.FormTemplate{}, err
	}
	form, err := s.forms.GetLibraryForm(ctx, binding.FormTemplateID, binding.FormTemplateVersion)
	if err != nil {
		return MatterAggregate{}, monitoring.FormTemplate{}, err
	}
	if form.Status != monitoring.LifecycleActive {
		return MatterAggregate{}, monitoring.FormTemplate{}, ErrMatterFormBindingInvalid
	}
	return matter, form, nil
}

func (s *MatterFormRemediationService) stateForRequest(ctx context.Context, principalID string, binding MatterFormRemediationBinding, request evidence.Request) (MatterFormRemediationState, error) {
	state := MatterFormRemediationState{Binding: binding, Request: &request, NextAction: "Open response"}
	if s.distributions != nil {
		if bundle, err := s.distributions.GetForRequest(ctx, binding.TenantID, binding.LegalEntityID, request.ID); err == nil {
			state.Distribution = &bundle.Distribution
			page, pageErr := s.distributions.ListCompletedResponses(ctx, evidence.CompletedResponseQuery{TenantID: binding.TenantID, LegalEntityID: binding.LegalEntityID, PrincipalID: principalID, SubjectType: "MATTER", SubjectID: binding.MatterID, FormTemplateID: binding.FormTemplateID, FormTemplateVersion: binding.FormTemplateVersion, CurrentOnly: true, Sort: evidence.ResponseSortNewest, Limit: 1})
			if pageErr == nil && len(page.Items) > 0 {
				state.Response = &page.Items[0]
				if scoreAcceptable(binding, page.Items[0].Score) {
					state.NextAction = "Review evidence"
				} else {
					state.NextAction = "Request correction"
				}
			}
		}
	}
	if application, err := s.repo.GetMatterFormApplication(ctx, binding.TenantID, binding.ID, ""); err == nil {
		state.Application = &application
		state.NextAction = "Check outcome"
	}
	return state, nil
}

func validateMatterFormMappings(matter MatterAggregate, form monitoring.FormTemplate, input CreateMatterFormBindingInput) error {
	if len(input.Mappings) == 0 || strings.TrimSpace(input.VerificationContractID) == "" || input.MinimumScore != nil && (*input.MinimumScore < 0 || *input.MinimumScore > 100) || input.MaximumAdverseScore != nil && (*input.MaximumAdverseScore < 0 || *input.MaximumAdverseScore > 100) {
		return ErrMatterFormBindingInvalid
	}
	fields := map[string]struct{}{}
	for _, field := range form.Fields {
		fields[field.ID] = struct{}{}
	}
	missing := map[string]struct{}{}
	var missingItems []string
	if err := json.Unmarshal(matter.Matter.MissingFacts, &missingItems); err != nil {
		return ErrMatterFormBindingInvalid
	}
	for _, item := range missingItems {
		missing[strings.ToLower(strings.TrimSpace(item))] = struct{}{}
	}
	seenFields, seenMissing, seenFacts := map[string]struct{}{}, map[string]struct{}{}, map[string]struct{}{}
	for _, mapping := range normalizeMatterFormMappings(input.Mappings) {
		if _, ok := fields[mapping.FieldID]; !ok || mapping.FactKey == "" {
			return ErrMatterFormBindingInvalid
		}
		if _, ok := missing[strings.ToLower(mapping.MissingItem)]; !ok {
			return ErrMatterFormBindingInvalid
		}
		if _, ok := seenFields[mapping.FieldID]; ok {
			return ErrMatterFormBindingInvalid
		}
		seenFields[mapping.FieldID] = struct{}{}
		key := strings.ToLower(mapping.MissingItem)
		if _, ok := seenMissing[key]; ok {
			return ErrMatterFormBindingInvalid
		}
		seenMissing[key] = struct{}{}
		if _, ok := seenFacts[mapping.FactKey]; ok {
			return ErrMatterFormBindingInvalid
		}
		seenFacts[mapping.FactKey] = struct{}{}
	}
	if input.ActionID != "" {
		found := false
		for _, action := range matter.Actions {
			if action.ID == input.ActionID && action.Status != ActionCancelled {
				found = true
			}
		}
		if !found {
			return ErrMatterFormBindingInvalid
		}
	}
	foundContract := false
	for _, contract := range matter.VerificationContracts {
		if contract.ID == input.VerificationContractID && contract.Status == VerificationActive && (input.ActionID == "" || contract.ActionID == input.ActionID) {
			foundContract = true
		}
	}
	if !foundContract {
		return ErrMatterFormBindingInvalid
	}
	return nil
}

func normalizeMatterFormMappings(values []MatterFormFieldMapping) []MatterFormFieldMapping {
	result := make([]MatterFormFieldMapping, len(values))
	for index, value := range values {
		result[index] = MatterFormFieldMapping{FieldID: strings.TrimSpace(value.FieldID), MissingItem: strings.TrimSpace(value.MissingItem), FactKey: strings.ToLower(strings.TrimSpace(value.FactKey))}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].FieldID < result[j].FieldID })
	return result
}
func containsFoldString(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), target) {
			return true
		}
	}
	return false
}
func cloneFloat(value *float64) *float64 {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}
func scoreAcceptable(binding MatterFormRemediationBinding, score *evidence.ResponseScoreResult) bool {
	if binding.MinimumScore == nil && binding.MaximumAdverseScore == nil {
		return true
	}
	if score == nil || score.State != evidence.ResponseScoreFinal || !score.Final {
		return false
	}
	if binding.MinimumScore != nil && (score.RawScore == nil || *score.RawScore < *binding.MinimumScore) {
		return false
	}
	return binding.MaximumAdverseScore == nil || score.AdverseScore != nil && *score.AdverseScore <= *binding.MaximumAdverseScore
}

func mappedMatterFormArtifactsAvailable(ctx context.Context, requests matterFormRequestReader, request evidence.Request, binding MatterFormRemediationBinding, answers map[string]formcontract.AnswerValue) error {
	for _, mapping := range binding.Mappings {
		answer := answers[mapping.FieldID]
		artifactIDs := append([]string(nil), answer.ArtifactIDs...)
		if answer.Document != nil && strings.TrimSpace(answer.Document.ArtifactID) != "" {
			artifactIDs = append(artifactIDs, answer.Document.ArtifactID)
		}
		for _, artifactID := range artifactIDs {
			artifact, err := requests.GetArtifact(ctx, request.TenantID, request.ID, strings.TrimSpace(artifactID))
			if err != nil || artifact.Status != evidence.ArtifactAvailable {
				return ErrMatterFormResponseRejected
			}
		}
	}
	return nil
}
func (s *MatterFormRemediationService) currentTime() time.Time {
	if s.now == nil {
		return time.Now().UTC()
	}
	return s.now().UTC()
}

func matterRemediationRequestInput(binding MatterFormRemediationBinding, matter MatterAggregate, form monitoring.FormTemplate, recipient evidence.DistributionRecipientInput, actorID string, deadline time.Time) evidence.CreateRequestInput {
	fields := make([]evidence.Field, len(form.Fields))
	for index, field := range form.Fields {
		fields[index] = evidence.Field{ID: field.ID, SectionID: field.SectionID, Label: field.Label, Type: string(field.Type), Required: field.Required, Description: field.Description, Options: append([]string(nil), field.Options...), AcceptedFormats: append([]string(nil), field.AcceptedFormats...), Attestation: field.Attestation, Constraints: field.Constraints, Condition: field.Condition, Scoring: field.Scoring, CollectionIntent: field.CollectionIntent, RecordTarget: field.RecordTarget, BrowserCachePolicy: field.BrowserCachePolicy}
	}
	audienceType := "INTERNAL"
	recipientInput := evidence.RecipientInput{Type: recipient.Type, PrincipalID: recipient.PrincipalID}
	if recipient.Type == evidence.RecipientExternalAudience {
		audienceType = "EXTERNAL"
		recipientInput.Audience = recipient.Address
	}
	return evidence.CreateRequestInput{TenantID: binding.TenantID, LegalEntityID: binding.LegalEntityID, SubjectType: "MATTER", SubjectID: binding.MatterID, Title: form.Name, Purpose: "Provide the information mapped to issue " + matter.Matter.Reference + ".", WhyYou: "Your response supplies the selected missing information for this issue and will be reviewed before any outcome is confirmed.", Sensitivity: form.Sensitivity, AudienceType: audienceType, Recipient: recipientInput, EstimatedMinutes: max(5, len(fields)*2), Deadline: deadline.UTC(), KnownFacts: map[string]string{"Issue reference": matter.Matter.Reference, "Issue title": matter.Matter.Title}, Presentation: form.Presentation, ScoringMode: form.ScoringMode, ScoreProfile: form.ScoreProfile, Sections: form.Sections, Fields: fields, FormTemplateID: form.ID, FormTemplateVersion: form.Version, Origin: evidence.RequestOrigin{Type: MatterFormRemediationOrigin, ID: binding.ID, Version: binding.Version}, CreatedBy: actorID}
}

func applyMatterFormAnswers(matter Matter, binding MatterFormRemediationBinding, answers map[string]formcontract.AnswerValue, responseRevisionID string) (Matter, []string, error) {
	facts, err := decodeRawObject(matter.KnownFacts)
	if err != nil {
		return Matter{}, nil, err
	}
	missing, err := decodeStringList(matter.MissingFacts)
	if err != nil {
		return Matter{}, nil, err
	}
	applied := make([]string, 0, len(binding.Mappings))
	for _, mapping := range binding.Mappings {
		answer, ok := answers[mapping.FieldID]
		if !ok || !answer.Answered() {
			return Matter{}, nil, ErrMatterFormResponseRejected
		}
		if _, exists := facts[mapping.FactKey]; exists {
			return Matter{}, nil, ErrMatterFormResponseRejected
		}
		var found bool
		missing, found = removeLabel(missing, mapping.MissingItem)
		if !found {
			return Matter{}, nil, ErrMatterFormResponseRejected
		}
		factReference, _ := json.Marshal(map[string]any{"supplied": true, "response_revision_id": responseRevisionID, "field_id": mapping.FieldID})
		facts[mapping.FactKey] = factReference
		applied = append(applied, mapping.FieldID)
	}
	matter.KnownFacts, _ = json.Marshal(facts)
	matter.MissingFacts, _ = json.Marshal(missing)
	return matter, applied, nil
}
