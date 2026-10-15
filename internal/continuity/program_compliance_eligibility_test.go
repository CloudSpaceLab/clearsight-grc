package continuity

import (
	"testing"
	"time"
)

// Program status and requirement evidence coverage must use the same historical
// link membership. Persisted aggregate snapshots can contain retired links.
func TestRequirementCoverageAndProgramStateUseEffectiveControlLinks(t *testing.T) {
	now := time.Date(2026, 10, 15, 8, 0, 0, 0, time.UTC)
	created := now.Add(-time.Hour)
	retired := now.Add(-15 * time.Minute)
	assessed := now.Add(-40 * time.Minute)

	aggregate := ProgramAggregate{
		Program: Program{ID: "p", TenantID: "bank", Status: ProgramActive, EffectiveFrom: created, Version: 1},
		Requirements: []Requirement{{
			ID: "r", ProgramID: "p", Status: RequirementApproved, EffectiveFrom: created,
		}},
		Applicability: []Applicability{{
			ID: "a", RequirementID: "r", Status: ApplicabilityApplicable,
			EffectiveFrom: created, CreatedAt: created,
		}},
		ControlImplementations: []ControlImplementation{{
			ID: "c", ProgramID: "p", Status: ImplementationImplemented, EffectiveFrom: created,
		}},
		RequirementControlLinks: []RequirementControlLink{{
			ID: "link", ProgramID: "p", RequirementID: "r", ImplementationID: "c",
			CreatedAt: created, RetiredAt: &retired,
		}},
		EvidenceContracts: []EvidenceContract{{
			ID: "e", ProgramID: "p", RequirementID: "r", Status: EvidenceContractActive,
			MinimumCoverage: 1, FreshnessMinutes: 180,
		}},
		EvidenceAssessments: []EvidenceAssessment{{
			ID: "assessment", ProgramID: "p", ContractID: "e",
			Conclusion: EvidenceSupported, Coverage: 1, AssessedAt: assessed, CreatedAt: assessed,
		}},
	}

	t.Run("before retirement", func(t *testing.T) {
		at := retired.Add(-time.Second)
		coverage := CurrentRequirementCoverage(aggregate, at)["r"]
		if !coverage.Complete || !coverage.ControlImplemented {
			t.Fatalf("historically valid implementation was lost: %+v", coverage)
		}
		state := deriveProgramStateWithSourceState(aggregate, 0, at, ProgramSourceState{Known: true})
		if state.Dimensions.ControlDesign != StateCurrent || state.Dimensions.Implementation != StateCurrent {
			t.Fatalf("historical control state incorrectly degraded: %+v", state.Dimensions)
		}
	})

	t.Run("at and after retirement", func(t *testing.T) {
		for _, at := range []time.Time{retired, now} {
			coverage := CurrentRequirementCoverage(aggregate, at)["r"]
			if coverage.ControlImplemented || coverage.EvidenceSupported || coverage.Complete || len(coverage.ControlIDs) != 0 {
				t.Fatalf("retired link produced false compliance at %s: %+v", at, coverage)
			}
			state := deriveProgramStateWithSourceState(aggregate, 0, at, ProgramSourceState{Known: true})
			if state.Dimensions.ControlDesign != StateGapIdentified {
				t.Fatalf("retired link kept control design current at %s: %+v", at, state.Dimensions)
			}
			if state.Dimensions.Implementation != StateGapIdentified {
				t.Fatalf("retired link kept implementation current at %s: %+v", at, state.Dimensions)
			}
		}
	})

	t.Run("link cannot apply before creation", func(t *testing.T) {
		future := now.Add(time.Hour)
		aggregate.RequirementControlLinks[0].CreatedAt = future
		aggregate.RequirementControlLinks[0].RetiredAt = nil
		coverage := CurrentRequirementCoverage(aggregate, now)["r"]
		if coverage.ControlImplemented || coverage.Complete {
			t.Fatalf("future control mapping entered current denominator: %+v", coverage)
		}
		state := deriveProgramStateWithSourceState(aggregate, 0, now, ProgramSourceState{Known: true})
		if state.Dimensions.ControlDesign != StateGapIdentified {
			t.Fatalf("future link masked unmapped requirement: %+v", state.Dimensions)
		}
	})
}

func TestPartiallyApplicableRequirementDoesNotCountAsProved(t *testing.T) {
	now := time.Date(2026, 10, 15, 8, 0, 0, 0, time.UTC)
	created := now.Add(-time.Hour)
	aggregate := ProgramAggregate{
		Program:      Program{ID: "p", TenantID: "bank", Status: ProgramActive, EffectiveFrom: created, Version: 1},
		Requirements: []Requirement{{ID: "r", Status: RequirementApproved, EffectiveFrom: created}},
		Applicability: []Applicability{{
			RequirementID: "r", Status: ApplicabilityPartial, EffectiveFrom: created, CreatedAt: created,
		}},
		ControlImplementations: []ControlImplementation{{
			ID: "c", Status: ImplementationImplemented, EffectiveFrom: created,
		}},
		RequirementControlLinks: []RequirementControlLink{{
			RequirementID: "r", ImplementationID: "c", CreatedAt: created,
		}},
	}
	coverage := CurrentRequirementCoverage(aggregate, now)["r"]
	if coverage.Complete || coverage.Applicable {
		t.Fatalf("unresolved partial applicability was treated as proved: %+v", coverage)
	}
	state := deriveProgramStateWithSourceState(aggregate, 0, now, ProgramSourceState{Known: true})
	if state.Dimensions.Applicability != StateUnderReview || state.Overall == StateCurrent {
		t.Fatalf("Program became current without complete applicability: %+v", state)
	}
}
