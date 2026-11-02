package continuity

import (
	"sort"
	"strings"
	"time"
)

// ProgramRequirementComplianceInputs converts the existing authoritative
// Requirement → Control → Evidence chain into weighted calculation inputs.
// It cannot establish a Program-wide score without other approved categories.
func ProgramRequirementComplianceInputs(
	aggregate ProgramAggregate,
	at time.Time,
	category string,
	itemWeights map[string]int64,
) ([]ProgramComplianceInput, error) {
	if aggregate.Program.ID == "" || strings.TrimSpace(category) == "" || category != strings.TrimSpace(category) {
		return nil, ErrProgramComplianceCalculationInvalid
	}
	coverage := CurrentRequirementCoverage(aggregate, at)
	result := make([]ProgramComplianceInput, 0, len(coverage))
	for _, requirement := range aggregate.Requirements {
		current, ok := coverage[requirement.ID]
		if !ok {
			continue // Superseded, retired, future or expired obligation.
		}
		if requirement.ProgramID != "" && requirement.ProgramID != aggregate.Program.ID {
			return nil, ErrProgramComplianceCalculationInvalid
		}
		weight := int64(1)
		if override, exists := itemWeights[requirement.ID]; exists {
			weight = override
		}
		if weight < 1 || weight > programComplianceBasisPoints {
			return nil, ErrProgramComplianceCalculationInvalid
		}
		input := ProgramComplianceInput{
			Key:      "REQUIREMENT:" + requirement.ID,
			Category: category,
			Weight:   weight,
		}
		switch {
		case current.Applicability == ApplicabilityNotApplicable:
			// Applicability decisions are maintained through approved Program
			// operations; blank/unreviewed applicability is not an exclusion.
			input.State = ProgramComplianceNotApplicable
			input.Reason = "APPROVED_NOT_APPLICABLE"
		case current.Applicability != ApplicabilityApplicable:
			input.State = ProgramComplianceUnknown
			input.Reason = "APPLICABILITY_UNRESOLVED"
		case !current.ControlImplemented:
			// Missing, planned or inactive safeguards are known failures of
			// implementation, not evidence-review uncertainty.
			input.State = ProgramComplianceAssessed
			score := int64(0)
			input.ScoreBps = &score
			input.Reason = "CONTROL_NOT_IMPLEMENTED"
		case current.Complete:
			input.State = ProgramComplianceAssessed
			score := programComplianceBasisPoints
			input.ScoreBps = &score
			input.Reason = "REQUIREMENT_SUPPORTED"
		default:
			// The coverage contract intentionally does not certify whether
			// failing evidence represents rejection, expiry or no review.
			// Do not manufacture a 'failed' assessment from an unknown source.
			input.State = ProgramComplianceUnknown
			input.Reason = "EVIDENCE_UNVERIFIED_OR_NOT_CURRENT"
		}
		result = append(result, input)
	}
	for id := range itemWeights {
		if _, ok := coverage[id]; !ok {
			return nil, ErrProgramComplianceCalculationInvalid
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Key < result[j].Key })
	return result, nil
}
