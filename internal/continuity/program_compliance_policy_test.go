package continuity

import "testing"

func TestProgramCompliancePolicyValidationMatchesRuntime(t *testing.T) {
	valid := ProgramComplianceCalculationPolicy{
		Revision:           "committee-approved-v1",
		MinimumCoverageBps: 8500,
		Categories: []ProgramComplianceCategoryWeight{
			{Code: "REQUIREMENTS", WeightBps: 6000},
			{Code: "VENDORS", WeightBps: 4000},
		},
	}
	if err := ValidateProgramComplianceCalculationPolicy(valid); err != nil {
		t.Fatalf("valid policy rejected before activation: %v", err)
	}
	for _, tc := range []struct {
		name   string
		change func(*ProgramComplianceCalculationPolicy)
	}{
		{"missing revision", func(p *ProgramComplianceCalculationPolicy) { p.Revision = "" }},
		{"invalid coverage threshold", func(p *ProgramComplianceCalculationPolicy) { p.MinimumCoverageBps = 10001 }},
		{"invalid minimum", func(p *ProgramComplianceCalculationPolicy) { p.MinimumCoverageBps = 0 }},
		{"unbalanced categories", func(p *ProgramComplianceCalculationPolicy) { p.Categories[0].WeightBps = 5900 }},
		{"duplicate category", func(p *ProgramComplianceCalculationPolicy) { p.Categories[1].Code = p.Categories[0].Code }},
		{"zero category", func(p *ProgramComplianceCalculationPolicy) { p.Categories[0].WeightBps = 0 }},
		{"invalid category name", func(p *ProgramComplianceCalculationPolicy) { p.Categories[0].Code = " REQUIREMENTS" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			policy := valid
			policy.Categories = append([]ProgramComplianceCategoryWeight(nil), valid.Categories...)
			tc.change(&policy)
			if err := ValidateProgramComplianceCalculationPolicy(policy); err == nil {
				t.Fatalf("invalid policy could be activated: %+v", policy)
			}
			if _, err := CalculateProgramCompliance(policy, nil); err == nil {
				t.Fatalf("runtime accepted policy rejected by approval gate: %+v", policy)
			}
		})
	}
}
