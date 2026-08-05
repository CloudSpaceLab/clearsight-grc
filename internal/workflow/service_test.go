package workflow

import (
	"context"
	"errors"
	"testing"
)

func TestTransitionUsesOptimisticVersion(t *testing.T) {
	repo := NewMemoryRepository(DemoTasks())
	service := NewService(repo)
	task, err := service.Get(context.Background(), "task_review_cbn")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := service.Transition(context.Background(), task.ID, TransitionInput{Status: StatusInProgress, ExpectedVersion: task.Version})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Version != 2 {
		t.Fatalf("expected version 2, got %d", updated.Version)
	}
	_, err = service.Transition(context.Background(), task.ID, TransitionInput{Status: StatusCompleted, ExpectedVersion: 1})
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("expected version conflict, got %v", err)
	}
}
