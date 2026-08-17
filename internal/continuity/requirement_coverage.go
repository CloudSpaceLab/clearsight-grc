package continuity

import (
	"sort"
	"time"
)

// RequirementCoverage is the authoritative current requirement-level chain
// used by document coverage and other consumers. It deliberately reuses the
// same effective-period and evidence-validity rules as the Program projection.
type RequirementCoverage struct {
	RequirementID       string              `json:"requirement_id"`
	Applicability       ApplicabilityStatus `json:"applicability"`
	Applicable          bool                `json:"applicable"`
	ControlImplemented  bool                `json:"control_implemented"`
	EvidenceSupported   bool                `json:"evidence_supported"`
	Complete            bool                `json:"complete"`
	ControlIDs          []string            `json:"control_ids"`
	EvidenceContractIDs []string            `json:"evidence_contract_ids"`
	Reasons             []string            `json:"reasons"`
}

func CurrentRequirementCoverage(aggregate ProgramAggregate, at time.Time) map[string]RequirementCoverage {
	at = at.UTC()
	approved := make(map[string]Requirement)
	for _, requirement := range aggregate.Requirements {
		if requirement.Status == RequirementApproved && effectiveAt(requirement.EffectiveFrom, requirement.EffectiveUntil, at) {
			approved[requirement.ID] = requirement
		}
	}

	applicability := make(map[string]Applicability)
	for _, item := range aggregate.Applicability {
		if _, ok := approved[item.RequirementID]; !ok || !effectiveAt(item.EffectiveFrom, item.EffectiveUntil, at) {
			continue
		}
		current, ok := applicability[item.RequirementID]
		if !ok || item.EffectiveFrom.After(current.EffectiveFrom) ||
			(item.EffectiveFrom.Equal(current.EffectiveFrom) && item.CreatedAt.After(current.CreatedAt)) {
			applicability[item.RequirementID] = item
		}
	}

	implementations := make(map[string]ControlImplementation)
	for _, implementation := range aggregate.ControlImplementations {
		if effectiveAt(implementation.EffectiveFrom, implementation.EffectiveUntil, at) {
			implementations[implementation.ID] = implementation
		}
	}
	implementedByRequirement := make(map[string]map[string]struct{})
	for _, link := range aggregate.RequirementControlLinks {
		if _, ok := approved[link.RequirementID]; !ok {
			continue
		}
		implementation, ok := implementations[link.ImplementationID]
		if !ok || implementation.Status != ImplementationImplemented {
			continue
		}
		if implementedByRequirement[link.RequirementID] == nil {
			implementedByRequirement[link.RequirementID] = make(map[string]struct{})
		}
		implementedByRequirement[link.RequirementID][link.ImplementationID] = struct{}{}
	}

	latestAssessments := make(map[string]EvidenceAssessment)
	for _, assessment := range aggregate.EvidenceAssessments {
		if assessment.AssessedAt.After(at) {
			continue
		}
		current, ok := latestAssessments[assessment.ContractID]
		if !ok || assessment.AssessedAt.After(current.AssessedAt) ||
			(assessment.AssessedAt.Equal(current.AssessedAt) && assessment.CreatedAt.After(current.CreatedAt)) {
			latestAssessments[assessment.ContractID] = assessment
		}
	}
	contracts := effectiveEvidenceContracts(aggregate, at)

	result := make(map[string]RequirementCoverage, len(approved))
	for requirementID := range approved {
		coverage := RequirementCoverage{RequirementID: requirementID, ControlIDs: []string{}, EvidenceContractIDs: []string{}, Reasons: []string{}}
		if item, ok := applicability[requirementID]; ok {
			coverage.Applicability = item.Status
			coverage.Applicable = item.Status == ApplicabilityApplicable
		} else {
			coverage.Reasons = append(coverage.Reasons, "APPLICABILITY_NOT_RECORDED")
		}
		if !coverage.Applicable && coverage.Applicability != "" {
			coverage.Reasons = append(coverage.Reasons, "REQUIREMENT_NOT_APPLICABLE")
		}

		implemented := implementedByRequirement[requirementID]
		for implementationID := range implemented {
			coverage.ControlIDs = append(coverage.ControlIDs, implementationID)
		}
		sort.Strings(coverage.ControlIDs)
		coverage.ControlImplemented = coverage.Applicable && len(coverage.ControlIDs) > 0
		if coverage.Applicable && !coverage.ControlImplemented {
			coverage.Reasons = append(coverage.Reasons, "CONTROL_NOT_IMPLEMENTED")
		}

		allContractsSupported := true
		for _, contract := range contracts {
			_, controlRelevant := implemented[contract.ControlImplementationID]
			if contract.RequirementID != requirementID && !controlRelevant {
				continue
			}
			coverage.EvidenceContractIDs = append(coverage.EvidenceContractIDs, contract.ID)
			assessment, ok := latestAssessments[contract.ID]
			if !ok {
				allContractsSupported = false
				coverage.Reasons = append(coverage.Reasons, "EVIDENCE_NOT_ASSESSED")
				continue
			}
			validUntil := boundedAssessmentValidity(assessment, contract)
			supported := assessment.Conclusion == EvidenceSupported &&
				assessment.Coverage >= contract.MinimumCoverage &&
				!validUntil.IsZero() && at.Before(validUntil)
			if !supported {
				allContractsSupported = false
				coverage.Reasons = append(coverage.Reasons, "EVIDENCE_NOT_CURRENTLY_SUPPORTED")
			}
		}
		sort.Strings(coverage.EvidenceContractIDs)
		if len(coverage.EvidenceContractIDs) == 0 {
			allContractsSupported = false
			if coverage.Applicable {
				coverage.Reasons = append(coverage.Reasons, "NO_EVIDENCE_CONTRACTS")
			}
		}
		coverage.EvidenceSupported = coverage.ControlImplemented && allContractsSupported
		coverage.Complete = coverage.Applicable && coverage.ControlImplemented && coverage.EvidenceSupported
		sort.Strings(coverage.Reasons)
		result[requirementID] = coverage
	}
	return result
}
