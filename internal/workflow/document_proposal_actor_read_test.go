package workflow

import (
	"context"
	"testing"
	"time"
)

type actorWorkMergeRepository struct {
	generic  []Task
	document []Task
}

func (r *actorWorkMergeRepository) List(context.Context, ListFilter) ([]Task, error) {
	return append([]Task(nil), r.generic...), nil
}

func (r *actorWorkMergeRepository) ListDocumentProposalActorWork(context.Context, ListFilter) ([]Task, error) {
	return append([]Task(nil), r.document...), nil
}

func TestActorWorkMergeLetsDocumentProposalCompeteForBoundedQueue(t *testing.T) {
	now := time.Date(2026, 8, 22, 14, 0, 0, 0, time.UTC)
	later := now.Add(2 * time.Hour)
	sooner := now.Add(time.Hour)
	repo := &actorWorkMergeRepository{
		generic: []Task{
			{ID: "generic-a", DueAt: &later, UpdatedAt: now},
			{ID: "generic-b", DueAt: nil, UpdatedAt: now.Add(time.Minute)},
		},
		document: []Task{
			{ID: "document-review", WorkflowKind: DocumentProposalWorkflowKind, DueAt: &sooner, UpdatedAt: now.Add(2 * time.Minute)},
		},
	}
	service := NewService(repo)

	values, err := service.List(context.Background(), ListFilter{
		TenantID: "tenant-a", PrincipalID: "reviewer-a", VisibleActorWorkOnly: true, ActiveOnly: true, Limit: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 2 {
		t.Fatalf("expected bounded two-item queue, got %d", len(values))
	}
	if values[0].ID != "document-review" || values[1].ID != "generic-a" {
		t.Fatalf("document work did not compete by queue ordering: %#v", values)
	}
}
