package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestActorWorkPagesIncludeEveryAuthorizedAssignmentBeyondFifty(t *testing.T) {
	base := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	due := base.Add(72 * time.Hour)
	tasks := make([]Task, 0, 83)
	for i := 0; i < 81; i++ {
		tasks = append(tasks, actorPageTestTask(i, base.Add(time.Duration(i) * time.Minute), due))
	}
	// Higher-priority times in other scopes may not occupy actor page slots.
	otherEntity := actorPageTestTask(81, base.Add(100 * time.Minute), due)
	otherEntity.LegalEntityID = "other-entity"
	otherActor := actorPageTestTask(82, base.Add(101 * time.Minute), due)
	otherActor.PrincipalID = "other-actor"
	tasks = append(tasks, otherEntity, otherActor)
	service := NewService(NewMemoryRepository(tasks))
	query := ActorWorkPageQuery{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "actor", Limit: 30}
	seen := map[string]bool{}
	for pageNumber := 0; pageNumber < 4; pageNumber++ {
		page, err := service.ListActorWorkPage(t.Context(), query)
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Items) == 0 {
			t.Fatalf("empty page %d before all assignments were retrieved", pageNumber)
		}
		for _, task := range page.Items {
			if seen[task.ID] {
				t.Fatalf("duplicate actor work across page boundary: %s", task.ID)
			}
			seen[task.ID] = true
			if task.LegalEntityID != "entity" || task.PrincipalID != "actor" {
				t.Fatalf("cross-scope assignment: %#v", task)
			}
		}
		if !page.HasMore {
			if len(seen) != 81 || page.NextCursor != "" {
				t.Fatalf("last page count=%d cursor=%q", len(seen), page.NextCursor)
			}
			return
		}
		if page.NextCursor == "" {
			t.Fatal("page reports more assignments without a cursor")
		}
		query.Cursor = page.NextCursor
	}
	t.Fatal("page traversal failed to terminate")
}

func TestActorWorkCursorDoesNotCrossScopeOrFilter(t *testing.T) {
	base := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	due := base.Add(72 * time.Hour)
	service := NewService(NewMemoryRepository([]Task{
		actorPageTestTask(1, base, due),
		actorPageTestTask(2, base.Add(time.Minute), due),
	}))
	query := ActorWorkPageQuery{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "actor", Limit: 1}
	page, err := service.ListActorWorkPage(t.Context(), query)
	if err != nil {
		t.Fatal(err)
	}
	if !page.HasMore {
		t.Fatal("expected the next assignment to be available")
	}
	for _, change := range []func(*ActorWorkPageQuery){
		func(q *ActorWorkPageQuery) { q.PrincipalID = "other" },
		func(q *ActorWorkPageQuery) { q.LegalEntityID = "other" },
		func(q *ActorWorkPageQuery) { q.Search = "different" },
		func(q *ActorWorkPageQuery) { q.Status = StatusBlocked },
		func(q *ActorWorkPageQuery) {
			value := base.Add(time.Hour)
			q.DueFrom = &value
		},
	} {
		changed := query
		changed.Cursor = page.NextCursor
		change(&changed)
		if _, err := service.ListActorWorkPage(t.Context(), changed); !errors.Is(err, ErrInvalidWorkCursor) {
			t.Fatalf("cursor scope change was accepted: %v", err)
		}
	}
	query.Cursor = "invalid!cursor"
	if _, err := service.ListActorWorkPage(t.Context(), query); !errors.Is(err, ErrInvalidWorkCursor) {
		t.Fatalf("malformed cursor must be rejected: %v", err)
	}
}

func TestActorWorkCursorFencesUpdatedAndRevokedRecords(t *testing.T) {
	base := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	due := base.Add(72 * time.Hour)
	repo := NewMemoryRepository([]Task{
		actorPageTestTask(0, base, due),
		actorPageTestTask(1, base.Add(time.Minute), due),
		actorPageTestTask(2, base.Add(2 * time.Minute), due),
	})
	service := NewService(repo)
	query := ActorWorkPageQuery{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "actor", Limit: 1}
	first, err := service.ListActorWorkPage(t.Context(), query)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 1 || !first.HasMore {
		t.Fatalf("unexpected first page: %#v", first)
	}
	// A later revision and a revoked assignment may not reappear under the
	// original page cursor, even if the record's former rank was below it.
	repo.mu.Lock()
	revised := repo.tasks[actorPageTestID(1)]
	revised.UpdatedAt = first.AsOf.Add(time.Second)
	repo.tasks[revised.ID] = revised
	revoked := repo.tasks[actorPageTestID(0)]
	revoked.PrincipalID = "other"
	repo.tasks[revoked.ID] = revoked
	repo.mu.Unlock()
	query.Cursor = first.NextCursor
	second, err := service.ListActorWorkPage(t.Context(), query)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Items) != 0 || second.HasMore {
		t.Fatalf("mutated/revoked actor work leaked into cursor continuation: %#v", second)
	}
}

func TestActorWorkCursorRejectsExpiredAndTerminalSelections(t *testing.T) {
	service := NewService(NewMemoryRepository(nil))
	query := ActorWorkPageQuery{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "actor"}
	query.Status = StatusCompleted
	if _, err := service.ListActorWorkPage(t.Context(), query); err == nil {
		t.Fatal("terminal work must not appear as current work")
	}
	query.Status = ""
	query.Cursor = encodeActorWorkCursor(actorWorkCursor{
		Version:  1,
		Scope:    actorWorkScopeFingerprint(query),
		AsOf:     time.Now().UTC().Add(-time.Hour),
		Position: WorkPosition{UpdatedAt: time.Now().UTC().Add(-2 * time.Hour), ID: actorPageTestID(0)},
	})
	if _, err := service.ListActorWorkPage(t.Context(), query); !errors.Is(err, ErrInvalidWorkCursor) {
		t.Fatalf("expired cursor accepted: %v", err)
	}
}

func actorPageTestID(number int) string {
	return fmt.Sprintf("00000000-0000-4000-8000-%012x", number+1)
}

func actorPageTestTask(number int, updated, due time.Time) Task {
	return Task{
		ID: actorPageTestID(number), TenantID: "bank", LegalEntityID: "entity",
		PrincipalID: "actor", WorkflowKind: MatterActionWorkflowKind,
		MatterID:    actorPageTestID(number + 200),
		MatterScope: json.RawMessage(`{"access":"RESTRICTED"}`),
		Title:       fmt.Sprintf("Review %03d", number),
		Status:      StatusReady, DueAt: &due, UpdatedAt: updated,
	}
}
