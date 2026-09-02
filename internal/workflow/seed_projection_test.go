package workflow

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

func TestTasksFromMatterAggregatesUseOnlyStoredActiveAssignments(t *testing.T) {
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	values := []continuity.MatterAggregate{{
		Matter: continuity.Matter{ID: "matter-1", TenantID: "bank", LegalEntityID: "entity-a", Priority: 4, Scope: json.RawMessage(`{"access":"RESTRICTED","allowed_principal_ids":["owner-1"]}`)},
		Actions: []continuity.Action{
			{ID: "action-open", MatterID: "matter-1", Title: "Confirm the current control owner", OwnerPrincipalID: "owner-1", RequiredResponsibility: "PERFORMER", Status: continuity.ActionInProgress, CreatedAt: now, UpdatedAt: now, Version: 2},
			{ID: "action-done", MatterID: "matter-1", Title: "Already completed", OwnerPrincipalID: "owner-1", Status: continuity.ActionImplemented, CreatedAt: now, UpdatedAt: now, Version: 3},
		},
	}}

	tasks := TasksFromMatterAggregates(values)
	if len(tasks) != 1 || tasks[0].PrincipalID != "owner-1" || tasks[0].Title != "Confirm the current control owner" || tasks[0].LegalEntityID != "entity-a" {
		t.Fatalf("projected tasks = %#v", tasks)
	}
	repo := NewMemoryRepository(tasks)
	outside, err := repo.List(t.Context(), ListFilter{TenantID: "bank", LegalEntityID: "entity-b", PrincipalID: "owner-1", ActiveOnly: true, VisibleActorWorkOnly: true, Limit: 20})
	if err != nil || len(outside) != 0 {
		t.Fatalf("cross-entity tasks = %#v, err = %v", outside, err)
	}
}
