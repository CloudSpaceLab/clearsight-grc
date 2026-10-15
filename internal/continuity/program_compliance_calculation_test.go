package continuity

import (
	"reflect"
	"testing"
)

func compliancePoints(value int64) *int64 { return &value }

func TestProgramComplianceWeightedAggregationPreservesMissingCoverage(t *testing.T) {
	policy := ProgramComplianceCalculationPolicy{
		Revision: "v1", MinimumCoverageBps: 10000,
		Categories: []ProgramComplianceCategoryWeight{
			{Code: "REQUIREMENTS", WeightBps: 6000},
			{Code: "VENDORS", WeightBps: 4000},
		},
	}
	inputs := []ProgramComplianceInput{
		{Key: "requirement:1", Category: "REQUIREMENTS", Weight: 3, State: ProgramComplianceAssessed, ScoreBps: compliancePoints(8000)},
		{Key: "requirement:2", Category: "REQUIREMENTS", Weight: 1, State: ProgramComplianceUnknown},
		{Key: "vendor:1:approved-form", Category: "VENDORS", Weight: 1, State: ProgramComplianceAssessed, ScoreBps: compliancePoints(10000)},
	}

	result, err := CalculateProgramCompliance(policy, inputs)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != ProgramComplianceProvisional || result.ScoreBps != nil {
		t.Fatalf("incomplete coverage was presented as a final compliance score: %+v", result)
	}
	if result.CoverageBps != 8500 || result.AssessedScoreBps == nil || *result.AssessedScoreBps != 8941 {
		t.Fatalf("unexpected weighted numerator or denominator: %+v", result)
	}
	if result.EligibleCount != 3 || result.AssessedCount != 2 || result.UnknownCount != 1 {
		t.Fatalf("missing expected record changed the denominator: %+v", result)
	}
	if result.Categories[0].CoverageBps != 7500 || result.Categories[1].CoverageBps != 10000 {
		t.Fatalf("incorrect category coverages: %+v", result.Categories)
	}

	// An explicitly approved minimum allows release at 85% coverage.
	// Unavailable contributions remain visible as missing rather than green.
	policy.MinimumCoverageBps = 8500
	result, err = CalculateProgramCompliance(policy, inputs)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != ProgramComplianceFinal || result.ScoreBps == nil || *result.ScoreBps != 8941 ||
		result.CoverageBps != 8500 {
		t.Fatalf("approved partial-coverage threshold was not honoured: %+v", result)
	}
}

func TestProgramComplianceMissingCannotBePassedOrAutomaticallyExcluded(t *testing.T) {
	policy := ProgramComplianceCalculationPolicy{Revision: "v1", MinimumCoverageBps: 10000,
		Categories: []ProgramComplianceCategoryWeight{{Code: "CONTROLS", WeightBps: 10000}},
	}
	pending, err := CalculateProgramCompliance(policy, []ProgramComplianceInput{{
		Key: "unassessed", Category: "CONTROLS", Weight: 100, State: ProgramComplianceUnknown,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if pending.State != ProgramComplianceNotAssessed || pending.ScoreBps != nil || pending.CoverageBps != 0 {
		t.Fatalf("unknown evidence generated compliance: %+v", pending)
	}

	naOnly, err := CalculateProgramCompliance(policy, []ProgramComplianceInput{{
		Key: "out-of-scope", Category: "CONTROLS", Weight: 100,
		State: ProgramComplianceNotApplicable, Reason: "approved scoped applicability decision",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if naOnly.State != ProgramComplianceNotAssessed || naOnly.ScoreBps != nil ||
		naOnly.EligibleCount != 0 || naOnly.ExcludedCount != 1 {
		t.Fatalf("all-not-applicable did not stay unscored: %+v", naOnly)
	}

	// A separately validated overdue obligation may be explicitly scored as
	// failed. Unlike UNKNOWN, a confirmed failure counts toward assessed mass.
	failed, err := CalculateProgramCompliance(policy, []ProgramComplianceInput{
		{Key: "good", Category: "CONTROLS", Weight: 1, State: ProgramComplianceAssessed, ScoreBps: compliancePoints(10000)},
		{Key: "failed", Category: "CONTROLS", Weight: 1, State: ProgramComplianceAssessed, ScoreBps: compliancePoints(0)},
		{Key: "excluded", Category: "CONTROLS", Weight: 1, State: ProgramComplianceNotApplicable, Reason: "approved non-applicability"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if failed.State != ProgramComplianceFinal || failed.ScoreBps == nil || *failed.ScoreBps != 5000 ||
		failed.CoverageBps != 10000 || failed.ExcludedCount != 1 {
		t.Fatalf("confirmed failure or approved exclusion miscalculated: %+v", failed)
	}
}

func TestProgramComplianceAggregationIgnoresInputOrder(t *testing.T) {
	policy := ProgramComplianceCalculationPolicy{
		Revision: "check", MinimumCoverageBps: 10000,
		Categories: []ProgramComplianceCategoryWeight{
			{Code: "CONTROL", WeightBps: 4000}, {Code: "ASSESSMENT", WeightBps: 6000},
		},
	}
	a := []ProgramComplianceInput{
		{Key: "c2", Category: "CONTROL", Weight: 2, State: ProgramComplianceAssessed, ScoreBps: compliancePoints(9000)},
		{Key: "a1", Category: "ASSESSMENT", Weight: 1, State: ProgramComplianceAssessed, ScoreBps: compliancePoints(7000)},
		{Key: "c1", Category: "CONTROL", Weight: 1, State: ProgramComplianceAssessed, ScoreBps: compliancePoints(6000)},
	}
	b := []ProgramComplianceInput{a[2], a[0], a[1]}
	first, e1 := CalculateProgramCompliance(policy, a)
	second, e2 := CalculateProgramCompliance(policy, b)
	if e1 != nil || e2 != nil || !reflect.DeepEqual(first, second) {
		t.Fatalf("input order changed the calculation: first=%+v second=%+v errors=%v/%v", first, second, e1, e2)
	}
	if first.State != ProgramComplianceFinal || first.ScoreBps == nil || *first.ScoreBps != 7400 {
		// Control: (2×90 + 1×60)/3=80; Assessment: 70; 40%×80 + 60%×70 = 74.
		t.Fatalf("wrong weighted score: %+v", first)
	}
}

func TestProgramComplianceRejectsInvalidDefinitionsAndInputs(t *testing.T) {
	base := ProgramComplianceCalculationPolicy{
		Revision: "v1", MinimumCoverageBps: 10000,
		Categories: []ProgramComplianceCategoryWeight{{Code: "A", WeightBps: 10000}},
	}
	good := ProgramComplianceInput{
		Key: "a", Category: "A", Weight: 1, State: ProgramComplianceAssessed, ScoreBps: compliancePoints(9500),
	}
	for _, tc := range []struct {
		name   string
		policy ProgramComplianceCalculationPolicy
		items  []ProgramComplianceInput
	}{
		{"unapproved weight total", ProgramComplianceCalculationPolicy{Revision: "v1", MinimumCoverageBps: 10000, Categories: []ProgramComplianceCategoryWeight{{Code: "A", WeightBps: 9000}}}, []ProgramComplianceInput{good}},
		{"missing policy revision", ProgramComplianceCalculationPolicy{MinimumCoverageBps: 10000, Categories: base.Categories}, []ProgramComplianceInput{good}},
		{"invalid coverage minimum", ProgramComplianceCalculationPolicy{Revision: "v1", Categories: base.Categories}, []ProgramComplianceInput{good}},
		{"duplicate input", base, []ProgramComplianceInput{good, good}},
		{"wrong category", base, []ProgramComplianceInput{{Key: "b", Category: "B", Weight: 1, State: ProgramComplianceUnknown}}},
		{"missing score", base, []ProgramComplianceInput{{Key: "b", Category: "A", Weight: 1, State: ProgramComplianceAssessed}}},
		{"unknown with score", base, []ProgramComplianceInput{{Key: "b", Category: "A", Weight: 1, State: ProgramComplianceUnknown, ScoreBps: compliancePoints(10000)}}},
		{"score outside percentage", base, []ProgramComplianceInput{{Key: "b", Category: "A", Weight: 1, State: ProgramComplianceAssessed, ScoreBps: compliancePoints(10001)}}},
		{"N/A without approval reason", base, []ProgramComplianceInput{{Key: "b", Category: "A", Weight: 1, State: ProgramComplianceNotApplicable}}},
		{"zero input weight", base, []ProgramComplianceInput{{Key: "b", Category: "A", State: ProgramComplianceUnknown}}},
		{"duplicate categories", ProgramComplianceCalculationPolicy{Revision: "v1", MinimumCoverageBps: 10000, Categories: []ProgramComplianceCategoryWeight{{Code: "A", WeightBps: 5000}, {Code: "A", WeightBps: 5000}}}, []ProgramComplianceInput{good}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if result, err := CalculateProgramCompliance(tc.policy, tc.items); err == nil {
				t.Fatalf("unsafe calculation accepted: %+v", result)
			}
		})
	}
}
