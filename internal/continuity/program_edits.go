package continuity

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

type UpdateProgramDetailsInput struct {
	TenantID        string          `json:"tenant_id"`
	ProgramID       string          `json:"program_id"`
	ExpectedVersion int64           `json:"expected_version"`
	Name            string          `json:"name"`
	OwningFunction  string          `json:"owning_function"`
	Jurisdiction    string          `json:"jurisdiction,omitempty"`
	Scope           json.RawMessage `json:"scope"`
	EffectiveFrom   time.Time       `json:"effective_from"`
	EffectiveUntil  *time.Time      `json:"effective_until,omitempty"`
	ActorID         string          `json:"actor_id,omitempty"`
	Rationale       string          `json:"rationale"`
}

type AssignProgramInput struct {
	TenantID         string `json:"tenant_id"`
	ProgramID        string `json:"program_id"`
	ExpectedVersion  int64  `json:"expected_version"`
	OwnerPrincipalID string `json:"owner_principal_id"`
	ActorID          string `json:"actor_id,omitempty"`
	Rationale        string `json:"rationale"`
}

type AssignProgramApprovalAuthorityInput struct {
	TenantID             string `json:"tenant_id"`
	ProgramID            string `json:"program_id"`
	ExpectedVersion      int64  `json:"expected_version"`
	AuthorityPrincipalID string `json:"authority_principal_id"`
	ActorID              string `json:"actor_id,omitempty"`
	Rationale            string `json:"rationale"`
}

type SupersedeRequirementInput struct {
	TenantID        string    `json:"tenant_id"`
	ProgramID       string    `json:"program_id"`
	RequirementID   string    `json:"requirement_id"`
	ExpectedVersion int64     `json:"expected_version"`
	SourceID        string    `json:"source_id,omitempty"`
	Code            string    `json:"code"`
	Title           string    `json:"title"`
	Statement       string    `json:"statement"`
	SourceAnchor    string    `json:"source_anchor"`
	Modality        string    `json:"modality"`
	Actor           string    `json:"actor,omitempty"`
	Action          string    `json:"action,omitempty"`
	Object          string    `json:"object,omitempty"`
	EffectiveFrom   time.Time `json:"effective_from"`
	ActorID         string    `json:"actor_id,omitempty"`
	Rationale       string    `json:"rationale"`
}

type programDetailsUpdatedEvent struct {
	Program   Program `json:"program"`
	Previous  Program `json:"previous"`
	Rationale string  `json:"rationale"`
}

type programOwnerChangedEvent struct {
	Program          Program `json:"program"`
	PreviousOwnerID  string  `json:"previous_owner_principal_id,omitempty"`
	OwnerPrincipalID string  `json:"owner_principal_id"`
	Rationale        string  `json:"rationale"`
}

type programApprovalAuthorityChangedEvent struct {
	Program                      Program `json:"program"`
	PreviousAuthorityPrincipalID string  `json:"previous_authority_principal_id,omitempty"`
	AuthorityPrincipalID         string  `json:"authority_principal_id"`
	Rationale                    string  `json:"rationale"`
}

type requirementSupersededEvent struct {
	Prior       Requirement `json:"prior"`
	Replacement Requirement `json:"replacement"`
	Rationale   string      `json:"rationale"`
}

func (s *Service) UpdateProgramDetails(ctx context.Context, input UpdateProgramDetailsInput) (ProgramAggregate, error) {
	aggregate, err := s.programForMutation(ctx, input.TenantID, input.ProgramID, input.ExpectedVersion)
	if err != nil {
		return ProgramAggregate{}, err
	}
	if strings.TrimSpace(input.ActorID) == "" || strings.TrimSpace(input.Rationale) == "" {
		return ProgramAggregate{}, fmt.Errorf("actor_id and rationale are required")
	}
	if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.OwningFunction) == "" || input.EffectiveFrom.IsZero() {
		return ProgramAggregate{}, fmt.Errorf("name, owning_function and effective_from are required")
	}
	if input.EffectiveUntil != nil && !input.EffectiveFrom.Before(*input.EffectiveUntil) {
		return ProgramAggregate{}, fmt.Errorf("effective_until must be after effective_from")
	}
	scope, err := normalizedJSONObject(input.Scope, `{}`)
	if err != nil {
		return ProgramAggregate{}, fmt.Errorf("scope: %w", err)
	}
	previous := aggregate.Program
	program := previous
	program.Name = strings.TrimSpace(input.Name)
	program.OwningFunction = strings.TrimSpace(input.OwningFunction)
	program.Jurisdiction = strings.TrimSpace(input.Jurisdiction)
	program.Scope = scope
	program.EffectiveFrom = input.EffectiveFrom.UTC()
	program.EffectiveUntil = nil
	if input.EffectiveUntil != nil {
		until := input.EffectiveUntil.UTC()
		program.EffectiveUntil = &until
	}
	if program.Name == previous.Name && program.OwningFunction == previous.OwningFunction && program.Jurisdiction == previous.Jurisdiction && bytes.Equal(program.Scope, previous.Scope) && program.EffectiveFrom.Equal(previous.EffectiveFrom) && sameOptionalTime(program.EffectiveUntil, previous.EffectiveUntil) {
		return ProgramAggregate{}, fmt.Errorf("program details are unchanged")
	}
	program.UpdatedAt = s.now().UTC()
	event := programDetailsUpdatedEvent{Program: program, Previous: previous, Rationale: strings.TrimSpace(input.Rationale)}
	if err := s.applyProgramValue(ctx, input.TenantID, input.ProgramID, input.ExpectedVersion, EventProgramDetailsUpdated, event, input.ActorID); err != nil {
		return ProgramAggregate{}, err
	}
	return s.refreshAndGetProgram(ctx, input.TenantID, input.ProgramID, EventProgramDetailsUpdated, input.ProgramID)
}

func (s *Service) AssignProgram(ctx context.Context, input AssignProgramInput) (ProgramAggregate, error) {
	aggregate, err := s.programForMutation(ctx, input.TenantID, input.ProgramID, input.ExpectedVersion)
	if err != nil {
		return ProgramAggregate{}, err
	}
	ownerID := strings.TrimSpace(input.OwnerPrincipalID)
	if strings.TrimSpace(input.ActorID) == "" || strings.TrimSpace(input.Rationale) == "" || ownerID == "" {
		return ProgramAggregate{}, fmt.Errorf("owner_principal_id, actor_id and rationale are required")
	}
	if ownerID == aggregate.Program.OwnerPrincipalID {
		return ProgramAggregate{}, fmt.Errorf("the Program is already assigned to that owner")
	}
	if ownerID == aggregate.Program.AuthorityPrincipalID {
		return ProgramAggregate{}, fmt.Errorf("the Program owner and approval authority must be different people")
	}
	program := aggregate.Program
	program.OwnerPrincipalID = ownerID
	program.UpdatedAt = s.now().UTC()
	event := programOwnerChangedEvent{Program: program, PreviousOwnerID: aggregate.Program.OwnerPrincipalID, OwnerPrincipalID: ownerID, Rationale: strings.TrimSpace(input.Rationale)}
	if err := s.applyProgramValue(ctx, input.TenantID, input.ProgramID, input.ExpectedVersion, EventProgramOwnerChanged, event, input.ActorID); err != nil {
		return ProgramAggregate{}, err
	}
	return s.refreshAndGetProgram(ctx, input.TenantID, input.ProgramID, EventProgramOwnerChanged, input.ProgramID)
}

func (s *Service) AssignProgramApprovalAuthority(ctx context.Context, input AssignProgramApprovalAuthorityInput) (ProgramAggregate, error) {
	aggregate, err := s.programForMutation(ctx, input.TenantID, input.ProgramID, input.ExpectedVersion)
	if err != nil {
		return ProgramAggregate{}, err
	}
	authorityID := strings.TrimSpace(input.AuthorityPrincipalID)
	if strings.TrimSpace(input.ActorID) == "" || strings.TrimSpace(input.Rationale) == "" || authorityID == "" {
		return ProgramAggregate{}, fmt.Errorf("authority_principal_id, actor_id and rationale are required")
	}
	if authorityID == aggregate.Program.AuthorityPrincipalID {
		return ProgramAggregate{}, fmt.Errorf("that person already holds the Program approval authority")
	}
	if authorityID == aggregate.Program.OwnerPrincipalID {
		return ProgramAggregate{}, fmt.Errorf("the Program owner and approval authority must be different people")
	}
	program := aggregate.Program
	program.AuthorityPrincipalID = authorityID
	program.UpdatedAt = s.now().UTC()
	event := programApprovalAuthorityChangedEvent{Program: program, PreviousAuthorityPrincipalID: aggregate.Program.AuthorityPrincipalID, AuthorityPrincipalID: authorityID, Rationale: strings.TrimSpace(input.Rationale)}
	if err := s.applyProgramValue(ctx, input.TenantID, input.ProgramID, input.ExpectedVersion, EventProgramApprovalAuthorityChanged, event, input.ActorID); err != nil {
		return ProgramAggregate{}, err
	}
	return s.refreshAndGetProgram(ctx, input.TenantID, input.ProgramID, EventProgramApprovalAuthorityChanged, input.ProgramID)
}

func (s *Service) SupersedeRequirement(ctx context.Context, input SupersedeRequirementInput) (ProgramAggregate, error) {
	aggregate, err := s.programForMutation(ctx, input.TenantID, input.ProgramID, input.ExpectedVersion)
	if err != nil {
		return ProgramAggregate{}, err
	}
	if strings.TrimSpace(input.RequirementID) == "" || strings.TrimSpace(input.ActorID) == "" || strings.TrimSpace(input.Rationale) == "" {
		return ProgramAggregate{}, fmt.Errorf("requirement_id, actor_id and rationale are required")
	}
	if strings.TrimSpace(input.Code) == "" || strings.TrimSpace(input.Title) == "" || strings.TrimSpace(input.Statement) == "" || strings.TrimSpace(input.SourceAnchor) == "" || input.EffectiveFrom.IsZero() {
		return ProgramAggregate{}, fmt.Errorf("code, title, statement, source_anchor and effective_from are required")
	}
	priorIndex := -1
	for index := range aggregate.Requirements {
		if aggregate.Requirements[index].ID == input.RequirementID {
			priorIndex = index
			break
		}
	}
	if priorIndex < 0 {
		return ProgramAggregate{}, fmt.Errorf("requirement_id does not belong to this program")
	}
	prior := aggregate.Requirements[priorIndex]
	if prior.Status != RequirementApproved || prior.EffectiveUntil != nil {
		return ProgramAggregate{}, fmt.Errorf("only a current approved requirement can be superseded")
	}
	effectiveFrom := input.EffectiveFrom.UTC()
	if !effectiveFrom.After(prior.EffectiveFrom) {
		return ProgramAggregate{}, fmt.Errorf("effective_from must be after the current requirement effective date")
	}
	modality := strings.ToUpper(strings.TrimSpace(input.Modality))
	if modality == "" {
		modality = prior.Modality
	}
	if modality == "" {
		modality = "MUST"
	}
	if !validModality(modality) {
		return ProgramAggregate{}, fmt.Errorf("modality must be MUST, MUST_NOT, MAY, SHOULD or EXPECTED")
	}
	replacementID, err := id.NewUUIDv7()
	if err != nil {
		return ProgramAggregate{}, err
	}
	until := effectiveFrom
	prior.Status = RequirementSuperseded
	prior.EffectiveUntil = &until
	prior.Version++
	sourceID := strings.TrimSpace(input.SourceID)
	if sourceID == "" {
		sourceID = prior.SourceID
	}
	now := s.now().UTC()
	replacement := Requirement{
		ID: replacementID, TenantID: input.TenantID, ProgramID: input.ProgramID, SourceID: sourceID,
		Code: strings.ToUpper(strings.TrimSpace(input.Code)), Title: strings.TrimSpace(input.Title),
		Statement: strings.TrimSpace(input.Statement), SourceAnchor: strings.TrimSpace(input.SourceAnchor),
		Modality: modality, Actor: strings.TrimSpace(input.Actor), Action: strings.TrimSpace(input.Action),
		Object: strings.TrimSpace(input.Object), Status: RequirementApproved, EffectiveFrom: effectiveFrom,
		CreatedAt: now, Version: 1,
	}
	event := requirementSupersededEvent{Prior: prior, Replacement: replacement, Rationale: strings.TrimSpace(input.Rationale)}
	if err := s.applyProgramValue(ctx, input.TenantID, input.ProgramID, input.ExpectedVersion, EventRequirementSuperseded, event, input.ActorID); err != nil {
		return ProgramAggregate{}, err
	}
	return s.refreshAndGetProgram(ctx, input.TenantID, input.ProgramID, EventRequirementSuperseded, replacement.ID)
}
