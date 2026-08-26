package continuity

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestMemoryAssessmentPersistenceUsesContractFreshnessBoundary(t *testing.T) {
	now := time.Date(2026, 8, 7, 15, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository()
	repo.programs["bank"] = map[string]ProgramAggregate{
		"program-1": {
			Program: Program{ID: "program-1", TenantID: "bank", Version: 1},
			EvidenceContracts: []EvidenceContract{{
				ID: "contract-1", ProgramID: "program-1", Status: EvidenceContractActive, FreshnessMinutes: 60,
			}},
		},
	}
	repo.programEvents["bank"] = map[string][]Event{"program-1": {}}

	assessment := EvidenceAssessment{
		ID: "assessment-1", TenantID: "bank", ProgramID: "program-1", ContractID: "contract-1",
		Conclusion: EvidenceSupported, Coverage: 1, AssessedAt: now, CreatedAt: now,
	}
	overlong := now.Add(24 * time.Hour)
	assessment.ValidUntil = &overlong
	payload, err := json.Marshal(assessment)
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.ApplyProgramEvent(WithTrustedSystemScope(context.Background()), "bank", "program-1", 1, Event{
		AggregateVersion: 2, Type: EventEvidenceAssessmentRecorded, Payload: payload, OccurredAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}

	aggregate, err := repo.GetProgram(WithTrustedSystemScope(context.Background()), "bank", "program-1")
	if err != nil {
		t.Fatal(err)
	}
	want := now.Add(time.Hour)
	if len(aggregate.EvidenceAssessments) != 1 || aggregate.EvidenceAssessments[0].ValidUntil == nil || !aggregate.EvidenceAssessments[0].ValidUntil.Equal(want) {
		t.Fatalf("memory aggregate validity = %#v, want %s", aggregate.EvidenceAssessments, want)
	}
	events, err := repo.ProgramEvents(WithTrustedSystemScope(context.Background()), "bank", "program-1", nil)
	if err != nil {
		t.Fatal(err)
	}
	var persisted EvidenceAssessment
	if err := json.Unmarshal(events[len(events)-1].Payload, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.ValidUntil == nil || !persisted.ValidUntil.Equal(want) {
		t.Fatalf("memory replay event retained invalid validity: %#v", persisted.ValidUntil)
	}
}

func TestStateDerivationIgnoresEvidenceContractWhoseTargetIsNotCurrent(t *testing.T) {
	now := time.Date(2026, 8, 7, 15, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	aggregate := ProgramAggregate{
		Program: Program{ID: "program-1", TenantID: "bank", Status: ProgramActive, EffectiveFrom: past, Version: 3},
		Requirements: []Requirement{
			{ID: "req-current", Title: "Current", Status: RequirementApproved, EffectiveFrom: past},
			{ID: "req-future", Title: "Future", Status: RequirementApproved, EffectiveFrom: future},
		},
		Applicability: []Applicability{{ID: "app-current", RequirementID: "req-current", Status: ApplicabilityNotApplicable, EffectiveFrom: past}},
		EvidenceContracts: []EvidenceContract{{
			ID: "future-contract", RequirementID: "req-future", Name: "Future evidence", Status: EvidenceContractActive,
			FreshnessMinutes: 60, AcceptableSourceIDs: []string{"source-future"},
		}},
	}

	state := deriveProgramState(aggregate, 0, now)
	if state.Dimensions.Applicability != StateNotApplicable {
		t.Fatalf("current applicability = %s", state.Dimensions.Applicability)
	}
	if state.Dimensions.EvidenceSufficiency != StateNotApplicable {
		t.Fatalf("future-target evidence contract polluted current sufficiency: %s", state.Dimensions.EvidenceSufficiency)
	}
	for _, reason := range state.Reasons {
		if reason.ObjectID == "future-contract" {
			t.Fatalf("future evidence contract leaked into current reasons: %#v", state.Reasons)
		}
	}
}
