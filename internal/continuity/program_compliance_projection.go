package continuity

import (
	"fmt"
	"time"
)

// requirementsOnlyProgramCalculation is deliberately limited to an approved
// single-category policy. Multi-source scoring must first prove the exact
// expected vendor, issue and evidence populations and permission-safe drills.
// An unsupported policy never produces a misleading requirements-only total.
func requirementsOnlyProgramCalculation(
	aggregate ProgramAggregate,
	at time.Time,
	sourceState ProgramSourceState,
) (*ProgramComplianceCalculation, error) {
	position := aggregate.Program.CalculationPolicy
	if aggregate.Program.Status != ProgramActive || position == nil || position.Active == nil {
		return nil, nil
	}
	active := position.Active
	if active.ApprovedAt == nil || active.ApprovedAt.After(at) {
		return nil, nil
	}
	policy := active.Definition
	if len(policy.Categories) != 1 ||
		policy.Categories[0].Code != "REQUIREMENTS" ||
		len(policy.ExpectedVendorAssessments) != 0 {
		return nil, nil
	}
	// Existing evidence-source projection is authoritative for source health.
	// Never advertise a finalized percentage during a source outage.
	if !sourceState.Known || (sourceState.Required > 0 && !sourceState.Current) {
		return nil, nil
	}
	if err := ValidateProgramComplianceCalculationPolicy(policy); err != nil {
		return nil, fmt.Errorf("%w: invalid active policy", ErrProgramComplianceCalculationInvalid)
	}
	if active.Number < 1 || active.MakerID == "" || active.ApproverID == "" ||
		active.MakerID == active.ApproverID || policy.Revision != fmt.Sprintf("program-v%d", active.Number) {
		return nil, fmt.Errorf("%w: invalid approved policy revision", ErrProgramComplianceCalculationInvalid)
	}
	checksum, err := programCompliancePolicyChecksum(policy)
	if err != nil {
		return nil, err
	}
	if checksum != active.Checksum {
		return nil, fmt.Errorf("%w: approved policy checksum mismatch", ErrProgramComplianceCalculationInvalid)
	}
	items, err := ProgramRequirementComplianceInputs(aggregate, at, "REQUIREMENTS", nil)
	if err != nil {
		return nil, err
	}
	result, err := CalculateProgramCompliance(policy, items)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
