package monitoring

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

var (
	ErrMakerChecker = errors.New("submitter cannot approve the same monitoring revision")
	ErrInactive     = errors.New("monitoring revision is not active")
)

type Actor struct {
	TenantID    string
	PrincipalID string
}

type CreateFormInput struct {
	Code    string          `json:"code"`
	Name    string          `json:"name"`
	Purpose string          `json:"purpose"`
	Fields  []TemplateField `json:"fields"`
}

type TransitionInput struct {
	ID              string          `json:"id"`
	ExpectedVersion int64           `json:"expected_version"`
	To              LifecycleStatus `json:"to"`
}

type StartCollectionInput struct {
	FormTemplateID        string    `json:"form_template_id"`
	FormTemplateVersion   int64     `json:"form_template_version"`
	ProgramID             string    `json:"program_id"`
	RespondentPrincipalID string    `json:"respondent_principal_id"`
	ReviewerPrincipalID   string    `json:"reviewer_principal_id"`
	PeriodStart           time.Time `json:"period_start"`
	PeriodEnd             time.Time `json:"period_end"`
	Deadline              time.Time `json:"deadline"`
}

type CreateCheckInput struct {
	ProgramID               string        `json:"program_id"`
	RequirementID           string        `json:"requirement_id,omitempty"`
	ControlImplementationID string        `json:"control_implementation_id,omitempty"`
	EvidenceContractID      string        `json:"evidence_contract_id,omitempty"`
	Code                    string        `json:"code"`
	Name                    string        `json:"name"`
	Claim                   string        `json:"claim"`
	InputKind               InputKind     `json:"input_kind"`
	FormTemplateID          string        `json:"form_template_id,omitempty"`
	FormTemplateVersion     int64         `json:"form_template_version,omitempty"`
	BindingID               string        `json:"binding_id,omitempty"`
	BindingVersion          int64         `json:"binding_version,omitempty"`
	SourceRules             []SourceRule  `json:"source_rules,omitempty"`
	Thresholds              Thresholds    `json:"thresholds"`
	FreshnessMinutes        int           `json:"freshness_minutes"`
	MinimumCoverage         float64       `json:"minimum_coverage"`
	OwnerPrincipalID        string        `json:"owner_principal_id,omitempty"`
	ReviewerPrincipalID     string        `json:"reviewer_principal_id,omitempty"`
	FailureAction           FailureAction `json:"failure_action"`
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

type Service struct {
	repo     Repository
	requests requestCreator
	evidence evidenceReader
	sources  sourceReader
	now      func() time.Time
	newID    func() (string, error)
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

func (s *Service) CreateForm(ctx context.Context, actor Actor, input CreateFormInput) (FormTemplate, error) {
	if err := validateActor(actor); err != nil {
		return FormTemplate{}, err
	}
	fields, err := normalizeTemplateFields(input.Fields)
	if err != nil {
		return FormTemplate{}, err
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
		ID: valueID, TenantID: actor.TenantID, Code: strings.TrimSpace(input.Code), Name: strings.TrimSpace(input.Name),
		Purpose: strings.TrimSpace(input.Purpose), Fields: fields,
		Lifecycle: Lifecycle{Status: LifecycleDraft, Version: 1, CreatedBy: actor.PrincipalID, CreatedAt: now, UpdatedAt: now},
	})
}

func (s *Service) ListForms(ctx context.Context, actor Actor, limit int) ([]FormTemplate, error) {
	if err := validateActor(actor); err != nil {
		return nil, err
	}
	return s.repo.ListFormRevisions(ctx, actor.TenantID, limit)
}

func (s *Service) TransitionForm(ctx context.Context, actor Actor, input TransitionInput) (FormTemplate, error) {
	if err := validateActor(actor); err != nil {
		return FormTemplate{}, err
	}
	current, err := s.repo.FormRevision(ctx, actor.TenantID, strings.TrimSpace(input.ID), input.ExpectedVersion)
	if err != nil {
		return FormTemplate{}, err
	}
	if input.To == LifecycleActive && current.Status == LifecyclePendingApproval && current.SubmittedBy == actor.PrincipalID {
		return FormTemplate{}, ErrMakerChecker
	}
	return s.repo.TransitionForm(ctx, LifecycleTransition{TenantID: actor.TenantID, ID: current.ID, ExpectedVersion: input.ExpectedVersion, To: input.To, ActorID: actor.PrincipalID, At: s.now().UTC()})
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
	form, err := s.repo.FormRevision(ctx, actor.TenantID, input.FormTemplateID, input.FormTemplateVersion)
	if err != nil {
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
		fields[index] = evidence.Field{ID: field.ID, Label: field.Label, Type: field.Type, Required: field.Required, Description: field.Description, Options: append([]string(nil), field.Options...), AcceptedFormats: append([]string(nil), field.AcceptedFormats...)}
	}
	periodStart := input.PeriodStart.UTC()
	periodEnd := input.PeriodEnd.UTC()
	return s.requests.CreateRequest(ctx, evidence.CreateRequestInput{
		TenantID: actor.TenantID, SubjectType: "PROGRAM", SubjectID: input.ProgramID, Title: form.Name,
		Purpose: form.Purpose, WhyYou: "You are responsible for completing this control review.",
		Sensitivity: "INTERNAL", AudienceType: "INTERNAL",
		Recipient:        evidence.RecipientInput{Type: evidence.RecipientInternalPrincipal, PrincipalID: input.RespondentPrincipalID},
		EstimatedMinutes: estimateMinutes(len(fields)), Deadline: input.Deadline.UTC(),
		KnownFacts: map[string]string{"reviewer": input.ReviewerPrincipalID, "reporting_period_start": periodStart.Format(time.RFC3339), "reporting_period_end": periodEnd.Format(time.RFC3339)},
		Fields:     fields, FormTemplateID: form.ID, FormTemplateVersion: form.Version,
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
		form, err := s.repo.FormRevision(ctx, actor.TenantID, input.FormTemplateID, input.FormTemplateVersion)
		if err != nil {
			return MonitoringCheck{}, err
		}
		if form.Status != LifecycleActive || !form.IsCurrent {
			return MonitoringCheck{}, ErrInactive
		}
	case InputSource:
		if input.BindingID == "" || input.BindingVersion < 1 || input.FormTemplateID != "" || len(input.SourceRules) == 0 {
			return MonitoringCheck{}, errors.Join(ErrInvalid, fmt.Errorf("source checks require one binding revision and at least one rule"))
		}
		for _, rule := range input.SourceRules {
			if err := validateSourceRule(rule); err != nil {
				return MonitoringCheck{}, errors.Join(ErrInvalid, err)
			}
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
		BindingID: strings.TrimSpace(input.BindingID), BindingVersion: input.BindingVersion, SourceRules: append([]SourceRule(nil), input.SourceRules...),
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

func (s *Service) EvaluateSource(ctx context.Context, actor Actor, input EvaluateSourceInput) (MonitoringResult, error) {
	if err := validateActor(actor); err != nil {
		return MonitoringResult{}, err
	}
	if strings.TrimSpace(input.CheckID) == "" || input.CheckVersion < 1 {
		return MonitoringResult{}, errors.Join(ErrInvalid, fmt.Errorf("monitoring check and version are required"))
	}
	if s.sources == nil {
		return MonitoringResult{}, fmt.Errorf("connected-source reads are unavailable")
	}
	check, err := s.repo.CheckRevision(ctx, actor.TenantID, input.CheckID, input.CheckVersion)
	if err != nil {
		return MonitoringResult{}, err
	}
	if check.Status != LifecycleActive || !check.IsCurrent || check.InputKind != InputSource {
		return MonitoringResult{}, ErrInactive
	}
	binding, err := s.sources.Binding(ctx, actor.TenantID, check.BindingID, check.BindingVersion)
	if err != nil {
		return MonitoringResult{}, err
	}
	if binding.BindingID != check.BindingID || binding.Version != check.BindingVersion || binding.TenantID != actor.TenantID {
		return MonitoringResult{}, errors.Join(ErrInvalid, fmt.Errorf("connected-source revision does not match the monitoring check"))
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
	form, err := s.repo.FormRevision(ctx, check.TenantID, check.FormTemplateID, check.FormTemplateVersion)
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
		form, loadErr := s.repo.FormRevision(ctx, actor.TenantID, current.FormTemplateID, current.FormTemplateVersion)
		if loadErr != nil {
			return MonitoringCheck{}, loadErr
		}
		if form.Status != LifecycleActive || !form.IsCurrent {
			return MonitoringCheck{}, ErrInactive
		}
	}
	return s.repo.TransitionCheck(ctx, LifecycleTransition{TenantID: actor.TenantID, ID: current.ID, ExpectedVersion: input.ExpectedVersion, To: input.To, ActorID: actor.PrincipalID, At: s.now().UTC()})
}

func validateActor(actor Actor) error {
	if strings.TrimSpace(actor.TenantID) == "" || strings.TrimSpace(actor.PrincipalID) == "" {
		return errors.Join(ErrInvalid, fmt.Errorf("verified tenant and principal are required"))
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
	if len(fields) < 1 || len(fields) > 50 {
		return nil, errors.Join(ErrInvalid, fmt.Errorf("a form requires 1-50 fields"))
	}
	seen := map[string]struct{}{}
	normalized := make([]TemplateField, len(fields))
	for index, field := range fields {
		field.ID = strings.TrimSpace(field.ID)
		field.Label = strings.TrimSpace(field.Label)
		field.Type = strings.ToLower(strings.TrimSpace(field.Type))
		if field.ID == "" || field.Label == "" {
			return nil, errors.Join(ErrInvalid, fmt.Errorf("every field requires an id and label"))
		}
		if _, duplicate := seen[field.ID]; duplicate {
			return nil, errors.Join(ErrInvalid, fmt.Errorf("field ids must be unique"))
		}
		seen[field.ID] = struct{}{}
		switch field.Type {
		case "text", "short_text", "long_text", "date", "number", "photo", "file", "signature":
			if len(field.Options) != 0 {
				return nil, errors.Join(ErrInvalid, fmt.Errorf("%s cannot define choices", field.Label))
			}
		case "single_select":
			if len(field.Options) < 2 || len(field.Options) > 50 {
				return nil, errors.Join(ErrInvalid, fmt.Errorf("%s requires 2-50 choices", field.Label))
			}
		default:
			return nil, errors.Join(ErrInvalid, fmt.Errorf("%s uses an unsupported response type", field.Label))
		}
		if field.Scoring != nil {
			field.Scoring.ID = field.ID
			field.Scoring.Required = field.Required
			if field.Type != "single_select" || field.Scoring.Weight < 1 || field.Scoring.Weight > 100 || len(field.Scoring.AnswerScores) == 0 {
				return nil, errors.Join(ErrInvalid, fmt.Errorf("%s has invalid risk scoring", field.Label))
			}
			for answer, points := range field.Scoring.AnswerScores {
				if !contains(field.Options, answer) || points < 0 || points > 100 {
					return nil, errors.Join(ErrInvalid, fmt.Errorf("%s has an invalid answer score", field.Label))
				}
			}
			for _, answer := range field.Scoring.CriticalAnswers {
				if !contains(field.Options, answer) {
					return nil, errors.Join(ErrInvalid, fmt.Errorf("%s has an invalid critical answer", field.Label))
				}
			}
		}
		normalized[index] = field
	}
	return normalized, nil
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
