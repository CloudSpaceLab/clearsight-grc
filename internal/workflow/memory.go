package workflow

import (
	"context"
	"sort"
	"sync"

	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

type MemoryRepository struct {
	mu    sync.RWMutex
	tasks map[string]Task
}

func NewMemoryRepository(seed []Task) *MemoryRepository {
	tasks := make(map[string]Task, len(seed))
	for _, task := range seed {
		tasks[task.ID] = cloneTask(task)
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
		if filter.LegalEntityID != "" && task.LegalEntityID != filter.LegalEntityID {
			continue
		}
		if filter.PrincipalID != "" && task.PrincipalID != filter.PrincipalID {
			continue
		}
		if filter.Status != "" && task.Status != filter.Status {
			continue
		}
		if filter.WorkflowKind != "" && task.WorkflowKind != filter.WorkflowKind {
			continue
		}
		if filter.ActiveOnly && (task.Status == StatusCompleted || task.Status == StatusCancelled) {
			continue
		}
		if filter.VisibleMatterWorkOnly && !MatterWorkVisibleTo(task, filter.PrincipalID) {
			continue
		}
		if filter.VisibleActorWorkOnly && !ActorWorkVisibleTo(task, filter.PrincipalID) {
			continue
		}
		values = append(values, cloneTask(task))
	}
	sort.Slice(values, func(i, j int) bool {
		if filter.ActiveOnly {
			leftDue, rightDue := values[i].DueAt, values[j].DueAt
			if leftDue == nil && rightDue != nil {
				return false
			}
			if leftDue != nil && rightDue == nil {
				return true
			}
			if leftDue != nil && rightDue != nil && !leftDue.Equal(*rightDue) {
				return leftDue.Before(*rightDue)
			}
		}
		if !values[i].UpdatedAt.Equal(values[j].UpdatedAt) {
			return values[i].UpdatedAt.After(values[j].UpdatedAt)
		}
		return values[i].ID < values[j].ID
	})
	if len(values) > filter.Limit {
		values = values[:filter.Limit]
	}
	return values, nil
}

func cloneTask(task Task) Task {
	task.Context = clone(task.Context)
	task.SourceBindings = append([]sourceaccess.BindingReference(nil), task.SourceBindings...)
	task.MatterScope = append([]byte(nil), task.MatterScope...)
	return task
}

func clone(input map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range input {
		out[key] = value
	}
	return out
}
