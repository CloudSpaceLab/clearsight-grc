package workflow

import (
	"context"
	"testing"
	"time"
)

func TestListRequiresTenant(t *testing.T) {
	service := NewService(NewMemoryRepository(testTasks()))
	if _, err := service.List(context.Background(), ListFilter{}); err == nil {
		t.Fatal("expected tenant_id requirement")
	}
}

func TestListScopesTasksAndBoundsLimit(t *testing.T) {
	service := NewService(NewMemoryRepository(testTasks()))
	values, err := service.List(context.Background(), ListFilter{TenantID: "bank-demo", PrincipalID: "team-control-assurance", Limit: 500})
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 || values[0].ID != "task-review" {
		t.Fatalf("unexpected scoped tasks: %#v", values)
	}
}

func testTasks() []Task {
	return []Task{{ID: "task-review", TenantID: "bank-demo", PrincipalID: "team-control-assurance", Title: "Review stored obligations", Status: StatusReady, Version: 1}}
}

func TestListActiveOnlyFiltersBeforeLimitAndOrdersUrgentWork(t *testing.T) {
	now := time.Date(2026, 8, 7, 20, 0, 0, 0, time.UTC)
	overdue := now.Add(-time.Hour)
	later := now.Add(24 * time.Hour)
	tasks := []Task{
		{
			ID: "recent-terminal", TenantID: "bank", PrincipalID: "actor", WorkflowKind: MatterActionWorkflowKind,
			Status: StatusCompleted, DueAt: &overdue, UpdatedAt: now.Add(time.Hour),
		},
		{
			ID: "urgent-active", TenantID: "bank", PrincipalID: "actor", WorkflowKind: MatterActionWorkflowKind,
			Status: StatusReady, DueAt: &overdue, UpdatedAt: now.Add(-time.Hour),
		},
		{
			ID: "later-active", TenantID: "bank", PrincipalID: "actor", WorkflowKind: MatterActionWorkflowKind,
			Status: StatusReady, DueAt: &later, UpdatedAt: now,
		},
		{
			ID: "legacy-active", TenantID: "bank", PrincipalID: "actor", WorkflowKind: "REVIEW",
			Status: StatusReady, DueAt: &overdue, UpdatedAt: now.Add(2 * time.Hour),
		},
	}
	service := NewService(NewMemoryRepository(tasks))
	values, err := service.List(context.Background(), ListFilter{
		TenantID: "bank", PrincipalID: "actor", WorkflowKind: MatterActionWorkflowKind,
		ActiveOnly: true, Limit: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 || values[0].ID != "urgent-active" {
		t.Fatalf("expected overdue active Matter Action before terminal/legacy work, got %#v", values)
	}
}
