package workflow

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

type MemoryRepository struct {
	mu    sync.RWMutex
	tasks map[string]Task
	now   func() time.Time
}

func NewMemoryRepository(seed []Task) *MemoryRepository {
	tasks := make(map[string]Task, len(seed))
	for _, task := range seed {
		task.Context = clone(task.Context)
		tasks[task.ID] = task
	}
	return &MemoryRepository{tasks: tasks, now: time.Now}
}

func (r *MemoryRepository) Create(_ context.Context, input CreateInput) (Task, error) {
	taskID, err := id.New("tsk", 18)
	if err != nil {
		return Task{}, err
	}
	now := r.now().UTC()
	task := Task{ID: taskID, TenantID: input.TenantID, WorkflowID: input.WorkflowID, StepKey: input.StepKey, Responsibility: input.Responsibility, PrincipalID: input.PrincipalID, Title: input.Title, Status: StatusReady, DueAt: input.DueAt, Context: clone(input.Context), Version: 1, CreatedAt: now, UpdatedAt: now}
	r.mu.Lock()
	r.tasks[task.ID] = task
	r.mu.Unlock()
	return task, nil
}

func (r *MemoryRepository) Get(_ context.Context, tenantID, id string) (Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	task, ok := r.tasks[id]
	if !ok || task.TenantID != tenantID {
		return Task{}, ErrTaskNotFound
	}
	task.Context = clone(task.Context)
	return task, nil
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

func (r *MemoryRepository) Transition(_ context.Context, id string, input TransitionInput) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	task, ok := r.tasks[id]
	if !ok || task.TenantID != input.TenantID {
		return Task{}, ErrTaskNotFound
	}
	if task.Version != input.ExpectedVersion {
		return Task{}, ErrVersionConflict
	}
	now := r.now().UTC()
	task.Status = input.Status
	if input.Status == StatusInProgress && task.ClaimedAt == nil {
		task.ClaimedAt = &now
	}
	if input.Status == StatusCompleted {
		task.CompletedAt = &now
	}
	task.Version++
	task.UpdatedAt = now
	r.tasks[id] = task
	task.Context = clone(task.Context)
	return task, nil
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
	return []Task{
		{ID: "task_review_cbn", TenantID: "bank-demo", WorkflowID: "wf_cbn_change", StepKey: "applicability-review", Responsibility: "REVIEWER", PrincipalID: "team-control-assurance", Title: "Review seven proposed obligations", Status: StatusReady, DueAt: now.Add(3 * 24 * time.Hour), Context: map[string]string{"program": "CBN Digital Channels", "scope": "Bank NG"}, Version: 1, CreatedAt: now, UpdatedAt: now},
		{ID: "task_access_evidence", TenantID: "bank-demo", WorkflowID: "wf_access_review", StepKey: "focused-evidence", Responsibility: "PERFORMER", PrincipalID: "queue-risk-owners", Title: "Confirm four account owners", Status: StatusInProgress, DueAt: now.Add(36 * time.Hour), Context: map[string]string{"population": "4 of 1,250", "scope": "Treasury Operations"}, Version: 1, CreatedAt: now, UpdatedAt: now},
	}
}
