package continuity

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestProgramStateDoesNotAdvanceCommandVersion(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	service := NewService(repo)
	now := time.Date(2026, 8, 5, 18, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	program, err := service.CreateProgram(ctx, CreateProgramInput{TenantID: "bank", Code: "PRIVACY", Name: "Privacy", Type: "PRIVACY", OwningFunction: "Privacy", Scope: json.RawMessage(`{}`), EffectiveFrom: now})
	if err != nil {
		t.Fatal(err)
	}
	if program.Program.Version != 1 {
		t.Fatalf("calculated status advanced command version: %d", program.Program.Version)
	}
	if program.CurrentState == nil || program.CurrentState.ProgramVersion != 1 {
		t.Fatalf("status was not tied to command version 1: %#v", program.CurrentState)
	}
	program, err = service.AddRequirement(ctx, AddRequirementInput{TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: 1, Code: "REQ-1", Title: "Keep records", Statement: "Records must be current.", Modality: "MUST", Status: RequirementApproved, EffectiveFrom: now})
	if err != nil {
		t.Fatal(err)
	}
	if program.Program.Version != 2 || program.CurrentState == nil || program.CurrentState.ProgramVersion != 2 {
		t.Fatalf("unexpected command/projection versions: command=%d state=%#v", program.Program.Version, program.CurrentState)
	}
}

func TestProjectionRebuildQueueAndMaintainer(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	service := NewService(repo)
	now := time.Date(2026, 8, 5, 18, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	program, err := service.CreateProgram(ctx, CreateProgramInput{TenantID: "bank", Code: "CYBER", Name: "Cyber", Type: "CYBER", OwningFunction: "Security", Scope: json.RawMessage(`{}`), EffectiveFrom: now})
	if err != nil {
		t.Fatal(err)
	}
	job, err := service.QueueProgramStateRebuild(ctx, "bank", program.Program.ID, "person-1", "Check the latest Program status")
	if err != nil || job.Status != ProjectionJobReady {
		t.Fatalf("unexpected rebuild job %#v err=%v", job, err)
	}
	health, err := service.ProjectionHealth(ctx, "bank")
	if err != nil || len(health) != 1 || health[0].Pending != 1 {
		t.Fatalf("unexpected pending health %#v err=%v", health, err)
	}
	maintenanceTime := job.AvailableAt.Add(time.Second)
	maintainer := &ProjectionMaintainer{Service: service, Repo: repo, WorkerID: "worker-1", Now: func() time.Time { return maintenanceTime }}
	completed, err := maintainer.Maintain(ctx, maintenanceTime, 10)
	if err != nil || completed != 1 {
		t.Fatalf("unexpected maintain result completed=%d err=%v", completed, err)
	}
	health, err = service.ProjectionHealth(ctx, "bank")
	if err != nil || health[0].Pending != 0 || health[0].Failed != 0 || health[0].LastCompleted == nil {
		t.Fatalf("unexpected completed health %#v err=%v", health, err)
	}
}

func TestMatterCreationAndInitialProgramLinkAreAtomicInMemory(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	service := NewService(repo)
	now := time.Date(2026, 8, 5, 18, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	program, err := service.CreateProgram(ctx, CreateProgramInput{TenantID: "bank", Code: "OPS", Name: "Operations", Type: "OPERATIONS", OwningFunction: "Operations", Scope: json.RawMessage(`{}`), EffectiveFrom: now})
	if err != nil {
		t.Fatal(err)
	}
	matter, err := service.CreateMatter(ctx, CreateMatterInput{TenantID: "bank", Type: MatterControlGap, Priority: 3, Title: "Resolve missing review", Summary: "A required review is missing.", Scope: json.RawMessage(`{}`), ProgramID: program.Program.ID})
	if err != nil {
		t.Fatal(err)
	}
	if matter.Matter.Version != 2 || len(matter.Links) != 1 || matter.Links[0].ProgramID != program.Program.ID {
		t.Fatalf("matter and first link were not committed together: %#v", matter)
	}
	events, err := repo.MatterEvents(ctx, "bank", matter.Matter.ID, nil)
	if err != nil || len(events) != 2 || events[0].Type != EventMatterCreated || events[1].Type != EventMatterLinked {
		t.Fatalf("unexpected atomic event history %#v err=%v", events, err)
	}
}
