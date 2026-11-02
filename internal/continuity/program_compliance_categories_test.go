package continuity

import (
	"reflect"
	"testing"
)

func TestProgramComplianceUnpopulatedCategoryBlocksFinalEvenAtLowCoverageThreshold(t *testing.T) {
	policy := ProgramComplianceCalculationPolicy{
		Revision: "v1", MinimumCoverageBps: 5000,
		Categories: []ProgramComplianceCategoryWeight{
			{Code: "REQUIREMENTS", WeightBps: 5000},
			{Code: "VENDOR_ASSESSMENTS", WeightBps: 5000},
		},
	}
	result, err := CalculateProgramCompliance(policy, []ProgramComplianceInput{{
		Key: "requirement:1", Category: "REQUIREMENTS", Weight: 1,
		State: ProgramComplianceAssessed, ScoreBps: compliancePoints(10000),
	}})
	if err != nil {
		t.Fatal(err)
	}
	if result.State != ProgramComplianceProvisional || result.ScoreBps != nil || result.CoverageBps != 5000 ||
		!reflect.DeepEqual(result.UnpopulatedCategories, []string{"VENDOR_ASSESSMENTS"}) {
		t.Fatalf("empty expected category silently passed: %+v", result)
	}

	// One legitimate but still-unassessed vendor is visible in the population.
	result, err = CalculateProgramCompliance(policy, []ProgramComplianceInput{
		{Key: "requirement:1", Category: "REQUIREMENTS", Weight: 1, State: ProgramComplianceAssessed, ScoreBps: compliancePoints(10000)},
		{Key: "vendor:expected:1", Category: "VENDOR_ASSESSMENTS", Weight: 1, State: ProgramComplianceUnknown},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.State != ProgramComplianceFinal || result.ScoreBps == nil || *result.ScoreBps != 10000 ||
		result.CoverageBps != 5000 || len(result.UnpopulatedCategories) != 0 {
		t.Fatalf("approved coverage threshold did not apply to known expected population: %+v", result)
	}
}

func TestProgramComplianceExcludesOnlyExplicitlyNonApplicableCategories(t *testing.T) {
	policy := ProgramComplianceCalculationPolicy{
		Revision: "v1", MinimumCoverageBps: 10000,
		Categories: []ProgramComplianceCategoryWeight{
			{Code: "REQUIREMENTS", WeightBps: 4000},
			{Code: "VENDOR_ASSESSMENTS", WeightBps: 6000},
		},
	}
	result, err := CalculateProgramCompliance(policy, []ProgramComplianceInput{
		{Key: "requirement:1", Category: "REQUIREMENTS", Weight: 1, State: ProgramComplianceAssessed, ScoreBps: compliancePoints(7500)},
		{Key: "vendor:not-applicable", Category: "VENDOR_ASSESSMENTS", Weight: 1, State: ProgramComplianceNotApplicable,
			Reason: "approved: no third-party service dependency in this Program"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.State != ProgramComplianceFinal || result.ScoreBps == nil || *result.ScoreBps != 7500 ||
		result.CoverageBps != 10000 || result.ExcludedCount != 1 || len(result.UnpopulatedCategories) != 0 {
		t.Fatalf("explicitly excluded category did not preserve valid percentage: %+v", result)
	}
	onlyNotApplicable, err := CalculateProgramCompliance(policy, []ProgramComplianceInput{
		{Key: "r:n/a", Category: "REQUIREMENTS", Weight: 1, State: ProgramComplianceNotApplicable, Reason: "approved not-applicable requirement"},
		{Key: "v:n/a", Category: "VENDOR_ASSESSMENTS", Weight: 1, State: ProgramComplianceNotApplicable, Reason: "approved no vendors"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if onlyNotApplicable.State != ProgramComplianceNotAssessed || onlyNotApplicable.ScoreBps != nil ||
		onlyNotApplicable.CoverageBps != 0 {
		t.Fatalf("all-N/A population must not become 100%% compliant: %+v", onlyNotApplicable)
	}
}
