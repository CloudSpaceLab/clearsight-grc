package continuity

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestProgramStateForVisibleMattersRemovesHiddenMatterContribution(t *testing.T) {
	canonical := ProgramStateSnapshot{
		ID:                "canonical-snapshot",
		TenantID:          "bank",
		ProgramID:         "program-1",
		Overall:           StateAtRisk,
		Dimensions:        allCurrentDimensions(),
		Reasons:           []StateReason{{Code: "OPEN_MATTERS", Summary: "2 open issue(s) or change(s) affect this program."}, {Code: "OTHER", Summary: "Visible non-Matter reason."}},
		OpenMatterCount:   2,
		TriggerType:       EventMatterStateChanged,
		TriggerID:         "hidden-matter",
		GeneratedAt:       time.Date(2026, 8, 22, 18, 0, 0, 0, time.UTC),
		ProgramVersion:    7,
		ProjectionVersion: 91,
	}
	canonical.Dimensions.Exception = StateAtRisk

	hidden := programStateForVisibleMatters(canonical, 0)
	if hidden.OpenMatterCount != 0 || hidden.Dimensions.Exception != StateCurrent {
		t.Fatalf("hidden Matter contribution survived: %#v", hidden)
	}
	if hidden.Overall != StateCurrent {
		t.Fatalf("overall state still includes hidden Matter: %s", hidden.Overall)
	}
	if hasReasonCode(hidden.Reasons, "OPEN_MATTERS") {
		t.Fatalf("OPEN_MATTERS reason leaked: %#v", hidden.Reasons)
	}
	if !hasReasonCode(hidden.Reasons, "OTHER") {
		t.Fatalf("non-Matter reason was removed: %#v", hidden.Reasons)
	}
	if hidden.ID != "" || hidden.TriggerType != "" || hidden.TriggerID != "" {
		t.Fatalf("internal projection provenance leaked: %#v", hidden)
	}
	if hidden.ProjectionVersion < 1 {
		t.Fatalf("actor semantic version is invalid: %d", hidden.ProjectionVersion)
	}

	visible := programStateForVisibleMatters(canonical, 1)
	if visible.OpenMatterCount != 1 || visible.Dimensions.Exception != StateAtRisk || visible.Overall != StateAtRisk {
		t.Fatalf("visible Matter contribution was lost: %#v", visible)
	}
	if len(visible.Reasons) != 2 || !hasReasonCode(visible.Reasons, "OPEN_MATTERS") {
		t.Fatalf("visible Matter reason missing: %#v", visible.Reasons)
	}
	if visible.ProjectionVersion == hidden.ProjectionVersion {
		t.Fatal("different visible semantic states share one actor projection version")
	}
}

func TestVisibleProgramStateVersionIgnoresCanonicalProjectionChurn(t *testing.T) {
	first := ProgramStateSnapshot{
		ID:                "snapshot-a",
		TenantID:          "bank",
		ProgramID:         "program-1",
		Overall:           StateAtRisk,
		Dimensions:        allCurrentDimensions(),
		Reasons:           []StateReason{{Code: "OPEN_MATTERS", Summary: "1 open issue(s) or change(s) affect this program."}},
		OpenMatterCount:   1,
		TriggerType:       EventMatterStateChanged,
		TriggerID:         "matter-a",
		GeneratedAt:       time.Date(2026, 8, 22, 18, 0, 0, 0, time.UTC),
		ProgramVersion:    4,
		ProjectionVersion: 10,
	}
	first.Dimensions.Exception = StateAtRisk
	second := first
	second.ID = "snapshot-b"
	second.TriggerID = "matter-b"
	second.GeneratedAt = first.GeneratedAt.Add(time.Hour)
	second.ProjectionVersion = 99

	firstVisible := programStateForVisibleMatters(first, 0)
	secondVisible := programStateForVisibleMatters(second, 0)
	if firstVisible.ProjectionVersion != secondVisible.ProjectionVersion {
		t.Fatalf("canonical hidden projection churn changed actor version: %d != %d", firstVisible.ProjectionVersion, secondVisible.ProjectionVersion)
	}
}

func TestMemoryProgramForPrincipalCountsOnlyReadableOpenMatters(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	service := NewService(repo)
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)

	program := Program{ID: "program-1", TenantID: "bank", Status: ProgramActive, Version: 1, UpdatedAt: now}
	state := ProgramStateSnapshot{TenantID: "bank", ProgramID: program.ID, ProgramVersion: 1, ProjectionVersion: 1, Overall: StateAtRisk, Dimensions: allCurrentDimensions(), OpenMatterCount: 2, Reasons: []StateReason{{Code: "OPEN_MATTERS", Summary: "2 open issue(s) or change(s) affect this program."}}}
	state.Dimensions.Exception = StateAtRisk
	aggregate := ProgramAggregate{Program: program, CurrentState: &state}
	repo.programs["bank"] = map[string]ProgramAggregate{program.ID: aggregate}

	repo.matters["bank"] = map[string]MatterAggregate{
		"visible": {
			Matter: Matter{ID: "visible", TenantID: "bank", Status: MatterAssessment, Scope: json.RawMessage(`{"access":"INTERNAL"}`)},
			Links:  []MatterLink{{MatterID: "visible", ProgramID: program.ID}},
		},
		"hidden": {
			Matter: Matter{ID: "hidden", TenantID: "bank", Status: MatterAssessment, Scope: json.RawMessage(`{"access":"RESTRICTED","allowed_principal_ids":["person-b"]}`)},
			Links:  []MatterLink{{MatterID: "hidden", ProgramID: program.ID}},
		},
	}

	forA, err := service.ProgramForPrincipal(ctx, aggregate, "person-a", nil)
	if err != nil {
		t.Fatal(err)
	}
	if forA.CurrentState == nil || forA.CurrentState.OpenMatterCount != 1 {
		t.Fatalf("person-a saw wrong Matter count: %#v", forA.CurrentState)
	}
	forB, err := service.ProgramForPrincipal(ctx, aggregate, "person-b", nil)
	if err != nil {
		t.Fatal(err)
	}
	if forB.CurrentState == nil || forB.CurrentState.OpenMatterCount != 2 {
		t.Fatalf("allowed principal lost restricted Matter contribution: %#v", forB.CurrentState)
	}
}

func allCurrentDimensions() ComplianceDimensions {
	return ComplianceDimensions{
		Interpretation:         StateCurrent,
		Applicability:          StateCurrent,
		ControlDesign:          StateCurrent,
		Implementation:         StateCurrent,
		EvidenceSufficiency:    StateCurrent,
		OperatingEffectiveness: StateCurrent,
		Exception:              StateCurrent,
		Assurance:              StateCurrent,
		Deadline:               StateCurrent,
		SourceQuality:          StateCurrent,
	}
}

func hasReasonCode(values []StateReason, code string) bool {
	for _, value := range values {
		if value.Code == code {
			return true
		}
	}
	return false
}
