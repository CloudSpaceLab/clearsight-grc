package continuity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

type Service struct {
	repo            Repository
	now             func() time.Time
	sourceValidator EvidenceSourceValidator
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

func (s *Service) ConfigureEvidenceSourceValidator(validator EvidenceSourceValidator) {
	s.sourceValidator = validator
}

func (s *Service) validateEvidenceSources(ctx context.Context, tenant, legalEntity string, sourceIDs []string) error {
	ids := make([]string, 0, len(sourceIDs))
	seen := map[string]struct{}{}
	for _, sourceID := range sourceIDs {
		sourceID = strings.TrimSpace(sourceID)
		if sourceID == "" {
			continue
		}
		if _, ok := seen[sourceID]; ok {
			continue
		}
		seen[sourceID] = struct{}{}
		ids = append(ids, sourceID)
	}
	if len(ids) == 0 {
		return nil
	}
	if s.sourceValidator == nil {
		return ErrEvidenceSourceUnavailable
	}
	if err := s.sourceValidator.ValidateActiveSourcesForEntity(ctx, tenant, legalEntity, ids); err != nil {
		return errors.Join(ErrEvidenceSourceInvalid, err)
	}
	return nil
}

func (s *Service) resolveLegalEntity(ctx context.Context, tenant, identifier string) (string, error) {
	resolver, ok := s.repo.(LegalEntityResolver)
	if !ok {
		return "", ErrNotFound
	}
	return resolver.ResolveLegalEntity(ctx, strings.TrimSpace(tenant), strings.TrimSpace(identifier))
}

func (s *Service) ResolveLegalEntity(ctx context.Context, tenant, identifier string) (string, error) {
	return s.resolveLegalEntity(ctx, tenant, identifier)
}

type AddRequirementInput struct {
	TenantID        string            `json:"tenant_id"`
	ProgramID       string            `json:"program_id"`
	ExpectedVersion int64             `json:"expected_version"`
	SourceID        string            `json:"source_id,omitempty"`
	Code            string            `json:"code"`
	Title           string            `json:"title"`
	Statement       string            `json:"statement"`
	SourceAnchor    string            `json:"source_anchor,omitempty"`
	Modality        string            `json:"modality"`
	Actor           string            `json:"actor,omitempty"`
	Action          string            `json:"action,omitempty"`
	Object          string            `json:"object,omitempty"`
	Status          RequirementStatus `json:"status"`
	EffectiveFrom   time.Time         `json:"effective_from"`
	ActorID         string            `json:"actor_id,omitempty"`
}

type DetermineApplicabilityInput struct {
	TenantID        string              `json:"tenant_id"`
	ProgramID       string              `json:"program_id"`
	ExpectedVersion int64               `json:"expected_version"`
	RequirementID   string              `json:"requirement_id"`
	Status          ApplicabilityStatus `json:"status"`
	Scope           json.RawMessage     `json:"scope"`
	Rationale       string              `json:"rationale"`
	ApprovedBy      string              `json:"approved_by"`
	EffectiveFrom   time.Time           `json:"effective_from"`
}

type AddControlObjectiveInput struct {
	TenantID        string                 `json:"tenant_id"`
	ProgramID       string                 `json:"program_id"`
	ExpectedVersion int64                  `json:"expected_version"`
	Code            string                 `json:"code"`
	Name            string                 `json:"name"`
	Outcome         string                 `json:"outcome"`
	Status          ControlObjectiveStatus `json:"status"`
	ActorID         string                 `json:"actor_id,omitempty"`
}

type AddControlImplementationInput struct {
	TenantID           string                      `json:"tenant_id"`
	ProgramID          string                      `json:"program_id"`
	ExpectedVersion    int64                       `json:"expected_version"`
	ObjectiveID        string                      `json:"objective_id"`
	Name               string                      `json:"name"`
	Description        string                      `json:"description"`
	ImplementationType string                      `json:"implementation_type"`
	OwnerPrincipalID   string                      `json:"owner_principal_id,omitempty"`
	Scope              json.RawMessage             `json:"scope"`
	Status             ControlImplementationStatus `json:"status"`
	EffectiveFrom      time.Time                   `json:"effective_from"`
	ActorID            string                      `json:"actor_id,omitempty"`
}

type LinkRequirementControlInput struct {
	TenantID         string `json:"tenant_id"`
	ProgramID        string `json:"program_id"`
	ExpectedVersion  int64  `json:"expected_version"`
	RequirementID    string `json:"requirement_id"`
	ImplementationID string `json:"implementation_id"`
	ActorID          string `json:"actor_id,omitempty"`
}

type AddEvidenceContractInput struct {
	TenantID                string                 `json:"tenant_id"`
	ProgramID               string                 `json:"program_id"`
	ExpectedVersion         int64                  `json:"expected_version"`
	RequirementID           string                 `json:"requirement_id,omitempty"`
	ControlImplementationID string                 `json:"control_implementation_id,omitempty"`
	Code                    string                 `json:"code"`
	Name                    string                 `json:"name"`
	Claim                   string                 `json:"claim"`
	AcceptableSourceIDs     []string               `json:"acceptable_source_ids"`
	PopulationScope         json.RawMessage        `json:"population_scope"`
	FreshnessMinutes        int                    `json:"freshness_minutes"`
	MinimumCoverage         float64                `json:"minimum_coverage"`
	IndependenceRequired    bool                   `json:"independence_required"`
	ContradictionPolicy     string                 `json:"contradiction_policy"`
	FailureAction           string                 `json:"failure_action"`
	Status                  EvidenceContractStatus `json:"status"`
	ActorID                 string                 `json:"actor_id,omitempty"`
}

type RecordEvidenceAssessmentInput struct {
	TenantID        string             `json:"tenant_id"`
	ProgramID       string             `json:"program_id"`
	ExpectedVersion int64              `json:"expected_version"`
	ContractID      string             `json:"contract_id"`
	Conclusion      EvidenceConclusion `json:"conclusion"`
	Coverage        float64            `json:"coverage"`
	Basis           json.RawMessage    `json:"basis"`
	ValidUntil      *time.Time         `json:"valid_until,omitempty"`
	AssessedBy      string             `json:"assessed_by,omitempty"`
	AssessedAt      time.Time          `json:"assessed_at"`
}

type AddDecisionInput struct {
	TenantID             string          `json:"tenant_id"`
	MatterID             string          `json:"matter_id"`
	ExpectedVersion      int64           `json:"expected_version"`
	Type                 string          `json:"type"`
	Status               DecisionStatus  `json:"status"`
	Options              json.RawMessage `json:"options"`
	SelectedOption       string          `json:"selected_option,omitempty"`
	Rationale            string          `json:"rationale"`
	Conditions           json.RawMessage `json:"conditions"`
	AuthorityPrincipalID string          `json:"authority_principal_id,omitempty"`
	ExpiresAt            *time.Time      `json:"expires_at,omitempty"`
}

type AddActionInput struct {
	TenantID         string     `json:"tenant_id"`
	MatterID         string     `json:"matter_id"`
	ExpectedVersion  int64      `json:"expected_version"`
	Title            string     `json:"title"`
	Description      string     `json:"description"`
	OwnerPrincipalID string     `json:"owner_principal_id,omitempty"`
	DueAt            *time.Time `json:"due_at,omitempty"`
	ActorID          string     `json:"actor_id,omitempty"`
	OriginKey        string     `json:"-"`
}

type TransitionActionInput struct {
	TenantID        string       `json:"tenant_id"`
	MatterID        string       `json:"matter_id"`
	ActionID        string       `json:"action_id"`
	ExpectedVersion int64        `json:"expected_version"`
	To              ActionStatus `json:"to"`
	ActorID         string       `json:"actor_id,omitempty"`
	Rationale       string       `json:"rationale"`
}

type AddVerificationContractInput struct {
	TenantID                 string          `json:"tenant_id"`
	MatterID                 string          `json:"matter_id"`
	ExpectedVersion          int64           `json:"expected_version"`
	ActionID                 string          `json:"action_id,omitempty"`
	ExpectedOutcome          string          `json:"expected_outcome"`
	Baseline                 json.RawMessage `json:"baseline"`
	Scope                    json.RawMessage `json:"scope"`
	MeasurementSourceID      string          `json:"measurement_source_id,omitempty"`
	Threshold                json.RawMessage `json:"threshold"`
	ObservationPeriodMinutes int             `json:"observation_period_minutes"`
	ReviewerCandidateID      string          `json:"reviewer_candidate_id,omitempty"`
	AuthorityPrincipalID     string          `json:"authority_principal_id,omitempty"`
	FailureResponse          string          `json:"failure_response"`
	ActorID                  string          `json:"actor_id,omitempty"`
}

type SupersedeVerificationContractInput struct {
	TenantID                 string          `json:"tenant_id"`
	MatterID                 string          `json:"matter_id"`
	ContractID               string          `json:"contract_id"`
	ExpectedVersion          int64           `json:"expected_version"`
	ActionID                 string          `json:"action_id,omitempty"`
	ExpectedOutcome          string          `json:"expected_outcome"`
	Baseline                 json.RawMessage `json:"baseline"`
	Scope                    json.RawMessage `json:"scope"`
	MeasurementSourceID      string          `json:"measurement_source_id,omitempty"`
	Threshold                json.RawMessage `json:"threshold"`
	ObservationPeriodMinutes int             `json:"observation_period_minutes"`
	ReviewerCandidateID      string          `json:"reviewer_candidate_id,omitempty"`
	AuthorityPrincipalID     string          `json:"authority_principal_id,omitempty"`
	FailureResponse          string          `json:"failure_response"`
	Rationale                string          `json:"rationale"`
	ActorID                  string          `json:"actor_id,omitempty"`
}

type RetireVerificationContractInput struct {
	TenantID        string `json:"tenant_id"`
	MatterID        string `json:"matter_id"`
	ContractID      string `json:"contract_id"`
	ExpectedVersion int64  `json:"expected_version"`
	Rationale       string `json:"rationale"`
	ActorID         string `json:"actor_id,omitempty"`
}

type RecordVerificationResultInput struct {
	TenantID                     string                   `json:"tenant_id"`
	MatterID                     string                   `json:"matter_id"`
	ExpectedVersion              int64                    `json:"expected_version"`
	ContractID                   string                   `json:"contract_id"`
	Result                       VerificationResultStatus `json:"result"`
	Observations                 json.RawMessage          `json:"observations"`
	EvidenceReferences           json.RawMessage          `json:"evidence_references"`
	ReviewerPrincipalID          string                   `json:"reviewer_principal_id,omitempty"`
	ReviewerAuthorityPrincipalID string                   `json:"reviewer_authority_principal_id,omitempty"`
	EscalationPrincipalID        string                   `json:"escalation_principal_id,omitempty"`
	Rationale                    string                   `json:"rationale"`
	ObservedAt                   time.Time                `json:"observed_at"`
}

type AddResponsePackageInput struct {
	TenantID        string          `json:"tenant_id"`
	MatterID        string          `json:"matter_id"`
	ExpectedVersion int64           `json:"expected_version"`
	Purpose         string          `json:"purpose"`
	Audience        string          `json:"audience"`
	Manifest        json.RawMessage `json:"manifest"`
	ActorID         string          `json:"actor_id,omitempty"`
}

type TransitionResponseInput struct {
	TenantID        string         `json:"tenant_id"`
	MatterID        string         `json:"matter_id"`
	ResponseID      string         `json:"response_id"`
	ExpectedVersion int64          `json:"expected_version"`
	To              ResponseStatus `json:"to"`
	ActorID         string         `json:"actor_id,omitempty"`
	Rationale       string         `json:"rationale"`
}

func (s *Service) ListPrograms(ctx context.Context, tenant string, limit int) ([]ProgramAggregate, error) {
	if strings.TrimSpace(tenant) == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}
	return s.repo.ListPrograms(ctx, tenant, boundedLimit(limit))
}

func (s *Service) GetProgram(ctx context.Context, tenant, id string) (ProgramAggregate, error) {
	if strings.TrimSpace(tenant) == "" || strings.TrimSpace(id) == "" {
		return ProgramAggregate{}, fmt.Errorf("tenant_id and program id are required")
	}
	return s.repo.GetProgram(ctx, tenant, id)
}

func (s *Service) CreateProgram(ctx context.Context, input CreateProgramInput) (ProgramAggregate, error) {
	if strings.TrimSpace(input.TenantID) == "" || strings.TrimSpace(input.Code) == "" || strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Type) == "" || strings.TrimSpace(input.OwningFunction) == "" {
		return ProgramAggregate{}, fmt.Errorf("tenant_id, code, name, type and owning_function are required")
	}
	if strings.TrimSpace(input.OwnerPrincipalID) != "" && strings.TrimSpace(input.OwnerPrincipalID) == strings.TrimSpace(input.AuthorityPrincipalID) {
		return ProgramAggregate{}, fmt.Errorf("the Program owner and approval authority must be different people")
	}
	legalEntityID, scopeOK := actorLegalEntity(ctx, input.TenantID, input.LegalEntityID)
	if !scopeOK {
		return ProgramAggregate{}, ErrNotFound
	}
	canonicalEntityID, err := s.resolveLegalEntity(ctx, input.TenantID, legalEntityID)
	if err != nil {
		return ProgramAggregate{}, err
	}
	input.LegalEntityID = canonicalEntityID
	ctx = withCanonicalLegalEntity(ctx, input.TenantID, canonicalEntityID)
	if input.EffectiveFrom.IsZero() {
		input.EffectiveFrom = s.now().UTC()
	}
	if input.EffectiveUntil != nil && !input.EffectiveFrom.Before(*input.EffectiveUntil) {
		return ProgramAggregate{}, fmt.Errorf("effective_until must be after effective_from")
	}
	scope, err := normalizedJSON(input.Scope, `{}`)
	if err != nil {
		return ProgramAggregate{}, fmt.Errorf("scope: %w", err)
	}
	programID, err := id.NewUUIDv7()
	if err != nil {
		return ProgramAggregate{}, err
	}
	now := s.now().UTC()
	var effectiveUntil *time.Time
	if input.EffectiveUntil != nil {
		value := input.EffectiveUntil.UTC()
		effectiveUntil = &value
	}
	program := Program{ID: programID, TenantID: input.TenantID, LegalEntityID: input.LegalEntityID, Code: strings.ToUpper(strings.TrimSpace(input.Code)), Name: strings.TrimSpace(input.Name), Type: strings.TrimSpace(input.Type), Status: ProgramDraft, OwningFunction: strings.TrimSpace(input.OwningFunction), OwnerPrincipalID: input.OwnerPrincipalID, AuthorityPrincipalID: input.AuthorityPrincipalID, Jurisdiction: strings.TrimSpace(input.Jurisdiction), Scope: scope, EffectiveFrom: input.EffectiveFrom.UTC(), EffectiveUntil: effectiveUntil, CreatedAt: now, UpdatedAt: now, Version: 1}
	event, err := newEvent(input.TenantID, "PROGRAM", program.ID, 1, EventProgramCreated, program, actorFor(input.ActorID), input.ActorID, now)
	if err != nil {
		return ProgramAggregate{}, err
	}
	if _, err = s.repo.CreateProgram(ctx, program, event); err != nil {
		return ProgramAggregate{}, err
	}
	_ = s.requestProgramRefresh(ctx, input.TenantID, program.ID, EventProgramCreated, program.ID, "system")
	if current, readErr := s.repo.GetProgram(ctx, input.TenantID, program.ID); readErr == nil {
		return current, nil
	}
	return decorateProgram(ProgramAggregate{Program: program}), nil
}

func (s *Service) TransitionProgram(ctx context.Context, input ProgramTransitionInput) (ProgramAggregate, error) {
	if strings.TrimSpace(input.TenantID) == "" || strings.TrimSpace(input.ID) == "" || input.ExpectedVersion < 1 {
		return ProgramAggregate{}, fmt.Errorf("tenant_id, id and positive expected_version are required")
	}
	aggregate, err := s.GetProgram(ctx, input.TenantID, input.ID)
	if err != nil {
		return ProgramAggregate{}, err
	}
	if aggregate.Program.Version != input.ExpectedVersion {
		return ProgramAggregate{}, ErrVersionConflict
	}
	if !allowedProgramTransition(aggregate.Program.Status, input.To) {
		return ProgramAggregate{}, ErrInvalidState
	}
	if input.To == ProgramActive {
		if aggregate.Program.EffectiveUntil != nil && !s.now().UTC().Before(*aggregate.Program.EffectiveUntil) {
			return ProgramAggregate{}, fmt.Errorf("the program period has ended")
		}
		if strings.TrimSpace(input.ActorID) == "" || strings.TrimSpace(input.Rationale) == "" {
			return ProgramAggregate{}, fmt.Errorf("actor_id and rationale are required to activate a program")
		}
		if strings.TrimSpace(aggregate.Program.OwnerPrincipalID) == "" || strings.TrimSpace(aggregate.Program.AuthorityPrincipalID) == "" {
			return ProgramAggregate{}, fmt.Errorf("an accountable owner and approval authority are required before activation")
		}
		approved := false
		for _, requirement := range aggregate.Requirements {
			if requirement.Status == RequirementApproved {
				approved = true
				break
			}
		}
		if !approved {
			return ProgramAggregate{}, fmt.Errorf("at least one approved requirement is required before activation")
		}
	}
	if (input.To == ProgramPaused || input.To == ProgramRetired) && strings.TrimSpace(input.Rationale) == "" {
		return ProgramAggregate{}, fmt.Errorf("rationale is required when a program is paused or retired")
	}
	program := aggregate.Program
	program.Status = input.To
	program.UpdatedAt = s.now().UTC()
	if input.To == ProgramRetired {
		ended := program.UpdatedAt
		program.EffectiveUntil = &ended
	}
	if aggregate.Program.Status == ProgramPaused && input.To == ProgramActive {
		program.EffectiveUntil = nil
	}
	return s.applyProgramValueAndResult(ctx, aggregate, input.TenantID, input.ID, input.ExpectedVersion, EventProgramStatusChanged, program, input.ActorID, input.ID)
}

func (s *Service) AddRequirement(ctx context.Context, input AddRequirementInput) (ProgramAggregate, error) {
	aggregate, err := s.programForMutation(ctx, input.TenantID, input.ProgramID, input.ExpectedVersion)
	if err != nil {
		return ProgramAggregate{}, err
	}
	if strings.TrimSpace(input.Code) == "" || strings.TrimSpace(input.Title) == "" || strings.TrimSpace(input.Statement) == "" {
		return ProgramAggregate{}, fmt.Errorf("code, title and statement are required")
	}
	if input.Status == "" {
		input.Status = RequirementApproved
	}
	if !validRequirementStatus(input.Status) {
		return ProgramAggregate{}, fmt.Errorf("unsupported requirement status")
	}
	modality := strings.ToUpper(strings.TrimSpace(input.Modality))
	if modality == "" {
		modality = "MUST"
	}
	if !validModality(modality) {
		return ProgramAggregate{}, fmt.Errorf("modality must be MUST, MUST_NOT, MAY, SHOULD or EXPECTED")
	}
	if input.EffectiveFrom.IsZero() {
		input.EffectiveFrom = s.now().UTC()
	}
	valueID, err := id.NewUUIDv7()
	if err != nil {
		return ProgramAggregate{}, err
	}
	now := s.now().UTC()
	value := Requirement{ID: valueID, TenantID: input.TenantID, ProgramID: input.ProgramID, SourceID: input.SourceID, Code: strings.ToUpper(strings.TrimSpace(input.Code)), Title: strings.TrimSpace(input.Title), Statement: strings.TrimSpace(input.Statement), SourceAnchor: strings.TrimSpace(input.SourceAnchor), Modality: modality, Actor: strings.TrimSpace(input.Actor), Action: strings.TrimSpace(input.Action), Object: strings.TrimSpace(input.Object), Status: input.Status, EffectiveFrom: input.EffectiveFrom.UTC(), CreatedAt: now, Version: 1}
	return s.applyProgramValueAndResult(ctx, aggregate, input.TenantID, input.ProgramID, input.ExpectedVersion, EventRequirementAdded, value, input.ActorID, value.ID)
}

func (s *Service) DetermineApplicability(ctx context.Context, input DetermineApplicabilityInput) (ProgramAggregate, error) {
	aggregate, err := s.programForMutation(ctx, input.TenantID, input.ProgramID, input.ExpectedVersion)
	if err != nil {
		return ProgramAggregate{}, err
	}
	if strings.TrimSpace(input.RequirementID) == "" || strings.TrimSpace(input.Rationale) == "" || strings.TrimSpace(input.ApprovedBy) == "" {
		return ProgramAggregate{}, fmt.Errorf("requirement_id, rationale and approved_by are required")
	}
	if !containsRequirement(aggregate.Requirements, input.RequirementID) {
		return ProgramAggregate{}, fmt.Errorf("requirement_id does not belong to this program")
	}
	if !validApplicability(input.Status) {
		return ProgramAggregate{}, fmt.Errorf("unsupported applicability status")
	}
	scope, err := normalizedJSON(input.Scope, `{}`)
	if err != nil {
		return ProgramAggregate{}, err
	}
	valueID, err := id.NewUUIDv7()
	if err != nil {
		return ProgramAggregate{}, err
	}
	if input.EffectiveFrom.IsZero() {
		input.EffectiveFrom = s.now().UTC()
	}
	value := Applicability{ID: valueID, TenantID: input.TenantID, ProgramID: input.ProgramID, RequirementID: input.RequirementID, Status: input.Status, Scope: scope, Rationale: strings.TrimSpace(input.Rationale), ApprovedBy: input.ApprovedBy, EffectiveFrom: input.EffectiveFrom.UTC(), CreatedAt: s.now().UTC(), Version: 1}
	return s.applyProgramValueAndResult(ctx, aggregate, input.TenantID, input.ProgramID, input.ExpectedVersion, EventApplicabilityDetermined, value, input.ApprovedBy, value.ID)
}

func (s *Service) AddControlObjective(ctx context.Context, input AddControlObjectiveInput) (ProgramAggregate, error) {
	aggregate, err := s.programForMutation(ctx, input.TenantID, input.ProgramID, input.ExpectedVersion)
	if err != nil {
		return ProgramAggregate{}, err
	}
	if strings.TrimSpace(input.Code) == "" || strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Outcome) == "" {
		return ProgramAggregate{}, fmt.Errorf("code, name and outcome are required")
	}
	if input.Status == "" {
		input.Status = ObjectiveActive
	}
	if !validObjectiveStatus(input.Status) {
		return ProgramAggregate{}, fmt.Errorf("unsupported control objective status")
	}
	valueID, err := id.NewUUIDv7()
	if err != nil {
		return ProgramAggregate{}, err
	}
	value := ControlObjective{ID: valueID, TenantID: input.TenantID, ProgramID: input.ProgramID, Code: strings.ToUpper(strings.TrimSpace(input.Code)), Name: strings.TrimSpace(input.Name), Outcome: strings.TrimSpace(input.Outcome), Status: input.Status, CreatedAt: s.now().UTC(), Version: 1}
	return s.applyProgramValueAndResult(ctx, aggregate, input.TenantID, input.ProgramID, input.ExpectedVersion, EventControlObjectiveAdded, value, input.ActorID, value.ID)
}

func (s *Service) AddControlImplementation(ctx context.Context, input AddControlImplementationInput) (ProgramAggregate, error) {
	aggregate, err := s.programForMutation(ctx, input.TenantID, input.ProgramID, input.ExpectedVersion)
	if err != nil {
		return ProgramAggregate{}, err
	}
	if strings.TrimSpace(input.ObjectiveID) == "" || strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Description) == "" || strings.TrimSpace(input.ImplementationType) == "" {
		return ProgramAggregate{}, fmt.Errorf("objective_id, name, description and implementation_type are required")
	}
	if !containsObjective(aggregate.ControlObjectives, input.ObjectiveID) {
		return ProgramAggregate{}, fmt.Errorf("objective_id does not belong to this program")
	}
	if input.Status == "" {
		input.Status = ImplementationPlanned
	}
	if input.Status != ImplementationPlanned {
		return ProgramAggregate{}, fmt.Errorf("new safeguards must start as planned")
	}
	if input.EffectiveFrom.IsZero() {
		input.EffectiveFrom = s.now().UTC()
	}
	scope, err := normalizedJSON(input.Scope, `{}`)
	if err != nil {
		return ProgramAggregate{}, err
	}
	valueID, err := id.NewUUIDv7()
	if err != nil {
		return ProgramAggregate{}, err
	}
	now := s.now().UTC()
	value := ControlImplementation{ID: valueID, TenantID: input.TenantID, ProgramID: input.ProgramID, ObjectiveID: input.ObjectiveID, Name: strings.TrimSpace(input.Name), Description: strings.TrimSpace(input.Description), ImplementationType: strings.TrimSpace(input.ImplementationType), OwnerPrincipalID: input.OwnerPrincipalID, Scope: scope, Status: input.Status, EffectiveFrom: input.EffectiveFrom.UTC(), CreatedAt: now, UpdatedAt: now, Version: 1}
	return s.applyProgramValueAndResult(ctx, aggregate, input.TenantID, input.ProgramID, input.ExpectedVersion, EventControlImplementationAdded, value, input.ActorID, value.ID)
}

func (s *Service) LinkRequirementControl(ctx context.Context, input LinkRequirementControlInput) (ProgramAggregate, error) {
	aggregate, err := s.programForMutation(ctx, input.TenantID, input.ProgramID, input.ExpectedVersion)
	if err != nil {
		return ProgramAggregate{}, err
	}
	if strings.TrimSpace(input.RequirementID) == "" || strings.TrimSpace(input.ImplementationID) == "" {
		return ProgramAggregate{}, fmt.Errorf("requirement_id and implementation_id are required")
	}
	if !containsRequirement(aggregate.Requirements, input.RequirementID) || !containsImplementation(aggregate.ControlImplementations, input.ImplementationID) {
		return ProgramAggregate{}, fmt.Errorf("requirement_id and implementation_id must belong to this program")
	}
	valueID, err := id.NewUUIDv7()
	if err != nil {
		return ProgramAggregate{}, err
	}
	value := RequirementControlLink{ID: valueID, TenantID: input.TenantID, ProgramID: input.ProgramID, RequirementID: input.RequirementID, ImplementationID: input.ImplementationID, CreatedAt: s.now().UTC()}
	return s.applyProgramValueAndResult(ctx, aggregate, input.TenantID, input.ProgramID, input.ExpectedVersion, EventRequirementControlLinked, value, input.ActorID, value.ID)
}

func (s *Service) AddEvidenceContract(ctx context.Context, input AddEvidenceContractInput) (ProgramAggregate, error) {
	aggregate, err := s.programForMutation(ctx, input.TenantID, input.ProgramID, input.ExpectedVersion)
	if err != nil {
		return ProgramAggregate{}, err
	}
	if (input.RequirementID == "") == (input.ControlImplementationID == "") {
		return ProgramAggregate{}, fmt.Errorf("exactly one of requirement_id or control_implementation_id is required")
	}
	if strings.TrimSpace(input.Code) == "" || strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Claim) == "" {
		return ProgramAggregate{}, fmt.Errorf("code, name and claim are required")
	}
	if (strings.TrimSpace(input.RequirementID) == "") == (strings.TrimSpace(input.ControlImplementationID) == "") {
		return ProgramAggregate{}, fmt.Errorf("provide either requirement_id or control_implementation_id")
	}
	if input.RequirementID != "" && !containsRequirement(aggregate.Requirements, input.RequirementID) {
		return ProgramAggregate{}, fmt.Errorf("requirement_id does not belong to this program")
	}
	if input.ControlImplementationID != "" && !containsImplementation(aggregate.ControlImplementations, input.ControlImplementationID) {
		return ProgramAggregate{}, fmt.Errorf("control_implementation_id does not belong to this program")
	}
	if input.FreshnessMinutes < 1 || input.FreshnessMinutes > 525600 {
		return ProgramAggregate{}, fmt.Errorf("freshness_minutes must be between 1 and 525600")
	}
	if input.MinimumCoverage < 0 || input.MinimumCoverage > 1 {
		return ProgramAggregate{}, fmt.Errorf("minimum_coverage must be between 0 and 1")
	}
	if input.Status == "" {
		input.Status = EvidenceContractDraft
	}
	if input.Status != EvidenceContractDraft {
		return ProgramAggregate{}, fmt.Errorf("new evidence checks must start as drafts")
	}
	contradictionPolicy := strings.ToUpper(strings.TrimSpace(input.ContradictionPolicy))
	if contradictionPolicy == "" {
		contradictionPolicy = "REVIEW"
	}
	if !validContradictionPolicy(contradictionPolicy) {
		return ProgramAggregate{}, fmt.Errorf("contradiction_policy must be HOLD, REVIEW or FAIL")
	}
	failureAction := strings.ToUpper(strings.TrimSpace(input.FailureAction))
	if failureAction == "" {
		failureAction = "MATTER"
	}
	if !validEvidenceFailureAction(failureAction) {
		return ProgramAggregate{}, fmt.Errorf("failed evidence results must create a linked issue")
	}
	if err = s.validateEvidenceSources(ctx, input.TenantID, aggregate.Program.LegalEntityID, input.AcceptableSourceIDs); err != nil {
		return ProgramAggregate{}, err
	}
	population, err := normalizedJSON(input.PopulationScope, `{}`)
	if err != nil {
		return ProgramAggregate{}, err
	}
	valueID, err := id.NewUUIDv7()
	if err != nil {
		return ProgramAggregate{}, err
	}
	now := s.now().UTC()
	value := EvidenceContract{ID: valueID, TenantID: input.TenantID, ProgramID: input.ProgramID, RequirementID: input.RequirementID, ControlImplementationID: input.ControlImplementationID, Code: strings.ToUpper(strings.TrimSpace(input.Code)), Name: strings.TrimSpace(input.Name), Claim: strings.TrimSpace(input.Claim), AcceptableSourceIDs: append([]string(nil), input.AcceptableSourceIDs...), PopulationScope: population, FreshnessMinutes: input.FreshnessMinutes, MinimumCoverage: input.MinimumCoverage, IndependenceRequired: input.IndependenceRequired, ContradictionPolicy: contradictionPolicy, FailureAction: failureAction, ConfiguredBy: strings.TrimSpace(input.ActorID), Status: input.Status, CreatedAt: now, UpdatedAt: now, Version: 1}
	return s.applyProgramValueAndResult(ctx, aggregate, input.TenantID, input.ProgramID, input.ExpectedVersion, EventEvidenceContractAdded, value, input.ActorID, value.ID)
}

func (s *Service) RecordEvidenceAssessment(ctx context.Context, input RecordEvidenceAssessmentInput) (ProgramAggregate, error) {
	aggregate, err := s.programForMutation(ctx, input.TenantID, input.ProgramID, input.ExpectedVersion)
	if err != nil {
		return ProgramAggregate{}, err
	}
	if strings.TrimSpace(input.ContractID) == "" || !validEvidenceConclusion(input.Conclusion) {
		return ProgramAggregate{}, fmt.Errorf("contract_id and supported conclusion are required")
	}
	contract, ok := evidenceContractByID(aggregate.EvidenceContracts, input.ContractID)
	if !ok {
		return ProgramAggregate{}, fmt.Errorf("contract_id does not belong to this program")
	}
	if contract.Status != EvidenceContractActive {
		return ProgramAggregate{}, fmt.Errorf("the evidence check is not active")
	}
	if contract.IndependenceRequired {
		if strings.TrimSpace(input.AssessedBy) == "" {
			return ProgramAggregate{}, fmt.Errorf("an independent reviewer is required for this evidence check")
		}
		if contract.ControlImplementationID != "" {
			if implementation, found := implementationByID(aggregate.ControlImplementations, contract.ControlImplementationID); found && implementation.OwnerPrincipalID != "" && implementation.OwnerPrincipalID == input.AssessedBy {
				return ProgramAggregate{}, fmt.Errorf("the control owner cannot perform this independent evidence review")
			}
		}
	}
	if input.Coverage < 0 || input.Coverage > 1 {
		return ProgramAggregate{}, fmt.Errorf("coverage must be between 0 and 1")
	}
	basis, err := normalizedJSON(input.Basis, `{}`)
	if err != nil {
		return ProgramAggregate{}, err
	}
	if input.AssessedAt.IsZero() {
		input.AssessedAt = s.now().UTC()
	}
	if input.ValidUntil == nil {
		validUntil := input.AssessedAt.UTC().Add(time.Duration(contract.FreshnessMinutes) * time.Minute)
		input.ValidUntil = &validUntil
	}
	valueID, err := id.NewUUIDv7()
	if err != nil {
		return ProgramAggregate{}, err
	}
	value := EvidenceAssessment{ID: valueID, TenantID: input.TenantID, ProgramID: input.ProgramID, ContractID: input.ContractID, Conclusion: input.Conclusion, Coverage: input.Coverage, Basis: basis, ValidUntil: input.ValidUntil, AssessedBy: input.AssessedBy, AssessedAt: input.AssessedAt.UTC(), CreatedAt: s.now().UTC()}
	if evidenceAssessmentNeedsFailureAction(value, contract) {
		switch contract.FailureAction {
		case "FLAG", "BLOCK":
			// Legacy checks used these values before failure handling was narrowed
			// to linked issues. The result itself drives Program attention and
			// blocking state and remains a valid material history event.
		case "MATTER", "REQUEST":
			if repo, ok := s.repo.(EvidenceAssessmentFailureRepository); ok {
				return s.recordEvidenceAssessmentWithFailure(ctx, aggregate, contract, value, repo)
			}
			return ProgramAggregate{}, fmt.Errorf("atomic evidence failure handling is unavailable")
		default:
			return ProgramAggregate{}, fmt.Errorf("the evidence check failure action is not recognised")
		}
	}
	if err = s.applyProgramValue(ctx, input.TenantID, input.ProgramID, input.ExpectedVersion, EventEvidenceAssessmentRecorded, value, input.AssessedBy); err != nil {
		return ProgramAggregate{}, err
	}
	// The assessment event is committed before projection refresh and the
	// immediate aggregate read. Neither derived work nor a read outage may turn
	// that successful material command into a reported failure.
	_ = s.requestProgramRefresh(ctx, input.TenantID, input.ProgramID, EventEvidenceAssessmentRecorded, value.ID, "system")
	aggregate.EvidenceAssessments = append(aggregate.EvidenceAssessments, value)
	return s.programResourceResult(ctx, input.TenantID, input.ProgramID, aggregate, value.CreatedAt), nil
}

func (s *Service) RefreshProgram(ctx context.Context, tenant, programID, triggerType, triggerID string) (ProgramAggregate, error) {
	if err := s.refreshProgram(ctx, tenant, programID, triggerType, triggerID); err != nil {
		return ProgramAggregate{}, err
	}
	return s.repo.GetProgram(ctx, tenant, programID)
}

func (s *Service) ApplyTrigger(ctx context.Context, trigger Trigger) (ProgramAggregate, *Matter, bool, error) {
	if strings.TrimSpace(trigger.TenantID) == "" || strings.TrimSpace(trigger.ProgramID) == "" || strings.TrimSpace(trigger.Type) == "" || strings.TrimSpace(trigger.DedupeKey) == "" || strings.TrimSpace(trigger.Source) == "" {
		return ProgramAggregate{}, nil, false, fmt.Errorf("tenant_id, program_id, type, dedupe_key and source are required")
	}
	if trigger.ObservedAt.IsZero() {
		trigger.ObservedAt = s.now().UTC()
	}
	payload, err := normalizedJSON(trigger.Payload, `{}`)
	if err != nil {
		return ProgramAggregate{}, nil, false, err
	}
	trigger.Payload = payload
	if trigger.ID == "" {
		trigger.ID, err = id.NewUUIDv7()
		if err != nil {
			return ProgramAggregate{}, nil, false, err
		}
	}
	if bundleRepo, ok := s.repo.(TriggerBundleRepository); ok {
		aggregate, getErr := s.repo.GetProgram(ctx, trigger.TenantID, trigger.ProgramID)
		if getErr != nil {
			return ProgramAggregate{}, nil, false, getErr
		}
		return s.applyTriggerBundle(ctx, trigger, aggregate, bundleRepo)
	}
	inserted, err := s.repo.RecordProgramTrigger(ctx, trigger)
	if err != nil {
		return ProgramAggregate{}, nil, false, err
	}
	aggregate, err := s.repo.GetProgram(ctx, trigger.TenantID, trigger.ProgramID)
	if err != nil {
		return ProgramAggregate{}, nil, false, err
	}
	alreadyInStream := false
	for _, recorded := range aggregate.Triggers {
		if recorded.DedupeKey == trigger.DedupeKey {
			alreadyInStream = true
			trigger = recorded
			break
		}
	}
	if !alreadyInStream {
		if err = s.applyProgramValue(ctx, trigger.TenantID, trigger.ProgramID, aggregate.Program.Version, EventProgramTriggerRecorded, trigger, trigger.ActorID); err != nil {
			return ProgramAggregate{}, nil, false, err
		}
	}

	createdMatter, err := s.ensureTriggerMatter(ctx, trigger)
	if err != nil {
		return ProgramAggregate{}, nil, false, err
	}
	program, err := s.refreshAndGetProgram(ctx, trigger.TenantID, trigger.ProgramID, trigger.Type, trigger.ID)
	return program, createdMatter, inserted, err
}

func (s *Service) ensureTriggerMatter(ctx context.Context, trigger Trigger) (*Matter, error) {
	matterType, title, summary, create := matterForTrigger(trigger)
	if !create {
		return nil, nil
	}
	existingAggregate, err := s.MatterByTriggerKey(ctx, trigger.TenantID, trigger.DedupeKey)
	if err == nil {
		if matterLinkedToProgram(existingAggregate, trigger.ProgramID) {
			existing := existingAggregate.Matter
			return &existing, nil
		}
		return nil, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	matterAggregate, err := s.CreateMatter(ctx, CreateMatterInput{TenantID: trigger.TenantID, Type: matterType, Priority: triggerPriority(trigger.Type), Title: title, Summary: summary, Scope: trigger.Payload, TriggerType: trigger.Type, TriggerID: trigger.ID, TriggerKey: trigger.DedupeKey, KnownFacts: trigger.Payload, MissingFacts: json.RawMessage(`[]`), Contradictions: json.RawMessage(`[]`), ProgramID: trigger.ProgramID, ActorID: trigger.ActorID})
	if errors.Is(err, ErrDuplicate) {
		existingAggregate, lookupErr := s.MatterByTriggerKey(ctx, trigger.TenantID, trigger.DedupeKey)
		if lookupErr != nil {
			if errors.Is(lookupErr, ErrNotFound) {
				return nil, nil
			}
			return nil, lookupErr
		}
		if !matterLinkedToProgram(existingAggregate, trigger.ProgramID) {
			return nil, nil
		}
		existing := existingAggregate.Matter
		return &existing, nil
	}
	if err != nil {
		return nil, err
	}
	return &matterAggregate.Matter, nil
}

func (s *Service) ProgramAt(ctx context.Context, tenant, id string, at time.Time) (ProgramAggregate, error) {
	events, err := s.repo.ProgramEvents(ctx, tenant, id, &at)
	if err != nil {
		return ProgramAggregate{}, err
	}
	aggregate, err := reconstructProgram(events)
	if err != nil {
		return ProgramAggregate{}, err
	}
	state, err := s.programStateAt(ctx, tenant, id, &at)
	if err != nil {
		return ProgramAggregate{}, err
	}
	if state != nil {
		aggregate.CurrentState = state
	}
	return decorateProgram(aggregate), nil
}

func (s *Service) ListMatters(ctx context.Context, tenant, status string, limit int) ([]MatterAggregate, error) {
	if strings.TrimSpace(tenant) == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}
	return s.repo.ListMatters(ctx, tenant, strings.ToUpper(strings.TrimSpace(status)), boundedLimit(limit))
}

func (s *Service) GetMatter(ctx context.Context, tenant, id string) (MatterAggregate, error) {
	if strings.TrimSpace(tenant) == "" || strings.TrimSpace(id) == "" {
		return MatterAggregate{}, fmt.Errorf("tenant_id and matter id are required")
	}
	aggregate, err := s.repo.GetMatter(ctx, tenant, id)
	if err != nil {
		return MatterAggregate{}, err
	}
	aggregate.Closure = assessClosure(aggregate)
	return aggregate, nil
}

func (s *Service) CreateMatter(ctx context.Context, input CreateMatterInput) (MatterAggregate, error) {
	if strings.TrimSpace(input.TenantID) == "" || !validMatterType(input.Type) || strings.TrimSpace(input.Title) == "" || strings.TrimSpace(input.Summary) == "" {
		return MatterAggregate{}, fmt.Errorf("tenant_id, supported type, title and summary are required")
	}
	legalEntityID, scopeOK := actorLegalEntity(ctx, input.TenantID, input.LegalEntityID)
	if !scopeOK {
		return MatterAggregate{}, ErrNotFound
	}
	if input.Priority < 1 || input.Priority > 5 {
		return MatterAggregate{}, fmt.Errorf("priority must be between 1 and 5")
	}
	if (strings.TrimSpace(input.RequirementID) != "" || strings.TrimSpace(input.ControlID) != "") && strings.TrimSpace(input.ProgramID) == "" {
		return MatterAggregate{}, fmt.Errorf("program_id is required when linking a requirement or control")
	}
	canonicalEntityID := ""
	var err error
	if strings.TrimSpace(legalEntityID) != "" {
		canonicalEntityID, err = s.resolveLegalEntity(ctx, input.TenantID, legalEntityID)
		if err != nil {
			return MatterAggregate{}, err
		}
		ctx = withCanonicalLegalEntity(ctx, input.TenantID, canonicalEntityID)
	}
	if strings.TrimSpace(input.ProgramID) != "" {
		program, getErr := s.GetProgram(ctx, input.TenantID, input.ProgramID)
		if getErr != nil {
			return MatterAggregate{}, getErr
		}
		if input.RequirementID != "" && !containsRequirement(program.Requirements, input.RequirementID) {
			return MatterAggregate{}, fmt.Errorf("requirement_id does not belong to this program")
		}
		if input.ControlID != "" && !containsImplementation(program.ControlImplementations, input.ControlID) {
			return MatterAggregate{}, fmt.Errorf("control_id does not belong to this program")
		}
		if canonicalEntityID == "" {
			canonicalEntityID, err = s.resolveLegalEntity(ctx, input.TenantID, program.Program.LegalEntityID)
			if err != nil {
				return MatterAggregate{}, err
			}
			ctx = withCanonicalLegalEntity(ctx, input.TenantID, canonicalEntityID)
		}
		if program.Program.LegalEntityID != canonicalEntityID {
			return MatterAggregate{}, ErrNotFound
		}
	}
	if canonicalEntityID == "" {
		return MatterAggregate{}, ErrNotFound
	}
	input.LegalEntityID = canonicalEntityID
	scope, err := normalizedJSON(input.Scope, `{}`)
	if err != nil {
		return MatterAggregate{}, err
	}
	if _, valid := ParseMatterAccessPolicy(scope); !valid {
		return MatterAggregate{}, fmt.Errorf("scope contains invalid Matter access metadata")
	}
	known, err := normalizedJSON(input.KnownFacts, `{}`)
	if err != nil {
		return MatterAggregate{}, err
	}
	missing, err := normalizedJSON(input.MissingFacts, `[]`)
	if err != nil {
		return MatterAggregate{}, err
	}
	contradictions, err := normalizedJSON(input.Contradictions, `[]`)
	if err != nil {
		return MatterAggregate{}, err
	}
	matterID, err := id.NewUUIDv7()
	if err != nil {
		return MatterAggregate{}, err
	}
	now := s.now().UTC()
	status := MatterDraft
	if input.TriggerType != "" {
		status = MatterInitialReview
	}
	matter := Matter{ID: matterID, TenantID: input.TenantID, LegalEntityID: input.LegalEntityID, Reference: matterReference(matterID), Type: input.Type, Status: status, Priority: input.Priority, Title: strings.TrimSpace(input.Title), Summary: strings.TrimSpace(input.Summary), Scope: scope, SourceType: input.SourceType, SourceID: input.SourceID, TriggerType: input.TriggerType, TriggerID: input.TriggerID, TriggerKey: input.TriggerKey, KnownFacts: known, MissingFacts: missing, Contradictions: contradictions, OwnerPrincipalID: input.OwnerPrincipalID, RequiredAuthority: input.RequiredAuthority, DueAt: input.DueAt, CreatedAt: now, UpdatedAt: now, Version: 1}
	event, err := newEvent(input.TenantID, "MATTER", matter.ID, 1, EventMatterCreated, matter, actorFor(input.ActorID), input.ActorID, now)
	if err != nil {
		return MatterAggregate{}, err
	}
	if input.ProgramID != "" {
		if compoundRepo, ok := s.repo.(CompoundRepository); ok {
			return s.createMatterWithInitialLink(ctx, input, matter, event, compoundRepo)
		}
	}
	if _, err = s.repo.CreateMatter(ctx, matter, event); err != nil {
		return MatterAggregate{}, err
	}
	if input.ProgramID != "" {
		linked, linkErr := s.AddMatterLink(ctx, AddMatterLinkInput{TenantID: input.TenantID, MatterID: matter.ID, ExpectedVersion: matter.Version, ProgramID: input.ProgramID, RequirementID: input.RequirementID, ControlID: input.ControlID, Relationship: "AFFECTS", ActorID: input.ActorID})
		if linkErr != nil {
			return MatterAggregate{}, linkErr
		}
		return linked, nil
	}
	return s.currentMatterOrFallback(ctx, input.TenantID, matter.ID, decorateMatter(MatterAggregate{Matter: matter})), nil
}

func (s *Service) AddMatterLink(ctx context.Context, input AddMatterLinkInput) (MatterAggregate, error) {
	aggregate, err := s.matterForMutation(ctx, input.TenantID, input.MatterID, input.ExpectedVersion)
	if err != nil {
		return MatterAggregate{}, err
	}
	if strings.TrimSpace(input.ProgramID) == "" {
		return MatterAggregate{}, fmt.Errorf("program_id is required")
	}
	program, err := s.GetProgram(ctx, input.TenantID, input.ProgramID)
	if err != nil {
		return MatterAggregate{}, err
	}
	if strings.TrimSpace(aggregate.Matter.LegalEntityID) == "" || aggregate.Matter.LegalEntityID != program.Program.LegalEntityID {
		return MatterAggregate{}, ErrNotFound
	}
	if input.RequirementID != "" && !containsRequirement(program.Requirements, input.RequirementID) {
		return MatterAggregate{}, fmt.Errorf("requirement_id does not belong to this program")
	}
	if input.ControlID != "" && !containsImplementation(program.ControlImplementations, input.ControlID) {
		return MatterAggregate{}, fmt.Errorf("control_id does not belong to this program")
	}
	relationship := strings.ToUpper(strings.TrimSpace(input.Relationship))
	if relationship == "" {
		relationship = "AFFECTS"
	}
	for _, existing := range aggregate.Links {
		if existing.ProgramID == input.ProgramID && existing.RequirementID == input.RequirementID && existing.ControlID == input.ControlID && existing.Relationship == relationship {
			return aggregate, nil
		}
	}
	linkID, err := id.NewUUIDv7()
	if err != nil {
		return MatterAggregate{}, err
	}
	link := MatterLink{ID: linkID, TenantID: input.TenantID, MatterID: input.MatterID, ProgramID: input.ProgramID, RequirementID: input.RequirementID, ControlID: input.ControlID, Relationship: relationship, CreatedAt: s.now().UTC()}
	result, err := s.applyMatterValueAndResult(ctx, aggregate, input.TenantID, input.MatterID, input.ExpectedVersion, EventMatterLinked, link, input.ActorID)
	if err != nil {
		return MatterAggregate{}, err
	}
	_ = s.requestProgramRefresh(ctx, input.TenantID, input.ProgramID, EventMatterLinked, input.MatterID, input.ActorID)
	return result, nil
}

func (s *Service) TransitionMatter(ctx context.Context, input TransitionInput) (MatterAggregate, error) {
	if strings.TrimSpace(input.TenantID) == "" || strings.TrimSpace(input.ID) == "" || input.ExpectedVersion < 1 {
		return MatterAggregate{}, fmt.Errorf("tenant_id, id and expected_version are required")
	}
	aggregate, err := s.GetMatter(ctx, input.TenantID, input.ID)
	if err != nil {
		return MatterAggregate{}, err
	}
	if aggregate.Matter.Version != input.ExpectedVersion {
		return MatterAggregate{}, ErrVersionConflict
	}
	if !allowedMatterTransition(aggregate.Matter.Status, input.To) {
		return MatterAggregate{}, ErrInvalidState
	}
	if (input.To == MatterClosed || input.To == MatterCancelled || aggregate.Matter.Status == MatterClosed) && strings.TrimSpace(input.Rationale) == "" {
		return MatterAggregate{}, fmt.Errorf("rationale is required for closure, cancellation or reopening")
	}
	if (input.To == MatterDecisionRequired || input.To == MatterClosed || input.To == MatterCancelled || aggregate.Matter.Status == MatterClosed) && strings.TrimSpace(input.ActorID) == "" {
		return MatterAggregate{}, fmt.Errorf("actor_id is required for this status change")
	}
	if input.To == MatterCancelled {
		for _, response := range aggregate.ResponsePackages {
			if response.Status == ResponseTransmitted || response.Status == ResponseAcknowledged {
				return MatterAggregate{}, fmt.Errorf("a matter with a transmitted external response cannot be cancelled")
			}
		}
	}
	if input.To == MatterClosed {
		closure := assessClosureAt(aggregate, s.now().UTC())
		if !closure.Ready {
			return MatterAggregate{}, fmt.Errorf("%w: %s", ErrClosureBlocked, strings.Join(closure.Reasons, " "))
		}
	}
	matter := aggregate.Matter
	matter.Status = input.To
	matter.UpdatedAt = s.now().UTC()
	if input.To == MatterClosed {
		closed := matter.UpdatedAt
		matter.ClosedAt = &closed
		matter.ClosureReason = strings.TrimSpace(input.Rationale)
	}
	if aggregate.Matter.Status == MatterClosed && input.To == MatterAssessment {
		matter.ClosedAt = nil
		matter.ClosureReason = ""
		matter.ReopenCount++
	}
	if input.To == MatterCancelled {
		matter.ClosureReason = strings.TrimSpace(input.Rationale)
	}
	result, err := s.applyMatterValueAndResult(ctx, aggregate, input.TenantID, input.ID, input.ExpectedVersion, EventMatterStateChanged, matter, input.ActorID)
	if err != nil {
		return MatterAggregate{}, err
	}
	s.refreshLinkedPrograms(ctx, input.TenantID, input.ID, EventMatterStateChanged)
	return result, nil
}

func (s *Service) AddDecision(ctx context.Context, input AddDecisionInput) (MatterAggregate, error) {
	aggregate, err := s.matterForMutation(ctx, input.TenantID, input.MatterID, input.ExpectedVersion)
	if err != nil {
		return MatterAggregate{}, err
	}
	if strings.TrimSpace(input.Type) == "" || strings.TrimSpace(input.Rationale) == "" || !validDecisionStatus(input.Status) {
		return MatterAggregate{}, fmt.Errorf("type, rationale and supported decision status are required")
	}
	if input.Status == DecisionApproved || input.Status == DecisionConditionallyApproved || input.Status == DecisionRejected {
		if strings.TrimSpace(input.AuthorityPrincipalID) == "" || strings.TrimSpace(input.SelectedOption) == "" {
			return MatterAggregate{}, fmt.Errorf("authority_principal_id and selected_option are required for a decided outcome")
		}
	}
	options, err := normalizedJSON(input.Options, `[]`)
	if err != nil {
		return MatterAggregate{}, err
	}
	conditions, err := normalizedJSON(input.Conditions, `[]`)
	if err != nil {
		return MatterAggregate{}, err
	}
	valueID, err := id.NewUUIDv7()
	if err != nil {
		return MatterAggregate{}, err
	}
	now := s.now().UTC()
	value := Decision{ID: valueID, TenantID: input.TenantID, MatterID: input.MatterID, Type: strings.TrimSpace(input.Type), Status: input.Status, Options: options, SelectedOption: strings.TrimSpace(input.SelectedOption), Rationale: strings.TrimSpace(input.Rationale), Conditions: conditions, AuthorityPrincipalID: input.AuthorityPrincipalID, ExpiresAt: input.ExpiresAt, CreatedAt: now, UpdatedAt: now, Version: 1}
	if input.Status == DecisionApproved || input.Status == DecisionConditionallyApproved || input.Status == DecisionRejected {
		value.DecidedAt = &now
	}
	return s.applyMatterValueAndResult(ctx, aggregate, input.TenantID, input.MatterID, input.ExpectedVersion, EventDecisionAdded, value, input.AuthorityPrincipalID)
}

func (s *Service) AddAction(ctx context.Context, input AddActionInput) (MatterAggregate, error) {
	aggregate, err := s.matterForMutation(ctx, input.TenantID, input.MatterID, input.ExpectedVersion)
	if err != nil {
		return MatterAggregate{}, err
	}
	ownerID := strings.TrimSpace(input.OwnerPrincipalID)
	originKey := strings.TrimSpace(input.OriginKey)
	if strings.TrimSpace(input.Title) == "" || strings.TrimSpace(input.Description) == "" || ownerID == "" {
		return MatterAggregate{}, fmt.Errorf("title, description and owner_principal_id are required")
	}
	if len(originKey) > 160 {
		return MatterAggregate{}, fmt.Errorf("origin_key must not exceed 160 characters")
	}
	if originKey != "" {
		for _, action := range aggregate.Actions {
			if action.OriginKey == originKey {
				return MatterAggregate{}, ErrDuplicate
			}
		}
	}
	valueID, err := id.NewUUIDv7()
	if err != nil {
		return MatterAggregate{}, err
	}
	now := s.now().UTC()
	value := Action{ID: valueID, TenantID: input.TenantID, MatterID: input.MatterID, OriginKey: originKey, Title: strings.TrimSpace(input.Title), Description: strings.TrimSpace(input.Description), OwnerPrincipalID: ownerID, RequiredResponsibility: "PERFORMER", Status: ActionPlanned, DueAt: input.DueAt, CreatedAt: now, UpdatedAt: now, Version: 1}
	return s.applyMatterValueAndResult(ctx, aggregate, input.TenantID, input.MatterID, input.ExpectedVersion, EventActionAdded, value, input.ActorID)
}

func (s *Service) TransitionAction(ctx context.Context, input TransitionActionInput) (MatterAggregate, error) {
	aggregate, err := s.matterForMutation(ctx, input.TenantID, input.MatterID, input.ExpectedVersion)
	if err != nil {
		return MatterAggregate{}, err
	}
	var action *Action
	for index := range aggregate.Actions {
		if aggregate.Actions[index].ID == input.ActionID {
			action = &aggregate.Actions[index]
			break
		}
	}
	if action == nil {
		return MatterAggregate{}, ErrNotFound
	}
	if !allowedActionTransition(action.Status, input.To) {
		return MatterAggregate{}, ErrInvalidState
	}
	if (input.To == ActionBlocked || input.To == ActionCancelled) && strings.TrimSpace(input.Rationale) == "" {
		return MatterAggregate{}, fmt.Errorf("rationale is required when an action is blocked or cancelled")
	}
	action.Status = input.To
	action.UpdatedAt = s.now().UTC()
	action.Version++
	if input.To == ActionImplemented {
		implemented := action.UpdatedAt
		action.ImplementedAt = &implemented
	}
	return s.applyMatterValueAndResult(ctx, aggregate, input.TenantID, input.MatterID, input.ExpectedVersion, EventActionStateChanged, *action, input.ActorID)
}

func (s *Service) AddVerificationContract(ctx context.Context, input AddVerificationContractInput) (MatterAggregate, error) {
	aggregate, err := s.matterForMutation(ctx, input.TenantID, input.MatterID, input.ExpectedVersion)
	if err != nil {
		return MatterAggregate{}, err
	}
	if strings.TrimSpace(input.ExpectedOutcome) == "" || strings.TrimSpace(input.FailureResponse) == "" {
		return MatterAggregate{}, fmt.Errorf("expected_outcome and failure_response are required")
	}
	if input.ActionID != "" && !containsAction(aggregate.Actions, input.ActionID) {
		return MatterAggregate{}, fmt.Errorf("action_id does not belong to this matter")
	}
	if !validFailureResponse(input.FailureResponse) {
		return MatterAggregate{}, fmt.Errorf("failure_response must be REOPEN, CREATE_MATTER, ESCALATE or BLOCK_CLOSE")
	}
	if input.ObservationPeriodMinutes < 0 || input.ObservationPeriodMinutes > 525600 {
		return MatterAggregate{}, fmt.Errorf("observation_period_minutes is outside the supported range")
	}
	if err = s.validateEvidenceSources(ctx, input.TenantID, aggregate.Matter.LegalEntityID, []string{input.MeasurementSourceID}); err != nil {
		return MatterAggregate{}, err
	}
	baseline, err := normalizedJSON(input.Baseline, `{}`)
	if err != nil {
		return MatterAggregate{}, err
	}
	scope, err := normalizedJSON(input.Scope, `{}`)
	if err != nil {
		return MatterAggregate{}, err
	}
	threshold, err := normalizedJSON(input.Threshold, `{}`)
	if err != nil {
		return MatterAggregate{}, err
	}
	valueID, err := id.NewUUIDv7()
	if err != nil {
		return MatterAggregate{}, err
	}
	now := s.now().UTC()
	value := VerificationContract{ID: valueID, TenantID: input.TenantID, MatterID: input.MatterID, ActionID: input.ActionID, ExpectedOutcome: strings.TrimSpace(input.ExpectedOutcome), Baseline: baseline, Scope: scope, MeasurementSourceID: input.MeasurementSourceID, Threshold: threshold, ObservationPeriodMinutes: input.ObservationPeriodMinutes, AuthorityPrincipalID: input.AuthorityPrincipalID, FailureResponse: strings.ToUpper(strings.TrimSpace(input.FailureResponse)), Status: VerificationActive, CreatedAt: now, UpdatedAt: now, Version: 1}
	return s.applyMatterValueAndResult(ctx, aggregate, input.TenantID, input.MatterID, input.ExpectedVersion, EventVerificationContractAdded, value, input.ActorID)
}

func (s *Service) RecordVerificationResult(ctx context.Context, input RecordVerificationResultInput) (MatterAggregate, error) {
	return s.recordVerificationResult(ctx, input)
}

func (s *Service) AddResponsePackage(ctx context.Context, input AddResponsePackageInput) (MatterAggregate, error) {
	aggregate, err := s.matterForMutation(ctx, input.TenantID, input.MatterID, input.ExpectedVersion)
	if err != nil {
		return MatterAggregate{}, err
	}
	if strings.TrimSpace(input.Purpose) == "" || strings.TrimSpace(input.Audience) == "" {
		return MatterAggregate{}, fmt.Errorf("purpose and audience are required")
	}
	manifest, err := normalizedJSON(input.Manifest, `[]`)
	if err != nil {
		return MatterAggregate{}, err
	}
	valueID, err := id.NewUUIDv7()
	if err != nil {
		return MatterAggregate{}, err
	}
	now := s.now().UTC()
	value := ResponsePackage{ID: valueID, TenantID: input.TenantID, MatterID: input.MatterID, Purpose: strings.TrimSpace(input.Purpose), Audience: strings.TrimSpace(input.Audience), Status: ResponseDraft, Manifest: manifest, CreatedAt: now, UpdatedAt: now, Version: 1}
	return s.applyMatterValueAndResult(ctx, aggregate, input.TenantID, input.MatterID, input.ExpectedVersion, EventResponsePackageAdded, value, input.ActorID)
}

func (s *Service) TransitionResponsePackage(ctx context.Context, input TransitionResponseInput) (MatterAggregate, error) {
	aggregate, err := s.matterForMutation(ctx, input.TenantID, input.MatterID, input.ExpectedVersion)
	if err != nil {
		return MatterAggregate{}, err
	}
	var response *ResponsePackage
	for index := range aggregate.ResponsePackages {
		if aggregate.ResponsePackages[index].ID == input.ResponseID {
			response = &aggregate.ResponsePackages[index]
			break
		}
	}
	if response == nil {
		return MatterAggregate{}, ErrNotFound
	}
	if !allowedResponseTransition(response.Status, input.To) {
		return MatterAggregate{}, ErrInvalidState
	}
	if (input.To == ResponseApproved || input.To == ResponseRejected || input.To == ResponseWithdrawn) && strings.TrimSpace(input.ActorID) == "" {
		return MatterAggregate{}, fmt.Errorf("actor_id is required for approval, rejection or withdrawal")
	}
	response.Status = input.To
	response.UpdatedAt = s.now().UTC()
	response.Version++
	switch input.To {
	case ResponseApproved:
		response.ApprovedBy = input.ActorID
	case ResponseTransmitted:
		value := response.UpdatedAt
		response.TransmittedAt = &value
	case ResponseAcknowledged:
		value := response.UpdatedAt
		response.AcknowledgedAt = &value
	}
	return s.applyMatterValueAndResult(ctx, aggregate, input.TenantID, input.MatterID, input.ExpectedVersion, EventResponsePackageStateChanged, *response, input.ActorID)
}

func (s *Service) MatterAt(ctx context.Context, tenant, id string, at time.Time) (MatterAggregate, error) {
	events, err := s.repo.MatterEvents(ctx, tenant, id, &at)
	if err != nil {
		return MatterAggregate{}, err
	}
	return reconstructMatter(events)
}

func (s *Service) ResponsePackageHistory(ctx context.Context, tenant, matterID, responseID string, limit int) (ResponseHistoryPage, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	items, hasMore, err := s.repo.ResponsePackageHistory(ctx, tenant, matterID, responseID, limit)
	if err != nil {
		return ResponseHistoryPage{}, err
	}
	return ResponseHistoryPage{Items: items, HasMore: hasMore, GeneratedAt: s.now().UTC()}, nil
}

func (s *Service) applyProgramValue(ctx context.Context, tenant, programID string, expected int64, eventType string, value any, actorID string) error {
	event, err := newEvent(tenant, "PROGRAM", programID, expected+1, eventType, value, actorFor(actorID), actorID, s.now().UTC())
	if err != nil {
		return err
	}
	_, err = s.repo.ApplyProgramEvent(ctx, tenant, programID, expected, event)
	return err
}

func (s *Service) applyProgramValueAndResult(ctx context.Context, fallback ProgramAggregate, tenant, programID string, expected int64, eventType string, value any, actorID, triggerID string) (ProgramAggregate, error) {
	event, err := newEvent(tenant, "PROGRAM", programID, expected+1, eventType, value, actorFor(actorID), actorID, s.now().UTC())
	if err != nil {
		return ProgramAggregate{}, err
	}
	committed := fallback
	if err = applyProgramEventToAggregate(&committed, event); err != nil {
		return ProgramAggregate{}, err
	}
	committed.Program.Version = event.AggregateVersion
	committed.Program.UpdatedAt = event.OccurredAt
	if _, err = s.repo.ApplyProgramEvent(ctx, tenant, programID, expected, event); err != nil {
		return ProgramAggregate{}, err
	}
	_ = s.requestProgramRefresh(ctx, tenant, programID, eventType, triggerID, "system")
	if current, readErr := s.repo.GetProgram(ctx, tenant, programID); readErr == nil {
		return current, nil
	}
	return decorateProgram(committed), nil
}

func (s *Service) applyMatterValue(ctx context.Context, tenant, matterID string, expected int64, eventType string, value any, actorID string) error {
	event, err := newEvent(tenant, "MATTER", matterID, expected+1, eventType, value, actorFor(actorID), actorID, s.now().UTC())
	if err != nil {
		return err
	}
	_, err = s.repo.ApplyMatterEvent(ctx, tenant, matterID, expected, event)
	return err
}

func (s *Service) refreshAndGetProgram(ctx context.Context, tenant, programID, triggerType, triggerID string) (ProgramAggregate, error) {
	if err := s.requestProgramRefresh(ctx, tenant, programID, triggerType, triggerID, "system"); err != nil {
		return ProgramAggregate{}, err
	}
	return s.repo.GetProgram(ctx, tenant, programID)
}

func (s *Service) refreshProgram(ctx context.Context, tenant, programID, triggerType, triggerID string) error {
	aggregate, err := s.repo.GetProgram(ctx, tenant, programID)
	if err != nil {
		return err
	}
	openMatters, err := s.repo.OpenMatterCount(ctx, tenant, programID)
	if err != nil {
		return err
	}
	state := deriveProgramState(aggregate, openMatters, s.now().UTC())
	state.ID, err = id.NewUUIDv7()
	if err != nil {
		return err
	}
	state.TriggerType = triggerType
	state.TriggerID = triggerID
	state.ProgramVersion = aggregate.Program.Version
	if aggregate.CurrentState != nil && stateEquivalent(*aggregate.CurrentState, state) && aggregate.CurrentState.ProgramVersion == aggregate.Program.Version {
		return nil
	}
	if projectionRepo, ok := s.repo.(ProgramStateRepository); ok {
		_, err = projectionRepo.SaveProgramState(ctx, tenant, programID, aggregate.Program.Version, state)
		return err
	}
	event, err := newEvent(tenant, "PROGRAM", programID, aggregate.Program.Version+1, EventProgramStateUpdated, state, ActorSystem, "", s.now().UTC())
	if err != nil {
		return err
	}
	_, err = s.repo.ApplyProgramEvent(ctx, tenant, programID, aggregate.Program.Version, event)
	return err
}

func (s *Service) refreshLinkedPrograms(ctx context.Context, tenant, matterID, triggerType string) {
	programIDs, err := s.repo.LinkedProgramIDs(ctx, tenant, matterID)
	if err != nil {
		return
	}
	for _, programID := range programIDs {
		_ = s.requestProgramRefresh(ctx, tenant, programID, triggerType, matterID, "system")
	}
}

func stateEquivalent(left, right ProgramStateSnapshot) bool {
	return left.Overall == right.Overall && reflect.DeepEqual(left.Dimensions, right.Dimensions) && reflect.DeepEqual(left.Reasons, right.Reasons) && left.OpenMatterCount == right.OpenMatterCount
}

func newEvent(tenant, aggregateType, aggregateID string, version int64, eventType string, value any, actorType ActorType, actorID string, occurredAt time.Time) (Event, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return Event{}, err
	}
	eventID, err := id.NewUUIDv7()
	if err != nil {
		return Event{}, err
	}
	return Event{ID: eventID, TenantID: tenant, AggregateType: aggregateType, AggregateID: aggregateID, AggregateVersion: version, Type: eventType, Payload: payload, ActorType: actorType, ActorID: actorID, OccurredAt: occurredAt.UTC()}, nil
}

func actorFor(actorID string) ActorType {
	if strings.TrimSpace(actorID) == "" {
		return ActorSystem
	}
	return ActorPerson
}

func normalizedJSON(value json.RawMessage, fallback string) (json.RawMessage, error) {
	if len(bytes.TrimSpace(value)) == 0 {
		value = json.RawMessage(fallback)
	}
	if !json.Valid(value) {
		return nil, fmt.Errorf("value must be valid JSON")
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, value); err != nil {
		return nil, err
	}
	return json.RawMessage(compact.Bytes()), nil
}

func (s *Service) programForMutation(ctx context.Context, tenant, programID string, expected int64) (ProgramAggregate, error) {
	if err := validateProgramMutation(tenant, programID, expected); err != nil {
		return ProgramAggregate{}, err
	}
	aggregate, err := s.GetProgram(ctx, tenant, programID)
	if err != nil {
		return ProgramAggregate{}, err
	}
	if aggregate.Program.Version != expected {
		return ProgramAggregate{}, ErrVersionConflict
	}
	if aggregate.Program.Status == ProgramRetired {
		return ProgramAggregate{}, ErrInvalidState
	}
	return aggregate, nil
}

func (s *Service) matterForMutation(ctx context.Context, tenant, matterID string, expected int64) (MatterAggregate, error) {
	if err := validateMatterMutation(tenant, matterID, expected); err != nil {
		return MatterAggregate{}, err
	}
	aggregate, err := s.GetMatter(ctx, tenant, matterID)
	if err != nil {
		return MatterAggregate{}, err
	}
	if aggregate.Matter.Version != expected {
		return MatterAggregate{}, ErrVersionConflict
	}
	if aggregate.Matter.Status == MatterClosed || aggregate.Matter.Status == MatterCancelled {
		return MatterAggregate{}, ErrInvalidState
	}
	return aggregate, nil
}

func validateProgramMutation(tenant, programID string, expected int64) error {
	if strings.TrimSpace(tenant) == "" || strings.TrimSpace(programID) == "" || expected < 1 {
		return fmt.Errorf("tenant_id, program_id and positive expected_version are required")
	}
	return nil
}

func validateMatterMutation(tenant, matterID string, expected int64) error {
	if strings.TrimSpace(tenant) == "" || strings.TrimSpace(matterID) == "" || expected < 1 {
		return fmt.Errorf("tenant_id, matter_id and positive expected_version are required")
	}
	return nil
}

func boundedLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func matterReference(value string) string {
	clean := strings.ToUpper(strings.ReplaceAll(value, "-", ""))
	// UUIDv7 prefixes are time-dominant and collide for records created in the
	// same millisecond. Use the entropy-bearing suffix for human references.
	if len(clean) > 16 {
		clean = clean[len(clean)-16:]
	}
	return "MAT-" + clean
}

func containsRequirement(values []Requirement, id string) bool {
	for _, value := range values {
		if value.ID == id {
			return true
		}
	}
	return false
}
func containsObjective(values []ControlObjective, id string) bool {
	for _, value := range values {
		if value.ID == id {
			return true
		}
	}
	return false
}
func containsImplementation(values []ControlImplementation, id string) bool {
	for _, value := range values {
		if value.ID == id {
			return true
		}
	}
	return false
}
func containsEvidenceContract(values []EvidenceContract, id string) bool {
	for _, value := range values {
		if value.ID == id {
			return true
		}
	}
	return false
}
func containsAction(values []Action, id string) bool {
	for _, value := range values {
		if value.ID == id {
			return true
		}
	}
	return false
}
func containsVerificationContract(values []VerificationContract, id string) bool {
	for _, value := range values {
		if value.ID == id {
			return true
		}
	}
	return false
}
func validFailureResponse(value string) bool {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "REOPEN", "CREATE_MATTER", "ESCALATE", "BLOCK_CLOSE":
		return true
	default:
		return false
	}
}

func allowedProgramTransition(from, to ProgramStatus) bool {
	allowed := map[ProgramStatus]map[ProgramStatus]bool{
		ProgramDraft:  {ProgramActive: true, ProgramRetired: true},
		ProgramActive: {ProgramPaused: true, ProgramRetired: true},
		ProgramPaused: {ProgramActive: true, ProgramRetired: true},
	}
	return allowed[from][to]
}

func AllowedProgramTargets(from ProgramStatus) []ProgramStatus {
	candidates := []ProgramStatus{ProgramDraft, ProgramActive, ProgramPaused, ProgramRetired}
	values := make([]ProgramStatus, 0, len(candidates))
	for _, candidate := range candidates {
		if allowedProgramTransition(from, candidate) {
			values = append(values, candidate)
		}
	}
	return values
}

func validApplicability(value ApplicabilityStatus) bool {
	switch value {
	case ApplicabilityPotential, ApplicabilityApplicable, ApplicabilityPartial, ApplicabilityNotApplicable, ApplicabilityLater, ApplicabilitySuperseded:
		return true
	default:
		return false
	}
}

func validEvidenceConclusion(value EvidenceConclusion) bool {
	switch value {
	case EvidenceSupported, EvidencePartiallySupported, EvidenceUnsupported, EvidenceContradicted, EvidenceIndeterminate, EvidenceExpired:
		return true
	default:
		return false
	}
}

func validMatterType(value MatterType) bool {
	switch value {
	case MatterRegulatoryChange, MatterSupervisoryFinding, MatterAuthorityRequest, MatterRiskSituation, MatterControlGap, MatterAuditFinding, MatterException, MatterIncident, MatterOperationalLoss, MatterDataBreach, MatterVendorReview, MatterVendorDeficiency, MatterCustomerConcern, MatterOverdueObligation, MatterFailedVerification, MatterEvidenceContradiction, MatterKRIBreach:
		return true
	default:
		return false
	}
}

func validDecisionStatus(value DecisionStatus) bool {
	switch value {
	case DecisionProposed, DecisionApproved, DecisionConditionallyApproved, DecisionRejected, DecisionReturned:
		return true
	default:
		return false
	}
}

func validVerificationResult(value VerificationResultStatus) bool {
	switch value {
	case VerificationPassed, VerificationFailed, VerificationInconclusive:
		return true
	default:
		return false
	}
}

func validRequirementStatus(value RequirementStatus) bool {
	switch value {
	case RequirementDraft, RequirementApproved, RequirementSuperseded, RequirementRetired:
		return true
	default:
		return false
	}
}

func validModality(value string) bool {
	switch value {
	case "MUST", "MUST_NOT", "MAY", "SHOULD", "EXPECTED":
		return true
	default:
		return false
	}
}

func validObjectiveStatus(value ControlObjectiveStatus) bool {
	switch value {
	case ObjectiveDraft, ObjectiveActive, ObjectiveRetired:
		return true
	default:
		return false
	}
}

func validImplementationStatus(value ControlImplementationStatus) bool {
	switch value {
	case ImplementationPlanned, ImplementationInProgress, ImplementationImplemented, ImplementationInactive, ImplementationRetired:
		return true
	default:
		return false
	}
}

func validEvidenceContractStatus(value EvidenceContractStatus) bool {
	switch value {
	case EvidenceContractDraft, EvidenceContractActive, EvidenceContractRetired:
		return true
	default:
		return false
	}
}

func validContradictionPolicy(value string) bool {
	return value == "HOLD" || value == "REVIEW" || value == "FAIL"
}

func validEvidenceFailureAction(value string) bool {
	return value == "MATTER"
}

func evidenceContractByID(values []EvidenceContract, id string) (EvidenceContract, bool) {
	for _, value := range values {
		if value.ID == id {
			return value, true
		}
	}
	return EvidenceContract{}, false
}

func implementationByID(values []ControlImplementation, id string) (ControlImplementation, bool) {
	for _, value := range values {
		if value.ID == id {
			return value, true
		}
	}
	return ControlImplementation{}, false
}

func verificationContractByID(values []VerificationContract, id string) (VerificationContract, bool) {
	for _, value := range values {
		if value.ID == id {
			return value, true
		}
	}
	return VerificationContract{}, false
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func matterForTrigger(trigger Trigger) (MatterType, string, string, bool) {
	switch strings.ToUpper(trigger.Type) {
	case "REQUIREMENT_CHANGED":
		return MatterRegulatoryChange, "Review changed requirement", "A requirement changed and its effect on this program needs review.", true
	case "SOURCE_DEGRADED":
		return MatterControlGap, "Restore an unavailable source", "A source used by this program is stale, degraded or unavailable.", true
	case "EVIDENCE_EXPIRED":
		return MatterOverdueObligation, "Replace out-of-date evidence", "Evidence used by this program is no longer current.", true
	case "EVIDENCE_CONTRADICTION":
		return MatterEvidenceContradiction, "Resolve conflicting evidence", "Current evidence contains values that do not agree.", true
	case "CONTROL_FAILED":
		return MatterControlGap, "Resolve a failed control", "A control did not operate as expected.", true
	case "MONITORING_RESULT_ADVERSE":
		return MatterControlGap, "Review an adverse monitoring result", "The latest monitoring result requires control assurance review and follow-up.", true
	case "VERIFICATION_FAILED":
		return MatterFailedVerification, "Complete further remediation", "The latest outcome check did not pass.", true
	case "DEADLINE_MISSED":
		return MatterOverdueObligation, "Complete an overdue obligation", "A required activity or response passed its due date.", true
	default:
		return "", "", "", false
	}
}

func triggerPriority(triggerType string) int {
	switch strings.ToUpper(triggerType) {
	case "VERIFICATION_FAILED", "CONTROL_FAILED", "MONITORING_RESULT_ADVERSE", "DEADLINE_MISSED":
		return 4
	case "EVIDENCE_CONTRADICTION", "SOURCE_DEGRADED":
		return 3
	default:
		return 2
	}
}
