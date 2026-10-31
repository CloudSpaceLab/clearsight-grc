package workflow

import (
	"testing"
	"time"
)

func TestActorWorkDueCountsAreSourceCompleteBeyondFirstPage(t *testing.T) {
	base := time.Now().UTC().Add(-2 * time.Hour)
	late := base.Add(-24 * time.Hour)
	soon := base.Add(24 * time.Hour)
	later := base.Add(14 * 24 * time.Hour)
	seed := make([]Task, 0, 115)
	for i := 0; i < 110; i++ {
		task := actorPageTestTask(i, base.Add(time.Duration(i)*time.Second), later)
		switch {
		case i < 10:
			task.DueAt = &late
		case i < 75:
			task.DueAt = &soon
		case i >= 101:
			task.DueAt = nil
		}
		seed = append(seed, task)
	}
	otherEntity := actorPageTestTask(200, base, late)
	otherEntity.LegalEntityID = "other"
	otherActor := actorPageTestTask(201, base, soon)
	otherActor.PrincipalID = "other"
	completed := actorPageTestTask(202, base, late)
	completed.Status = StatusCompleted
	unsupported := actorPageTestTask(203, base, late)
	unsupported.WorkflowKind = "UNSUPPORTED"
	seed = append(seed, otherEntity, otherActor, completed, unsupported)

	service := NewService(NewMemoryRepository(seed))
	counts, err := service.ReadActorWorkDueCounts(t.Context(), "bank", "entity", "actor")
	if err != nil {
		t.Fatal(err)
	}
	if counts.Total != 110 || counts.Overdue != 10 || counts.DueSoon != 65 || counts.Undated != 9 {
		t.Fatalf("unexpected source-complete counts: %#v", counts)
	}
	if counts.AsOf.IsZero() || counts.CountBasis != "SOURCE_SNAPSHOT" ||
		counts.Coverage != "WORKFLOW_ASSIGNMENTS_ONLY" {
		t.Fatalf("counts must identify their actual authorized source: %#v", counts)
	}
}

func TestActorWorkDueCountsFailClosedWithoutVerifiedIdentity(t *testing.T) {
	service := NewService(NewMemoryRepository(nil))
	for _, scope := range []struct{ tenant, entity, principal string }{
		{"", "entity", "actor"},
		{"bank", "", "actor"},
		{"bank", "entity", ""},
	} {
		if _, err := service.ReadActorWorkDueCounts(t.Context(), scope.tenant, scope.entity, scope.principal); err == nil {
			t.Fatal("missing verified scope must not produce zero counts")
		}
	}
}
