package continuity

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	platformid "github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

// FrameworkClauseReference identifies one immutable clause within one edition.
// Published clauses are attribution references, not Program scoring units.
type FrameworkClauseReference struct {
	ID        string
	EditionID string
}

type FrameworkClauseLookup interface {
	GetFrameworkClause(context.Context, string, string) (FrameworkClauseReference, error)
}

type FrameworkMappingReader interface {
	GetFrameworkMapping(context.Context, string, string, string) (FrameworkClauseMapping, error)
}

// FrameworkMappingListReader supplies one bounded, scoped review queue without
// loading clause mappings into the continuously refreshed Program aggregate.
type FrameworkMappingListReader interface {
	ListFrameworkMappings(context.Context, string, string, string, string, int) ([]FrameworkClauseMapping, error)
}

type FrameworkMappingStatus string

const (
	FrameworkMappingPending  FrameworkMappingStatus = "PENDING"
	FrameworkMappingApproved FrameworkMappingStatus = "APPROVED"
	FrameworkMappingRejected FrameworkMappingStatus = "REJECTED"
)

type FrameworkMappingKind string

const (
	FrameworkMappingExact      FrameworkMappingKind = "EXACT"
	FrameworkMappingPartial    FrameworkMappingKind = "PARTIAL"
	FrameworkMappingSupporting FrameworkMappingKind = "SUPPORTING"
)

type FrameworkClauseMapping struct {
	ID            string                 `json:"id"`
	TenantID      string                 `json:"tenant_id"`
	LegalEntityID string                 `json:"legal_entity_id"`
	ProgramID     string                 `json:"program_id"`
	AdoptionID    string                 `json:"adoption_id"`
	EditionID     string                 `json:"edition_id"`
	ClauseID      string                 `json:"clause_id"`
	ClauseCode    string                 `json:"clause_code,omitempty"`
	RequirementID string                 `json:"requirement_id"`
	MappingKind   FrameworkMappingKind   `json:"mapping_kind"`
	Rationale     string                 `json:"rationale"`
	MappedBy      string                 `json:"mapped_by"`
	ReviewedBy    string                 `json:"reviewed_by,omitempty"`
	ProposedAt    time.Time              `json:"proposed_at"`
	ReviewedAt    *time.Time             `json:"reviewed_at,omitempty"`
	EffectiveFrom *time.Time             `json:"effective_from,omitempty"`
	Status        FrameworkMappingStatus `json:"status"`
	Version       int64                  `json:"version"`
}

type ProposeFrameworkMappingInput struct {
	TenantID        string               `json:"tenant_id"`
	ProgramID       string               `json:"program_id"`
	AdoptionID      string               `json:"adoption_id"`
	ClauseID        string               `json:"clause_id"`
	RequirementID   string               `json:"requirement_id"`
	MappingKind     FrameworkMappingKind `json:"mapping_kind"`
	ExpectedVersion int64                `json:"expected_version"`
	ActorID         string               `json:"actor_id"`
	Rationale       string               `json:"rationale"`
}

type DecideFrameworkMappingInput struct {
	AdoptionID      string `json:"adoption_id,omitempty"`
	TenantID        string `json:"tenant_id"`
	ProgramID       string `json:"program_id"`
	MappingID       string `json:"mapping_id"`
	ExpectedVersion int64  `json:"expected_version"`
	ActorID         string `json:"actor_id"`
	Rationale       string `json:"rationale"`
}

type frameworkMappingDecision struct {
	Mapping   FrameworkClauseMapping `json:"mapping"`
	Rationale string                 `json:"rationale"`
}

// ProposeFrameworkMapping creates a review-only attribution reference through
// the canonical Program event/outbox. It never inserts a scored Requirement.
func (s *Service) ProposeFrameworkMapping(ctx context.Context, input ProposeFrameworkMappingInput) (FrameworkClauseMapping, ProgramAggregate, error) {
	program, err := s.programForMutation(ctx, input.TenantID, input.ProgramID, input.ExpectedVersion)
	if err != nil {
		return FrameworkClauseMapping{}, ProgramAggregate{}, err
	}
	maker := strings.TrimSpace(input.ActorID)
	if !programComplianceActorMatches(ctx, input.TenantID, maker) ||
		maker != program.Program.OwnerPrincipalID ||
		!validFrameworkAdoptionRationale(strings.TrimSpace(input.Rationale)) ||
		!validFrameworkMappingKind(input.MappingKind) {
		return FrameworkClauseMapping{}, ProgramAggregate{}, ErrInvalidState
	}
	var activeEdition string
	for _, adoption := range program.FrameworkAdoptions {
		if adoption.ID == input.AdoptionID && adoption.Status == FrameworkAdoptionActive &&
			adoption.LegalEntityID == program.Program.LegalEntityID {
			activeEdition = adoption.EditionID
			break
		}
	}
	now := s.now().UTC()
	if activeEdition == "" || !approvedFrameworkRequirement(program.Requirements, input.RequirementID, now) {
		return FrameworkClauseMapping{}, ProgramAggregate{}, ErrInvalidState
	}
	resolver, ok := s.frameworkEditionLookup.(FrameworkClauseLookup)
	if !ok {
		return FrameworkClauseMapping{}, ProgramAggregate{}, ErrFrameworkEditionUnavailable
	}
	clause, err := resolver.GetFrameworkClause(ctx, activeEdition, input.ClauseID)
	if err != nil {
		return FrameworkClauseMapping{}, ProgramAggregate{}, err
	}
	if clause.ID != input.ClauseID || clause.EditionID != activeEdition {
		return FrameworkClauseMapping{}, ProgramAggregate{}, ErrInvalidState
	}
	id, err := platformid.NewUUIDv7()
	if err != nil {
		return FrameworkClauseMapping{}, ProgramAggregate{}, err
	}
	proposal := FrameworkClauseMapping{
		ID: id, TenantID: program.Program.TenantID, LegalEntityID: program.Program.LegalEntityID,
		ProgramID: program.Program.ID, AdoptionID: input.AdoptionID, EditionID: activeEdition,
		ClauseID: clause.ID, RequirementID: input.RequirementID, MappingKind: input.MappingKind,
		Rationale: strings.TrimSpace(input.Rationale), MappedBy: maker,
		ProposedAt: now, Status: FrameworkMappingPending, Version: 1,
	}
	result, err := s.applyProgramValueAndResult(ctx, program, input.TenantID, input.ProgramID,
		input.ExpectedVersion, EventFrameworkMappingProposed, proposal, maker, proposal.ID)
	if err != nil {
		return FrameworkClauseMapping{}, ProgramAggregate{}, err
	}
	return proposal, result, nil
}

func (s *Service) ApproveFrameworkMapping(ctx context.Context, input DecideFrameworkMappingInput) (FrameworkClauseMapping, ProgramAggregate, error) {
	return s.decideFrameworkMapping(ctx, input, true)
}

func (s *Service) RejectFrameworkMapping(ctx context.Context, input DecideFrameworkMappingInput) (FrameworkClauseMapping, ProgramAggregate, error) {
	return s.decideFrameworkMapping(ctx, input, false)
}

func (s *Service) decideFrameworkMapping(ctx context.Context, input DecideFrameworkMappingInput, approve bool) (FrameworkClauseMapping, ProgramAggregate, error) {
	program, err := s.programForMutation(ctx, input.TenantID, input.ProgramID, input.ExpectedVersion)
	if err != nil {
		return FrameworkClauseMapping{}, ProgramAggregate{}, err
	}
	checker := strings.TrimSpace(input.ActorID)
	if !programComplianceActorMatches(ctx, input.TenantID, checker) ||
		checker != program.Program.AuthorityPrincipalID ||
		!validFrameworkAdoptionRationale(strings.TrimSpace(input.Rationale)) {
		return FrameworkClauseMapping{}, ProgramAggregate{}, ErrInvalidState
	}
	reader, ok := s.repo.(FrameworkMappingReader)
	if !ok {
		return FrameworkClauseMapping{}, ProgramAggregate{}, ErrFrameworkEditionUnavailable
	}
	current, err := reader.GetFrameworkMapping(ctx, input.TenantID, input.ProgramID, input.MappingID)
	if err != nil {
		return FrameworkClauseMapping{}, ProgramAggregate{}, err
	}
	if current.Status != FrameworkMappingPending || current.Version != 1 ||
		current.MappedBy == checker || current.LegalEntityID != program.Program.LegalEntityID ||
		(input.AdoptionID != "" && current.AdoptionID != input.AdoptionID) {
		return FrameworkClauseMapping{}, ProgramAggregate{}, ErrInvalidState
	}
	now := s.now().UTC()
	if now.Before(current.ProposedAt) {
		return FrameworkClauseMapping{}, ProgramAggregate{}, ErrInvalidState
	}
	foundAdoption := false
	for _, adoption := range program.FrameworkAdoptions {
		if adoption.ID == current.AdoptionID && adoption.EditionID == current.EditionID &&
			adoption.Status == FrameworkAdoptionActive {
			foundAdoption = true
			break
		}
	}
	if approve && (!foundAdoption || !approvedFrameworkRequirement(program.Requirements, current.RequirementID, now)) {
		return FrameworkClauseMapping{}, ProgramAggregate{}, ErrInvalidState
	}
	current.ReviewedBy = checker
	current.ReviewedAt = &now
	current.Version++
	eventType := EventFrameworkMappingRejected
	current.Status = FrameworkMappingRejected
	if approve {
		current.Status = FrameworkMappingApproved
		current.EffectiveFrom = &now
		eventType = EventFrameworkMappingApproved
	}
	result, err := s.applyProgramValueAndResult(ctx, program, input.TenantID, input.ProgramID,
		input.ExpectedVersion, eventType,
		frameworkMappingDecision{Mapping: current, Rationale: strings.TrimSpace(input.Rationale)}, checker, current.ID)
	if err != nil {
		return FrameworkClauseMapping{}, ProgramAggregate{}, err
	}
	return current, result, nil
}

func validFrameworkMappingKind(value FrameworkMappingKind) bool {
	return value == FrameworkMappingExact || value == FrameworkMappingPartial || value == FrameworkMappingSupporting
}

func approvedFrameworkRequirement(requirements []Requirement, id string, at time.Time) bool {
	for _, requirement := range requirements {
		if requirement.ID == id && requirement.Status == RequirementApproved &&
			!at.Before(requirement.EffectiveFrom) &&
			(requirement.EffectiveUntil == nil || at.Before(*requirement.EffectiveUntil)) {
			return true
		}
	}
	return false
}

// FrameworkMappingAt reconstructs a single mapping from immutable Program
// history. This is an audit read, not an unbounded addition to Program lists.
func (s *Service) FrameworkMappingAt(ctx context.Context, tenant, programID, mappingID string, at time.Time) (FrameworkClauseMapping, error) {
	if at.IsZero() {
		return FrameworkClauseMapping{}, ErrInvalidState
	}
	events, err := s.repo.ProgramEvents(ctx, tenant, programID, &at)
	if err != nil {
		return FrameworkClauseMapping{}, err
	}
	var result FrameworkClauseMapping
	for _, event := range events {
		if event.Type != EventFrameworkMappingProposed &&
			event.Type != EventFrameworkMappingApproved &&
			event.Type != EventFrameworkMappingRejected {
			continue
		}
		value, err := frameworkMappingFromEvent(event)
		if err != nil {
			return FrameworkClauseMapping{}, err
		}
		if value.ID == mappingID {
			result = value
		}
	}
	if result.ID == "" {
		return FrameworkClauseMapping{}, ErrNotFound
	}
	return result, nil
}

func frameworkMappingFromEvent(event Event) (FrameworkClauseMapping, error) {
	if event.Type == EventFrameworkMappingProposed {
		var value FrameworkClauseMapping
		if err := json.Unmarshal(event.Payload, &value); err != nil {
			return FrameworkClauseMapping{}, err
		}
		return value, nil
	}
	var value frameworkMappingDecision
	if err := json.Unmarshal(event.Payload, &value); err != nil {
		return FrameworkClauseMapping{}, err
	}
	return value.Mapping, nil
}

func (s *Service) ListFrameworkMappings(
	ctx context.Context, tenantID, programID, adoptionID, afterID string, limit int,
) ([]FrameworkClauseMapping, error) {
	reader, ok := s.repo.(FrameworkMappingListReader)
	if !ok {
		return nil, ErrFrameworkEditionUnavailable
	}
	if limit < 1 || limit > 100 || tenantID == "" || programID == "" || adoptionID == "" {
		return nil, ErrInvalidState
	}
	return reader.ListFrameworkMappings(ctx, tenantID, programID, adoptionID, afterID, limit)
}
