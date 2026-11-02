package continuity

import (
	"context"
	"strings"
	"testing"
	"time"
)

func approvedRequirementsPolicyForProjection(t *testing.T, now time.Time) *ProgramCompliancePolicyPosition {
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
	approved := now.Add(-time.Minute)
	return &ProgramCompliancePolicyPosition{
		LastRevision: 1,
		Active: &ProgramCompliancePolicyRevision{
			Number:     1,
			Definition: policy,
			Checksum:   checksum,
			MakerID:    "owner",
			ApproverID: "checker",
			ApprovedAt: &approved,
		},
	}
}

func requirementsProjectionFixture(t *testing.T, now time.Time) ProgramAggregate {
	t.Helper()
	start := now.Add(-2 * time.Hour)
	return ProgramAggregate{
		Program: Program{
			ID:                "program",
			TenantID:          "bank",
			LegalEntityID:     "entity",
			Status:            ProgramActive,
			Version:           4,
			EffectiveFrom:     start,
			CalculationPolicy: approvedRequirementsPolicyForProjection(t, now),
		},
		Requirements: []Requirement{{
			ID:            "requirement",
			ProgramID:     "program",
			Status:        RequirementApproved,
			EffectiveFrom: start,
		}},
		Applicability: []Applicability{{
			RequirementID: "requirement",
			Status:        ApplicabilityApplicable,
			EffectiveFrom: start,
			CreatedAt:     start,
		}},
		ControlImplementations: []ControlImplementation{{
			ID:            "control",
			Status:        ImplementationImplemented,
			EffectiveFrom: start,
		}},
		RequirementControlLinks: []RequirementControlLink{{
			ID:               "link",
			RequirementID:    "requirement",
			ImplementationID: "control",
			CreatedAt:        start,
		}},
		EvidenceContracts: []EvidenceContract{{
			ID:               "contract",
			RequirementID:    "requirement",
			Status:           EvidenceContractActive,
			FreshnessMinutes: 120,
			MinimumCoverage:  1,
		}},
		EvidenceAssessments: []EvidenceAssessment{{
			ID:         "assessment",
			ContractID: "contract",
			Conclusion: EvidenceSupported,
			Coverage:   1,
			AssessedAt: now.Add(-time.Minute),
		}},
	}
}

func TestRequirementsOnlyCalculationUsesApprovedSourceCoverage(t *testing.T) {
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	aggregate := requirementsProjectionFixture(t, now)
	healthy := ProgramSourceState{Known: true, Current: true, Required: 1}
	value, err := requirementsOnlyProgramCalculation(aggregate, now, healthy)
	if err != nil || value == nil || value.State != ProgramComplianceFinal ||
		value.ScoreBps == nil || *value.ScoreBps != 10000 ||
		value.CoverageBps != 10000 || value.PolicyRevision != "program-v1" {
		t.Fatalf("approved fully supported requirement not scored: %+v, %v", value, err)
	}

	// An implemented control without assessed current evidence is unknown,
	// never an assumed success nor a confirmed failed control.
	aggregate.EvidenceAssessments = nil
	value, err = requirementsOnlyProgramCalculation(aggregate, now, healthy)
	if err != nil || value == nil || value.ScoreBps != nil ||
		value.CoverageBps != 0 || value.State != ProgramComplianceNotAssessed {
		t.Fatalf("missing evidence manufactured compliance: %+v, %v", value, err)
	}

	// A proven missing safeguard is a known compliance failure.
	aggregate.ControlImplementations = nil
	value, err = requirementsOnlyProgramCalculation(aggregate, now, healthy)
	if err != nil || value == nil || value.ScoreBps == nil ||
		*value.ScoreBps != 0 || value.CoverageBps != 10000 {
		t.Fatalf("confirmed safeguard failure became unknown: %+v, %v", value, err)
	}
}

func TestRequirementsOnlyCalculationGatesSourcePolicyAndPeriod(t *testing.T) {
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	value := requirementsProjectionFixture(t, now)
	healthy := ProgramSourceState{Known: true, Current: true, Required: 1}
	expectNoScore := func(name string, scope ProgramAggregate, at time.Time, source ProgramSourceState) {
		t.Helper()
		calculation, err := requirementsOnlyProgramCalculation(scope, at, source)
		if err != nil || calculation != nil {
			t.Fatalf("%s published an unsupported score: %+v, %v", name, calculation, err)
		}
	}
	expectNoScore("stale source", value, now, ProgramSourceState{Known: true, Required: 1})
	expectNoScore("unavailable source", value, now, ProgramSourceState{Required: 1})
	expectNoScore("approval in future", value, now.Add(-2*time.Minute), healthy)

	draft := value
	draft.Program.Status = ProgramDraft
	expectNoScore("draft Program", draft, now, healthy)
	pending := value
	pending.Program.CalculationPolicy = &ProgramCompliancePolicyPosition{Pending: value.Program.CalculationPolicy.Active}
	expectNoScore("pending policy", pending, now, healthy)

	multi := value
	multi.Program.CalculationPolicy = clonePolicyForProjection(t, value.Program.CalculationPolicy, func(p *ProgramComplianceCalculationPolicy) {
		p.Categories = []ProgramComplianceCategoryWeight{{Code: "REQUIREMENTS", WeightBps: 7000}, {Code: "VENDORS", WeightBps: 3000}}
	})
	expectNoScore("vendor category needs authoritative source", multi, now, healthy)

	multi.Program.CalculationPolicy = clonePolicyForProjection(t, value.Program.CalculationPolicy, func(p *ProgramComplianceCalculationPolicy) {
		p.Categories = []ProgramComplianceCategoryWeight{{Code: "REQUIREMENTS", WeightBps: 8000}, {Code: "ISSUES", WeightBps: 2000}}
	})
	expectNoScore("issues category needs actor-safe source", multi, now, healthy)

	bad := value
	revision := cloneProgramCompliancePolicyPosition(value.Program.CalculationPolicy)
	revision.Active.Checksum = strings.Repeat("0", 64)
	bad.Program.CalculationPolicy = &revision
	if calculation, err := requirementsOnlyProgramCalculation(bad, now, healthy); err == nil || calculation != nil {
		t.Fatalf("tampered approved policy produced score: %+v, %v", calculation, err)
	}
}

func clonePolicyForProjection(t *testing.T, position *ProgramCompliancePolicyPosition, edit func(*ProgramComplianceCalculationPolicy)) *ProgramCompliancePolicyPosition {
	t.Helper()
	value := cloneProgramCompliancePolicyPosition(position)
	edit(&value.Active.Definition)
	checksum, err := programCompliancePolicyChecksum(value.Active.Definition)
	if err != nil {
		t.Fatal(err)
	}
	value.Active.Checksum = checksum
	return &value
}

func TestProgramRefreshRetainsRequirementsCalculationWithoutAnotherProjection(t *testing.T) {
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	aggregate := requirementsProjectionFixture(t, now)
	repo := NewMemoryRepository()
	repo.programs["bank"] = map[string]ProgramAggregate{"program": aggregate}
	repo.programEvents["bank"] = map[string][]Event{"program": {}}
	service := NewService(repo)
	service.now = func() time.Time { return now }
	ctx := WithTrustedSystemScope(context.Background())
	if err := service.refreshProgramCurrent(ctx, "bank", "program", "POLICY_APPROVED", ""); err != nil {
		t.Fatal(err)
	}
	saved, err := service.GetProgram(ctx, "bank", "program")
	if err != nil {
		t.Fatal(err)
	}
	if saved.CurrentState == nil || saved.CurrentState.Calculation == nil ||
		saved.CurrentState.Calculation.ScoreBps == nil ||
		*saved.CurrentState.Calculation.ScoreBps != 10000 ||
		saved.CurrentState.Calculation.PolicyRevision != "program-v1" {
		t.Fatalf("approved score did not use existing Program projection: %+v", saved.CurrentState)
	}
}
