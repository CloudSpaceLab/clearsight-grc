package continuity

import (
	"errors"
	"testing"
)

func TestDecisionAuthorityRequirementUsesExactLifecycleTarget(t *testing.T) {
	values := MatterAggregate{Matter: Matter{ID: "matter", Priority: 5}, Decisions: []Decision{
		{ID: "old", Type: "K", Status: DecisionProposed},
		{ID: "other", Type: "OTHER", Status: DecisionProposed},
		{ID: "latest", Type: "K", Status: DecisionInReview},
	}}
	got, err := DecisionAuthorityRequirement(values, " K ", DecisionApproved)
	if err != nil {
		t.Fatal(err)
	}
	if got.ObjectType != "DECISION" || got.ObjectID != "latest" || got.Responsibility != "AUTHORIZER" || got.Materiality != 5 {
		t.Fatalf("unexpected decision authority requirement: %+v", got)
	}
}

func TestDecisionAuthorityRequirementUsesMatterForInitialProposal(t *testing.T) {
	got, err := DecisionAuthorityRequirement(MatterAggregate{Matter: Matter{ID: "matter", Priority: 1}}, "POSITION", DecisionProposed)
	if err != nil {
		t.Fatal(err)
	}
	if got.ObjectType != "MATTER" || got.ObjectID != "matter" || got.Responsibility != "PROPOSER" || got.Materiality != 2 {
		t.Fatalf("unexpected initial decision authority requirement: %+v", got)
	}
}

func TestDecisionAuthorityRequirementRejectsInvalidTransition(t *testing.T) {
	values := MatterAggregate{Matter: Matter{ID: "matter", Priority: 3}, Decisions: []Decision{{ID: "prior", Type: "POSITION", Status: DecisionApproved}}}
	if _, err := DecisionAuthorityRequirement(values, "POSITION", DecisionInReview); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expected lifecycle rejection, got %v", err)
	}
}
