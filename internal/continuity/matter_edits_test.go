package continuity

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestAddActionRequiresAssignedOwner(t *testing.T) {
	service := NewService(NewMemoryRepository())
	matter, err := service.CreateMatter(WithTrustedSystemScope(t.Context()), CreateMatterInput{
		TenantID:      "bank",
		LegalEntityID: "entity-a",
		Type:          MatterControlGap,
		Priority:      3,
		Title:         "Account review gap",
		Summary:       "Resolve accounts without current ownership evidence.",
		Scope:         json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.AddAction(WithTrustedSystemScope(t.Context()), AddActionInput{
		TenantID:         "bank",
		MatterID:         matter.Matter.ID,
		ExpectedVersion:  matter.Matter.Version,
		Title:            "Confirm account owners",
		Description:      "Obtain current ownership confirmation for every unresolved account.",
		OwnerPrincipalID: "   ",
	})
	if err == nil || !strings.Contains(err.Error(), "owner_principal_id") {
		t.Fatalf("expected assigned-owner validation error, got %v", err)
	}

	current, err := service.GetMatter(WithTrustedSystemScope(t.Context()), "bank", matter.Matter.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(current.Actions) != 0 {
		t.Fatalf("invalid action was persisted: %#v", current.Actions)
	}
}

func TestUpdateMatterDetailsAndResolveMissingFact(t *testing.T) {
	service := NewService(NewMemoryRepository())
	matter, err := service.CreateMatter(WithTrustedSystemScope(t.Context()), CreateMatterInput{
		TenantID:       "bank",
		LegalEntityID:  "entity-a",
		Type:           MatterRegulatoryChange,
		Priority:       4,
		Title:          "Annual return",
		Summary:        "Update the filing process.",
		KnownFacts:     json.RawMessage(`{"filing_channel":"licensed DPCO"}`),
		MissingFacts:   json.RawMessage(`["final DPCO checklist"]`),
		Contradictions: json.RawMessage(`[]`),
	})
	if err != nil {
		t.Fatal(err)
	}

	updated, err := service.UpdateMatterDetails(WithTrustedSystemScope(t.Context()), UpdateMatterDetailsInput{
		TenantID:        "bank",
		MatterID:        matter.Matter.ID,
		ExpectedVersion: matter.Matter.Version,
		Title:           "Annual return filing",
		Summary:         "Update the filing process and evidence ownership.",
		Priority:        4,
		DueAt:           matter.Matter.DueAt,
		Scope:           matter.Matter.Scope,
		ActorID:         "owner",
		Rationale:       "Clarify the approved scope.",
	})
	if err != nil {
		t.Fatal(err)
	}

	resolved, err := service.ChangeMatterContext(WithTrustedSystemScope(t.Context()), ChangeMatterContextInput{
		TenantID:           "bank",
		MatterID:           matter.Matter.ID,
		ExpectedVersion:    updated.Matter.Version,
		Kind:               MatterContextResolveMissing,
		Key:                "final_dpco_checklist",
		Label:              "final DPCO checklist",
		Value:              json.RawMessage(`"Checklist v3"`),
		Rationale:          "DPCO approved version 3.",
		EvidenceReferences: json.RawMessage(`["artifact-checklist-v3"]`),
		ActorID:            "owner",
	})
	if err != nil {
		t.Fatal(err)
	}

	var facts map[string]any
	if err := json.Unmarshal(resolved.Matter.KnownFacts, &facts); err != nil {
		t.Fatal(err)
	}
	if facts["final_dpco_checklist"] != "Checklist v3" {
		t.Fatalf("fact missing: %#v", facts)
	}
	if string(resolved.Matter.MissingFacts) != "[]" {
		t.Fatalf("missing fact not resolved: %s", resolved.Matter.MissingFacts)
	}
	if resolved.Matter.Title != "Annual return filing" || resolved.Matter.Version != matter.Matter.Version+2 {
		t.Fatalf("details or aggregate version not updated: %#v", resolved.Matter)
	}
}

func TestUpdateMatterDetailsPreservesAuditMetadataAndRejectsNoOp(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewService(repo)
	matter, err := service.CreateMatter(WithTrustedSystemScope(t.Context()), CreateMatterInput{
		TenantID:       "bank",
		LegalEntityID:  "entity-a",
		Type:           MatterRegulatoryChange,
		Priority:       4,
		Title:          "Annual return",
		Summary:        "Update the filing process.",
		Scope:          json.RawMessage(`{"area":"Privacy"}`),
		KnownFacts:     json.RawMessage(`{}`),
		MissingFacts:   json.RawMessage(`[]`),
		Contradictions: json.RawMessage(`[]`),
	})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := service.UpdateMatterDetails(WithTrustedSystemScope(t.Context()), UpdateMatterDetailsInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
		Title: "Annual return filing", Summary: matter.Matter.Summary, Priority: matter.Matter.Priority,
		Scope: matter.Matter.Scope, DueAt: matter.Matter.DueAt, ActorID: "owner", Rationale: "Use the approved working title.",
	})
	if err != nil {
		t.Fatal(err)
	}
	events, err := repo.MatterEvents(WithTrustedSystemScope(t.Context()), "bank", matter.Matter.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	var changed matterDetailsUpdatedEvent
	if err := json.Unmarshal(events[1].Payload, &changed); err != nil {
		t.Fatal(err)
	}
	if changed.Previous.Title != "Annual return" || changed.Matter.Title != "Annual return filing" || changed.Rationale == "" {
		t.Fatalf("detail audit metadata missing: %#v", changed)
	}

	_, err = service.UpdateMatterDetails(WithTrustedSystemScope(t.Context()), UpdateMatterDetailsInput{
		TenantID: "bank", MatterID: updated.Matter.ID, ExpectedVersion: updated.Matter.Version,
		Title: updated.Matter.Title, Summary: updated.Matter.Summary, Priority: updated.Matter.Priority,
		Scope: updated.Matter.Scope, DueAt: updated.Matter.DueAt, ActorID: "owner", Rationale: "No material change.",
	})
	if err == nil {
		t.Fatal("expected an unchanged detail command to be rejected")
	}
	eventsAfterNoOp, err := repo.MatterEvents(WithTrustedSystemScope(t.Context()), "bank", matter.Matter.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(eventsAfterNoOp) != len(events) {
		t.Fatalf("no-op appended an event: %d before, %d after", len(events), len(eventsAfterNoOp))
	}
}

func TestChangeMatterContextSupportsExplicitFactAndIssueChanges(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewService(repo)
	matter, err := service.CreateMatter(WithTrustedSystemScope(t.Context()), CreateMatterInput{
		TenantID:       "bank",
		LegalEntityID:  "entity-a",
		Type:           MatterRegulatoryChange,
		Priority:       4,
		Title:          "Annual return",
		Summary:        "Update the filing process.",
		KnownFacts:     json.RawMessage(`{"filing_channel":"email"}`),
		MissingFacts:   json.RawMessage(`[]`),
		Contradictions: json.RawMessage(`[]`),
	})
	if err != nil {
		t.Fatal(err)
	}

	changes := []ChangeMatterContextInput{
		{Kind: MatterContextAddFact, Key: "filing_deadline", Label: "Filing deadline", Value: json.RawMessage(`"31 March"`), Rationale: "Record the published timetable.", EvidenceReferences: json.RawMessage(`["schedule-2026"]`)},
		{Kind: MatterContextCorrectFact, Key: "filing_channel", Label: "Filing channel", Value: json.RawMessage(`"licensed DPCO portal"`), Rationale: "Replace the superseded submission route.", EvidenceReferences: json.RawMessage(`["dpco-guidance-2026"]`)},
		{Kind: MatterContextRetireFact, Key: "filing_deadline", Label: "Filing deadline", Rationale: "The timetable is being reissued."},
		{Kind: MatterContextAddMissing, Label: "approved section owner", Rationale: "Ownership is not recorded."},
		{Kind: MatterContextAddContradiction, Label: "The filing notice and DPCO portal show different dates", Rationale: "Both current sources are authoritative."},
		{Kind: MatterContextResolveContradiction, Label: "The filing notice and DPCO portal show different dates", Rationale: "The DPCO confirmed the portal date."},
	}
	for index := range changes {
		changes[index].TenantID = "bank"
		changes[index].MatterID = matter.Matter.ID
		changes[index].ExpectedVersion = matter.Matter.Version
		changes[index].ActorID = "owner"
		matter, err = service.ChangeMatterContext(WithTrustedSystemScope(t.Context()), changes[index])
		if err != nil {
			t.Fatalf("change %s failed: %v", changes[index].Kind, err)
		}
	}

	var facts map[string]any
	if err := json.Unmarshal(matter.Matter.KnownFacts, &facts); err != nil {
		t.Fatal(err)
	}
	if len(facts) != 1 || facts["filing_channel"] != "licensed DPCO portal" {
		t.Fatalf("unexpected facts: %#v", facts)
	}
	if string(matter.Matter.MissingFacts) != `["approved section owner"]` {
		t.Fatalf("unexpected missing information: %s", matter.Matter.MissingFacts)
	}
	if string(matter.Matter.Contradictions) != "[]" {
		t.Fatalf("contradiction was not resolved: %s", matter.Matter.Contradictions)
	}

	events, err := repo.MatterEvents(WithTrustedSystemScope(context.Background()), "bank", matter.Matter.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	var corrected matterContextChangedEvent
	if err := json.Unmarshal(events[2].Payload, &corrected); err != nil {
		t.Fatal(err)
	}
	if string(corrected.PreviousValue) != `"email"` || corrected.Rationale == "" || len(corrected.EvidenceReferences) != 1 {
		t.Fatalf("correction audit metadata missing: %#v", corrected)
	}

	replayed, err := service.MatterAt(WithTrustedSystemScope(t.Context()), "bank", matter.Matter.ID, events[len(events)-1].OccurredAt)
	if err != nil {
		t.Fatal(err)
	}
	if string(replayed.Matter.KnownFacts) != string(matter.Matter.KnownFacts) || string(replayed.Matter.MissingFacts) != string(matter.Matter.MissingFacts) {
		t.Fatalf("replayed context differs: %#v", replayed.Matter)
	}
}

func TestChangeMatterContextRejectsAmbiguousOrUngovernedChanges(t *testing.T) {
	service := NewService(NewMemoryRepository())
	matter, err := service.CreateMatter(WithTrustedSystemScope(t.Context()), CreateMatterInput{
		TenantID:       "bank",
		LegalEntityID:  "entity-a",
		Type:           MatterRegulatoryChange,
		Priority:       4,
		Title:          "Annual return",
		Summary:        "Update the filing process.",
		KnownFacts:     json.RawMessage(`{"filing_channel":"email"}`),
		MissingFacts:   json.RawMessage(`["approved owner"]`),
		Contradictions: json.RawMessage(`[]`),
	})
	if err != nil {
		t.Fatal(err)
	}

	base := ChangeMatterContextInput{TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, ActorID: "owner", Rationale: "Correct the record."}
	tests := []struct {
		name  string
		input ChangeMatterContextInput
	}{
		{name: "structured fact value", input: withMatterContext(base, MatterContextAddFact, "sections", "Sections", json.RawMessage(`[1,2]`))},
		{name: "duplicate missing item", input: withMatterContext(base, MatterContextAddMissing, "", "approved owner", nil)},
		{name: "absent fact correction", input: withMatterContext(base, MatterContextCorrectFact, "deadline", "Deadline", json.RawMessage(`"31 March"`))},
		{name: "missing actor", input: withMatterContext(base, MatterContextAddMissing, "", "signatory", nil)},
	}
	tests[3].input.ActorID = ""
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := service.ChangeMatterContext(WithTrustedSystemScope(t.Context()), test.input); err == nil || errors.Is(err, ErrVersionConflict) {
				t.Fatalf("expected a validation error, got %v", err)
			}
		})
	}
}

func withMatterContext(base ChangeMatterContextInput, kind MatterContextChangeKind, key, label string, value json.RawMessage) ChangeMatterContextInput {
	base.Kind = kind
	base.Key = key
	base.Label = label
	base.Value = value
	return base
}

func TestAssignMatterAndUpdateAction(t *testing.T) {
	service, matter := editableMatterFixture(t)
	action := matter.Actions[0]

	assigned, err := service.AssignMatter(WithTrustedSystemScope(t.Context()), AssignMatterInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
		OwnerPrincipalID: "owner-2", ActorID: "owner-1", Rationale: "Move accountability to privacy operations.",
	})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := service.UpdateAction(WithTrustedSystemScope(t.Context()), UpdateActionInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ActionID: action.ID,
		ExpectedVersion: assigned.Matter.Version, Title: action.Title,
		Description: "Assign every annual-return section and attach its source.",
		DueAt:       action.DueAt, ActorID: "owner-2", Rationale: "Clarify the required evidence.",
	})
	if err != nil {
		t.Fatal(err)
	}
	assignedAction, err := service.AssignAction(WithTrustedSystemScope(t.Context()), AssignActionInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ActionID: action.ID,
		ExpectedVersion: updated.Matter.Version, OwnerPrincipalID: "performer-2",
		ActorID: "owner-2", Rationale: "Assign the active privacy operations owner.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if assignedAction.Matter.OwnerPrincipalID != "owner-2" || assignedAction.Actions[0].OwnerPrincipalID != "performer-2" {
		t.Fatalf("assignments not updated: %#v", assignedAction)
	}
	if assignedAction.Actions[0].Description != "Assign every annual-return section and attach its source." {
		t.Fatalf("action description not updated: %#v", assignedAction.Actions[0])
	}
	if assignedAction.Actions[0].Version != action.Version+2 {
		t.Fatalf("action version should increment once per action command: %#v", assignedAction.Actions[0])
	}
}

func editableMatterFixture(t *testing.T) (*Service, MatterAggregate) {
	t.Helper()
	service := NewService(NewMemoryRepository())
	matter, err := service.CreateMatter(WithTrustedSystemScope(t.Context()), CreateMatterInput{
		TenantID: "bank", LegalEntityID: "entity-a", Type: MatterRegulatoryChange, Priority: 4,
		Title: "Annual return", Summary: "Update the filing process.",
		Scope: json.RawMessage(`{}`), KnownFacts: json.RawMessage(`{}`),
		MissingFacts: json.RawMessage(`[]`), Contradictions: json.RawMessage(`[]`),
		OwnerPrincipalID: "owner-1", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.AddAction(WithTrustedSystemScope(t.Context()), AddActionInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
		Title: "Update annual-return evidence checklist", Description: "Map each return section to current evidence.",
		OwnerPrincipalID: "performer-1", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	return service, matter
}

func TestActionEditsPreserveHistoryAndRejectTerminalActions(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewService(repo)
	matter, err := service.CreateMatter(WithTrustedSystemScope(t.Context()), CreateMatterInput{
		TenantID: "bank", LegalEntityID: "entity-a", Type: MatterRegulatoryChange, Priority: 4,
		Title: "Annual return", Summary: "Update the filing process.",
		Scope: json.RawMessage(`{}`), KnownFacts: json.RawMessage(`{}`),
		MissingFacts: json.RawMessage(`[]`), Contradictions: json.RawMessage(`[]`),
		OwnerPrincipalID: "owner-1", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.AddAction(WithTrustedSystemScope(t.Context()), AddActionInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
		Title: "Update checklist", Description: "Map every section.", OwnerPrincipalID: "performer-1", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	action := matter.Actions[0]
	matter, err = service.UpdateAction(WithTrustedSystemScope(t.Context()), UpdateActionInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ActionID: action.ID, ExpectedVersion: matter.Matter.Version,
		Title: action.Title, Description: "Map every section to its current source.", DueAt: action.DueAt,
		ActorID: "owner-1", Rationale: "Make the evidence requirement explicit.",
	})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.AssignAction(WithTrustedSystemScope(t.Context()), AssignActionInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ActionID: action.ID, ExpectedVersion: matter.Matter.Version,
		OwnerPrincipalID: "performer-2", ActorID: "owner-1", Rationale: "Assign the current process owner.",
	})
	if err != nil {
		t.Fatal(err)
	}
	events, err := repo.MatterEvents(WithTrustedSystemScope(t.Context()), "bank", matter.Matter.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	var update actionUpdatedEvent
	if err := json.Unmarshal(events[2].Payload, &update); err != nil {
		t.Fatal(err)
	}
	var assignment actionAssignedEvent
	if err := json.Unmarshal(events[3].Payload, &assignment); err != nil {
		t.Fatal(err)
	}
	if update.Previous.Description != "Map every section." || update.Rationale == "" || assignment.PreviousOwnerID != "performer-1" || assignment.OwnerPrincipalID != "performer-2" {
		t.Fatalf("action audit metadata missing: update=%#v assignment=%#v", update, assignment)
	}

	for _, state := range []ActionStatus{ActionInProgress, ActionImplemented} {
		matter, err = service.TransitionAction(WithTrustedSystemScope(t.Context()), TransitionActionInput{
			TenantID: "bank", MatterID: matter.Matter.ID, ActionID: action.ID,
			ExpectedVersion: matter.Matter.Version, To: state, ActorID: "performer-2", Rationale: "Complete assigned work.",
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	terminal := matter.Actions[0]
	if _, err := service.UpdateAction(WithTrustedSystemScope(t.Context()), UpdateActionInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ActionID: action.ID, ExpectedVersion: matter.Matter.Version,
		Title: terminal.Title, Description: "Rewrite completed work.", DueAt: terminal.DueAt,
		ActorID: "owner-1", Rationale: "Attempt to rewrite completion.",
	}); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expected terminal action update to fail closed, got %v", err)
	}
	if _, err := service.AssignAction(WithTrustedSystemScope(t.Context()), AssignActionInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ActionID: action.ID, ExpectedVersion: matter.Matter.Version,
		OwnerPrincipalID: "performer-3", ActorID: "owner-1", Rationale: "Attempt to reassign completed work.",
	}); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expected terminal action assignment to fail closed, got %v", err)
	}
}
