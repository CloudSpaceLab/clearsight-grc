package continuity

import (
	"encoding/json"
	"testing"
)

func TestProgramCalculationSnapshotRoundTripAndEquivalence(t *testing.T) {
	score := int64(7500)
	snapshot := ProgramStateSnapshot{
		ProgramID: "program", ProgramVersion: 7, ProjectionVersion: 3,
		Calculation: &ProgramComplianceCalculation{
			PolicyRevision: "program-v2",
			State: ProgramComplianceFinal,
			ScoreBps: &score,
			AssessedScoreBps: &score,
			CoverageBps: 10000,
			EligibleCount: 2,
			AssessedCount: 2,
			Categories: []ProgramComplianceCategoryResult{{
				Code: "REQUIREMENTS", WeightBps: 10000, EligibleCount: 2,
				AssessedCount: 2, CoverageBps: 10000, ScoreBps: &score,
			}},
		},
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	var restored ProgramStateSnapshot
	if err := json.Unmarshal(raw, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.Calculation == nil || restored.Calculation.ScoreBps == nil ||
		*restored.Calculation.ScoreBps != 7500 ||
		restored.Calculation.PolicyRevision != "program-v2" ||
		!stateEquivalent(snapshot, restored) {
		t.Fatalf("retained snapshot lost calculation provenance: %+v", restored.Calculation)
	}
	without := snapshot
	without.Calculation = nil
	if stateEquivalent(snapshot, without) {
		t.Fatal("scoring change was suppressed by Program state equivalence")
	}
	raw, err = json.Marshal(without)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) == "" || json.Valid(raw) == false {
		t.Fatal("unconfigured Program must remain serializable")
	}
	if _, present := func() (json.RawMessage, bool) {
		var value map[string]json.RawMessage
		if err := json.Unmarshal(raw, &value); err != nil {
			t.Fatal(err)
		}
		data, ok := value["calculation"]
		return data, ok
	}(); present {
		t.Fatal("unconfigured Program emitted a fabricated score")
	}
}
