package continuity

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestMonitoringAdverseTriggerCreatesOneGovernedEntityScopedControlGap(t *testing.T) {
	ctx := WithTrustedSystemScope(t.Context())
	repo := NewMemoryRepository()
	service := NewService(repo)
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	program, err := service.CreateProgram(ctx, CreateProgramInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "ACCESS", Name: "Access control monitoring", Type: "COMPLIANCE",
		OwningFunction: "Information Security", OwnerPrincipalID: "program-owner", AuthorityPrincipalID: "program-authorizer",
		Scope: json.RawMessage(`{"systems":["core-banking"]}`), EffectiveFrom: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	trigger := Trigger{
		TenantID: "bank", ProgramID: program.Program.ID, Type: "MONITORING_RESULT_ADVERSE", SubjectType: "MONITORING_CHECK", SubjectID: "check-1",
		DedupeKey: "monitoring-adverse:check-1:period-2026-08", Source: "monitoring", ObservedAt: now,
		Payload: json.RawMessage(`{"monitoring_check_id":"check-1","monitoring_result_id":"result-1","risk_band":"HIGH","score":72}`), ActorID: "reviewer-1",
	}
	updated, matter, inserted, err := service.ApplyTrigger(ctx, trigger)
	if err != nil {
		t.Fatal(err)
	}
	if !inserted || matter == nil {
		t.Fatalf("adverse trigger did not create work: inserted=%v matter=%#v", inserted, matter)
	}
	if matter.Type != MatterControlGap || matter.Priority != 4 || matter.LegalEntityID != "entity-a" || matter.OwnerPrincipalID != "program-owner" || matter.RequiredAuthority != "CONTROL_ASSURANCE" {
		t.Fatalf("adverse monitoring work is not governed: %#v", matter)
	}
	if matter.SourceType != "MONITORING_CHECK" || matter.SourceID != "check-1" {
		t.Fatalf("adverse monitoring source lineage is incomplete: %#v", matter)
	}
	if matter.TriggerType != trigger.Type || matter.TriggerKey != trigger.DedupeKey || matter.TriggerID == "" {
		t.Fatalf("adverse monitoring lineage is incomplete: %#v", matter)
	}
	if updated.Program.Version != program.Program.Version+1 || len(updated.Triggers) != 1 {
		t.Fatalf("Program trigger history was not preserved: %#v", updated)
	}
	stored, err := service.GetMatter(ctx, "bank", matter.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.Links) != 1 || stored.Links[0].ProgramID != program.Program.ID {
		t.Fatalf("adverse monitoring work is not linked to the exact Program: %#v", stored.Links)
	}
	programEvents, err := repo.ProgramEvents(ctx, "bank", program.Program.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	matterEvents, err := repo.MatterEvents(ctx, "bank", matter.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if programEvents[len(programEvents)-1].Type != EventProgramTriggerRecorded || len(matterEvents) != 2 || matterEvents[0].Type != EventMatterCreated || matterEvents[1].Type != EventMatterLinked {
		t.Fatalf("atomic trigger history is incomplete: program=%#v matter=%#v", programEvents, matterEvents)
	}

	replayed, duplicate, inserted, err := service.ApplyTrigger(ctx, trigger)
	if err != nil {
		t.Fatal(err)
	}
	if inserted || duplicate == nil || duplicate.ID != matter.ID || replayed.Program.Version != updated.Program.Version || len(replayed.Triggers) != 1 {
		t.Fatalf("adverse trigger retry was not idempotent: inserted=%v matter=%#v program=%#v", inserted, duplicate, replayed)
	}
	matters, err := service.ListMatters(ctx, "bank", "OPEN", 20)
	if err != nil || len(matters) != 1 {
		t.Fatalf("adverse trigger created duplicate work: matters=%d err=%v", len(matters), err)
	}
}

func TestMemoryTriggerBundleRollsBackEveryRowWhenMatterLinkIsInvalid(t *testing.T) {
	ctx := WithTrustedSystemScope(t.Context())
	repo := NewMemoryRepository()
	service := NewService(repo)
	now := time.Date(2026, 8, 26, 13, 0, 0, 0, time.UTC)
	program, err := service.CreateProgram(ctx, CreateProgramInput{TenantID: "bank", LegalEntityID: "entity-a", Code: "ROLLBACK", Name: "Rollback proof", Type: "COMPLIANCE", OwningFunction: "Risk", OwnerPrincipalID: "owner", Scope: json.RawMessage(`{}`), EffectiveFrom: now})
	if err != nil {
		t.Fatal(err)
	}
	trigger := Trigger{ID: "trigger-rollback", TenantID: "bank", ProgramID: program.Program.ID, Type: "MONITORING_RESULT_ADVERSE", DedupeKey: "monitoring-adverse:rollback", Payload: json.RawMessage(`{}`), ObservedAt: now, Source: "monitoring"}
	programEvent, err := newEvent("bank", "PROGRAM", program.Program.ID, program.Program.Version+1, EventProgramTriggerRecorded, trigger, ActorSystem, "", now)
	if err != nil {
		t.Fatal(err)
	}
	matter := Matter{ID: "matter-rollback", TenantID: "bank", LegalEntityID: "entity-a", Reference: "MAT-ROLLBACK", Type: MatterControlGap, Status: MatterInitialReview, Priority: 4, Title: "Rollback", Summary: "Rollback", Scope: json.RawMessage(`{}`), KnownFacts: json.RawMessage(`{}`), MissingFacts: json.RawMessage(`[]`), Contradictions: json.RawMessage(`[]`), CreatedAt: now, UpdatedAt: now, Version: 1}
	matterEvent, err := newEvent("bank", "MATTER", matter.ID, 1, EventMatterCreated, matter, ActorSystem, "", now)
	if err != nil {
		t.Fatal(err)
	}
	link := MatterLink{ID: "link-rollback", TenantID: "bank", MatterID: matter.ID, ProgramID: program.Program.ID, Relationship: "AFFECTS", CreatedAt: now}
	linkEvent := Event{ID: "event-link-invalid", TenantID: "bank", AggregateType: "MATTER", AggregateID: matter.ID, AggregateVersion: 2, Type: EventMatterLinked, Payload: json.RawMessage(`[]`), ActorType: ActorSystem, OccurredAt: now}
	_, err = repo.ApplyTriggerBundle(ctx, TriggerBundle{Trigger: trigger, ProgramEvent: programEvent, Matter: &matter, MatterEvent: &matterEvent, Link: &link, LinkEvent: &linkEvent})
	if err == nil {
		t.Fatal("invalid link payload did not fail the memory trigger bundle")
	}
	after, err := service.GetProgram(ctx, "bank", program.Program.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Program.Version != program.Program.Version || len(after.Triggers) != 0 {
		t.Fatalf("failed bundle advanced Program state: %#v", after)
	}
	if _, err := service.GetMatter(ctx, "bank", matter.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("failed bundle retained Matter: %v", err)
	}
	events, err := repo.ProgramEvents(ctx, "bank", program.Program.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.ID == programEvent.ID {
			t.Fatalf("failed bundle retained Program event: %#v", event)
		}
	}
	data := projectionData(repo)
	data.mu.Lock()
	defer data.mu.Unlock()
	for _, job := range data.jobs {
		if job.TriggerID == trigger.ID {
			t.Fatalf("failed bundle retained projection job: %#v", job)
		}
	}
}
