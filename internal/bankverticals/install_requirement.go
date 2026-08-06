package bankverticals

import (
	"context"
	"fmt"
	"strings"
	"time"

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
		if value.ObjectiveID == objective.ID && strings.EqualFold(value.Name, spec.implementationName) && value.Status == continuity.ImplementationImplemented {
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
			Status:             continuity.ImplementationImplemented,
			EffectiveFrom:      config.Now.AddDate(0, -3, 0),
			ActorID:            config.ActorID,
		})
		if err != nil {
			return program, fmt.Errorf("repair safeguard %s: %w", spec.code, err)
		}
		value := program.ControlImplementations[len(program.ControlImplementations)-1]
		implementation = &value
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

	var contract *continuity.EvidenceContract
	for index := range program.EvidenceContracts {
		if strings.EqualFold(program.EvidenceContracts[index].Code, spec.evidenceCode) && program.EvidenceContracts[index].Status == continuity.EvidenceContractActive {
			value := program.EvidenceContracts[index]
			contract = &value
			break
		}
	}
	if contract == nil {
		acceptable := make([]string, 0, len(spec.sourceCodes))
		for _, code := range spec.sourceCodes {
			acceptable = append(acceptable, sourceIDs[code])
		}
		program, err = s.continuity.AddEvidenceContract(ctx, continuity.AddEvidenceContractInput{
			TenantID:                config.TenantID,
			ProgramID:               program.Program.ID,
			ExpectedVersion:         program.Program.Version,
			ControlImplementationID: implementation.ID,
			Code:                    spec.evidenceCode,
			Name:                    spec.evidenceName,
			Claim:                   spec.claim,
			AcceptableSourceIDs:     acceptable,
			PopulationScope:         mustJSON(spec.population),
			FreshnessMinutes:        spec.freshnessMinutes,
			MinimumCoverage:         spec.minimumCoverage,
			IndependenceRequired:    true,
			ContradictionPolicy:     "REVIEW",
			FailureAction:           "MATTER",
			Status:                  continuity.EvidenceContractActive,
			ActorID:                 config.ActorID,
		})
		if err != nil {
			return program, fmt.Errorf("repair evidence check %s: %w", spec.code, err)
		}
		value := program.EvidenceContracts[len(program.EvidenceContracts)-1]
		contract = &value
	}

	current := false
	for _, assessment := range program.EvidenceAssessments {
		if assessment.ContractID == contract.ID && (assessment.ValidUntil == nil || assessment.ValidUntil.After(config.Now)) {
			current = true
			break
		}
	}
	if !current {
		validUntil := config.Now.Add(30 * 24 * time.Hour)
		program, err = s.continuity.RecordEvidenceAssessment(ctx, continuity.RecordEvidenceAssessmentInput{
			TenantID:        config.TenantID,
			ProgramID:       program.Program.ID,
			ExpectedVersion: program.Program.Version,
			ContractID:      contract.ID,
			Conclusion:      spec.conclusion,
			Coverage:        spec.coverage,
			Basis:           mustJSON(spec.basis),
			ValidUntil:      &validUntil,
			AssessedBy:      config.ReviewerPrincipalID,
			AssessedAt:      config.Now.Add(-2 * time.Hour),
		})
		if err != nil {
			return program, fmt.Errorf("repair evidence assessment %s: %w", spec.code, err)
		}
	}
	return program, nil
}
