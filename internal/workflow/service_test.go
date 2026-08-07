package workflow

import (
	"context"
	"testing"
)

func TestListRequiresTenant(t *testing.T) {
	service := NewService(NewMemoryRepository(DemoTasks()))
	if _, err := service.List(context.Background(), ListFilter{}); err == nil {
		t.Fatal("expected tenant_id requirement")
	}
}

func TestListScopesTasksAndBoundsLimit(t *testing.T) {
	service := NewService(NewMemoryRepository(DemoTasks()))
	values, err := service.List(context.Background(), ListFilter{TenantID: "bank-demo", PrincipalID: "team-control-assurance", Limit: 500})
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 || values[0].ID != "task_review_cbn" {
		t.Fatalf("unexpected scoped tasks: %#v", values)
	}
}
