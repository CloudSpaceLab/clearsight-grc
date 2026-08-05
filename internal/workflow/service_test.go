package workflow

import (
	"context"
	"errors"
	"testing"
)

func TestTransitionUsesOptimisticVersionAndTenantScope(t *testing.T) {
	repo := NewMemoryRepository(DemoTasks())
	service := NewService(repo)
	task, err := service.Get(context.Background(), "bank-demo", "task_review_cbn")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := service.Transition(context.Background(), task.ID, TransitionInput{TenantID: "bank-demo", Status: StatusInProgress, ExpectedVersion: task.Version})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Version != 2 || updated.ClaimedAt == nil {
		t.Fatalf("expected claimed version 2 task, got %#v", updated)
	}
	_, err = service.Transition(context.Background(), task.ID, TransitionInput{TenantID: "bank-demo", Status: StatusCompleted, ExpectedVersion: 1})
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("expected version conflict, got %v", err)
	}
	_, err = service.Get(context.Background(), "other-tenant", task.ID)
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("expected tenant-scoped not found, got %v", err)
	}
}

func TestListRequiresTenant(t *testing.T) {
	service := NewService(NewMemoryRepository(DemoTasks()))
	if _, err := service.List(context.Background(), ListFilter{}); err == nil {
		t.Fatal("expected tenant_id requirement")
	}
}

func TestCompletionSetsTimestamp(t *testing.T) {
	service := NewService(NewMemoryRepository(DemoTasks()))
	task, err := service.Get(context.Background(), "bank-demo", "task_access_evidence")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := service.Transition(context.Background(), task.ID, TransitionInput{TenantID: "bank-demo", Status: StatusCompleted, ExpectedVersion: task.Version, Reason: "Evidence accepted"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.CompletedAt == nil {
		t.Fatal("expected completed_at")
	}
}
