package continuity

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCompileMatterWorkProjectsOnlyUnambiguousResponseTransitions(t *testing.T) {
	now := time.Date(2026, 8, 7, 22, 30, 0, 0, time.UTC)
	due := now.Add(2 * time.Hour)
	aggregate := MatterAggregate{
		Matter: Matter{ID: "matter-1", TenantID: "bank", Status: MatterResponsePreparation, Priority: 4, DueAt: &due},
		ResponsePackages: []ResponsePackage{
			{ID: "response-transmitted", Purpose: "NDPC response", Audience: "NDPC", Status: ResponseTransmitted, UpdatedAt: now.Add(-time.Minute)},
			{ID: "response-review", Purpose: "CBN response", Audience: "CBN", Status: ResponseInReview, UpdatedAt: now.Add(-2 * time.Minute)},
		},
	}

	requirements, ambiguities := CompileMatterWork(aggregate, now)
	if len(requirements) != 1 {
		t.Fatalf("expected one executable requirement, got %#v", requirements)
	}
	got := requirements[0]
	if got.CommandName != "matter.response.transition" || got.TargetStatus != string(ResponseAcknowledged) || got.Responsibility != "ACKNOWLEDGEMENT_RECORDER" || got.Materiality != 4 {
		t.Fatalf("unexpected transmitted response requirement: %#v", got)
	}
	if len(ambiguities) != 1 || ambiguities[0].SubresourceID != "response-review" || len(ambiguities[0].AllowedTargets) < 2 {
		t.Fatalf("expected the in-review response to remain explicitly ambiguous, got %#v", ambiguities)
	}
}

func TestCompileMatterWorkDoesNotInventDecisionAssignee(t *testing.T) {
	now := time.Date(2026, 8, 7, 22, 30, 0, 0, time.UTC)
	aggregate := MatterAggregate{
		Matter:    Matter{ID: "matter-1", TenantID: "bank", Status: MatterDecisionRequired, Priority: 3},
		Decisions: []Decision{{ID: "decision-1", Type: "RISK_ACCEPTANCE", Status: DecisionInReview, UpdatedAt: now}},
	}

	requirements, ambiguities := CompileMatterWork(aggregate, now)
	if len(requirements) != 0 {
		t.Fatalf("decision state alone must not invent executable work: %#v", requirements)
	}
	if len(ambiguities) != 1 || ambiguities[0].SubresourceID != "decision-1" {
		t.Fatalf("expected explicit decision ambiguity, got %#v", ambiguities)
	}
}

func TestCompileMatterWorkWaitsForVerificationObservationPeriod(t *testing.T) {
	now := time.Date(2026, 8, 7, 22, 30, 0, 0, time.UTC)
	implemented := now.Add(-40 * time.Minute)
	contractCreated := now.Add(-2 * time.Hour)
	aggregate := MatterAggregate{
		Matter:                Matter{ID: "matter-1", TenantID: "bank", Status: MatterVerification, Priority: 5},
		Actions:               []Action{{ID: "action-1", Status: ActionImplemented, OwnerPrincipalID: "owner-1", ImplementedAt: &implemented}},
		VerificationContracts: []VerificationContract{{ID: "verify-1", ActionID: "action-1", ExpectedOutcome: "ATM is operating", ObservationPeriodMinutes: 60, AuthorityPrincipalID: "reviewer-1", Status: VerificationActive, CreatedAt: contractCreated}},
	}

	requirements, _ := CompileMatterWork(aggregate, now)
	if len(requirements) != 0 {
		t.Fatalf("verification must not appear before the observation period completes: %#v", requirements)
	}

	now = now.Add(25 * time.Minute)
	requirements, _ = CompileMatterWork(aggregate, now)
	if len(requirements) != 1 {
		t.Fatalf("expected one ready outcome check, got %#v", requirements)
	}
	got := requirements[0]
	if got.CommandName != "matter.outcome.record" || got.Responsibility != "REVIEWER" || got.RequiredPrincipalID != "reviewer-1" || got.Verification == nil || got.Verification.ContractID != "verify-1" {
		t.Fatalf("unexpected outcome-check requirement: %#v", got)
	}
}

func TestCompileMatterWorkDoesNotRepeatRecordedVerification(t *testing.T) {
	now := time.Date(2026, 8, 7, 22, 30, 0, 0, time.UTC)
	created := now.Add(-2 * time.Hour)
	aggregate := MatterAggregate{
		Matter:                Matter{ID: "matter-1", TenantID: "bank", Status: MatterVerification, Priority: 3},
		VerificationContracts: []VerificationContract{{ID: "verify-1", ExpectedOutcome: "Expected state", ObservationPeriodMinutes: 30, Status: VerificationActive, CreatedAt: created}},
		VerificationResults:   []VerificationResult{{ID: "result-1", ContractID: "verify-1", Result: VerificationInconclusive, Observations: json.RawMessage(`{}`), ObservedAt: now.Add(-time.Minute), CreatedAt: now.Add(-time.Minute)}},
	}

	requirements, _ := CompileMatterWork(aggregate, now)
	if len(requirements) != 0 {
		t.Fatalf("a contract with a current result must not create duplicate review work: %#v", requirements)
	}
}
