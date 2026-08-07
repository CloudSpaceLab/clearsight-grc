package continuity

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestDecisionLifecyclePreservesDistinctActors(t *testing.T) {
	ctx := context.Background()
	service := NewService(NewMemoryRepository())
	now := time.Date(2026, 8, 7, 16, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	matter, err := service.CreateMatter(ctx, CreateMatterInput{TenantID: "bank", Type: MatterRegulatoryChange, Priority: 3, Title: "Review a regulatory change", Summary: "A current regulatory position is required.", Scope: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}

	steps := []struct {
		status DecisionStatus
		actor  string
		want   func(Decision) bool
	}{
		{DecisionProposed, "proposer", func(v Decision) bool { return v.ProposedBy == "proposer" && v.AuthorityPrincipalID == "" }},
		{DecisionInReview, "reviewer", func(v Decision) bool { return v.ReviewedBy == "reviewer" && v.AuthorityPrincipalID == "" }},
		{DecisionChallenged, "challenger", func(v Decision) bool { return v.ChallengedBy == "challenger" && v.AuthorityPrincipalID == "" }},
		{DecisionApproved, "authorizer", func(v Decision) bool { return v.AuthorityPrincipalID == "authorizer" }},
	}
	for _, step := range steps {
		input := AddDecisionInput{TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, Type: "REGULATORY_POSITION", Status: step.status, Rationale: "Lifecycle progression.", AuthorityPrincipalID: step.actor}
		if step.status == DecisionApproved {
			input.SelectedOption = "IMPLEMENT"
		}
		matter, err = service.RecordDecisionLifecycle(ctx, input)
		if err != nil {
			t.Fatalf("%s: %v", step.status, err)
		}
		latest := CurrentDecisionForType(matter.Decisions, "REGULATORY_POSITION")
		if latest == nil || !step.want(*latest) {
			t.Fatalf("%s actor was not preserved: %#v", step.status, latest)
		}
	}
	if len(matter.Decisions) != len(steps) {
		t.Fatalf("expected append-only lifecycle history, got %d records", len(matter.Decisions))
	}
}

func TestResponseLifecyclePreservesActorsFromEventEnvelopeInMemory(t *testing.T) {
	ctx := context.Background()
	service := NewService(NewMemoryRepository())
	now := time.Date(2026, 8, 7, 16, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	matter, err := service.CreateMatter(ctx, CreateMatterInput{TenantID: "bank", Type: MatterAuthorityRequest, Priority: 4, Title: "Regulator response", Summary: "Prepare and send the response.", Scope: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.AddResponsePackage(ctx, AddResponsePackageInput{TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, Purpose: "Provide records", Audience: "Regulator", Manifest: json.RawMessage(`[]`), ActorID: "preparer"})
	if err != nil {
		t.Fatal(err)
	}
	responseID := matter.ResponsePackages[0].ID
	if matter.ResponsePackages[0].PreparedBy != "preparer" {
		t.Fatalf("preparer was not reconstructed from event actor: %#v", matter.ResponsePackages[0])
	}

	steps := []struct {
		status ResponseStatus
		actor  string
		want   func(ResponsePackage) bool
	}{
		{ResponseInReview, "reviewer", func(v ResponsePackage) bool { return v.ReviewedBy == "reviewer" }},
		{ResponseApproved, "signatory", func(v ResponsePackage) bool { return v.ApprovedBy == "signatory" }},
		{ResponseTransmitted, "transmitter", func(v ResponsePackage) bool { return v.TransmittedBy == "transmitter" }},
		{ResponseAcknowledged, "ack-recorder", func(v ResponsePackage) bool { return v.AcknowledgedBy == "ack-recorder" }},
	}
	for _, step := range steps {
		matter, err = service.TransitionResponsePackage(ctx, TransitionResponseInput{TenantID: "bank", MatterID: matter.Matter.ID, ResponseID: responseID, ExpectedVersion: matter.Matter.Version, To: step.status, ActorID: step.actor, Rationale: "Lifecycle progression."})
		if err != nil {
			t.Fatalf("%s: %v", step.status, err)
		}
		if !step.want(matter.ResponsePackages[0]) {
			t.Fatalf("%s actor was not preserved: %#v", step.status, matter.ResponsePackages[0])
		}
	}
}

func TestDecisionLifecycleRejectsInvalidTransition(t *testing.T) {
	values := []Decision{{ID: "d1", Type: "POSITION", Status: DecisionApproved, CreatedAt: time.Now().UTC()}}
	if err := ValidateDecisionLifecycle(values, "POSITION", DecisionInReview); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expected invalid transition, got %v", err)
	}
}

func TestHistoricalReplayNormalizesLifecycleActorsFromEventEnvelope(t *testing.T) {
	now := time.Date(2026, 8, 7, 16, 0, 0, 0, time.UTC)
	matter := Matter{ID: "m1", TenantID: "bank", Type: MatterRegulatoryChange, Status: MatterAssessment, Priority: 3, Title: "Matter", Summary: "Summary", Scope: json.RawMessage(`{}`), KnownFacts: json.RawMessage(`{}`), MissingFacts: json.RawMessage(`[]`), Contradictions: json.RawMessage(`[]`), CreatedAt: now, UpdatedAt: now, Version: 1}
	created, err := newEvent("bank", "MATTER", "m1", 1, EventMatterCreated, matter, ActorSystem, "", now)
	if err != nil {
		t.Fatal(err)
	}
	legacy := Decision{ID: "d1", TenantID: "bank", MatterID: "m1", Type: "POSITION", Status: DecisionProposed, Rationale: "Proposal", AuthorityPrincipalID: "legacy-wrong-field", CreatedAt: now.Add(time.Minute), UpdatedAt: now.Add(time.Minute), Version: 1}
	decisionEvent, err := newEvent("bank", "MATTER", "m1", 2, EventDecisionAdded, legacy, ActorPerson, "actual-proposer", now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := reconstructMatter([]Event{created, decisionEvent})
	if err != nil {
		t.Fatal(err)
	}
	if len(replayed.Decisions) != 1 || replayed.Decisions[0].ProposedBy != "actual-proposer" || replayed.Decisions[0].AuthorityPrincipalID != "" {
		t.Fatalf("event actor did not override legacy payload actor: %#v", replayed.Decisions)
	}
}
