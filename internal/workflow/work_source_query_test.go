package workflow

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestWorkSourceSearchFindsAuthorizedAssignmentsBeyondFirstFifty(t *testing.T) {
	start := time.Date(2026, time.October, 20, 12, 0, 0, 0, time.UTC)
	items := make([]Task, 0, 80)
	for i := 0; i < 75; i++ {
		items = append(items, Task{
			ID: fmt.Sprintf("work-%03d", i), TenantID: "bank", LegalEntityID: "entity-a",
			PrincipalID: "reviewer", WorkflowKind: MatterActionWorkflowKind,
			MatterID: fmt.Sprintf("matter-%03d", i),
			MatterScope: json.RawMessage(`{"access":"RESTRICTED"}`),
			Title: fmt.Sprintf("Routine check %03d", i),
			Status: StatusReady, UpdatedAt: start.Add(time.Duration(74-i) * time.Minute),
		})
	}
	items[74].Title = "Review attestation from September"
	items = append(items,
		Task{ID: "other-entity", TenantID: "bank", LegalEntityID: "entity-b", PrincipalID: "reviewer",
			WorkflowKind: MatterActionWorkflowKind, MatterID: "another", MatterScope: json.RawMessage(`{"access":"INTERNAL"}`),
			Title: "Review attestation from another entity", Status: StatusReady},
		Task{ID: "other-actor", TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "other",
			WorkflowKind: MatterActionWorkflowKind, MatterID: "private", MatterScope: json.RawMessage(`{"access":"RESTRICTED"}`),
			Title: "Review attestation from another actor", Status: StatusReady},
		Task{ID: "unsupported", TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "reviewer",
			WorkflowKind: "UNSUPPORTED", Title: "Review attestation unlinked", Status: StatusReady},
	)
	service := NewService(NewMemoryRepository(items))
	filter := ListFilter{
		TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "reviewer",
		VisibleActorWorkOnly: true, ActiveOnly: true, Limit: 50,
	}
	first, err := service.List(t.Context(), filter)
	if err != nil { t.Fatal(err) }
	if len(first) != 50 {
		t.Fatalf("expected bounded first 50 authorized tasks, got %d", len(first))
	}
	filter.Search = "  ATTESTATION  "
	matched, err := service.List(t.Context(), filter)
	if err != nil { t.Fatal(err) }
	if len(matched) != 1 || matched[0].ID != "work-074" {
		t.Fatalf("search must filter the entire authorized source before limit, got %#v", matched)
	}
}

func TestWorkSourceDueWindowUsesInclusiveStartExclusiveEnd(t *testing.T) {
	start := time.Date(2026, time.October, 20, 0, 0, 0, 0, time.UTC)
	middle := start.Add(24 * time.Hour)
	end := middle.Add(24 * time.Hour)
	items := []Task{
		{ID: "start", TenantID: "bank", PrincipalID: "actor", DueAt: &start, Title: "Due at start", Status: StatusReady},
		{ID: "middle", TenantID: "bank", PrincipalID: "actor", DueAt: &middle, Title: "Due in window", Status: StatusReady},
		{ID: "end", TenantID: "bank", PrincipalID: "actor", DueAt: &end, Title: "Due at end", Status: StatusReady},
		{ID: "undated", TenantID: "bank", PrincipalID: "actor", Title: "No deadline", Status: StatusReady},
	}
	service := NewService(NewMemoryRepository(items))
	values, err := service.List(t.Context(), ListFilter{
		TenantID: "bank", PrincipalID: "actor", ActiveOnly: true, DueFrom: &start, DueBefore: &end, Limit: 20,
	})
	if err != nil { t.Fatal(err) }
	if len(values) != 2 || values[0].ID != "start" || values[1].ID != "middle" {
		t.Fatalf("due window returned %#v", values)
	}
}

func TestWorkSourceQueryRejectsUnscopedAndInvalidRequests(t *testing.T) {
	service := NewService(NewMemoryRepository(nil))
	cases := []struct {
		name   string
		filter ListFilter
	}{
		{"actor without principal", ListFilter{TenantID: "bank", VisibleActorWorkOnly: true}},
		{"matter without principal", ListFilter{TenantID: "bank", VisibleMatterWorkOnly: true}},
		{"search too long", ListFilter{TenantID: "bank", Search: strings.Repeat("x", 121)}},
		{"reverse due window", ListFilter{TenantID: "bank",
			DueFrom: ptrWorkTime(time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC)),
			DueBefore: ptrWorkTime(time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC))}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := service.List(t.Context(), tc.filter); err == nil {
				t.Fatal("expected invalid work query to fail closed")
			}
		})
	}
}

func ptrWorkTime(value time.Time) *time.Time { return &value }
