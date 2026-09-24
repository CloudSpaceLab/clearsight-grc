package bankverticals

import (
	"context"
	"fmt"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

func (s *Service) ensureRequirementBundle(ctx context.Context, config SeedConfig, program continuity.ProgramAggregate, sourceIDs map[string]string, spec requirementSpec) (continuity.ProgramAggregate, error) {
	var requirement *continuity.Requirement
	for index := range program.Requirements {
		if strings.EqualFold(program.Requirements[index].Code, spec.code) && program.Requirements[index].Status == continuity.RequirementApproved {
			value := program.Requirements[index]
			requirement = &value
			break
		}
	}
	if requirement == nil {
		return s.addRequirementBundle(ctx, config, program, sourceIDs, spec)
	}

	applicable := false
	for _, value := range program.Applicability {
		if value.RequirementID == requirement.ID && value.Status == continuity.ApplicabilityApplicable {
			applicable = true
			break
		}
	}
	var err error
	if !applicable {
		program, err = s.continuity.DetermineApplicability(ctx, continuity.DetermineApplicabilityInput{
			TenantID:        config.TenantID,
			ProgramID:       program.Program.ID,
			ExpectedVersion: program.Program.Version,
			RequirementID:   requirement.ID,
			Status:          continuity.ApplicabilityApplicable,
			Scope:           mustJSON(map[string]any{"bank": config.BankName, "legal_entity_id": config.LegalEntityID}),
			Rationale:       "The bank processes customer, employee, vendor and other personal data in Nigeria.",
			ApprovedBy:      config.ReviewerPrincipalID,
			EffectiveFrom:   config.Now.AddDate(0, -6, 0),
		})
		if err != nil {
			return program, fmt.Errorf("repair applicability %s: %w", spec.code, err)
		}
	}

	var objective *continuity.ControlObjective
	for index := range program.ControlObjectives {
		if strings.EqualFold(program.ControlObjectives[index].Code, spec.objectiveCode) && program.ControlObjectives[index].Status == continuity.ObjectiveActive {
			value := program.ControlObjectives[index]
			objective = &value
			break
		}
	}
	if objective == nil {
		program, err = s.continuity.AddControlObjective(ctx, continuity.AddControlObjectiveInput{
			TenantID:        config.TenantID,
			ProgramID:       program.Program.ID,
			ExpectedVersion: program.Program.Version,
			Code:            spec.objectiveCode,
			Name:            spec.objectiveName,
			Outcome:         spec.outcome,
			Status:          continuity.ObjectiveActive,
			ActorID:         config.ActorID,
		})
		if err != nil {
			return program, fmt.Errorf("repair safeguard objective %s: %w", spec.code, err)
		}
		value := program.ControlObjectives[len(program.ControlObjectives)-1]
		objective = &value
	}

	var implementation *continuity.ControlImplementation
	for index := range program.ControlImplementations {
		value := program.ControlImplementations[index]
		if value.ObjectiveID == objective.ID && strings.EqualFold(value.Name, spec.implementationName) && value.Status != continuity.ImplementationRetired {
			copy := value
			implementation = &copy
			break
		}
	}
	if implementation == nil {
		program, err = s.continuity.AddControlImplementation(ctx, continuity.AddControlImplementationInput{
			TenantID:           config.TenantID,
			ProgramID:          program.Program.ID,
			ExpectedVersion:    program.Program.Version,
			ObjectiveID:        objective.ID,
			Name:               spec.implementationName,
			Description:        spec.implementationDetail,
			ImplementationType: spec.implementationType,
			OwnerPrincipalID:   config.OwnerPrincipalID,
			Scope:              mustJSON(map[string]any{"bank": config.BankName}),
			Status:             continuity.ImplementationPlanned,
			EffectiveFrom:      config.Now.AddDate(0, -3, 0),
			ActorID:            config.ActorID,
		})
		if err != nil {
			return program, fmt.Errorf("repair safeguard %s: %w", spec.code, err)
		}
		value := program.ControlImplementations[len(program.ControlImplementations)-1]
		implementation = &value
	}
	program, err = implementReferenceSafeguard(ctx, s.continuity, config, program, implementation.ID)
	if err != nil {
		return program, fmt.Errorf("repair safeguard implementation %s: %w", spec.code, err)
	}
	for index := range program.ControlImplementations {
		if program.ControlImplementations[index].ID == implementation.ID {
			value := program.ControlImplementations[index]
			implementation = &value
			break
		}
	}

	linked := false
	for _, value := range program.RequirementControlLinks {
		if value.RequirementID == requirement.ID && value.ImplementationID == implementation.ID {
			linked = true
			break
		}
	}
	if !linked {
		program, err = s.continuity.LinkRequirementControl(ctx, continuity.LinkRequirementControlInput{
			TenantID:         config.TenantID,
			ProgramID:        program.Program.ID,
			ExpectedVersion:  program.Program.Version,
			RequirementID:    requirement.ID,
			ImplementationID: implementation.ID,
			ActorID:          config.ActorID,
		})
		if err != nil {
			return program, fmt.Errorf("repair safeguard link %s: %w", spec.code, err)
		}
	}

	return program, nil
}
