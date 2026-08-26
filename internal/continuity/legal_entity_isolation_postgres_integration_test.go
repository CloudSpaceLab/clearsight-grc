//go:build postgres && postgresintegration

package continuity

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresCurrentAndReplayReadersEnforceLegalEntityScope(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	const (
		tenantID = "98777777-7777-7777-8777-777777777771"
		entityA  = "98777777-7777-7777-8777-777777777772"
		entityB  = "98777777-7777-7777-8777-777777777773"
	)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID) })
	if _, err = pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'entity-scope-postgres','Entity Scope Test'); INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES($2::uuid,$1::uuid,'ENTITY-A','Entity A','NG'),($3::uuid,$1::uuid,'ENTITY-B','Entity B','NG')`, tenantID, entityA, entityB); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	current := NewCurrentPostgresRepository(pool)
	service := NewServiceWithClock(current, func() time.Time { return now })
	actorA := identity.WithActor(ctx, identity.Actor{TenantID: "entity-scope-postgres", PrincipalID: "actor-a", LegalEntityID: "ENTITY-A"})
	actorB := identity.WithActor(ctx, identity.Actor{TenantID: "entity-scope-postgres", PrincipalID: "actor-b", LegalEntityID: entityB})
	programA, err := service.CreateProgram(actorA, CreateProgramInput{TenantID: "entity-scope-postgres", LegalEntityID: entityB, Code: "SHARED", Name: "Program A", Type: "COMPLIANCE", OwningFunction: "Compliance", Scope: json.RawMessage(`{}`), EffectiveFrom: now})
	if err != nil {
		t.Fatal(err)
	}
	programB, err := service.CreateProgram(actorB, CreateProgramInput{TenantID: "entity-scope-postgres", LegalEntityID: entityB, Code: "SHARED", Name: "Program B", Type: "COMPLIANCE", OwningFunction: "Compliance", Scope: json.RawMessage(`{}`), EffectiveFrom: now})
	if err != nil {
		t.Fatal(err)
	}
	matterA, err := service.CreateMatter(actorA, CreateMatterInput{TenantID: "entity-scope-postgres", LegalEntityID: entityB, Type: MatterControlGap, Priority: 3, Title: "Matter A", Summary: "Entity A", Scope: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	matterB, err := service.CreateMatter(actorB, CreateMatterInput{TenantID: "entity-scope-postgres", LegalEntityID: entityB, Type: MatterControlGap, Priority: 3, Title: "Matter B", Summary: "Entity B", Scope: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}

	wildcard := identity.WithActor(ctx, identity.Actor{TenantID: "entity-scope-postgres", PrincipalID: "oversight", LegalEntityID: "*"})
	codeActorA := identity.WithActor(ctx, identity.Actor{TenantID: "entity-scope-postgres", PrincipalID: "actor-code-a", LegalEntityID: "ENTITY-A"})
	if found, findErr := service.ProgramByCode(actorA, "entity-scope-postgres", "SHARED"); findErr != nil || found.Program.ID != programA.Program.ID {
		t.Fatalf("entity A ProgramByCode = %#v, %v", found.Program, findErr)
	}
	if found, findErr := service.ProgramByCode(actorB, "entity-scope-postgres", "SHARED"); findErr != nil || found.Program.ID != programB.Program.ID {
		t.Fatalf("entity B ProgramByCode = %#v, %v", found.Program, findErr)
	}
	if found, findErr := service.GetProgram(codeActorA, "entity-scope-postgres", programA.Program.ID); findErr != nil || found.Program.ID != programA.Program.ID {
		t.Fatalf("code actor Program read = %#v, %v", found.Program, findErr)
	}
	if found, findErr := service.GetMatter(codeActorA, "entity-scope-postgres", matterA.Matter.ID); findErr != nil || found.Matter.ID != matterA.Matter.ID {
		t.Fatalf("code actor Matter read = %#v, %v", found.Matter, findErr)
	}
	for name, repo := range map[string]Repository{"replay": NewPostgresRepository(pool), "current": current} {
		t.Run(name, func(t *testing.T) {
			if values, listErr := repo.ListPrograms(actorA, "entity-scope-postgres", 20); listErr != nil || len(values) != 1 || values[0].Program.ID != programA.Program.ID {
				t.Fatalf("entity-scoped Program list = %#v, %v", values, listErr)
			}
			if _, getErr := repo.GetProgram(actorA, "entity-scope-postgres", programB.Program.ID); !errors.Is(getErr, ErrNotFound) {
				t.Fatalf("wrong-entity Program read error = %v", getErr)
			}
			if values, listErr := repo.ListMatters(actorA, "entity-scope-postgres", "OPEN", 20); listErr != nil || len(values) != 1 || values[0].Matter.ID != matterA.Matter.ID {
				t.Fatalf("entity-scoped Matter list = %#v, %v", values, listErr)
			}
			if _, getErr := repo.GetMatter(actorA, "entity-scope-postgres", matterB.Matter.ID); !errors.Is(getErr, ErrNotFound) {
				t.Fatalf("wrong-entity Matter read error = %v", getErr)
			}
			if values, listErr := repo.ListPrograms(wildcard, "entity-scope-postgres", 20); listErr != nil || len(values) != 2 {
				t.Fatalf("wildcard Program list = %#v, %v", values, listErr)
			}
		})
	}

	if _, err = service.TransitionMatter(actorA, TransitionInput{TenantID: "entity-scope-postgres", ID: matterB.Matter.ID, ExpectedVersion: matterB.Matter.Version, To: MatterInitialReview, ActorID: "actor-a", Rationale: "Cross-entity attempt"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("wrong-entity Matter mutation error = %v", err)
	}
	if _, err = current.GetProgram(wildcard, "entity-scope-postgres", programA.Program.ID); err != nil {
		t.Fatalf("wildcard Program read: %v", err)
	}

	_, createdMatter, inserted, err := service.ApplyTrigger(codeActorA, Trigger{TenantID: "entity-scope-postgres", ProgramID: programA.Program.ID, Type: "CONTROL_FAILED", DedupeKey: "shared-dedupe", Source: "test", ObservedAt: now})
	if err != nil || !inserted || createdMatter == nil {
		t.Fatalf("entity A trigger = %#v, %v, inserted=%v", createdMatter, err, inserted)
	}
	_, disclosed, inserted, err := service.ApplyTrigger(actorB, Trigger{TenantID: "entity-scope-postgres", ProgramID: programB.Program.ID, Type: "CONTROL_FAILED", DedupeKey: "shared-dedupe", Source: "test", ObservedAt: now.Add(time.Minute)})
	if err != nil || !inserted || disclosed == nil || disclosed.LegalEntityID != entityB {
		t.Fatalf("second Program trigger was not independently created: %#v, %v, inserted=%v", disclosed, err, inserted)
	}
	_, replayed, inserted, err := service.ApplyTrigger(actorB, Trigger{TenantID: "entity-scope-postgres", ProgramID: programB.Program.ID, Type: "CONTROL_FAILED", DedupeKey: "shared-dedupe", Source: "test", ObservedAt: now.Add(2 * time.Minute)})
	if err != nil || inserted || replayed == nil || replayed.ID != disclosed.ID {
		t.Fatalf("same Program trigger did not dedupe: %#v, %v, inserted=%v", replayed, err, inserted)
	}

	if _, err = pool.Exec(ctx, `INSERT INTO matter_links(tenant_id,matter_id,program_id,relationship) VALUES($1::uuid,$2::uuid,$3::uuid,'AFFECTS')`, tenantID, matterA.Matter.ID, programB.Program.ID); err == nil {
		t.Fatal("direct cross-entity Matter link was accepted")
	}
	if _, err = pool.Exec(ctx, `INSERT INTO matter_links(tenant_id,matter_id,program_id,relationship) VALUES($1::uuid,$2::uuid,$3::uuid,'AFFECTS')`, tenantID, matterA.Matter.ID, programA.Program.ID); err != nil {
		t.Fatalf("same-entity Matter link was rejected: %v", err)
	}
	if _, err = pool.Exec(ctx, `UPDATE matters SET legal_entity_id=$3::uuid WHERE tenant_id=$1::uuid AND id=$2::uuid`, tenantID, matterA.Matter.ID, entityB); err == nil {
		t.Fatal("linked Matter legal entity was mutable")
	}
	if _, err = pool.Exec(ctx, `UPDATE programs SET legal_entity_id=$3::uuid WHERE tenant_id=$1::uuid AND id=$2::uuid`, tenantID, programA.Program.ID, entityB); err == nil {
		t.Fatal("linked Program legal entity was mutable")
	}
	var persistedMatterEntity, persistedProgramEntity string
	if err = pool.QueryRow(ctx, `SELECT legal_entity_id::text FROM matters WHERE tenant_id=$1::uuid AND id=$2::uuid`, tenantID, matterA.Matter.ID).Scan(&persistedMatterEntity); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT legal_entity_id::text FROM programs WHERE tenant_id=$1::uuid AND id=$2::uuid`, tenantID, programA.Program.ID).Scan(&persistedProgramEntity); err != nil {
		t.Fatal(err)
	}
	if persistedMatterEntity != entityA || persistedProgramEntity != entityA {
		t.Fatalf("failed parent update changed entity: Matter=%s Program=%s", persistedMatterEntity, persistedProgramEntity)
	}

	var eventEntity, outboxEntity string
	if err = pool.QueryRow(ctx, `SELECT payload->>'legal_entity_id' FROM continuity_events WHERE tenant_id=$1::uuid AND aggregate_type='PROGRAM' AND aggregate_id=$2::uuid AND event_type='PROGRAM_CREATED'`, tenantID, programA.Program.ID).Scan(&eventEntity); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT payload->>'legal_entity_id' FROM outbox_events WHERE tenant_id=$1::uuid AND aggregate_type='PROGRAM' AND aggregate_id=$2::uuid AND event_type='PROGRAM_CREATED'`, tenantID, programA.Program.ID).Scan(&outboxEntity); err != nil {
		t.Fatal(err)
	}
	if programA.Program.LegalEntityID != entityA || eventEntity != entityA || outboxEntity != entityA {
		t.Fatalf("Program entity row/event/outbox mismatch: row=%s event=%s outbox=%s", programA.Program.LegalEntityID, eventEntity, outboxEntity)
	}
	if err = pool.QueryRow(ctx, `SELECT payload->>'legal_entity_id' FROM continuity_events WHERE tenant_id=$1::uuid AND aggregate_type='MATTER' AND aggregate_id=$2::uuid AND event_type='MATTER_CREATED'`, tenantID, matterA.Matter.ID).Scan(&eventEntity); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT payload->>'legal_entity_id' FROM outbox_events WHERE tenant_id=$1::uuid AND aggregate_type='MATTER' AND aggregate_id=$2::uuid AND event_type='MATTER_CREATED'`, tenantID, matterA.Matter.ID).Scan(&outboxEntity); err != nil {
		t.Fatal(err)
	}
	if eventEntity != entityA || outboxEntity != entityA {
		t.Fatalf("Matter entity row/event/outbox mismatch: row=%s event=%s outbox=%s", matterA.Matter.LegalEntityID, eventEntity, outboxEntity)
	}

	unknownEntity := "98777777-7777-7777-8777-777777777799"
	unknownActor := identity.WithActor(ctx, identity.Actor{TenantID: "entity-scope-postgres", PrincipalID: "unknown", LegalEntityID: unknownEntity})
	if _, err = service.CreateMatter(unknownActor, CreateMatterInput{TenantID: "entity-scope-postgres", Type: MatterControlGap, Priority: 3, Title: "Unknown entity", Summary: "Must fail", Scope: json.RawMessage(`{}`)}); !errors.Is(err, ErrNotFound) {
		t.Fatal("unknown Matter legal entity was accepted")
	}
	if _, err = pool.Exec(ctx, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES('98777777-7777-7777-8777-777777777791'::uuid,$1::uuid,'AMBIG','Ambiguous A','NG',clock_timestamp()-interval '2 seconds'),('98777777-7777-7777-8777-777777777792'::uuid,$1::uuid,'AMBIG','Ambiguous B','NG',clock_timestamp()-interval '1 second')`, tenantID); err != nil {
		t.Fatal(err)
	}
	ambiguousActor := identity.WithActor(ctx, identity.Actor{TenantID: "entity-scope-postgres", PrincipalID: "ambiguous", LegalEntityID: "AMBIG"})
	if _, err = service.CreateProgram(ambiguousActor, CreateProgramInput{TenantID: "entity-scope-postgres", Code: "AMBIG-PROGRAM", Name: "Must not write", Type: "COMPLIANCE", OwningFunction: "Compliance", Scope: json.RawMessage(`{}`), EffectiveFrom: now}); !errors.Is(err, ErrLegalEntityAmbiguous) {
		t.Fatalf("ambiguous legal entity error=%v", err)
	}
	var rejectedWrites int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM programs WHERE tenant_id=$1::uuid AND code='AMBIG-PROGRAM'`, tenantID).Scan(&rejectedWrites); err != nil || rejectedWrites != 0 {
		t.Fatalf("ambiguous entity wrote Program: count=%d err=%v", rejectedWrites, err)
	}
}
