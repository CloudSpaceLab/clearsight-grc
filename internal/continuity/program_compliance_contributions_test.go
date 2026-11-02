package continuity

import (
	"testing"
	"time"
)

func requirementScoreDrillFixture(t *testing.T, now time.Time) (ProgramAggregate, ProgramStateSnapshot) {
	t.Helper()
	policy := ProgramComplianceCalculationPolicy{
		Revision:           "program-v1",
		MinimumCoverageBps: 10000,
		Categories:         []ProgramComplianceCategoryWeight{{Code: "REQUIREMENTS", WeightBps: 10000}},
	}
	checksum, err := programCompliancePolicyChecksum(policy)
	if err != nil {
		t.Fatal(err)
	}
	approved := now.Add(-time.Hour)
	aggregate := ProgramAggregate{
		Program: Program{
			ID: "program", TenantID: "bank", LegalEntityID: "entity", Version: 3, Status: ProgramActive,
			CalculationPolicy: &ProgramCompliancePolicyPosition{
				LastRevision: 1,
				Active: &ProgramCompliancePolicyRevision{
					Number: 1, Definition: policy, Checksum: checksum, MakerID: "owner",
					ApproverID: "reviewer", ApprovedAt: &approved,
				},
			},
		},
		Requirements: []Requirement{{
			ID: "requirement", ProgramID: "program", Status: RequirementApproved,
			EffectiveFrom: now.Add(-2 * time.Hour),
		}},
		Applicability: []Applicability{{
			RequirementID: "requirement", Status: ApplicabilityApplicable,
			EffectiveFrom: now.Add(-2 * time.Hour),
		}},
	}
	items, err := ProgramRequirementComplianceInputs(aggregate, now, "REQUIREMENTS", nil)
	if err != nil {
		t.Fatal(err)
	}
	calculated, err := CalculateProgramCompliance(policy, items)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := ProgramStateSnapshot{
		TenantID: "bank", ProgramID: "program", ProgramVersion: 3,
		GeneratedAt: now, Calculation: &calculated,
	}
	return aggregate, snapshot
}

func TestProgramRequirementScoreContributionsMatchRetainedSnapshot(t *testing.T) {
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	aggregate, snapshot := requirementScoreDrillFixture(t, now)
	inputs, err := ProgramRequirementScoreContributions(aggregate, snapshot)
	if err != nil || len(inputs) != 1 ||
		inputs[0].Key != "REQUIREMENT:requirement" ||
		inputs[0].ScoreBps == nil || *inputs[0].ScoreBps != 0 {
		t.Fatalf("retained score did not reconcile to its source: %+v, %v", inputs, err)
	}

	memory := aggregate
	memory.Program.Version++
	memorySnapshot := snapshot
	memorySnapshot.ID = "retained-snapshot"
	memory.CurrentState = &memorySnapshot
	if _, err := ProgramRequirementScoreContributions(memory, memorySnapshot); err != nil {
		t.Fatalf("valid memory fallback projection-only version was rejected: %v", err)
	}
	memory.CurrentState = nil
	if _, err := ProgramRequirementScoreContributions(memory, memorySnapshot); err == nil {
		t.Fatal("unverified one-version advance was incorrectly trusted")
	}

	changed := snapshot
	score := *snapshot.Calculation
	inflated := int64(10000)
	score.ScoreBps = &inflated
	changed.Calculation = &score
	if _, err := ProgramRequirementScoreContributions(aggregate, changed); err == nil {
		t.Fatal("tampered total received plausible contributors")
	}
	changed = snapshot
	score = *snapshot.Calculation
	score.CoverageBps = 5000
	changed.Calculation = &score
	if _, err := ProgramRequirementScoreContributions(aggregate, changed); err == nil {
		t.Fatal("different assessed denominator was silently explained")
	}
}

func TestProgramRequirementScoreContributionsRejectForeignOrRevisedPopulation(t *testing.T) {
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	aggregate, snapshot := requirementScoreDrillFixture(t, now)
	missing := snapshot
	missing.Calculation = nil
	if _, err := ProgramRequirementScoreContributions(aggregate, missing); err == nil {
		t.Fatal("unscored Program yielded fabricated requirement contributions")
	}
	foreign := snapshot
	foreign.TenantID = "another-tenant"
	if _, err := ProgramRequirementScoreContributions(aggregate, foreign); err == nil {
		t.Fatal("cross-tenant snapshot matched")
	}
	foreign = snapshot
	foreign.ProgramID = "another-program"
	if _, err := ProgramRequirementScoreContributions(aggregate, foreign); err == nil {
		t.Fatal("cross-Program snapshot matched")
	}
	revised := aggregate
	revised.Program.Version++
	if _, err := ProgramRequirementScoreContributions(revised, snapshot); err == nil {
		t.Fatal("later Program revision was used to explain previous assessment")
	}
	aggregate.Program.CalculationPolicy.Active.Definition.Categories[0].WeightBps = 9000
	if _, err := ProgramRequirementScoreContributions(aggregate, snapshot); err == nil {
		t.Fatal("edited policy was used to explain retained snapshot")
	}
}
