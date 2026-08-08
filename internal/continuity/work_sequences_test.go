package continuity

import (
	"slices"
	"testing"
	"time"
)

func TestApplyWorkSequenceChoicePreservesAuthorizerOutcomes(t *testing.T) {
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	aggregate := MatterAggregate{
		Matter:    Matter{ID: "matter-1", TenantID: "bank", Priority: 4, Status: MatterDecisionRequired},
		Decisions: []Decision{{ID: "decision-1", MatterID: "matter-1", Type: "EXCEPTION", Status: DecisionInReview}},
	}
	requirements, ambiguities := CompileMatterWork(aggregate, now)
	if len(ambiguities) != 1 {
		t.Fatalf("expected one ambiguity, got %#v", ambiguities)
	}

	requirements, unresolved := ApplyWorkSequenceChoices(aggregate, requirements, ambiguities, []WorkSequenceChoice{{
		AmbiguityKey:   ambiguities[0].Key,
		Responsibility: "AUTHORIZER",
		RuleID:         "RISK-SEQ/authorize",
		PolicyVersion:  "RISK-SEQ:v4",
	}})
	if len(unresolved) != 0 || len(requirements) != 1 {
		t.Fatalf("sequence choice did not resolve packet: requirements=%#v unresolved=%#v", requirements, unresolved)
	}
	packet := requirements[0]
	want := []string{string(DecisionApproved), string(DecisionConditionallyApproved), string(DecisionRejected), string(DecisionSuperseded)}
	if packet.Responsibility != "AUTHORIZER" || packet.TargetStatus != "" || !slices.Equal(packet.AllowedTargets, want) {
		t.Fatalf("authorizer packet pre-decided or lost legal outcomes: %#v", packet)
	}
	if packet.PrimaryAction != "Decide" || packet.InterventionClass != "AUTHORIZATION" || packet.SequenceRuleID != "RISK-SEQ/authorize" {
		t.Fatalf("unexpected packet context: %#v", packet)
	}
}

func TestApplyWorkSequenceChoiceRejectsResponsibilityOutsideCurrentLifecycle(t *testing.T) {
	aggregate := MatterAggregate{
		Matter:    Matter{ID: "matter-1", Priority: 3, Status: MatterDecisionRequired},
		Decisions: []Decision{{ID: "decision-1", Type: "EXCEPTION", Status: DecisionInReview}},
	}
	requirements, ambiguities := CompileMatterWork(aggregate, time.Now())
	requirements, unresolved := ApplyWorkSequenceChoices(aggregate, requirements, ambiguities, []WorkSequenceChoice{{
		AmbiguityKey:   ambiguities[0].Key,
		Responsibility: "TRANSMITTER",
		RuleID:         "bad",
		PolicyVersion:  "bad:v1",
	}})
	if len(requirements) != 0 || len(unresolved) != 1 {
		t.Fatalf("invalid responsibility became work: requirements=%#v unresolved=%#v", requirements, unresolved)
	}
}

func TestApplyWorkSequenceChoiceCanSelectReviewGateWithoutChoosingReviewOutcome(t *testing.T) {
	aggregate := MatterAggregate{
		Matter: Matter{ID: "matter-1", Priority: 2, Status: MatterResponsePreparation},
		ResponsePackages: []ResponsePackage{{
			ID: "response-1", MatterID: "matter-1", Purpose: "Regulator response", Audience: "NDPC", Status: ResponseDraft,
		}},
	}
	requirements, ambiguities := CompileMatterWork(aggregate, time.Now())
	if len(ambiguities) != 1 {
		t.Fatalf("expected response ambiguity, got %#v", ambiguities)
	}
	requirements, unresolved := ApplyWorkSequenceChoices(aggregate, requirements, ambiguities, []WorkSequenceChoice{{
		AmbiguityKey:   ambiguities[0].Key,
		Responsibility: "REVIEWER",
		RuleID:         "RESPONSE-SEQ/review",
		PolicyVersion:  "RESPONSE-SEQ:v2",
	}})
	if len(unresolved) != 0 || len(requirements) != 1 {
		t.Fatal("review gate did not compile")
	}
	packet := requirements[0]
	if packet.Responsibility != "REVIEWER" || packet.PrimaryAction != "Review response" || len(packet.AllowedTargets) == 0 {
		t.Fatalf("unexpected response review packet: %#v", packet)
	}
	for _, target := range packet.AllowedTargets {
		policy, err := ResponseLifecyclePolicy(ResponseDraft, ResponseStatus(target))
		if err != nil || policy.Responsibility != "REVIEWER" {
			t.Fatalf("packet leaked non-reviewer target %s", target)
		}
	}
}
