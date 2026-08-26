package continuity

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestVerifiedActorLegalEntityScopesProgramReadsAndMutation(t *testing.T) {
	repo, service, now := legalEntityIsolationFixture()
	ctx := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank", PrincipalID: "actor-a", LegalEntityID: "entity-a"})

	values, err := service.ListPrograms(ctx, "bank", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 || values[0].Program.ID != "program-a" {
		t.Fatalf("wrong-entity Program was listed: %#v", values)
	}
	if _, err = service.GetProgram(ctx, "bank", "program-b"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("wrong-entity Program read error = %v, want ErrNotFound", err)
	}
	_, err = service.UpdateProgramDetails(ctx, UpdateProgramDetailsInput{
		TenantID: "bank", ProgramID: "program-b", ExpectedVersion: 1,
		Name: "Changed", OwningFunction: "Compliance", Scope: json.RawMessage(`{}`), EffectiveFrom: now,
		ActorID: "actor-a", Rationale: "Adversarial cross-entity update",
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("wrong-entity Program mutation error = %v, want ErrNotFound", err)
	}
	if got := repo.programs["bank"]["program-b"].Program.Name; got != "Program B" {
		t.Fatalf("wrong-entity Program changed to %q", got)
	}
}

func TestMissingOrPartialActorCannotReadButTrustedSystemCan(t *testing.T) {
	_, service, _ := legalEntityIsolationFixture()
	for name, ctx := range map[string]context.Context{
		"missing":        context.Background(),
		"missing tenant": identity.WithActor(context.Background(), identity.Actor{PrincipalID: "actor-a", LegalEntityID: "entity-a"}),
		"missing entity": identity.WithActor(context.Background(), identity.Actor{TenantID: "bank", PrincipalID: "actor-a"}),
	} {
		t.Run(name, func(t *testing.T) {
			if values, err := service.ListPrograms(ctx, "bank", 20); err != nil || len(values) != 0 {
				t.Fatalf("unverified Program list = %#v, %v", values, err)
			}
			if _, err := service.GetMatter(ctx, "bank", "matter-a"); !errors.Is(err, ErrNotFound) {
				t.Fatalf("unverified Matter read error = %v", err)
			}
		})
	}
	trusted := WithTrustedSystemScope(context.Background())
	if values, err := service.ListPrograms(trusted, "bank", 20); err != nil || len(values) != 2 {
		t.Fatalf("trusted Program list = %#v, %v", values, err)
	}
	if _, err := service.GetMatter(trusted, "bank", "matter-a"); err != nil {
		t.Fatalf("trusted Matter read: %v", err)
	}
}

func TestProgramByCodeScopesBeforeSelectingDuplicateCode(t *testing.T) {
	repo, service, _ := legalEntityIsolationFixture()
	programA := repo.programs["bank"]["program-a"]
	programB := repo.programs["bank"]["program-b"]
	programA.Program.Code = "SHARED"
	programA.Program.UpdatedAt = programB.Program.UpdatedAt.Add(-time.Hour)
	programB.Program.Code = "SHARED"
	repo.programs["bank"]["program-a"] = programA
	repo.programs["bank"]["program-b"] = programB
	ctx := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank", PrincipalID: "actor-a", LegalEntityID: "entity-a"})

	got, err := service.ProgramByCode(ctx, "bank", "shared")
	if err != nil {
		t.Fatal(err)
	}
	if got.Program.ID != "program-a" {
		t.Fatalf("ProgramByCode selected wrong entity: %#v", got.Program)
	}
}

func TestVerifiedActorLegalEntityScopesMatterReadsAndMutation(t *testing.T) {
	repo, service, _ := legalEntityIsolationFixture()
	ctx := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank", PrincipalID: "actor-a", LegalEntityID: "entity-a"})

	values, err := service.ListMatters(ctx, "bank", "OPEN", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 || values[0].Matter.ID != "matter-a" {
		t.Fatalf("wrong-entity Matter was listed: %#v", values)
	}
	if _, err = service.GetMatter(ctx, "bank", "matter-b"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("wrong-entity Matter read error = %v, want ErrNotFound", err)
	}
	_, err = service.TransitionMatter(ctx, TransitionInput{
		TenantID: "bank", ID: "matter-b", ExpectedVersion: 1, To: MatterInitialReview,
		ActorID: "actor-a", Rationale: "Adversarial cross-entity update",
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("wrong-entity Matter mutation error = %v, want ErrNotFound", err)
	}
	if got := repo.matters["bank"]["matter-b"].Matter.Status; got != MatterDraft {
		t.Fatalf("wrong-entity Matter changed to %q", got)
	}
}

func TestWildcardOversightCanReadAndMutateAcrossLegalEntities(t *testing.T) {
	_, service, _ := legalEntityIsolationFixture()
	ctx := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank", PrincipalID: "oversight", LegalEntityID: "*"})

	programs, err := service.ListPrograms(ctx, "bank", 20)
	if err != nil || len(programs) != 2 {
		t.Fatalf("wildcard Program list = %#v, %v", programs, err)
	}
	matters, err := service.ListMatters(ctx, "bank", "OPEN", 20)
	if err != nil || len(matters) != 2 {
		t.Fatalf("wildcard Matter list = %#v, %v", matters, err)
	}
	if _, err = service.GetProgram(ctx, "bank", "program-b"); err != nil {
		t.Fatalf("wildcard Program read: %v", err)
	}
	if _, err = service.TransitionMatter(ctx, TransitionInput{
		TenantID: "bank", ID: "matter-b", ExpectedVersion: 1, To: MatterInitialReview,
		ActorID: "oversight", Rationale: "Oversight handling",
	}); err != nil {
		t.Fatalf("wildcard Matter mutation: %v", err)
	}
}

func TestCreateUsesVerifiedActorLegalEntity(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewService(repo)
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	ctx := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank", PrincipalID: "actor-a", LegalEntityID: "entity-a"})

	program, err := service.CreateProgram(ctx, CreateProgramInput{
		TenantID: "bank", LegalEntityID: "entity-b", Code: "ENTITY-A", Name: "Entity A Program",
		Type: "COMPLIANCE", OwningFunction: "Compliance", Scope: json.RawMessage(`{}`), EffectiveFrom: now,
		ActorID: "actor-a",
	})
	if err != nil {
		t.Fatal(err)
	}
	if program.Program.LegalEntityID != "entity-a" {
		t.Fatalf("Program legal entity = %q, want verified entity-a", program.Program.LegalEntityID)
	}

	var matterInput CreateMatterInput
	if err = json.Unmarshal([]byte(`{"tenant_id":"bank","legal_entity_id":"entity-b","type":"CONTROL_GAP","priority":3,"title":"Control gap","summary":"A gap owned by entity A","scope":{},"actor_id":"actor-a"}`), &matterInput); err != nil {
		t.Fatal(err)
	}
	matter, err := service.CreateMatter(ctx, matterInput)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(matter.Matter)
	if err != nil {
		t.Fatal(err)
	}
	var stored map[string]any
	if err = json.Unmarshal(encoded, &stored); err != nil {
		t.Fatal(err)
	}
	if stored["legal_entity_id"] != "entity-a" {
		t.Fatalf("Matter legal entity = %#v, want verified entity-a", stored["legal_entity_id"])
	}
}

func TestCreateCanonicalizesVerifiedLegalEntityBeforeEvents(t *testing.T) {
	repo := NewMemoryRepository()
	repo.RegisterLegalEntity("bank", "11111111-1111-4111-8111-111111111111", "ENTITY-A")
	service := NewService(repo)
	ctx := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank", PrincipalID: "actor-a", LegalEntityID: "ENTITY-A"})

	program, err := service.CreateProgram(ctx, CreateProgramInput{TenantID: "bank", LegalEntityID: "ignored", Code: "CANONICAL", Name: "Canonical", Type: "COMPLIANCE", OwningFunction: "Compliance", Scope: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	matter, err := service.CreateMatter(ctx, CreateMatterInput{TenantID: "bank", LegalEntityID: "ignored", Type: MatterControlGap, Priority: 3, Title: "Canonical matter", Summary: "Canonical scope", Scope: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	const canonical = "11111111-1111-4111-8111-111111111111"
	if program.Program.LegalEntityID != canonical || matter.Matter.LegalEntityID != canonical {
		t.Fatalf("row entities Program=%q Matter=%q, want %q", program.Program.LegalEntityID, matter.Matter.LegalEntityID, canonical)
	}
	for _, event := range []Event{repo.programEvents["bank"][program.Program.ID][0], repo.matterEvents["bank"][matter.Matter.ID][0]} {
		var payload struct {
			LegalEntityID string `json:"legal_entity_id"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.LegalEntityID != canonical {
			t.Fatalf("event entity=%q, want %q", payload.LegalEntityID, canonical)
		}
	}
}

func TestMemoryCodeScopedActorCanReadOnlyCanonicalEntity(t *testing.T) {
	repo := NewMemoryRepository()
	repo.RegisterLegalEntity("bank", "11111111-1111-4111-8111-111111111111", "ENTITY-A")
	repo.RegisterLegalEntity("bank", "22222222-2222-4222-8222-222222222222", "ENTITY-B")
	service := NewService(repo)
	actorA := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank", PrincipalID: "actor-a", LegalEntityID: "ENTITY-A"})
	actorB := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank", PrincipalID: "actor-b", LegalEntityID: "ENTITY-B"})
	programA, err := service.CreateProgram(actorA, CreateProgramInput{TenantID: "bank", Code: "A", Name: "A", Type: "COMPLIANCE", OwningFunction: "Compliance", Scope: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	programB, err := service.CreateProgram(actorB, CreateProgramInput{TenantID: "bank", Code: "B", Name: "B", Type: "COMPLIANCE", OwningFunction: "Compliance", Scope: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	matterA, err := service.CreateMatter(actorA, CreateMatterInput{TenantID: "bank", Type: MatterControlGap, Priority: 3, Title: "A", Summary: "A", Scope: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	if values, listErr := service.ListPrograms(actorA, "bank", 20); listErr != nil || len(values) != 1 || values[0].Program.ID != programA.Program.ID {
		t.Fatalf("code-scoped Program list=%#v err=%v", values, listErr)
	}
	if _, getErr := service.GetProgram(actorA, "bank", programA.Program.ID); getErr != nil {
		t.Fatal(getErr)
	}
	if _, getErr := service.GetMatter(actorA, "bank", matterA.Matter.ID); getErr != nil {
		t.Fatal(getErr)
	}
	if _, getErr := service.GetProgram(actorA, "bank", programB.Program.ID); !errors.Is(getErr, ErrNotFound) {
		t.Fatalf("other-entity Program error=%v", getErr)
	}
	bounded := WithTrustedSystemEntityScope(context.Background(), "bank", "ENTITY-A")
	if _, getErr := service.GetProgram(bounded, "bank", programA.Program.ID); getErr != nil {
		t.Fatal(getErr)
	}
	if _, getErr := service.GetProgram(bounded, "bank", programB.Program.ID); !errors.Is(getErr, ErrNotFound) {
		t.Fatalf("bounded trusted scope disclosed other entity: %v", getErr)
	}
}

func TestCreateRejectsUnknownOrAmbiguousLegalEntityBeforeWrite(t *testing.T) {
	repo := NewMemoryRepository()
	repo.RegisterLegalEntity("bank", "11111111-1111-4111-8111-111111111111", "SHARED")
	repo.RegisterLegalEntity("bank", "22222222-2222-4222-8222-222222222222", "SHARED")
	service := NewService(repo)
	for _, entity := range []string{"UNKNOWN", "SHARED"} {
		ctx := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank", PrincipalID: "actor", LegalEntityID: entity})
		_, err := service.CreateProgram(ctx, CreateProgramInput{TenantID: "bank", Code: entity, Name: entity, Type: "COMPLIANCE", OwningFunction: "Compliance", Scope: json.RawMessage(`{}`)})
		if !errors.Is(err, ErrNotFound) && !errors.Is(err, ErrLegalEntityAmbiguous) {
			t.Fatalf("entity %q error=%v", entity, err)
		}
	}
	if len(repo.programs["bank"]) != 0 || len(repo.programEvents["bank"]) != 0 {
		t.Fatalf("rejected entities wrote state: programs=%d events=%d", len(repo.programs["bank"]), len(repo.programEvents["bank"]))
	}
}

func TestTriggerDedupeIsScopedToProgram(t *testing.T) {
	repo, service, now := legalEntityIsolationFixture()
	repo.triggers["bank"] = map[string]Trigger{
		programTriggerDedupeKey("program-a", "shared-trigger"): {ID: "trigger-a", TenantID: "bank", ProgramID: "program-a", Type: "CONTROL_FAILED", DedupeKey: "shared-trigger", Source: "test", ObservedAt: now},
	}
	matterA := repo.matters["bank"]["matter-a"]
	matterA.Matter.TriggerKey = "shared-trigger"
	matterA.Links = []MatterLink{{ID: "link-a", TenantID: "bank", MatterID: "matter-a", ProgramID: "program-a", Relationship: "AFFECTS", CreatedAt: now}}
	repo.matters["bank"]["matter-a"] = matterA
	ctx := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank", PrincipalID: "actor-b", LegalEntityID: "entity-b"})

	_, returned, inserted, err := service.ApplyTrigger(ctx, Trigger{
		TenantID: "bank", ProgramID: "program-b", Type: "CONTROL_FAILED", DedupeKey: "shared-trigger", Source: "test", ObservedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !inserted || returned == nil || returned.LegalEntityID != "entity-b" {
		t.Fatalf("second Program trigger was not independently created: inserted=%v matter=%#v", inserted, returned)
	}
	_, replayed, inserted, err := service.ApplyTrigger(ctx, Trigger{
		TenantID: "bank", ProgramID: "program-b", Type: "CONTROL_FAILED", DedupeKey: "shared-trigger", Source: "test", ObservedAt: now.Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if inserted || replayed == nil || replayed.ID != returned.ID {
		t.Fatalf("same Program trigger did not dedupe: inserted=%v matter=%#v", inserted, replayed)
	}
}

func TestWildcardCannotLinkMatterToProgramFromAnotherLegalEntity(t *testing.T) {
	_, service, _ := legalEntityIsolationFixture()
	ctx := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank", PrincipalID: "oversight", LegalEntityID: "*"})

	if _, err := service.AddMatterLink(ctx, AddMatterLinkInput{TenantID: "bank", MatterID: "matter-a", ProgramID: "program-b", ExpectedVersion: 1, ActorID: "oversight"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-entity link error = %v, want ErrNotFound", err)
	}
	linked, err := service.AddMatterLink(ctx, AddMatterLinkInput{TenantID: "bank", MatterID: "matter-a", ProgramID: "program-a", ExpectedVersion: 1, ActorID: "oversight"})
	if err != nil {
		t.Fatalf("same-entity wildcard link: %v", err)
	}
	if len(linked.Links) != 1 || linked.Links[0].ProgramID != "program-a" {
		t.Fatalf("same-entity link missing: %#v", linked.Links)
	}
}

func TestWildcardCannotReadOrMutateUnresolvedMatter(t *testing.T) {
	repo, service, now := legalEntityIsolationFixture()
	repo.matters["bank"]["legacy"] = MatterAggregate{Matter: Matter{
		ID: "legacy", TenantID: "bank", Reference: "MAT-LEGACY", Type: MatterControlGap, Status: MatterDraft,
		Priority: 3, Title: "Legacy", Summary: "No resolved legal entity", Scope: json.RawMessage(`{}`),
		KnownFacts: json.RawMessage(`{}`), MissingFacts: json.RawMessage(`[]`), Contradictions: json.RawMessage(`[]`), CreatedAt: now, UpdatedAt: now, Version: 1,
	}}
	repo.matterEvents["bank"]["legacy"] = []Event{}
	ctx := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank", PrincipalID: "oversight", LegalEntityID: "*"})

	if _, err := service.GetMatter(ctx, "bank", "legacy"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unresolved wildcard read error = %v, want ErrNotFound", err)
	}
	if _, err := service.TransitionMatter(ctx, TransitionInput{TenantID: "bank", ID: "legacy", ExpectedVersion: 1, To: MatterInitialReview, ActorID: "oversight", Rationale: "Attempt"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unresolved wildcard mutation error = %v, want ErrNotFound", err)
	}
}

func legalEntityIsolationFixture() (*MemoryRepository, *Service, time.Time) {
	repo := NewMemoryRepository()
	now := time.Date(2026, 8, 26, 9, 0, 0, 0, time.UTC)
	repo.programs["bank"] = map[string]ProgramAggregate{
		"program-a": {Program: Program{ID: "program-a", TenantID: "bank", LegalEntityID: "entity-a", Code: "A", Name: "Program A", Type: "COMPLIANCE", Status: ProgramDraft, OwningFunction: "Compliance", Scope: json.RawMessage(`{}`), EffectiveFrom: now, CreatedAt: now, UpdatedAt: now, Version: 1}},
		"program-b": {Program: Program{ID: "program-b", TenantID: "bank", LegalEntityID: "entity-b", Code: "B", Name: "Program B", Type: "COMPLIANCE", Status: ProgramDraft, OwningFunction: "Compliance", Scope: json.RawMessage(`{}`), EffectiveFrom: now, CreatedAt: now, UpdatedAt: now, Version: 1}},
	}
	repo.programEvents["bank"] = map[string][]Event{"program-a": {}, "program-b": {}}
	var matterA, matterB Matter
	_ = json.Unmarshal([]byte(`{"id":"matter-a","tenant_id":"bank","legal_entity_id":"entity-a","reference":"MAT-A","type":"CONTROL_GAP","status":"DRAFT","priority":3,"title":"Matter A","summary":"Entity A","scope":{},"known_facts":{},"missing_facts":[],"contradictions":[],"created_at":"2026-08-26T09:00:00Z","updated_at":"2026-08-26T09:00:00Z","version":1}`), &matterA)
	_ = json.Unmarshal([]byte(`{"id":"matter-b","tenant_id":"bank","legal_entity_id":"entity-b","reference":"MAT-B","type":"CONTROL_GAP","status":"DRAFT","priority":3,"title":"Matter B","summary":"Entity B","scope":{},"known_facts":{},"missing_facts":[],"contradictions":[],"created_at":"2026-08-26T09:00:00Z","updated_at":"2026-08-26T09:00:00Z","version":1}`), &matterB)
	repo.matters["bank"] = map[string]MatterAggregate{"matter-a": {Matter: matterA}, "matter-b": {Matter: matterB}}
	repo.matterEvents["bank"] = map[string][]Event{"matter-a": {}, "matter-b": {}}
	service := NewService(repo)
	service.now = func() time.Time { return now.Add(time.Minute) }
	return repo, service, now
}
