package continuity

import (
	"testing"
	"time"
)

func TestEffectiveEvidenceContractsExcludeFutureTargets(t *testing.T) {
	now := time.Date(2026, 8, 7, 14, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	aggregate := ProgramAggregate{
		Requirements: []Requirement{
			{ID: "current-requirement", Status: RequirementApproved, EffectiveFrom: past},
			{ID: "future-requirement", Status: RequirementApproved, EffectiveFrom: future},
		},
		ControlImplementations: []ControlImplementation{
			{ID: "current-control", Status: ImplementationImplemented, EffectiveFrom: past},
			{ID: "future-control", Status: ImplementationImplemented, EffectiveFrom: future},
		},
		EvidenceContracts: []EvidenceContract{
			{ID: "current-requirement-contract", RequirementID: "current-requirement", Status: EvidenceContractActive},
			{ID: "future-requirement-contract", RequirementID: "future-requirement", Status: EvidenceContractActive},
			{ID: "current-control-contract", ControlImplementationID: "current-control", Status: EvidenceContractActive},
			{ID: "future-control-contract", ControlImplementationID: "future-control", Status: EvidenceContractActive},
		},
	}

	contracts := effectiveEvidenceContracts(aggregate, now)
	if len(contracts) != 2 {
		t.Fatalf("effective contracts = %#v", contracts)
	}
	if contracts[0].ID != "current-requirement-contract" || contracts[1].ID != "current-control-contract" {
		t.Fatalf("future target leaked into current evidence contracts: %#v", contracts)
	}
}
