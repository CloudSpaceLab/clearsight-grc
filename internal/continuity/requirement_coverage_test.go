package continuity

import (
	"testing"
	"time"
)

func TestCurrentRequirementCoverage(t *testing.T) {
	now := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	complete := ProgramAggregate{
		Program: Program{ID: "program-1", Status: ProgramActive},
		Requirements: []Requirement{{
			ID: "req-1", Status: RequirementApproved, EffectiveFrom: now.Add(-time.Hour),
		}},
		Applicability: []Applicability{{
			RequirementID: "req-1", Status: ApplicabilityApplicable,
			EffectiveFrom: now.Add(-time.Hour), CreatedAt: now.Add(-time.Hour),
		}},
		ControlImplementations: []ControlImplementation{{
			ID: "control-1", Status: ImplementationImplemented, EffectiveFrom: now.Add(-time.Hour),
		}},
		RequirementControlLinks: []RequirementControlLink{{
			RequirementID: "req-1", ImplementationID: "control-1",
		}},
		EvidenceContracts: []EvidenceContract{{
			ID: "contract-1", RequirementID: "req-1", ControlImplementationID: "control-1",
			Status: EvidenceContractActive, FreshnessMinutes: 60, MinimumCoverage: .9,
		}},
		EvidenceAssessments: []EvidenceAssessment{{
			ContractID: "contract-1", Conclusion: EvidenceSupported, Coverage: 1,
			AssessedAt: now.Add(-time.Minute), CreatedAt: now.Add(-time.Minute),
		}},
	}

	t.Run("complete chain", func(t *testing.T) {
		got := CurrentRequirementCoverage(complete, now)["req-1"]
		if !got.Applicable || !got.ControlImplemented || !got.EvidenceSupported || !got.Complete {
			t.Fatalf("expected complete chain, got %#v", got)
		}
		if len(got.ControlIDs) != 1 || got.ControlIDs[0] != "control-1" {
			t.Fatalf("unexpected controls: %#v", got.ControlIDs)
		}
		if len(got.EvidenceContractIDs) != 1 || got.EvidenceContractIDs[0] != "contract-1" {
			t.Fatalf("unexpected contracts: %#v", got.EvidenceContractIDs)
		}
	})

	t.Run("missing control", func(t *testing.T) {
		aggregate := complete
		aggregate.RequirementControlLinks = nil
		got := CurrentRequirementCoverage(aggregate, now)["req-1"]
		if !got.Applicable || got.ControlImplemented || got.EvidenceSupported || got.Complete {
			t.Fatalf("missing control must break the chain, got %#v", got)
		}
	})

	t.Run("stale evidence", func(t *testing.T) {
		aggregate := complete
		aggregate.EvidenceAssessments = append([]EvidenceAssessment(nil), complete.EvidenceAssessments...)
		aggregate.EvidenceAssessments[0].AssessedAt = now.Add(-2 * time.Hour)
		aggregate.EvidenceAssessments[0].CreatedAt = now.Add(-2 * time.Hour)
		got := CurrentRequirementCoverage(aggregate, now)["req-1"]
		if !got.Applicable || !got.ControlImplemented || got.EvidenceSupported || got.Complete {
			t.Fatalf("stale evidence must break the chain, got %#v", got)
		}
	})

	t.Run("one failed contract breaks evidence support", func(t *testing.T) {
		aggregate := complete
		aggregate.EvidenceContracts = append(append([]EvidenceContract(nil), complete.EvidenceContracts...), EvidenceContract{
			ID: "contract-2", RequirementID: "req-1", Status: EvidenceContractActive,
			FreshnessMinutes: 60, MinimumCoverage: .9,
		})
		aggregate.EvidenceAssessments = append(append([]EvidenceAssessment(nil), complete.EvidenceAssessments...), EvidenceAssessment{
			ContractID: "contract-2", Conclusion: EvidenceUnsupported, Coverage: 1,
			AssessedAt: now.Add(-time.Minute), CreatedAt: now.Add(-time.Minute),
		})
		got := CurrentRequirementCoverage(aggregate, now)["req-1"]
		if got.EvidenceSupported || got.Complete {
			t.Fatalf("all relevant contracts must be supported, got %#v", got)
		}
	})
}
