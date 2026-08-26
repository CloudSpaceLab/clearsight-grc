package continuity

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type ReviseControlImplementationInput struct {
	TenantID                      string          `json:"tenant_id"`
	ProgramID                     string          `json:"program_id"`
	ImplementationID              string          `json:"implementation_id"`
	ExpectedVersion               int64           `json:"expected_version"`
	ExpectedImplementationVersion int64           `json:"expected_implementation_version"`
	Name                          string          `json:"name"`
	Description                   string          `json:"description"`
	ImplementationType            string          `json:"implementation_type"`
	Scope                         json.RawMessage `json:"scope"`
	EffectiveFrom                 time.Time       `json:"effective_from"`
	Rationale                     string          `json:"rationale"`
	ActorID                       string          `json:"actor_id,omitempty"`
}

type AssignControlImplementationInput struct {
	TenantID                      string `json:"tenant_id"`
	ProgramID                     string `json:"program_id"`
	ImplementationID              string `json:"implementation_id"`
	ExpectedVersion               int64  `json:"expected_version"`
	ExpectedImplementationVersion int64  `json:"expected_implementation_version"`
	OwnerPrincipalID              string `json:"owner_principal_id"`
	Rationale                     string `json:"rationale"`
	ActorID                       string `json:"actor_id,omitempty"`
}

type TransitionControlImplementationInput struct {
	TenantID                      string                      `json:"tenant_id"`
	ProgramID                     string                      `json:"program_id"`
	ImplementationID              string                      `json:"implementation_id"`
	ExpectedVersion               int64                       `json:"expected_version"`
	ExpectedImplementationVersion int64                       `json:"expected_implementation_version"`
	To                            ControlImplementationStatus `json:"to"`
	Rationale                     string                      `json:"rationale"`
	ActorID                       string                      `json:"actor_id,omitempty"`
}

type ReviseEvidenceContractInput struct {
	TenantID                string          `json:"tenant_id"`
	ProgramID               string          `json:"program_id"`
	ContractID              string          `json:"contract_id"`
	ExpectedVersion         int64           `json:"expected_version"`
	ExpectedContractVersion int64           `json:"expected_contract_version"`
	Name                    string          `json:"name"`
	Claim                   string          `json:"claim"`
	AcceptableSourceIDs     []string        `json:"acceptable_source_ids"`
	PopulationScope         json.RawMessage `json:"population_scope"`
	FreshnessMinutes        int             `json:"freshness_minutes"`
	MinimumCoverage         float64         `json:"minimum_coverage"`
	IndependenceRequired    bool            `json:"independence_required"`
	ContradictionPolicy     string          `json:"contradiction_policy"`
	FailureAction           string          `json:"failure_action"`
	Rationale               string          `json:"rationale"`
	ActorID                 string          `json:"actor_id,omitempty"`
}

type TransitionEvidenceContractInput struct {
	TenantID                string                 `json:"tenant_id"`
	ProgramID               string                 `json:"program_id"`
	ContractID              string                 `json:"contract_id"`
	ExpectedVersion         int64                  `json:"expected_version"`
	ExpectedContractVersion int64                  `json:"expected_contract_version"`
	To                      EvidenceContractStatus `json:"to"`
	Rationale               string                 `json:"rationale"`
	ActorID                 string                 `json:"actor_id,omitempty"`
}

type controlImplementationLifecycleEvent struct {
	Prior     ControlImplementation `json:"prior"`
	Current   ControlImplementation `json:"current"`
	Rationale string                `json:"rationale"`
}

type evidenceContractLifecycleEvent struct {
	Prior     EvidenceContract `json:"prior"`
	Current   EvidenceContract `json:"current"`
	Rationale string           `json:"rationale"`
}

func (s *Service) ReviseControlImplementation(ctx context.Context, input ReviseControlImplementationInput) (ProgramAggregate, error) {
	aggregate, current, err := s.controlImplementationForMutation(ctx, input.TenantID, input.ProgramID, input.ImplementationID, input.ExpectedVersion, input.ExpectedImplementationVersion)
	if err != nil {
		return ProgramAggregate{}, err
	}
	if current.Status == ImplementationRetired {
		return ProgramAggregate{}, ErrInvalidState
	}
	if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Description) == "" || strings.TrimSpace(input.ImplementationType) == "" || strings.TrimSpace(input.Rationale) == "" {
		return ProgramAggregate{}, fmt.Errorf("name, description, implementation_type and rationale are required")
	}
	scope, err := normalizedJSON(input.Scope, `{}`)
	if err != nil {
		return ProgramAggregate{}, err
	}
	if input.EffectiveFrom.IsZero() {
		input.EffectiveFrom = current.EffectiveFrom
	}
	next := current
	next.Name = strings.TrimSpace(input.Name)
	next.Description = strings.TrimSpace(input.Description)
	next.ImplementationType = strings.TrimSpace(input.ImplementationType)
	next.Scope = scope
	next.EffectiveFrom = input.EffectiveFrom.UTC()
	if current.Status == ImplementationImplemented {
		next.Status = ImplementationInProgress
	}
	next.UpdatedAt = s.now().UTC()
	next.Version++
	payload := controlImplementationLifecycleEvent{Prior: current, Current: next, Rationale: strings.TrimSpace(input.Rationale)}
	if err = s.applyProgramValue(ctx, input.TenantID, input.ProgramID, aggregate.Program.Version, EventControlImplementationRevised, payload, input.ActorID); err != nil {
		return ProgramAggregate{}, err
	}
	aggregate.ControlImplementations = upsertImplementation(aggregate.ControlImplementations, next)
	return s.programResourceResult(ctx, input.TenantID, input.ProgramID, aggregate, next.UpdatedAt), nil
}

func (s *Service) AssignControlImplementation(ctx context.Context, input AssignControlImplementationInput) (ProgramAggregate, error) {
	aggregate, current, err := s.controlImplementationForMutation(ctx, input.TenantID, input.ProgramID, input.ImplementationID, input.ExpectedVersion, input.ExpectedImplementationVersion)
	if err != nil {
		return ProgramAggregate{}, err
	}
	if current.Status == ImplementationRetired {
		return ProgramAggregate{}, ErrInvalidState
	}
	if strings.TrimSpace(input.OwnerPrincipalID) == "" || strings.TrimSpace(input.Rationale) == "" {
		return ProgramAggregate{}, fmt.Errorf("owner_principal_id and rationale are required")
	}
	next := current
	next.OwnerPrincipalID = strings.TrimSpace(input.OwnerPrincipalID)
	next.UpdatedAt = s.now().UTC()
	next.Version++
	payload := controlImplementationLifecycleEvent{Prior: current, Current: next, Rationale: strings.TrimSpace(input.Rationale)}
	if err = s.applyProgramValue(ctx, input.TenantID, input.ProgramID, aggregate.Program.Version, EventControlImplementationOwnerChanged, payload, input.ActorID); err != nil {
		return ProgramAggregate{}, err
	}
	aggregate.ControlImplementations = upsertImplementation(aggregate.ControlImplementations, next)
	return s.programResourceResult(ctx, input.TenantID, input.ProgramID, aggregate, next.UpdatedAt), nil
}

func (s *Service) TransitionControlImplementation(ctx context.Context, input TransitionControlImplementationInput) (ProgramAggregate, error) {
	aggregate, current, err := s.controlImplementationForMutation(ctx, input.TenantID, input.ProgramID, input.ImplementationID, input.ExpectedVersion, input.ExpectedImplementationVersion)
	if err != nil {
		return ProgramAggregate{}, err
	}
	if strings.TrimSpace(input.Rationale) == "" || !controlImplementationTransitionAllowed(current.Status, input.To) {
		return ProgramAggregate{}, ErrInvalidState
	}
	next := current
	next.Status = input.To
	next.UpdatedAt = s.now().UTC()
	next.Version++
	if input.To == ImplementationRetired {
		retiredAt := next.UpdatedAt
		next.EffectiveUntil = &retiredAt
	}
	payload := controlImplementationLifecycleEvent{Prior: current, Current: next, Rationale: strings.TrimSpace(input.Rationale)}
	if err = s.applyProgramValue(ctx, input.TenantID, input.ProgramID, aggregate.Program.Version, EventControlImplementationStatusChanged, payload, input.ActorID); err != nil {
		return ProgramAggregate{}, err
	}
	aggregate.ControlImplementations = upsertImplementation(aggregate.ControlImplementations, next)
	return s.programResourceResult(ctx, input.TenantID, input.ProgramID, aggregate, next.UpdatedAt), nil
}

func (s *Service) ReviseEvidenceContract(ctx context.Context, input ReviseEvidenceContractInput) (ProgramAggregate, error) {
	aggregate, current, err := s.evidenceContractForMutation(ctx, input.TenantID, input.ProgramID, input.ContractID, input.ExpectedVersion, input.ExpectedContractVersion)
	if err != nil {
		return ProgramAggregate{}, err
	}
	if current.Status == EvidenceContractRetired {
		return ProgramAggregate{}, ErrInvalidState
	}
	if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Claim) == "" || strings.TrimSpace(input.Rationale) == "" {
		return ProgramAggregate{}, fmt.Errorf("name, claim and rationale are required")
	}
	if input.FreshnessMinutes < 1 || input.FreshnessMinutes > 525600 {
		return ProgramAggregate{}, fmt.Errorf("freshness_minutes must be between 1 and 525600")
	}
	if input.MinimumCoverage < 0 || input.MinimumCoverage > 1 {
		return ProgramAggregate{}, fmt.Errorf("minimum_coverage must be between 0 and 1")
	}
	contradictionPolicy := strings.ToUpper(strings.TrimSpace(input.ContradictionPolicy))
	if !validContradictionPolicy(contradictionPolicy) {
		return ProgramAggregate{}, fmt.Errorf("contradiction_policy must be HOLD, REVIEW or FAIL")
	}
	failureAction := strings.ToUpper(strings.TrimSpace(input.FailureAction))
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
	next := current
	next.Name = strings.TrimSpace(input.Name)
	next.Claim = strings.TrimSpace(input.Claim)
	next.AcceptableSourceIDs = append([]string(nil), input.AcceptableSourceIDs...)
	next.PopulationScope = population
	next.FreshnessMinutes = input.FreshnessMinutes
	next.MinimumCoverage = input.MinimumCoverage
	next.IndependenceRequired = input.IndependenceRequired
	next.ContradictionPolicy = contradictionPolicy
	next.FailureAction = failureAction
	next.ConfiguredBy = strings.TrimSpace(input.ActorID)
	if current.Status == EvidenceContractActive {
		next.Status = EvidenceContractDraft
	}
	next.UpdatedAt = s.now().UTC()
	next.Version++
	payload := evidenceContractLifecycleEvent{Prior: current, Current: next, Rationale: strings.TrimSpace(input.Rationale)}
	if err = s.applyProgramValue(ctx, input.TenantID, input.ProgramID, aggregate.Program.Version, EventEvidenceContractRevised, payload, input.ActorID); err != nil {
		return ProgramAggregate{}, err
	}
	aggregate.EvidenceContracts = upsertEvidenceContract(aggregate.EvidenceContracts, next)
	return s.programResourceResult(ctx, input.TenantID, input.ProgramID, aggregate, next.UpdatedAt), nil
}

func (s *Service) TransitionEvidenceContract(ctx context.Context, input TransitionEvidenceContractInput) (ProgramAggregate, error) {
	aggregate, current, err := s.evidenceContractForMutation(ctx, input.TenantID, input.ProgramID, input.ContractID, input.ExpectedVersion, input.ExpectedContractVersion)
	if err != nil {
		return ProgramAggregate{}, err
	}
	if strings.TrimSpace(input.Rationale) == "" || !evidenceContractTransitionAllowed(current.Status, input.To) {
		return ProgramAggregate{}, ErrInvalidState
	}
	if current.Status == EvidenceContractDraft && input.To == EvidenceContractActive {
		actorID := strings.TrimSpace(input.ActorID)
		if actorID == "" || strings.TrimSpace(current.ConfiguredBy) == "" || actorID == strings.TrimSpace(current.ConfiguredBy) {
			return ProgramAggregate{}, ErrMakerChecker
		}
	}
	next := current
	next.Status = input.To
	next.UpdatedAt = s.now().UTC()
	next.Version++
	payload := evidenceContractLifecycleEvent{Prior: current, Current: next, Rationale: strings.TrimSpace(input.Rationale)}
	if err = s.applyProgramValue(ctx, input.TenantID, input.ProgramID, aggregate.Program.Version, EventEvidenceContractStatusChanged, payload, input.ActorID); err != nil {
		return ProgramAggregate{}, err
	}
	aggregate.EvidenceContracts = upsertEvidenceContract(aggregate.EvidenceContracts, next)
	return s.programResourceResult(ctx, input.TenantID, input.ProgramID, aggregate, next.UpdatedAt), nil
}

func (s *Service) programResourceResult(ctx context.Context, tenantID, programID string, fallback ProgramAggregate, updatedAt time.Time) ProgramAggregate {
	if current, err := s.GetProgram(ctx, tenantID, programID); err == nil {
		return current
	}
	fallback.Program.Version++
	fallback.Program.UpdatedAt = updatedAt.UTC()
	return decorateProgram(fallback)
}

func (s *Service) controlImplementationForMutation(ctx context.Context, tenantID, programID, implementationID string, expectedProgramVersion, expectedImplementationVersion int64) (ProgramAggregate, ControlImplementation, error) {
	aggregate, err := s.programForMutation(ctx, tenantID, programID, expectedProgramVersion)
	if err != nil {
		return ProgramAggregate{}, ControlImplementation{}, err
	}
	current, ok := implementationByID(aggregate.ControlImplementations, implementationID)
	if !ok {
		return ProgramAggregate{}, ControlImplementation{}, ErrNotFound
	}
	if expectedImplementationVersion < 1 || current.Version != expectedImplementationVersion {
		return ProgramAggregate{}, ControlImplementation{}, ErrVersionConflict
	}
	return aggregate, current, nil
}

func (s *Service) evidenceContractForMutation(ctx context.Context, tenantID, programID, contractID string, expectedProgramVersion, expectedContractVersion int64) (ProgramAggregate, EvidenceContract, error) {
	aggregate, err := s.programForMutation(ctx, tenantID, programID, expectedProgramVersion)
	if err != nil {
		return ProgramAggregate{}, EvidenceContract{}, err
	}
	current, ok := evidenceContractByID(aggregate.EvidenceContracts, contractID)
	if !ok {
		return ProgramAggregate{}, EvidenceContract{}, ErrNotFound
	}
	if expectedContractVersion < 1 || current.Version != expectedContractVersion {
		return ProgramAggregate{}, EvidenceContract{}, ErrVersionConflict
	}
	return aggregate, current, nil
}

func controlImplementationTransitionAllowed(from, to ControlImplementationStatus) bool {
	switch from {
	case ImplementationPlanned:
		return to == ImplementationInProgress || to == ImplementationRetired
	case ImplementationInProgress:
		return to == ImplementationImplemented || to == ImplementationInactive || to == ImplementationRetired
	case ImplementationImplemented:
		return to == ImplementationInactive || to == ImplementationRetired
	case ImplementationInactive:
		return to == ImplementationInProgress || to == ImplementationRetired
	default:
		return false
	}
}

func evidenceContractTransitionAllowed(from, to EvidenceContractStatus) bool {
	return (from == EvidenceContractDraft && (to == EvidenceContractActive || to == EvidenceContractRetired)) ||
		(from == EvidenceContractActive && to == EvidenceContractRetired)
}
