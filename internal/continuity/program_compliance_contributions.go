package continuity

import (
	"reflect"
	"fmt"
)

// ProgramRequirementScoreContributions reconstructs a retained requirements-
// only score from the original Program facts. Callers must first enforce
// tenant/entity/Program access on the aggregate and snapshot they load.
// Historical reads must use ProgramAt(snapshot.GeneratedAt), never current
// mutable records or a browser-paginated requirements list.
func ProgramRequirementScoreContributions(
	aggregate ProgramAggregate,
	snapshot ProgramStateSnapshot,
) ([]ProgramComplianceInput, error) {
	if aggregate.Program.ID == "" || aggregate.Program.ID != snapshot.ProgramID ||
		aggregate.Program.TenantID == "" || aggregate.Program.TenantID != snapshot.TenantID ||
		aggregate.Program.Version != snapshot.ProgramVersion ||
		snapshot.GeneratedAt.IsZero() || snapshot.Calculation == nil {
		return nil, ErrProgramComplianceCalculationInvalid
	}
	position := aggregate.Program.CalculationPolicy
	if position == nil || position.Active == nil || position.Active.ApprovedAt == nil ||
		position.Active.ApprovedAt.After(snapshot.GeneratedAt) {
		return nil, ErrProgramComplianceCalculationInvalid
	}
	policy := position.Active.Definition
	if position.Active.Number < 1 || position.Active.MakerID == "" ||
		position.Active.ApproverID == "" || position.Active.MakerID == position.Active.ApproverID ||
		policy.Revision != fmt.Sprintf("program-v%d", position.Active.Number) {
		return nil, ErrProgramComplianceCalculationInvalid
	}
	if len(policy.Categories) != 1 || policy.Categories[0].Code != "REQUIREMENTS" ||
		len(policy.ExpectedVendorAssessments) != 0 ||
		policy.Revision != snapshot.Calculation.PolicyRevision {
		return nil, ErrProgramComplianceCalculationInvalid
	}
	if err := ValidateProgramComplianceCalculationPolicy(policy); err != nil {
		return nil, err
	}
	checksum, err := programCompliancePolicyChecksum(policy)
	if err != nil || position.Active.Checksum != checksum {
		return nil, ErrProgramComplianceCalculationInvalid
	}
	items, err := ProgramRequirementComplianceInputs(aggregate, snapshot.GeneratedAt, "REQUIREMENTS", nil)
	if err != nil {
		return nil, err
	}
	calculated, err := CalculateProgramCompliance(policy, items)
	if err != nil {
		return nil, err
	}
	// A retained percentage is never silently explained using a different
	// denominator, evidence revision or category population.
	if !reflect.DeepEqual(calculated, *snapshot.Calculation) {
		return nil, ErrProgramComplianceCalculationInvalid
	}
	return items, nil
}
