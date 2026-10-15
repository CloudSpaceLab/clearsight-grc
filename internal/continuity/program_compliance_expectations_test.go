package continuity

import "testing"

func TestProgramVendorExpectationPolicyPopulationAndRevisionCopies(t *testing.T) {
	valid := ProgramComplianceCalculationPolicy{
		Revision: "program-v1", MinimumCoverageBps: 10000,
		Categories: []ProgramComplianceCategoryWeight{
			{Code: "REQUIREMENTS", WeightBps: 6500},
			{Code: "VENDORS", WeightBps: 3500},
		},
		ExpectedVendorAssessments: []ProgramComplianceExpectedVendorAssessment{
			{Key: "vendor-a:quarter", RelationshipID: "vendor-a", FormTemplateID: "security",
				FormTemplateVersion: 3, ReviewKind: "PERIODIC", Weight: 2},
			{Key: "vendor-b:quarter", RelationshipID: "vendor-b", FormTemplateID: "security",
				FormTemplateVersion: 3, ReviewKind: "PERIODIC", Weight: 1},
		},
	}
	if err := ValidateProgramComplianceCalculationPolicy(valid); err != nil {
		t.Fatal(err)
	}
	copy := cloneProgramCompliancePolicyPosition(&ProgramCompliancePolicyPosition{
		LastRevision: 1, Active: &ProgramCompliancePolicyRevision{
			Number: 1, Definition: valid,
		},
	})
	copy.Active.Definition.ExpectedVendorAssessments[0].Weight = 10
	if valid.ExpectedVendorAssessments[0].Weight != 2 {
		t.Fatal("editing proposed expected assessments mutated active policy source")
	}
	for _, tc := range []struct {
		name string
		change func(*ProgramComplianceCalculationPolicy)
	}{
		{"duplicate key", func(p *ProgramComplianceCalculationPolicy) { p.ExpectedVendorAssessments[1].Key = p.ExpectedVendorAssessments[0].Key }},
		{"duplicate identity", func(p *ProgramComplianceCalculationPolicy) {
			p.ExpectedVendorAssessments[1].RelationshipID = "vendor-a"
			p.ExpectedVendorAssessments[1].Key = "different-key"
		}},
		{"empty form ID", func(p *ProgramComplianceCalculationPolicy) { p.ExpectedVendorAssessments[0].FormTemplateID = "" }},
		{"wrong review kind", func(p *ProgramComplianceCalculationPolicy) { p.ExpectedVendorAssessments[0].ReviewKind = "RISK" }},
		{"invalid form version", func(p *ProgramComplianceCalculationPolicy) { p.ExpectedVendorAssessments[0].FormTemplateVersion = 0 }},
		{"zero weight", func(p *ProgramComplianceCalculationPolicy) { p.ExpectedVendorAssessments[0].Weight = 0 }},
		{"no vendor scoring category", func(p *ProgramComplianceCalculationPolicy) {
			p.Categories = []ProgramComplianceCategoryWeight{{Code: "REQUIREMENTS", WeightBps: 10000}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := valid
			p.Categories = append([]ProgramComplianceCategoryWeight(nil), valid.Categories...)
			p.ExpectedVendorAssessments = append([]ProgramComplianceExpectedVendorAssessment(nil), valid.ExpectedVendorAssessments...)
			tc.change(&p)
			if err := ValidateProgramComplianceCalculationPolicy(p); err == nil {
				t.Fatal("invalid expected assessment entered an approved Program policy")
			}
		})
	}
}
