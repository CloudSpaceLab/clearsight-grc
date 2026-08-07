package continuity

import (
	"encoding/json"
	"testing"
	"time"
)

func TestProgramStateIgnoresFutureAndExpiredRecords(t *testing.T) {
	now := time.Date(2026, 8, 7, 14, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	expired := now.Add(-time.Minute)
	program := Program{
		ID:            "program-1",
		TenantID:      "bank",
		Status:        ProgramActive,
		EffectiveFrom: past,
		Version:       7,
	}

	aggregate := ProgramAggregate{
		Program: program,
		Requirements: []Requirement{
			{
				ID:            "req-current",
				TenantID:      "bank",
				ProgramID:     program.ID,
				Title:         "Current requirement",
				Status:        RequirementApproved,
				EffectiveFrom: past,
			},
			{
				ID:            "req-future",
				TenantID:      "bank",
				ProgramID:     program.ID,
				Title:         "Future requirement",
				Status:        RequirementApproved,
				EffectiveFrom: future,
			},
		},
		Applicability: []Applicability{
			{
				ID:             "app-expired",
				RequirementID:  "req-current",
				Status:         ApplicabilityApplicable,
				EffectiveFrom:  past.Add(-time.Hour),
				EffectiveUntil: &expired,
				CreatedAt:      past.Add(-time.Hour),
			},
		},
	}

	state := deriveProgramStateWithSourceState(aggregate, 0, now, ProgramSourceState{Known: true})
	if state.Dimensions.Interpretation != StateCurrent {
		t.Fatalf("current requirement was not selected: %#v", state.Dimensions)
	}
	if state.Dimensions.Applicability != StateUnderReview {
		t.Fatalf("expired applicability was treated as current: %#v", state.Dimensions)
	}
	for _, reason := range state.Reasons {
		if reason.ObjectID == "req-future" {
			t.Fatalf("future requirement leaked into current reasons: %#v", state.Reasons)
		}
	}
}

func TestProgramStateTreatsFutureImplementationAsNotCurrent(t *testing.T) {
	now := time.Date(2026, 8, 7, 14, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	aggregate := ProgramAggregate{
		Program: Program{
			ID:            "program-1",
			TenantID:      "bank",
			Status:        ProgramActive,
			EffectiveFrom: past,
			Version:       4,
		},
		Requirements: []Requirement{
			{
				ID:            "req",
				TenantID:      "bank",
				ProgramID:     "program-1",
				Title:         "Requirement",
				Status:        RequirementApproved,
				EffectiveFrom: past,
			},
		},
		Applicability: []Applicability{
			{
				ID:            "app",
				RequirementID: "req",
				Status:        ApplicabilityApplicable,
				EffectiveFrom: past,
				CreatedAt:     past,
			},
		},
		ControlImplementations: []ControlImplementation{
			{
				ID:            "control",
				ProgramID:     "program-1",
				Status:        ImplementationImplemented,
				EffectiveFrom: future,
			},
		},
		RequirementControlLinks: []RequirementControlLink{
			{RequirementID: "req", ImplementationID: "control"},
		},
	}

	state := deriveProgramStateWithSourceState(aggregate, 0, now, ProgramSourceState{Known: true})
	if state.Dimensions.ControlDesign != StateCurrent {
		t.Fatalf("control mapping itself should be current: %#v", state.Dimensions)
	}
	if state.Dimensions.Implementation != StateGapIdentified {
		t.Fatalf("future implementation was treated as operating: %#v", state.Dimensions)
	}
}

func TestEvidenceValidityIsBoundedByContractFreshness(t *testing.T) {
	now := time.Date(2026, 8, 7, 14, 0, 0, 0, time.UTC)
	past := now.Add(-2 * time.Hour)
	farFuture := now.Add(24 * time.Hour)
	contract := EvidenceContract{
		ID:               "contract",
		Status:           EvidenceContractActive,
		FreshnessMinutes: 60,
		MinimumCoverage:  1,
		Name:             "Daily evidence",
	}
	assessment := EvidenceAssessment{
		ContractID: contract.ID,
		Conclusion: EvidenceSupported,
		Coverage:   1,
		AssessedAt: past,
		ValidUntil: &farFuture,
		CreatedAt:  past,
	}

	bounded := boundedAssessmentValidity(assessment, contract)
	want := past.Add(time.Hour)
	if !bounded.Equal(want) {
		t.Fatalf("validity=%s want=%s", bounded, want)
	}

	aggregate := ProgramAggregate{
		Program: Program{
			ID:            "program-1",
			TenantID:      "bank",
			Status:        ProgramActive,
			EffectiveFrom: past.Add(-time.Hour),
			Version:       5,
		},
		Requirements: []Requirement{
			{
				ID:            "req",
				TenantID:      "bank",
				ProgramID:     "program-1",
				Title:         "Requirement",
				Status:        RequirementApproved,
				EffectiveFrom: past.Add(-time.Hour),
			},
		},
		Applicability: []Applicability{
			{
				ID:            "app",
				RequirementID: "req",
				Status:        ApplicabilityApplicable,
				EffectiveFrom: past.Add(-time.Hour),
				CreatedAt:     past.Add(-time.Hour),
			},
		},
		ControlImplementations: []ControlImplementation{
			{
				ID:            "control",
				ProgramID:     "program-1",
				Status:        ImplementationImplemented,
				EffectiveFrom: past.Add(-time.Hour),
			},
		},
		RequirementControlLinks: []RequirementControlLink{
			{RequirementID: "req", ImplementationID: "control"},
		},
		EvidenceContracts:   []EvidenceContract{contract},
		EvidenceAssessments: []EvidenceAssessment{assessment},
	}
	state := deriveProgramStateWithSourceState(aggregate, 0, now, ProgramSourceState{Known: true})
	if state.Dimensions.EvidenceSufficiency != StateEvidenceInsufficient {
		t.Fatalf("assessment remained current past contract freshness: %#v", state.Dimensions)
	}
}

func TestUnknownMandatoryDimensionsCannotBecomeCurrent(t *testing.T) {
	allCurrent := ComplianceDimensions{
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
	if got := chooseOverallState(allCurrent); got != StateCurrent {
		t.Fatalf("fully current dimensions = %s", got)
	}
	allCurrent.Assurance = StateUnknown
	if got := chooseOverallState(allCurrent); got != StateUnknown {
		t.Fatalf("unknown assurance became green: %s", got)
	}
	allCurrent.Assurance = StateCurrent
	allCurrent.SourceQuality = StateUnknown
	if got := chooseOverallState(allCurrent); got != StateUnknown {
		t.Fatalf("unknown source quality became green: %s", got)
	}
}

func TestAuthoritativeSourceStateControlsSourceQuality(t *testing.T) {
	now := time.Date(2026, 8, 7, 14, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	aggregate := ProgramAggregate{
		Program: Program{
			ID:            "program-1",
			TenantID:      "bank",
			Status:        ProgramActive,
			EffectiveFrom: past,
			Version:       2,
		},
		Requirements: []Requirement{
			{
				ID:            "req",
				TenantID:      "bank",
				ProgramID:     "program-1",
				SourceID:      "source-a",
				Title:         "Requirement",
				Status:        RequirementApproved,
				EffectiveFrom: past,
			},
		},
	}

	degraded := deriveProgramStateWithSourceState(aggregate, 0, now, ProgramSourceState{Required: 2, Current: false, Known: true})
	if degraded.Dimensions.SourceQuality != StateAtRisk {
		t.Fatalf("unhealthy required source did not make source quality at risk: %#v", degraded.Dimensions)
	}
	current := deriveProgramStateWithSourceState(aggregate, 0, now, ProgramSourceState{Required: 2, Current: true, Known: true})
	if current.Dimensions.SourceQuality != StateCurrent {
		t.Fatalf("all-current sources did not make source quality current: %#v", current.Dimensions)
	}
	none := deriveProgramStateWithSourceState(
		ProgramAggregate{Program: aggregate.Program},
		0,
		now,
		ProgramSourceState{Required: 0, Known: true},
	)
	if none.Dimensions.SourceQuality != StateNotApplicable {
		t.Fatalf("no source dependency should be explicit N/A: %#v", none.Dimensions)
	}
}

func TestReplayPreservesProgramPeriodOnResume(t *testing.T) {
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(90 * 24 * time.Hour)
	created := Program{
		ID:             "program-1",
		TenantID:       "bank",
		Status:         ProgramActive,
		EffectiveFrom:  start,
		EffectiveUntil: &end,
		Version:        1,
	}
	paused := created
	paused.Status = ProgramPaused
	resumed := paused
	resumed.Status = ProgramActive
	resumed.EffectiveUntil = nil // historical buggy payload

	marshal := func(value any) json.RawMessage {
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	aggregate, err := reconstructProgram([]Event{
		{AggregateVersion: 1, Type: EventProgramCreated, Payload: marshal(created)},
		{AggregateVersion: 2, Type: EventProgramStatusChanged, Payload: marshal(paused)},
		{AggregateVersion: 3, Type: EventProgramStatusChanged, Payload: marshal(resumed)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if aggregate.Program.EffectiveUntil == nil || !aggregate.Program.EffectiveUntil.Equal(end) {
		t.Fatalf("resume erased configured period: %#v", aggregate.Program.EffectiveUntil)
	}
}

func TestProgramSummaryReportsStalenessAndReasonOmissions(t *testing.T) {
	now := time.Now().UTC()
	reasons := make([]StateReason, 8)
	for index := range reasons {
		reasons[index] = StateReason{Code: "R", Summary: "reason"}
	}
	aggregate := ProgramAggregate{
		Program: Program{
			ID:       "program-1",
			TenantID: "bank",
			Status:   ProgramActive,
			Version:  9,
		},
		CurrentState: &ProgramStateSnapshot{
			Overall:        StateAtRisk,
			ProgramVersion: 7,
			GeneratedAt:    now,
			Reasons:        reasons,
		},
	}
	summary := summarizeProgram(aggregate)
	if summary.ProgramVersion != 9 || summary.AssessedProgramVersion != 7 || !summary.ProjectionStale {
		t.Fatalf("unexpected projection freshness: %#v", summary)
	}
	if summary.ReasonsTotal != 8 || summary.ReasonsOmitted != 2 || len(summary.Reasons) != 6 {
		t.Fatalf("silent reason truncation metadata is wrong: %#v", summary)
	}
}
