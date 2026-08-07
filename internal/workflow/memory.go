package workflow

import (
	"context"
	"sort"
	"sync"
	"time"
)

type MemoryRepository struct {
	mu    sync.RWMutex
	tasks map[string]Task
}

func NewMemoryRepository(seed []Task) *MemoryRepository {
	tasks := make(map[string]Task, len(seed))
	for _, task := range seed {
		task.Context = clone(task.Context)
		tasks[task.ID] = task
	}
	return &MemoryRepository{tasks: tasks}
}

func (r *MemoryRepository) List(_ context.Context, filter ListFilter) ([]Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := []Task{}
	for _, task := range r.tasks {
		if task.TenantID != filter.TenantID {
			continue
		}
		if filter.PrincipalID != "" && task.PrincipalID != filter.PrincipalID {
			continue
		}
		if filter.Status != "" && task.Status != filter.Status {
			continue
		}
		task.Context = clone(task.Context)
		values = append(values, task)
	}
	sort.Slice(values, func(i, j int) bool { return values[i].UpdatedAt.After(values[j].UpdatedAt) })
	if len(values) > filter.Limit {
		values = values[:filter.Limit]
	}
	return values, nil
}

func clone(input map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range input {
		out[key] = value
	}
	return out
}

func DemoTasks() []Task {
	now := time.Now().UTC()
	reviewDue := now.Add(3 * 24 * time.Hour)
	evidenceDue := now.Add(36 * time.Hour)
	return []Task{
		{ID: "task_review_cbn", TenantID: "bank-demo", WorkflowID: "wf_cbn_change", StepKey: "applicability-review", Responsibility: "REVIEWER", PrincipalID: "team-control-assurance", Title: "Review seven proposed obligations", Status: StatusReady, DueAt: &reviewDue, Context: map[string]string{"program": "CBN Digital Channels", "scope": "Bank NG"}, Version: 1, CreatedAt: now, UpdatedAt: now},
		{ID: "task_access_evidence", TenantID: "bank-demo", WorkflowID: "wf_access_review", StepKey: "focused-evidence", Responsibility: "PERFORMER", PrincipalID: "queue-risk-owners", Title: "Confirm four account owners", Status: StatusInProgress, DueAt: &evidenceDue, Context: map[string]string{"population": "4 of 1,250", "scope": "Treasury Operations"}, Version: 1, CreatedAt: now, UpdatedAt: now},
	}
}
