package continuity

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

type postCommitReadFailureRepository struct {
	*MemoryRepository
	failProgramRead      bool
	failMatterRead       bool
	failMatterAfterApply bool
}

func (r *postCommitReadFailureRepository) CreateProgram(ctx context.Context, program Program, event Event) (Program, error) {
	created, err := r.MemoryRepository.CreateProgram(ctx, program, event)
	if err == nil {
		r.failProgramRead = true
	}
	return created, err
}

func (r *postCommitReadFailureRepository) GetProgram(ctx context.Context, tenant, id string) (ProgramAggregate, error) {
	if r.failProgramRead {
		return ProgramAggregate{}, errors.New("current Program read unavailable")
	}
	return r.MemoryRepository.GetProgram(ctx, tenant, id)
}

func (r *postCommitReadFailureRepository) CreateMatterWithLink(ctx context.Context, bundle MatterLinkBundle) (Matter, error) {
	created, err := r.MemoryRepository.CreateMatterWithLink(ctx, bundle)
	if err == nil {
		r.failMatterRead = true
	}
	return created, err
}

func (r *postCommitReadFailureRepository) GetMatter(ctx context.Context, tenant, id string) (MatterAggregate, error) {
	if r.failMatterRead {
		return MatterAggregate{}, errors.New("current Matter read unavailable")
	}
	return r.MemoryRepository.GetMatter(ctx, tenant, id)
}

func (r *postCommitReadFailureRepository) ApplyMatterEvent(ctx context.Context, tenant, id string, expected int64, event Event) (int64, error) {
	version, err := r.MemoryRepository.ApplyMatterEvent(ctx, tenant, id, expected, event)
	if err == nil && r.failMatterAfterApply {
		r.failMatterRead = true
	}
	return version, err
}

func TestCommittedProgramReturnsCommandResultWhenRefreshReadFails(t *testing.T) {
	repository := &postCommitReadFailureRepository{MemoryRepository: NewMemoryRepository()}
	service := NewService(repository)
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	ctx := WithTrustedSystemScope(t.Context())

	program, err := service.CreateProgram(ctx, CreateProgramInput{TenantID: "bank", LegalEntityID: "entity-a", Code: "TPRM", Name: "Third-party risk", Type: "THIRD_PARTY", OwningFunction: "Risk", Scope: json.RawMessage(`{}`), EffectiveFrom: now})
	if err != nil {
		t.Fatalf("committed Program returned an error: %v", err)
	}
	if program.Program.ID == "" || program.Program.Version != 1 || program.Program.Code != "TPRM" {
		t.Fatalf("fallback command result = %#v", program)
	}
}

func TestCommittedMatterLinkReturnsCommandResultWhenCurrentReadFails(t *testing.T) {
	repository := &postCommitReadFailureRepository{MemoryRepository: NewMemoryRepository()}
	service := NewService(repository)
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	ctx := WithTrustedSystemScope(t.Context())
	program, err := repository.MemoryRepository.CreateProgram(ctx, Program{ID: "program-1", TenantID: "bank", LegalEntityID: "entity-a", Code: "TPRM", Name: "Third-party risk", Type: "THIRD_PARTY", Status: ProgramDraft, OwningFunction: "Risk", Scope: json.RawMessage(`{}`), EffectiveFrom: now, CreatedAt: now, UpdatedAt: now, Version: 1}, Event{ID: "event-1", TenantID: "bank", AggregateType: "PROGRAM", AggregateID: "program-1", AggregateVersion: 1, Type: EventProgramCreated, Payload: json.RawMessage(`{}`), OccurredAt: now})
	if err != nil {
		t.Fatal(err)
	}
	matter, err := service.CreateMatter(ctx, CreateMatterInput{TenantID: "bank", LegalEntityID: "entity-a", Type: MatterControlGap, Priority: 3, Title: "Obtain vendor assurance", Summary: "The vendor must provide current assurance evidence.", Scope: json.RawMessage(`{}`), ProgramID: program.ID})
	if err != nil {
		t.Fatalf("committed Matter returned an error: %v", err)
	}
	if matter.Matter.Version != 2 || len(matter.Links) != 1 || matter.Links[0].ProgramID != program.ID {
		t.Fatalf("fallback command result = %#v", matter)
	}
}

func TestCommittedMatterEventReturnsCommandResultWhenCurrentReadFails(t *testing.T) {
	repository := &postCommitReadFailureRepository{MemoryRepository: NewMemoryRepository()}
	service := NewService(repository)
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	ctx := WithTrustedSystemScope(t.Context())
	matter, err := repository.MemoryRepository.CreateMatter(ctx, Matter{ID: "matter-1", TenantID: "bank", LegalEntityID: "entity-a", Reference: "MAT-1", Type: MatterControlGap, Status: MatterDraft, Priority: 3, Title: "Obtain vendor assurance", Summary: "Current assurance evidence is required.", Scope: json.RawMessage(`{}`), KnownFacts: json.RawMessage(`{}`), MissingFacts: json.RawMessage(`[]`), Contradictions: json.RawMessage(`[]`), CreatedAt: now, UpdatedAt: now, Version: 1}, Event{ID: "event-1", TenantID: "bank", AggregateType: "MATTER", AggregateID: "matter-1", AggregateVersion: 1, Type: EventMatterCreated, Payload: json.RawMessage(`{}`), OccurredAt: now})
	if err != nil {
		t.Fatal(err)
	}
	repository.failMatterAfterApply = true

	result, err := service.AddAction(ctx, AddActionInput{TenantID: "bank", MatterID: matter.ID, ExpectedVersion: 1, Title: "Request current SOC report", Description: "The vendor must upload the current report.", OwnerPrincipalID: "owner-1", ActorID: "owner-1"})
	if err != nil {
		t.Fatalf("committed Matter event returned an error: %v", err)
	}
	if result.Matter.Version != 2 || len(result.Actions) != 1 || result.Actions[0].Title != "Request current SOC report" {
		t.Fatalf("fallback command result = %#v", result)
	}
}
