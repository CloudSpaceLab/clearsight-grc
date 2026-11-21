package continuity

import (
	"context"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestFrameworkProgramCountsRequireActiveScopedEffectiveProgramsAndAdoptions(t *testing.T) {
	now := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	earlier, later := now.Add(-time.Hour), now.Add(time.Hour)
	repo := NewMemoryRepository()
	repo.RegisterLegalEntity("bank", "entity-a", "EA")
	repo.RegisterLegalEntity("bank", "entity-b", "EB")
	adoption := FrameworkAdoption{TemplateCode: "ISO_27001_2022", Status: FrameworkAdoptionStatus("ACTIVE"), EffectiveFrom: &earlier}
	add := func(tenant, entity, id string, status ProgramStatus, adoptions ...FrameworkAdoption) {
		if repo.programs[tenant] == nil {
			repo.programs[tenant] = map[string]ProgramAggregate{}
		}
		repo.programs[tenant][id] = ProgramAggregate{Program: Program{ID: id, TenantID: tenant, LegalEntityID: entity, Status: status, EffectiveFrom: earlier}, FrameworkAdoptions: adoptions}
	}
	add("bank", "entity-a", "one", ProgramActive, adoption, adoption)
	add("bank", "entity-a", "two", ProgramActive, adoption)
	add("bank", "entity-a", "draft", ProgramDraft, adoption)
	add("bank", "entity-a", "paused", ProgramPaused, adoption)
	pending := adoption
	pending.Status = FrameworkAdoptionStatus("PENDING")
	add("bank", "entity-a", "pending", ProgramActive, pending)
	future := adoption
	future.EffectiveFrom = &later
	add("bank", "entity-a", "future-adoption", ProgramActive, future)
	expired := adoption
	expired.EffectiveUntil = &earlier
	add("bank", "entity-a", "expired-adoption", ProgramActive, expired)
	add("bank", "entity-b", "other-entity", ProgramActive, adoption)
	add("other-bank", "entity-a", "other-tenant", ProgramActive, adoption)
	add("bank", "entity-a", "future-program", ProgramActive, adoption)
	value := repo.programs["bank"]["future-program"]
	value.Program.EffectiveFrom = later
	repo.programs["bank"]["future-program"] = value
	add("bank", "entity-a", "expired-program", ProgramActive, adoption)
	value = repo.programs["bank"]["expired-program"]
	value.Program.EffectiveUntil = &earlier
	repo.programs["bank"]["expired-program"] = value
	service := NewService(repo)
	service.now = func() time.Time { return now }
	ctx := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "reader"})
	got, err := service.FrameworkProgramUsage(ctx)
	if err != nil || len(got) != 1 || got[0].TemplateCode != "ISO_27001_2022" || got[0].ActiveProgramCount != 2 {
		t.Fatalf("counts %+v %v", got, err)
	}
	for _, entity := range []string{"", "*"} {
		unscoped := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank", LegalEntityID: entity, PrincipalID: "reader"})
		if _, err = service.FrameworkProgramUsage(unscoped); err == nil {
			t.Fatal("accepted unscoped reader")
		}
	}
}
