package continuity

import (
	"testing"
	"time"
)

func TestProgramRequirementComplianceInputs(t *testing.T) {
	now := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	previous := now.Add(-time.Hour)
	aggregate := ProgramAggregate{
		Program: Program{ID: "program", TenantID: "bank", Status: ProgramActive},
		Requirements: []Requirement{
			{ID: "r-implemented", ProgramID: "program", Status: RequirementApproved, EffectiveFrom: previous},
			{ID: "r-missing", ProgramID: "program", Status: RequirementApproved, EffectiveFrom: previous},
			{ID: "r-review", ProgramID: "program", Status: RequirementApproved, EffectiveFrom: previous},
			{ID: "r-excluded", ProgramID: "program", Status: RequirementApproved, EffectiveFrom: previous},
			{ID: "r-future", ProgramID: "program", Status: RequirementApproved, EffectiveFrom: now.Add(time.Hour)},
			{ID: "r-retired", ProgramID: "program", Status: RequirementRetired, EffectiveFrom: previous},
		},
		Applicability: []Applicability{
			{RequirementID: "r-implemented", Status: ApplicabilityApplicable, EffectiveFrom: previous},
			{RequirementID: "r-missing", Status: ApplicabilityApplicable, EffectiveFrom: previous},
			{RequirementID: "r-review", Status: ApplicabilityApplicable, EffectiveFrom: previous},
			{RequirementID: "r-excluded", Status: ApplicabilityNotApplicable, EffectiveFrom: previous},
		},
		ControlImplementations: []ControlImplementation{
			{ID: "control-a", Status: ImplementationImplemented, EffectiveFrom: previous},
			{ID: "control-b", Status: ImplementationImplemented, EffectiveFrom: previous},
		},
		RequirementControlLinks: []RequirementControlLink{
			{RequirementID: "r-implemented", ImplementationID: "control-a", CreatedAt: previous},
			{RequirementID: "r-review", ImplementationID: "control-b", CreatedAt: previous},
		},
		EvidenceContracts: []EvidenceContract{
			{ID: "e-a", RequirementID: "r-implemented", Status: EvidenceContractActive, FreshnessMinutes: 60, MinimumCoverage: 1},
			{ID: "e-b", RequirementID: "r-review", Status: EvidenceContractActive, FreshnessMinutes: 60, MinimumCoverage: 1},
		},
		EvidenceAssessments: []EvidenceAssessment{
			{ContractID: "e-a", Conclusion: EvidenceSupported, Coverage: 1, AssessedAt: now.Add(-time.Minute)},
		},
	}
	result, err := ProgramRequirementComplianceInputs(aggregate, now, "REQUIREMENTS", map[string]int64{"r-implemented": 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 4 {
		t.Fatalf("required only four currently effective approved requirements, got %+v", result)
	}
	byID := map[string]ProgramComplianceInput{}
	for _, item := range result {
		byID[item.Key] = item
	}
	if item := byID["REQUIREMENT:r-implemented"]; item.Weight != 3 || item.State != ProgramComplianceAssessed ||
		item.ScoreBps == nil || *item.ScoreBps != 10000 {
		t.Fatalf("supported requirement not scored accurately: %+v", item)
	}
	if item := byID["REQUIREMENT:r-missing"]; item.State != ProgramComplianceAssessed ||
		item.ScoreBps == nil || *item.ScoreBps != 0 {
		t.Fatalf("unimplemented control must be a proven failure: %+v", item)
	}
	if item := byID["REQUIREMENT:r-review"]; item.State != ProgramComplianceUnknown || item.ScoreBps != nil {
		t.Fatalf("unassessed evidence must reduce assessment coverage: %+v", item)
	}
	if item := byID["REQUIREMENT:r-excluded"]; item.State != ProgramComplianceNotApplicable ||
		item.ScoreBps != nil || item.Reason == "" {
		t.Fatalf("approved non-applicability was not excluded: %+v", item)
	}
	policy := ProgramComplianceCalculationPolicy{Revision: "approved-v1", MinimumCoverageBps: 10000,
		Categories: []ProgramComplianceCategoryWeight{{Code: "REQUIREMENTS", WeightBps: 10000}}}
	calculated, err := CalculateProgramCompliance(policy, result)
	if err != nil {
		t.Fatal(err)
	}
	if calculated.State != ProgramComplianceProvisional || calculated.CoverageBps != 8000 ||
		calculated.ScoreBps != nil || calculated.AssessedScoreBps == nil || *calculated.AssessedScoreBps != 7500 {
		t.Fatalf("incorrect derived requirement population or weighted score: %+v", calculated)
	}

	for _, override := range []map[string]int64{
		{"r-implemented": 0},
		{"r-implemented": 10001},
		{"r-future": 1},
		{"r-retired": 1},
	} {
		if _, err := ProgramRequirementComplianceInputs(aggregate, now, "REQUIREMENTS", override); err == nil {
			t.Fatalf("invalid requirement weight override accepted: %+v", override)
		}
	}

	aggregate.Requirements[0].ProgramID = "other-program"
	if _, err := ProgramRequirementComplianceInputs(aggregate, now, "REQUIREMENTS", nil); err == nil {
		t.Fatal("cross-program requirement was accepted")
	}
}

func TestProgramRequirementComplianceInputsTracksFreshnessAndApplicability(t *testing.T) {
	now := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	start := now.Add(-24 * time.Hour)
	expired := now.Add(-time.Minute)
	aggregate := ProgramAggregate{
		Program:                 Program{ID: "program", Status: ProgramActive},
		Requirements:            []Requirement{{ID: "r", ProgramID: "program", Status: RequirementApproved, EffectiveFrom: start}},
		Applicability:           []Applicability{{RequirementID: "r", Status: ApplicabilityPartial, EffectiveFrom: start}},
		ControlImplementations:  []ControlImplementation{{ID: "c", Status: ImplementationImplemented, EffectiveFrom: start}},
		RequirementControlLinks: []RequirementControlLink{{RequirementID: "r", ImplementationID: "c", CreatedAt: start}},
		EvidenceContracts:       []EvidenceContract{{ID: "e", RequirementID: "r", Status: EvidenceContractActive, FreshnessMinutes: 60, MinimumCoverage: 1}},
		EvidenceAssessments:     []EvidenceAssessment{{ContractID: "e", Conclusion: EvidenceSupported, Coverage: 1, AssessedAt: now.Add(-time.Minute)}},
	}
	testState := func(expected ProgramComplianceInputState) {
		t.Helper()
		items, err := ProgramRequirementComplianceInputs(aggregate, now, "REQUIREMENTS", nil)
		if err != nil || len(items) != 1 || items[0].State != expected {
			t.Fatalf("want %s; items=%+v, err=%v", expected, items, err)
		}
	}
	testState(ProgramComplianceUnknown)
	aggregate.Applicability[0].Status = ApplicabilityApplicable
	testState(ProgramComplianceAssessed)
	aggregate.EvidenceAssessments[0].AssessedAt = now.Add(-3 * time.Hour)
	testState(ProgramComplianceUnknown)
	aggregate.EvidenceAssessments[0].AssessedAt = now.Add(-time.Minute)
	aggregate.RequirementControlLinks[0].RetiredAt = &expired
	items, err := ProgramRequirementComplianceInputs(aggregate, now, "REQUIREMENTS", nil)
	if err != nil || len(items) != 1 || items[0].ScoreBps == nil || *items[0].ScoreBps != 0 {
		t.Fatalf("retired control mapping remained compliant: %+v, %v", items, err)
	}
}
