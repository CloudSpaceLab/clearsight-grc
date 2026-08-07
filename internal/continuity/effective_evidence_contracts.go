package continuity

import "time"

func effectiveEvidenceContracts(aggregate ProgramAggregate, at time.Time) []EvidenceContract {
	currentRequirements := make(map[string]struct{}, len(aggregate.Requirements))
	for _, requirement := range aggregate.Requirements {
		if requirement.Status == RequirementApproved && effectiveAt(requirement.EffectiveFrom, requirement.EffectiveUntil, at) {
			currentRequirements[requirement.ID] = struct{}{}
		}
	}
	currentImplementations := make(map[string]struct{}, len(aggregate.ControlImplementations))
	for _, implementation := range aggregate.ControlImplementations {
		if implementation.Status != ImplementationInactive && implementation.Status != ImplementationRetired && effectiveAt(implementation.EffectiveFrom, implementation.EffectiveUntil, at) {
			currentImplementations[implementation.ID] = struct{}{}
		}
	}

	result := make([]EvidenceContract, 0, len(aggregate.EvidenceContracts))
	for _, contract := range aggregate.EvidenceContracts {
		if contract.Status != EvidenceContractActive {
			continue
		}
		if contract.RequirementID != "" {
			if _, ok := currentRequirements[contract.RequirementID]; !ok {
				continue
			}
		}
		if contract.ControlImplementationID != "" {
			if _, ok := currentImplementations[contract.ControlImplementationID]; !ok {
				continue
			}
		}
		result = append(result, contract)
	}
	return result
}
