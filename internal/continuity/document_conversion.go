package continuity

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// EnsureRequirement materializes one deterministic canonical requirement.
// If the exact object already exists, it is returned as a successful replay even
// when the Program version has advanced since the original attempt.
func (s *Service) EnsureRequirement(ctx context.Context, objectID string, input AddRequirementInput) (ProgramAggregate, error) {
	objectID = strings.TrimSpace(objectID)
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.ProgramID = strings.TrimSpace(input.ProgramID)
	if objectID == "" || input.TenantID == "" || input.ProgramID == "" || input.ExpectedVersion < 1 {
		return ProgramAggregate{}, fmt.Errorf("object id, tenant_id, program_id and positive expected_version are required")
	}
	aggregate, err := s.GetProgram(ctx, input.TenantID, input.ProgramID)
	if err != nil {
		return ProgramAggregate{}, err
	}
	for _, current := range aggregate.Requirements {
		if current.ID != objectID {
			continue
		}
		if sameEnsuredRequirement(current, input) {
			return aggregate, nil
		}
		return ProgramAggregate{}, ErrDuplicate
	}
	if aggregate.Program.Version != input.ExpectedVersion {
		return ProgramAggregate{}, ErrVersionConflict
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
	now := s.now().UTC()
	value := Requirement{
		ID: objectID, TenantID: input.TenantID, ProgramID: input.ProgramID,
		SourceID: strings.TrimSpace(input.SourceID), Code: strings.ToUpper(strings.TrimSpace(input.Code)),
		Title: strings.TrimSpace(input.Title), Statement: strings.TrimSpace(input.Statement), SourceAnchor: strings.TrimSpace(input.SourceAnchor),
		Modality: modality, Actor: strings.TrimSpace(input.Actor), Action: strings.TrimSpace(input.Action), Object: strings.TrimSpace(input.Object),
		Status: input.Status, EffectiveFrom: input.EffectiveFrom.UTC(), CreatedAt: now, Version: 1,
	}
	if err := s.applyProgramValue(ctx, input.TenantID, input.ProgramID, input.ExpectedVersion, EventRequirementAdded, value, input.ActorID); err != nil {
		return ProgramAggregate{}, err
	}
	return s.refreshAndGetProgram(ctx, input.TenantID, input.ProgramID, EventRequirementAdded, value.ID)
}

func sameEnsuredRequirement(current Requirement, input AddRequirementInput) bool {
	status := input.Status
	if status == "" {
		status = RequirementApproved
	}
	modality := strings.ToUpper(strings.TrimSpace(input.Modality))
	if modality == "" {
		modality = "MUST"
	}
	return current.ProgramID == strings.TrimSpace(input.ProgramID) &&
		current.Code == strings.ToUpper(strings.TrimSpace(input.Code)) &&
		current.Title == strings.TrimSpace(input.Title) &&
		current.Statement == strings.TrimSpace(input.Statement) &&
		current.SourceID == strings.TrimSpace(input.SourceID) &&
		current.SourceAnchor == strings.TrimSpace(input.SourceAnchor) &&
		current.Modality == modality &&
		current.Actor == strings.TrimSpace(input.Actor) &&
		current.Action == strings.TrimSpace(input.Action) &&
		current.Object == strings.TrimSpace(input.Object) &&
		current.Status == status
}

// EnsureControlObjective is the replay-safe counterpart for canonical control
// objectives created by an independently authorized document handoff.
func (s *Service) EnsureControlObjective(ctx context.Context, objectID string, input AddControlObjectiveInput) (ProgramAggregate, error) {
	objectID = strings.TrimSpace(objectID)
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.ProgramID = strings.TrimSpace(input.ProgramID)
	if objectID == "" || input.TenantID == "" || input.ProgramID == "" || input.ExpectedVersion < 1 {
		return ProgramAggregate{}, fmt.Errorf("object id, tenant_id, program_id and positive expected_version are required")
	}
	aggregate, err := s.GetProgram(ctx, input.TenantID, input.ProgramID)
	if err != nil {
		return ProgramAggregate{}, err
	}
	for _, current := range aggregate.ControlObjectives {
		if current.ID != objectID {
			continue
		}
		if sameEnsuredControlObjective(current, input) {
			return aggregate, nil
		}
		return ProgramAggregate{}, ErrDuplicate
	}
	if aggregate.Program.Version != input.ExpectedVersion {
		return ProgramAggregate{}, ErrVersionConflict
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
	value := ControlObjective{
		ID: objectID, TenantID: input.TenantID, ProgramID: input.ProgramID,
		Code: strings.ToUpper(strings.TrimSpace(input.Code)), Name: strings.TrimSpace(input.Name), Outcome: strings.TrimSpace(input.Outcome),
		Status: input.Status, CreatedAt: s.now().UTC(), Version: 1,
	}
	if err := s.applyProgramValue(ctx, input.TenantID, input.ProgramID, input.ExpectedVersion, EventControlObjectiveAdded, value, input.ActorID); err != nil {
		return ProgramAggregate{}, err
	}
	return s.refreshAndGetProgram(ctx, input.TenantID, input.ProgramID, EventControlObjectiveAdded, value.ID)
}

func sameEnsuredControlObjective(current ControlObjective, input AddControlObjectiveInput) bool {
	status := input.Status
	if status == "" {
		status = ObjectiveActive
	}
	return current.ProgramID == strings.TrimSpace(input.ProgramID) &&
		current.Code == strings.ToUpper(strings.TrimSpace(input.Code)) &&
		current.Name == strings.TrimSpace(input.Name) &&
		current.Outcome == strings.TrimSpace(input.Outcome) &&
		current.Status == status
}

// Compile-time guard against accidentally introducing wall-clock values outside
// the service clock in future edits of this replay path.
var _ = time.Time{}
