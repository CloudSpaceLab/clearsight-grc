package continuity

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
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
	program := aggregate.Program
	program.OwnerPrincipalID = ownerID
	program.UpdatedAt = s.now().UTC()
	event := programOwnerChangedEvent{Program: program, PreviousOwnerID: aggregate.Program.OwnerPrincipalID, OwnerPrincipalID: ownerID, Rationale: strings.TrimSpace(input.Rationale)}
	if err := s.applyProgramValue(ctx, input.TenantID, input.ProgramID, input.ExpectedVersion, EventProgramOwnerChanged, event, input.ActorID); err != nil {
		return ProgramAggregate{}, err
	}
	return s.refreshAndGetProgram(ctx, input.TenantID, input.ProgramID, EventProgramOwnerChanged, input.ProgramID)
}
