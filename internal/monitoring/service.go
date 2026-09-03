package monitoring

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

var (
	ErrMakerChecker                = errors.New("submitter cannot approve the same monitoring revision")
	ErrInactive                    = errors.New("monitoring revision is not active")
	ErrLinkedIssueIneligible       = errors.New("this monitoring result is not the latest adverse result configured to create a linked issue")
	ErrSourceValidationUnavailable = errors.New("connected-source validation is unavailable")
)

type Actor struct {
	TenantID      string
	LegalEntityID string
	PrincipalID   string
}

type CreateFormInput struct {
	ProgramID        string                     `json:"program_id"`
	LegalEntityID    string                     `json:"legal_entity_id"`
	Code             string                     `json:"code"`
	Name             string                     `json:"name"`
	Purpose          string                     `json:"purpose"`
	OwnerPrincipalID string                     `json:"owner_principal_id,omitempty"`
	ResponsibleTeam  string                     `json:"responsible_team,omitempty"`
	ApprovedUses     []string                   `json:"approved_uses,omitempty"`
	Tags             []string                   `json:"tags,omitempty"`
	Jurisdiction     string                     `json:"jurisdiction,omitempty"`
	Industry         string                     `json:"industry,omitempty"`
	Sensitivity      string                     `json:"sensitivity,omitempty"`
	ScoringMode      formcontract.ScoringMode   `json:"scoring_mode,omitempty"`
	ScoreProfile     *formcontract.ScoreProfile `json:"score_profile,omitempty"`
	NextReviewAt     *time.Time                 `json:"next_review_at,omitempty"`
	Presentation     formcontract.Presentation  `json:"presentation"`
	Sections         []formcontract.Section     `json:"sections"`
	Fields           []TemplateField            `json:"fields"`
}

type CreateFormRevisionInput struct {
	ExpectedVersion int64           `json:"expected_version"`
	Form            CreateFormInput `json:"form"`
}

type InstantiateStarterTemplateInput struct {
	Code             string     `json:"code,omitempty"`
	Name             string     `json:"name,omitempty"`
	Purpose          string     `json:"purpose,omitempty"`
	ProgramID        string     `json:"program_id,omitempty"`
	OwnerPrincipalID string     `json:"owner_principal_id,omitempty"`
	ResponsibleTeam  string     `json:"responsible_team,omitempty"`
	Jurisdiction     string     `json:"jurisdiction,omitempty"`
	Industry         string     `json:"industry,omitempty"`
	NextReviewAt     *time.Time `json:"next_review_at,omitempty"`
}

type TransitionInput struct {
	ID              string          `json:"id"`
	ProgramID       string          `json:"program_id,omitempty"`
	LegalEntityID   string          `json:"legal_entity_id,omitempty"`
	ExpectedVersion int64           `json:"expected_version"`
	To              LifecycleStatus `json:"to"`
}

type StartCollectionInput struct {
	FormTemplateID        string    `json:"form_template_id"`
	FormTemplateVersion   int64     `json:"form_template_version"`
	ProgramID             string    `json:"program_id"`
	LegalEntityID         string    `json:"legal_entity_id"`
	RespondentPrincipalID string    `json:"respondent_principal_id"`
	ReviewerPrincipalID   string    `json:"reviewer_principal_id"`
	PeriodStart           time.Time `json:"period_start"`
	PeriodEnd             time.Time `json:"period_end"`
	Deadline              time.Time `json:"deadline"`
}

type CreateCheckInput struct {
	ProgramID               string            `json:"program_id"`
	RequirementID           string            `json:"requirement_id,omitempty"`
	ControlImplementationID string            `json:"control_implementation_id,omitempty"`
	EvidenceContractID      string            `json:"evidence_contract_id,omitempty"`
	Code                    string            `json:"code"`
	Name                    string            `json:"name"`
	Claim                   string            `json:"claim"`
	InputKind               InputKind         `json:"input_kind"`
	FormTemplateID          string            `json:"form_template_id,omitempty"`
	FormTemplateVersion     int64             `json:"form_template_version,omitempty"`
	CollectionPolicy        *CollectionPolicy `json:"collection_policy,omitempty"`
	BindingID               string            `json:"binding_id,omitempty"`
	BindingVersion          int64             `json:"binding_version,omitempty"`
	SourceRules             []SourceRule      `json:"source_rules,omitempty"`
	Thresholds              Thresholds        `json:"thresholds"`
	FreshnessMinutes        int               `json:"freshness_minutes"`
	MinimumCoverage         float64           `json:"minimum_coverage"`
	OwnerPrincipalID        string            `json:"owner_principal_id,omitempty"`
	ReviewerPrincipalID     string            `json:"reviewer_principal_id,omitempty"`
	FailureAction           FailureAction     `json:"failure_action"`
}

type EvaluateSourceInput struct {
	CheckID      string `json:"check_id"`
	CheckVersion int64  `json:"check_version"`
}

type requestCreator interface {
	CreateRequest(context.Context, evidence.CreateRequestInput) (evidence.Request, error)
}

type evidenceReader interface {
	GetRequest(context.Context, string, string) (evidence.Request, error)
	GetSubmission(context.Context, string, string) (evidence.Submission, error)
}

type sourceReader interface {
	Binding(context.Context, string, string, int64) (sourceaccess.BindingRevision, error)
	PreviewBinding(context.Context, string, string, int64, sourceaccess.PageRequest) (sourceaccess.RecordPage, error)
}

type sourceScopeValidator interface {
	ValidateActiveSourcesForEntity(context.Context, string, string, []string) error
}

type Service struct {
	repo            Repository
	requests        requestCreator
	evidence        evidenceReader
	sources         sourceReader
	sourceValidator sourceScopeValidator
	now             func() time.Time
	newID           func() (string, error)
	commandGuard    *commandauth.Guard
}

func (s *Service) ConfigureCommandGuard(guard *commandauth.Guard) {
	s.commandGuard = guard
}

func NewService(repo Repository, requests requestCreator) *Service {
	service := &Service{repo: repo, requests: requests, now: time.Now, newID: id.NewUUIDv7}
	if reader, ok := requests.(evidenceReader); ok {
		service.evidence = reader
	}
	return service
}

func (s *Service) ConfigureEvidenceReader(reader evidenceReader) {
	s.evidence = reader
}

func (s *Service) ConfigureSourceReader(reader sourceReader) {
	s.sources = reader
}

func (s *Service) ConfigureSourceValidator(validator sourceScopeValidator) {
	s.sourceValidator = validator
}

func (s *Service) CreateLibraryForm(ctx context.Context, input CreateFormInput) (FormTemplate, error) {
	actor, err := s.requireFormActor(ctx)
	if err != nil {
		return FormTemplate{}, err
	}
	if err := s.authorizeFormCommand(ctx, actor, "LEGAL_ENTITY", actor.LegalEntityID, authority.ResponsibilityOwner, "forms.template.create", 2); err != nil {
		return FormTemplate{}, err
	}
	valueID, err := s.newID()
	if err != nil {
		return FormTemplate{}, err
	}
	return s.createLibraryRevision(ctx, actor, valueID, 1, input, FormTemplate{})
}

func (s *Service) CreateFormRevision(ctx context.Context, formID string, input CreateFormRevisionInput) (FormTemplate, error) {
	actor, err := s.requireFormActor(ctx)
	if err != nil {
		return FormTemplate{}, err
	}
	formID = strings.TrimSpace(formID)
	if formID == "" || input.ExpectedVersion < 1 {
		return FormTemplate{}, errors.Join(ErrInvalid, fmt.Errorf("form and expected version are required"))
	}
	base, err := s.repo.ReusableFormRevision(ctx, actor.TenantID, actor.LegalEntityID, formID, input.ExpectedVersion)
	if err != nil {
		return FormTemplate{}, err
	}
	if err := s.authorizeFormCommand(ctx, actor, "FORM_TEMPLATE", base.ID, authority.ResponsibilityOwner, "forms.template.revise", 2); err != nil {
		return FormTemplate{}, err
	}
	return s.createLibraryRevision(ctx, actor, base.ID, base.Version+1, input.Form, base)
}

func (s *Service) GetLibraryForm(ctx context.Context, formID string, version int64) (FormTemplate, error) {
	actor, err := s.requireFormActor(ctx)
	if err != nil {
		return FormTemplate{}, err
	}
	if strings.TrimSpace(formID) == "" || version < 1 {
		return FormTemplate{}, errors.Join(ErrInvalid, fmt.Errorf("form and version are required"))
	}
	return s.repo.ReusableFormRevision(ctx, actor.TenantID, actor.LegalEntityID, strings.TrimSpace(formID), version)
}

func (s *Service) ListFormLibrary(ctx context.Context, filter FormLibraryFilter) (FormTemplatePage, error) {
	actor, err := s.requireFormActor(ctx)
	if err != nil {
		return FormTemplatePage{}, err
	}
	filter.TenantID = actor.TenantID
	filter.LegalEntityID = actor.LegalEntityID
	return s.repo.ListFormLibrary(ctx, filter)
}

func (s *Service) ListStarterTemplates(ctx context.Context) ([]StarterTemplate, error) {
	if _, err := s.requireFormActor(ctx); err != nil {
		return nil, err
	}
	return s.repo.ListStarterTemplates(ctx)
}

func (s *Service) InstantiateStarterTemplate(ctx context.Context, starterCode string, input InstantiateStarterTemplateInput) (FormTemplate, error) {
	actor, err := s.requireFormActor(ctx)
	if err != nil {
		return FormTemplate{}, err
	}
	if err := s.authorizeFormCommand(ctx, actor, "LEGAL_ENTITY", actor.LegalEntityID, authority.ResponsibilityOwner, "forms.starter.instantiate", 2); err != nil {
		return FormTemplate{}, err
	}
	starter, err := s.repo.StarterTemplateByCode(ctx, starterCode)
	if err != nil {
		return FormTemplate{}, err
	}
	valueID, err := s.newID()
	if err != nil {
		return FormTemplate{}, err
	}
	value := cloneValue(starter.Template)
	value.ID = valueID
	value.TenantID = actor.TenantID
	value.LegalEntityID = actor.LegalEntityID
	value.ProgramID = strings.TrimSpace(input.ProgramID)
	value.OwnerPrincipalID = strings.TrimSpace(input.OwnerPrincipalID)
	if value.OwnerPrincipalID == "" {
		value.OwnerPrincipalID = actor.PrincipalID
	}
	value.ResponsibleTeam = strings.TrimSpace(input.ResponsibleTeam)
	value.Jurisdiction = strings.TrimSpace(input.Jurisdiction)
	value.Industry = strings.TrimSpace(input.Industry)
	value.NextReviewAt = input.NextReviewAt
	if strings.TrimSpace(input.Code) != "" {
		value.Code = strings.TrimSpace(input.Code)
	}
	if strings.TrimSpace(input.Name) != "" {
		value.Name = strings.TrimSpace(input.Name)
	}
	if strings.TrimSpace(input.Purpose) != "" {
		value.Purpose = strings.TrimSpace(input.Purpose)
	}
	if err := validateTextFields(value.Code, value.Name, value.Purpose); err != nil {
		return FormTemplate{}, err
	}
	now := s.now().UTC()
	value.Lifecycle = Lifecycle{Status: LifecycleDraft, Version: 1, CreatedBy: actor.PrincipalID, CreatedAt: now, UpdatedAt: now}
	return s.repo.CreateFormRevision(ctx, value)
}

func (s *Service) TransitionLibraryForm(ctx context.Context, formID string, input TransitionInput) (FormTemplate, error) {
	actor, err := s.requireFormActor(ctx)
	if err != nil {
		return FormTemplate{}, err
	}
	formID = strings.TrimSpace(formID)
	if formID == "" || input.ExpectedVersion < 1 {
		return FormTemplate{}, errors.Join(ErrInvalid, fmt.Errorf("form and expected version are required"))
	}
	current, err := s.repo.ReusableFormRevision(ctx, actor.TenantID, actor.LegalEntityID, formID, input.ExpectedVersion)
	if err != nil {
		return FormTemplate{}, err
	}
	responsibility := authority.ResponsibilityOwner
	if current.Status == LifecyclePendingApproval && (input.To == LifecycleActive || input.To == LifecycleRejected) {
		responsibility = authority.ResponsibilityReviewer
	}
	if err := s.authorizeFormCommand(ctx, actor, "FORM_TEMPLATE", current.ID, responsibility, "forms.template.transition", 3); err != nil {
		return FormTemplate{}, err
	}
	if input.To == LifecyclePendingApproval || input.To == LifecycleActive {
		if err := validateLibraryApprovalContract(current); err != nil {
			return FormTemplate{}, err
		}
	}
	if input.To == LifecycleActive && current.Status == LifecyclePendingApproval && current.SubmittedBy == actor.PrincipalID {
		return FormTemplate{}, ErrMakerChecker
	}
	// A reusable library form may intentionally have no Program binding. Transition the exact
	// entity-scoped revision already resolved above instead of re-reading it
	// through the Program form path, whose Program identifier is required.
	return s.repo.TransitionForm(ctx, LifecycleTransition{
		TenantID: actor.TenantID, LegalEntityID: current.LegalEntityID, ProgramID: current.ProgramID,
		ID: current.ID, ExpectedVersion: input.ExpectedVersion, To: input.To,
		ActorID: actor.PrincipalID, At: s.now().UTC(),
	})
}

func (s *Service) ListSavedFormViews(ctx context.Context) ([]SavedFormView, error) {
	actor, err := s.requireFormActor(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.ListSavedFormViews(ctx, actor.TenantID, actor.LegalEntityID, actor.PrincipalID)
}

func (s *Service) SaveFormView(ctx context.Context, view SavedFormView) (SavedFormView, error) {
	actor, err := s.requireFormActor(ctx)
	if err != nil {
		return SavedFormView{}, err
	}
	view.TenantID = actor.TenantID
	view.LegalEntityID = actor.LegalEntityID
	view.PrincipalID = actor.PrincipalID
	view.Filter.TenantID = ""
	view.Filter.LegalEntityID = ""
	view.Filter.Cursor = ""
	if strings.TrimSpace(view.ID) == "" {
		view.ID, err = s.newID()
		if err != nil {
			return SavedFormView{}, err
		}
	}
	now := s.now().UTC()
	if view.CreatedAt.IsZero() {
		view.CreatedAt = now
	}
	view.UpdatedAt = now
	return s.repo.SaveFormView(ctx, view)
}

func (s *Service) DeleteSavedFormView(ctx context.Context, viewID string) error {
	actor, err := s.requireFormActor(ctx)
	if err != nil {
		return err
	}
	return s.repo.DeleteSavedFormView(ctx, actor.TenantID, actor.LegalEntityID, actor.PrincipalID, strings.TrimSpace(viewID))
}

func (s *Service) createLibraryRevision(ctx context.Context, actor identity.Actor, id string, version int64, input CreateFormInput, base FormTemplate) (FormTemplate, error) {
	contract, err := normalizeLibraryDraft(input)
	if err != nil {
		return FormTemplate{}, err
	}
	if err := validateTextFields(input.Code, input.Name, input.Purpose); err != nil {
		return FormTemplate{}, err
	}
	ownerID := strings.TrimSpace(input.OwnerPrincipalID)
	if ownerID == "" {
		ownerID = actor.PrincipalID
	}
	now := s.now().UTC()
	value := FormTemplate{
		ID: id, TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ProgramID: strings.TrimSpace(base.ProgramID),
		Code: strings.TrimSpace(input.Code), Name: strings.TrimSpace(input.Name), Purpose: strings.TrimSpace(input.Purpose),
		OwnerPrincipalID: ownerID, ResponsibleTeam: strings.TrimSpace(input.ResponsibleTeam), ApprovedUses: append([]string(nil), input.ApprovedUses...), Tags: append([]string(nil), input.Tags...),
		Jurisdiction: strings.TrimSpace(input.Jurisdiction), Industry: strings.TrimSpace(input.Industry), Sensitivity: strings.TrimSpace(input.Sensitivity), ScoringMode: contract.ScoringMode, ScoreProfile: contract.ScoreProfile, NextReviewAt: input.NextReviewAt,
		StarterCatalogCode: base.StarterCatalogCode, StarterCatalogVersion: base.StarterCatalogVersion,
		Presentation: contract.Presentation, Sections: contract.Sections, Fields: contract.Fields,
		Lifecycle: Lifecycle{Status: LifecycleDraft, Version: version, CreatedBy: actor.PrincipalID, CreatedAt: now, UpdatedAt: now},
	}
	if base.ID == "" {
		value.ProgramID = strings.TrimSpace(input.ProgramID)
	}
	return s.repo.CreateFormRevision(ctx, value)
}

func (s *Service) requireFormActor(ctx context.Context) (identity.Actor, error) {
	actor, err := identity.Require(ctx)
	if err != nil {
		return identity.Actor{}, err
	}
	if err := actor.Valid(s.now().UTC()); err != nil {
		return identity.Actor{}, err
	}
	if actor.LegalEntityID == "*" {
		return identity.Actor{}, identity.ErrInvalidIdentity
	}
	return actor, nil
}

func (s *Service) authorizeFormCommand(ctx context.Context, actor identity.Actor, objectType, objectID string, responsibility authority.Responsibility, decisionType string, materiality int) error {
	if s.commandGuard == nil {
		return commandauth.ErrGuardUnavailable
	}
	_, err := s.commandGuard.Authorize(ctx, commandauth.Request{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ObjectType: objectType, ObjectID: objectID,
		Responsibility: responsibility, DecisionType: decisionType, Materiality: materiality,
	})
	return err
}

func (s *Service) CreateForm(ctx context.Context, actor Actor, input CreateFormInput) (FormTemplate, error) {
	if err := validateActor(actor); err != nil {
		return FormTemplate{}, err
	}
	if err := validateProgramScope(actor, input.ProgramID, input.LegalEntityID); err != nil {
		return FormTemplate{}, err
	}
	contract, err := formcontract.Normalize(formcontract.Contract{Presentation: input.Presentation, ScoringMode: input.ScoringMode, ScoreProfile: input.ScoreProfile, Sections: input.Sections, Fields: input.Fields})
	if err != nil {
		return FormTemplate{}, errors.Join(ErrInvalid, err)
	}
	if err := validateTextFields(input.Code, input.Name, input.Purpose); err != nil {
		return FormTemplate{}, err
	}
	valueID, err := s.newID()
	if err != nil {
		return FormTemplate{}, err
	}
	now := s.now().UTC()
	return s.repo.CreateFormRevision(ctx, FormTemplate{
		ID: valueID, TenantID: actor.TenantID, LegalEntityID: strings.TrimSpace(input.LegalEntityID), ProgramID: strings.TrimSpace(input.ProgramID), Code: strings.TrimSpace(input.Code), Name: strings.TrimSpace(input.Name),
		Purpose: strings.TrimSpace(input.Purpose), Presentation: contract.Presentation, ScoringMode: contract.ScoringMode, ScoreProfile: contract.ScoreProfile, Sections: contract.Sections, Fields: contract.Fields,
		Lifecycle: Lifecycle{Status: LifecycleDraft, Version: 1, CreatedBy: actor.PrincipalID, CreatedAt: now, UpdatedAt: now},
	})
}

func (s *Service) ListForms(ctx context.Context, actor Actor, programID string, limit int) ([]FormTemplate, error) {
	if err := validateActor(actor); err != nil {
		return nil, err
	}
	if err := validateProgramScope(actor, programID, actor.LegalEntityID); err != nil {
		return nil, err
	}
	return s.repo.ListFormRevisions(ctx, actor.TenantID, actor.LegalEntityID, strings.TrimSpace(programID), limit)
}

func (s *Service) ListReusableForms(ctx context.Context, actor Actor, limit int) ([]FormTemplate, error) {
	if err := validateActor(actor); err != nil {
		return nil, err
	}
	repo, ok := s.repo.(ReusableFormRepository)
	if !ok {
		return nil, errors.Join(ErrInvalid, fmt.Errorf("reusable form lookup is unavailable"))
	}
	return repo.ListReusableFormRevisions(ctx, actor.TenantID, actor.LegalEntityID, limit)
}

func (s *Service) Form(ctx context.Context, actor Actor, programID, id string, version int64) (FormTemplate, error) {
	if err := validateActor(actor); err != nil {
		return FormTemplate{}, err
	}
	if err := validateProgramScope(actor, programID, actor.LegalEntityID); err != nil {
		return FormTemplate{}, err
	}
	if strings.TrimSpace(id) == "" || version < 1 {
		return FormTemplate{}, errors.Join(ErrInvalid, fmt.Errorf("form and version are required"))
	}
	return s.repo.FormRevision(ctx, actor.TenantID, actor.LegalEntityID, strings.TrimSpace(programID), strings.TrimSpace(id), version)
}

func (s *Service) TransitionForm(ctx context.Context, actor Actor, input TransitionInput) (FormTemplate, error) {
	if err := validateActor(actor); err != nil {
		return FormTemplate{}, err
	}
	current, err := s.repo.FormRevision(ctx, actor.TenantID, strings.TrimSpace(input.LegalEntityID), strings.TrimSpace(input.ProgramID), strings.TrimSpace(input.ID), input.ExpectedVersion)
	if err != nil {
		return FormTemplate{}, err
	}
	if err := validateStoredFormScope(actor, current, input.ProgramID, input.LegalEntityID); err != nil {
		return FormTemplate{}, err
	}
	if input.To == LifecycleActive && current.Status == LifecyclePendingApproval && current.SubmittedBy == actor.PrincipalID {
		return FormTemplate{}, ErrMakerChecker
	}
	return s.repo.TransitionForm(ctx, LifecycleTransition{TenantID: actor.TenantID, LegalEntityID: current.LegalEntityID, ProgramID: current.ProgramID, ID: current.ID, ExpectedVersion: input.ExpectedVersion, To: input.To, ActorID: actor.PrincipalID, At: s.now().UTC()})
}

func (s *Service) StartCollection(ctx context.Context, actor Actor, input StartCollectionInput) (evidence.Request, error) {
	if err := validateActor(actor); err != nil {
		return evidence.Request{}, err
	}
	if s.requests == nil {
		return evidence.Request{}, fmt.Errorf("evidence request service is unavailable")
	}
	if strings.TrimSpace(input.FormTemplateID) == "" || input.FormTemplateVersion < 1 || strings.TrimSpace(input.ProgramID) == "" || strings.TrimSpace(input.RespondentPrincipalID) == "" || strings.TrimSpace(input.ReviewerPrincipalID) == "" {
		return evidence.Request{}, errors.Join(ErrInvalid, fmt.Errorf("form, program, respondent and reviewer are required"))
	}
	if input.PeriodStart.IsZero() || input.PeriodEnd.IsZero() || input.PeriodStart.After(input.PeriodEnd) || input.Deadline.IsZero() {
		return evidence.Request{}, errors.Join(ErrInvalid, fmt.Errorf("a valid reporting period and deadline are required"))
	}
	form, err := s.repo.FormRevision(ctx, actor.TenantID, input.LegalEntityID, input.ProgramID, input.FormTemplateID, input.FormTemplateVersion)
	if err != nil {
		return evidence.Request{}, err
	}
	if err := validateStoredFormScope(actor, form, input.ProgramID, input.LegalEntityID); err != nil {
		return evidence.Request{}, err
	}
	if form.Status != LifecycleActive || !form.IsCurrent {
		return evidence.Request{}, ErrInactive
	}
	checks, err := s.repo.ListCheckRevisions(ctx, actor.TenantID, input.ProgramID, 500)
	if err != nil {
		return evidence.Request{}, err
	}
	linked := false
	for _, check := range checks {
		if check.Status == LifecycleActive && check.IsCurrent && check.InputKind == InputForm && check.FormTemplateID == form.ID && check.FormTemplateVersion == form.Version {
			linked = true
			break
		}
	}
	if !linked {
		return evidence.Request{}, ErrInactive
	}
	fields := make([]evidence.Field, len(form.Fields))
	for index, field := range form.Fields {
		fields[index] = evidence.Field{ID: field.ID, SectionID: field.SectionID, Label: field.Label, Type: string(field.Type), Required: field.Required, Description: field.Description, Options: append([]string(nil), field.Options...), AcceptedFormats: append([]string(nil), field.AcceptedFormats...), Attestation: field.Attestation, Constraints: field.Constraints, Condition: field.Condition, Scoring: field.Scoring}
	}
	periodStart := input.PeriodStart.UTC()
	periodEnd := input.PeriodEnd.UTC()
	return s.requests.CreateRequest(ctx, evidence.CreateRequestInput{
		TenantID: actor.TenantID, SubjectType: "PROGRAM", SubjectID: input.ProgramID, Title: form.Name,
		Purpose: form.Purpose, WhyYou: "You are responsible for completing this control review.",
		Sensitivity: "INTERNAL", AudienceType: "INTERNAL",
		Recipient:        evidence.RecipientInput{Type: evidence.RecipientInternalPrincipal, PrincipalID: input.RespondentPrincipalID},
		EstimatedMinutes: estimateMinutes(len(fields)), Deadline: input.Deadline.UTC(),
		KnownFacts:   map[string]string{"reviewer": input.ReviewerPrincipalID, "legal_entity_id": input.LegalEntityID, "reporting_period_start": periodStart.Format(time.RFC3339), "reporting_period_end": periodEnd.Format(time.RFC3339)},
		Presentation: form.Presentation, ScoringMode: form.ScoringMode, ScoreProfile: form.ScoreProfile,
		Sections: form.Sections, Fields: fields, FormTemplateID: form.ID, FormTemplateVersion: form.Version,
		CollectionPeriodStart: &periodStart, CollectionPeriodEnd: &periodEnd, CreatedBy: actor.PrincipalID,
	})
}

func (s *Service) CreateCheck(ctx context.Context, actor Actor, input CreateCheckInput) (MonitoringCheck, error) {
	if err := validateActor(actor); err != nil {
		return MonitoringCheck{}, err
	}
	if err := validateTextFields(input.Code, input.Name, input.Claim); err != nil || strings.TrimSpace(input.ProgramID) == "" {
		if err == nil {
			err = fmt.Errorf("program is required")
		}
		return MonitoringCheck{}, errors.Join(ErrInvalid, err)
	}
	if input.Thresholds == (Thresholds{}) {
		input.Thresholds = DefaultThresholds()
	}
	if err := validateThresholds(input.Thresholds); err != nil {
		return MonitoringCheck{}, errors.Join(ErrInvalid, err)
	}
	if input.FreshnessMinutes < 1 || input.FreshnessMinutes > 525600 || input.MinimumCoverage < 0 || input.MinimumCoverage > 1 {
		return MonitoringCheck{}, errors.Join(ErrInvalid, fmt.Errorf("freshness and minimum coverage are invalid"))
	}
	if input.FailureAction == "" {
		input.FailureAction = FailureReview
	}
	switch input.FailureAction {
	case FailureReview, FailureRecommendMatter:
	default:
		return MonitoringCheck{}, errors.Join(ErrInvalid, fmt.Errorf("failure action is invalid"))
	}
	switch input.InputKind {
	case InputForm:
		if input.FormTemplateID == "" || input.FormTemplateVersion < 1 || input.BindingID != "" || len(input.SourceRules) != 0 {
			return MonitoringCheck{}, errors.Join(ErrInvalid, fmt.Errorf("form checks require exactly one form revision"))
		}
		if strings.TrimSpace(actor.LegalEntityID) == "" {
			return MonitoringCheck{}, errors.Join(ErrInvalid, fmt.Errorf("verified legal-entity scope is required for a form check"))
		}
		form, err := s.repo.FormRevision(ctx, actor.TenantID, actor.LegalEntityID, input.ProgramID, input.FormTemplateID, input.FormTemplateVersion)
		if err != nil {
			return MonitoringCheck{}, err
		}
		if form.Status != LifecycleActive || !form.IsCurrent {
			return MonitoringCheck{}, ErrInactive
		}
		policy, err := normalizeCollectionPolicy(input.CollectionPolicy)
		if err != nil {
			return MonitoringCheck{}, err
		}
		input.CollectionPolicy = &policy
	case InputSource:
		if input.BindingID == "" || input.BindingVersion < 1 || input.FormTemplateID != "" || len(input.SourceRules) == 0 || input.CollectionPolicy != nil {
			return MonitoringCheck{}, errors.Join(ErrInvalid, fmt.Errorf("source checks require one binding revision and at least one rule"))
		}
		for _, rule := range input.SourceRules {
			if err := validateSourceRule(rule); err != nil {
				return MonitoringCheck{}, errors.Join(ErrInvalid, err)
			}
		}
		if _, err := s.validateSourceBinding(ctx, actor, input.BindingID, input.BindingVersion); err != nil {
			return MonitoringCheck{}, err
		}
	default:
		return MonitoringCheck{}, errors.Join(ErrInvalid, fmt.Errorf("input kind is invalid"))
	}
	valueID, err := s.newID()
	if err != nil {
		return MonitoringCheck{}, err
	}
	now := s.now().UTC()
	return s.repo.CreateCheckRevision(ctx, MonitoringCheck{
		ID: valueID, TenantID: actor.TenantID, ProgramID: strings.TrimSpace(input.ProgramID), RequirementID: strings.TrimSpace(input.RequirementID),
		ControlImplementationID: strings.TrimSpace(input.ControlImplementationID), EvidenceContractID: strings.TrimSpace(input.EvidenceContractID),
		Code: strings.TrimSpace(input.Code), Name: strings.TrimSpace(input.Name), Claim: strings.TrimSpace(input.Claim), InputKind: input.InputKind,
		FormTemplateID: strings.TrimSpace(input.FormTemplateID), FormTemplateVersion: input.FormTemplateVersion,
		CollectionPolicy: input.CollectionPolicy,
		BindingID:        strings.TrimSpace(input.BindingID), BindingVersion: input.BindingVersion, SourceRules: append([]SourceRule(nil), input.SourceRules...),
		Thresholds: input.Thresholds, FreshnessMinutes: input.FreshnessMinutes, MinimumCoverage: input.MinimumCoverage,
		OwnerPrincipalID: strings.TrimSpace(input.OwnerPrincipalID), ReviewerPrincipalID: strings.TrimSpace(input.ReviewerPrincipalID), FailureAction: input.FailureAction,
		Lifecycle: Lifecycle{Status: LifecycleDraft, Version: 1, CreatedBy: actor.PrincipalID, CreatedAt: now, UpdatedAt: now},
	})
}

func (s *Service) ListChecks(ctx context.Context, actor Actor, programID string, limit int) ([]MonitoringCheck, error) {
	if err := validateActor(actor); err != nil {
		return nil, err
	}
	if strings.TrimSpace(programID) == "" {
		return nil, errors.Join(ErrInvalid, fmt.Errorf("program is required"))
	}
	return s.repo.ListCheckRevisions(ctx, actor.TenantID, programID, limit)
}

func (s *Service) Check(ctx context.Context, actor Actor, checkID string, version int64) (MonitoringCheck, error) {
	if err := validateActor(actor); err != nil {
		return MonitoringCheck{}, err
	}
	if strings.TrimSpace(checkID) == "" || version < 1 {
		return MonitoringCheck{}, errors.Join(ErrInvalid, fmt.Errorf("monitoring check and version are required"))
	}
	return s.repo.CheckRevision(ctx, actor.TenantID, strings.TrimSpace(checkID), version)
}

func (s *Service) LatestCheck(ctx context.Context, actor Actor, checkID string) (MonitoringCheck, error) {
	if err := validateActor(actor); err != nil {
		return MonitoringCheck{}, err
	}
	if strings.TrimSpace(checkID) == "" {
		return MonitoringCheck{}, errors.Join(ErrInvalid, fmt.Errorf("monitoring check is required"))
	}
	return s.repo.LatestCheckRevision(ctx, actor.TenantID, strings.TrimSpace(checkID))
}

func (s *Service) ListResults(ctx context.Context, actor Actor, checkID string, limit int) ([]MonitoringResult, error) {
	if err := validateActor(actor); err != nil {
		return nil, err
	}
	if strings.TrimSpace(checkID) == "" {
		return nil, errors.Join(ErrInvalid, fmt.Errorf("monitoring check is required"))
	}
	return s.repo.ListResults(ctx, actor.TenantID, checkID, limit)
}

func (s *Service) Result(ctx context.Context, actor Actor, resultID string) (MonitoringResult, error) {
	if err := validateActor(actor); err != nil {
		return MonitoringResult{}, err
	}
	if strings.TrimSpace(resultID) == "" {
		return MonitoringResult{}, errors.Join(ErrInvalid, fmt.Errorf("monitoring result is required"))
	}
	return s.repo.Result(ctx, actor.TenantID, strings.TrimSpace(resultID))
}

func EligibleForLinkedIssue(check MonitoringCheck, result MonitoringResult) bool {
	if check.Status != LifecycleActive || !check.IsCurrent || check.FailureAction != FailureRecommendMatter {
		return false
	}
	if result.TenantID != check.TenantID || result.ProgramID != check.ProgramID || result.MonitoringCheckID != check.ID || result.MonitoringCheckVersion != check.Version {
		return false
	}
	if result.Evaluation.Band == RiskHigh || result.Evaluation.Band == RiskCritical || result.Evaluation.Coverage < check.MinimumCoverage || len(result.Evaluation.CriticalFailures) > 0 {
		return true
	}
	for _, rule := range result.Evaluation.RuleResults {
		if rule.Critical && rule.Outcome != RulePassed {
			return true
		}
	}
	return false
}

func (s *Service) EvaluateSource(ctx context.Context, actor Actor, input EvaluateSourceInput) (MonitoringResult, error) {
	if err := validateActor(actor); err != nil {
		return MonitoringResult{}, err
	}
	if strings.TrimSpace(input.CheckID) == "" || input.CheckVersion < 1 {
		return MonitoringResult{}, errors.Join(ErrInvalid, fmt.Errorf("monitoring check and version are required"))
	}
	check, err := s.repo.CheckRevision(ctx, actor.TenantID, input.CheckID, input.CheckVersion)
	if err != nil {
		return MonitoringResult{}, err
	}
	if check.Status != LifecycleActive || !check.IsCurrent || check.InputKind != InputSource {
		return MonitoringResult{}, ErrInactive
	}
	binding, err := s.validateSourceBinding(ctx, actor, check.BindingID, check.BindingVersion)
	if err != nil {
		return MonitoringResult{}, err
	}
	page, err := s.sources.PreviewBinding(ctx, actor.TenantID, check.BindingID, check.BindingVersion, sourceaccess.PageRequest{Limit: 2})
	if err != nil {
		return MonitoringResult{}, err
	}
	now := s.now().UTC()
	state := evidence.SourceResolutionCurrent
	if page.NextCursor != nil || page.Receipt.Completeness != sourceaccess.CompletenessComplete {
		state = evidence.SourceResolutionPartial
	} else if check.FreshnessMinutes > 0 && (page.Receipt.ObservedAt.IsZero() || now.Sub(page.Receipt.ObservedAt) > time.Duration(check.FreshnessMinutes)*time.Minute) {
		state = evidence.SourceResolutionStale
	}
	resolution := evidence.SourceResolution{BindingID: binding.BindingID, BindingVersion: binding.Version, BindingName: binding.Name, SourceID: binding.SourceID, State: state, Records: page.Records, Receipt: &page.Receipt}
	evaluation, err := EvaluateSource(check.SourceRules, resolution, check.Thresholds, now)
	if err != nil {
		return MonitoringResult{}, err
	}
	receipt, err := json.Marshal(page.Receipt)
	if err != nil {
		return MonitoringResult{}, err
	}
	resultID, err := s.newID()
	if err != nil {
		return MonitoringResult{}, err
	}
	inputReference := check.BindingID + ":" + page.Receipt.ObservedAt.UTC().Format(time.RFC3339Nano)
	return s.repo.AppendResult(ctx, MonitoringResult{
		ID: resultID, TenantID: actor.TenantID, ProgramID: check.ProgramID, MonitoringCheckID: check.ID, MonitoringCheckVersion: check.Version,
		InputKind: InputSource, InputReferenceID: inputReference, InputReferenceVersion: check.BindingVersion,
		Evaluation: evaluation, SourceReceipt: receipt, EvaluatedAt: now, EvaluatorVersion: "monitoring-risk-v1", CreatedAt: now,
	})
}

func (s *Service) EvaluateSubmission(ctx context.Context, tenantID, submissionID string) ([]MonitoringResult, error) {
	if strings.TrimSpace(tenantID) == "" || strings.TrimSpace(submissionID) == "" {
		return nil, errors.Join(ErrInvalid, fmt.Errorf("tenant and submission are required"))
	}
	if s.evidence == nil {
		return nil, fmt.Errorf("evidence submission reads are unavailable")
	}
	submission, err := s.evidence.GetSubmission(ctx, tenantID, submissionID)
	if err != nil {
		return nil, err
	}
	request, err := s.evidence.GetRequest(ctx, tenantID, submission.RequestID)
	if err != nil {
		return nil, err
	}
	if request.SubjectType != "PROGRAM" || request.SubjectID == "" || request.FormTemplateID == "" || request.FormTemplateVersion < 1 {
		return nil, errors.Join(ErrInvalid, fmt.Errorf("submission is not linked to a Program form"))
	}
	checks, err := s.repo.ListCheckRevisions(ctx, tenantID, request.SubjectID, 500)
	if err != nil {
		return nil, err
	}
	results := make([]MonitoringResult, 0)
	for _, check := range checks {
		if check.Status != LifecycleActive || !check.IsCurrent || check.InputKind != InputForm || check.FormTemplateID != request.FormTemplateID || check.FormTemplateVersion != request.FormTemplateVersion {
			continue
		}
		result, evaluateErr := s.evaluateFormSubmission(ctx, check, request, submission)
		if evaluateErr != nil {
			return results, evaluateErr
		}
		results = append(results, result)
	}
	return results, nil
}

func (s *Service) evaluateFormSubmission(ctx context.Context, check MonitoringCheck, request evidence.Request, submission evidence.Submission) (MonitoringResult, error) {
	legalEntityID := strings.TrimSpace(request.KnownFacts["legal_entity_id"])
	if legalEntityID == "" {
		return MonitoringResult{}, errors.Join(ErrInvalid, fmt.Errorf("form submission legal-entity scope is missing"))
	}
	form, err := s.repo.FormRevision(ctx, check.TenantID, legalEntityID, check.ProgramID, check.FormTemplateID, check.FormTemplateVersion)
	if err != nil {
		return MonitoringResult{}, err
	}
	fields := make([]FormField, 0, len(form.Fields))
	for _, field := range form.Fields {
		if field.Scoring != nil {
			fields = append(fields, *field.Scoring)
		}
	}
	evaluation, err := EvaluateForm(fields, submission.Answers, check.Thresholds)
	if err != nil {
		return MonitoringResult{}, err
	}
	provenance, err := json.Marshal(map[string]any{
		"request_id": request.ID, "channel": submission.Channel, "submitted_by": submission.SubmittedBy,
		"submitted_at": submission.SubmittedAt.UTC(), "answer_provenance": submission.AnswerProvenance,
	})
	if err != nil {
		return MonitoringResult{}, err
	}
	resultID, err := s.newID()
	if err != nil {
		return MonitoringResult{}, err
	}
	now := s.now().UTC()
	return s.repo.AppendResult(ctx, MonitoringResult{
		ID: resultID, TenantID: check.TenantID, ProgramID: check.ProgramID, MonitoringCheckID: check.ID, MonitoringCheckVersion: check.Version,
		InputKind: InputForm, InputReferenceID: submission.ID, InputReferenceVersion: 1, Evaluation: evaluation,
		SubmissionProvenance: provenance, EvaluatedAt: now, EvaluatorVersion: "monitoring-risk-v1", CreatedAt: now,
	})
}

func (s *Service) TransitionCheck(ctx context.Context, actor Actor, input TransitionInput) (MonitoringCheck, error) {
	if err := validateActor(actor); err != nil {
		return MonitoringCheck{}, err
	}
	current, err := s.repo.CheckRevision(ctx, actor.TenantID, strings.TrimSpace(input.ID), input.ExpectedVersion)
	if err != nil {
		return MonitoringCheck{}, err
	}
	if input.To == LifecycleActive && current.Status == LifecyclePendingApproval && current.SubmittedBy == actor.PrincipalID {
		return MonitoringCheck{}, ErrMakerChecker
	}
	if input.To == LifecycleActive && current.InputKind == InputForm {
		if strings.TrimSpace(actor.LegalEntityID) == "" {
			return MonitoringCheck{}, errors.Join(ErrInvalid, fmt.Errorf("verified legal-entity scope is required for a form check"))
		}
		form, loadErr := s.repo.FormRevision(ctx, actor.TenantID, actor.LegalEntityID, current.ProgramID, current.FormTemplateID, current.FormTemplateVersion)
		if loadErr != nil {
			return MonitoringCheck{}, loadErr
		}
		if form.Status != LifecycleActive || !form.IsCurrent {
			return MonitoringCheck{}, ErrInactive
		}
	}
	if input.To == LifecycleActive && current.InputKind == InputSource {
		if _, err := s.validateSourceBinding(ctx, actor, current.BindingID, current.BindingVersion); err != nil {
			return MonitoringCheck{}, err
		}
	}
	return s.repo.TransitionCheck(ctx, LifecycleTransition{TenantID: actor.TenantID, ID: current.ID, ExpectedVersion: input.ExpectedVersion, To: input.To, ActorID: actor.PrincipalID, At: s.now().UTC()})
}

func (s *Service) validateSourceBinding(ctx context.Context, actor Actor, bindingID string, bindingVersion int64) (sourceaccess.BindingRevision, error) {
	tenantID := strings.TrimSpace(actor.TenantID)
	legalEntityID := strings.TrimSpace(actor.LegalEntityID)
	bindingID = strings.TrimSpace(bindingID)
	if legalEntityID == "" || legalEntityID == "*" {
		return sourceaccess.BindingRevision{}, errors.Join(ErrInvalid, fmt.Errorf("verified legal-entity scope is required for a connected-source check"))
	}
	if s.sources == nil {
		return sourceaccess.BindingRevision{}, fmt.Errorf("%w: connected-source reads are unavailable", ErrSourceValidationUnavailable)
	}
	if s.sourceValidator == nil {
		return sourceaccess.BindingRevision{}, fmt.Errorf("%w: connected-source legal-entity validation is unavailable", ErrSourceValidationUnavailable)
	}
	binding, err := s.sources.Binding(ctx, tenantID, bindingID, bindingVersion)
	if err != nil {
		if errors.Is(err, sourceaccess.ErrCatalogNotFound) || errors.Is(err, sourceaccess.ErrCatalogInvalid) {
			return sourceaccess.BindingRevision{}, errors.Join(ErrInvalid, fmt.Errorf("connected-source revision is missing or invalid: %w", err))
		}
		return sourceaccess.BindingRevision{}, fmt.Errorf("%w: connected-source revision could not be resolved: %v", ErrSourceValidationUnavailable, err)
	}
	if binding.BindingID != bindingID || binding.Version != bindingVersion || binding.TenantID != tenantID || strings.TrimSpace(binding.SourceID) == "" {
		return sourceaccess.BindingRevision{}, errors.Join(ErrInvalid, fmt.Errorf("connected-source revision does not match the monitoring check"))
	}
	now := s.now().UTC()
	if binding.Status != sourceaccess.RevisionActive || !binding.IsCurrent || binding.EffectiveFrom == nil || now.Before(binding.EffectiveFrom.UTC()) || (binding.EffectiveUntil != nil && !now.Before(binding.EffectiveUntil.UTC())) {
		return sourceaccess.BindingRevision{}, errors.Join(ErrInactive, fmt.Errorf("connected-source revision is not current and effective"))
	}
	allowsPage := false
	for _, operation := range binding.Operations {
		if operation == sourceaccess.OperationPage {
			allowsPage = true
			break
		}
	}
	if !allowsPage {
		return sourceaccess.BindingRevision{}, errors.Join(ErrInvalid, fmt.Errorf("connected-source revision does not allow page reads"))
	}
	if err := s.sourceValidator.ValidateActiveSourcesForEntity(ctx, tenantID, legalEntityID, []string{binding.SourceID}); err != nil {
		if errors.Is(err, evidence.ErrSourceScopeMismatch) || errors.Is(err, evidence.ErrSourceScopeRequired) {
			return sourceaccess.BindingRevision{}, errors.Join(ErrInvalid, fmt.Errorf("connected source is not active in the selected legal entity: %w", err))
		}
		return sourceaccess.BindingRevision{}, fmt.Errorf("%w: connected-source legal-entity validation failed: %v", ErrSourceValidationUnavailable, err)
	}
	return binding, nil
}

func validateActor(actor Actor) error {
	if strings.TrimSpace(actor.TenantID) == "" || strings.TrimSpace(actor.PrincipalID) == "" {
		return errors.Join(ErrInvalid, fmt.Errorf("verified tenant and principal are required"))
	}
	return nil
}

func validateProgramScope(actor Actor, programID, legalEntityID string) error {
	programID = strings.TrimSpace(programID)
	legalEntityID = strings.TrimSpace(legalEntityID)
	if programID == "" || legalEntityID == "" || strings.TrimSpace(actor.LegalEntityID) == "" {
		return errors.Join(ErrInvalid, fmt.Errorf("verified Program and legal-entity scope are required"))
	}
	if actor.LegalEntityID != legalEntityID {
		return ErrNotFound
	}
	return nil
}

func validateStoredFormScope(actor Actor, form FormTemplate, programID, legalEntityID string) error {
	if strings.TrimSpace(form.ProgramID) == "" && strings.TrimSpace(programID) == "" {
		legalEntityID = strings.TrimSpace(legalEntityID)
		if legalEntityID == "" || strings.TrimSpace(actor.LegalEntityID) == "" {
			return errors.Join(ErrInvalid, fmt.Errorf("verified legal-entity scope is required"))
		}
		if actor.LegalEntityID != legalEntityID || form.TenantID != actor.TenantID || form.LegalEntityID != legalEntityID {
			return ErrNotFound
		}
		return nil
	}
	if err := validateProgramScope(actor, programID, legalEntityID); err != nil {
		return err
	}
	if form.TenantID != actor.TenantID || form.ProgramID != strings.TrimSpace(programID) || form.LegalEntityID != strings.TrimSpace(legalEntityID) {
		return ErrNotFound
	}
	return nil
}

func validateTextFields(code, name, purpose string) error {
	if strings.TrimSpace(code) == "" || strings.TrimSpace(name) == "" || strings.TrimSpace(purpose) == "" {
		return errors.Join(ErrInvalid, fmt.Errorf("code, name and purpose are required"))
	}
	return nil
}

func normalizeTemplateFields(fields []TemplateField) ([]TemplateField, error) {
	contract, err := formcontract.Normalize(formcontract.Contract{Fields: fields})
	if err != nil {
		return nil, errors.Join(ErrInvalid, err)
	}
	return contract.Fields, nil
}

func validateSourceRule(rule SourceRule) error {
	if strings.TrimSpace(rule.ID) == "" || strings.TrimSpace(rule.Field) == "" || rule.RiskPoints < 0 || rule.RiskPoints > 100 {
		return fmt.Errorf("source rule id, field and risk points are required")
	}
	switch rule.Operator {
	case OperatorEquals, OperatorNotEquals, OperatorGreaterThan, OperatorGreaterOrEqual, OperatorLessThan, OperatorLessOrEqual, OperatorPresent, OperatorMaxAgeMinutes:
		return nil
	default:
		return fmt.Errorf("source rule operator is invalid")
	}
}

func estimateMinutes(fieldCount int) int {
	minutes := fieldCount
	if minutes < 2 {
		return 2
	}
	if minutes > 60 {
		return 60
	}
	return minutes
}
