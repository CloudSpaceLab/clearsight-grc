package people

import (
	"context"
	"sort"
	"sync"
	"time"
)

type MemoryRepository struct {
	mu          sync.RWMutex
	profiles    map[string]Profile
	work        map[string][]WorkItem
	assignments map[string][]AssignmentItem
	activity    map[string][]ActivityItem
}

func NewMemoryRepository(profiles ...Profile) *MemoryRepository {
	values := make(map[string]Profile, len(profiles))
	for _, profile := range profiles {
		values[profile.Person.ID] = profile
	}
	return &MemoryRepository{profiles: values, work: map[string][]WorkItem{}, assignments: map[string][]AssignmentItem{}, activity: map[string][]ActivityItem{}}
}

func (r *MemoryRepository) PutWork(personID string, values ...WorkItem) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.work[personID] = append([]WorkItem(nil), values...)
}

func (r *MemoryRepository) PutAssignments(personID string, values ...AssignmentItem) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.assignments[personID] = append([]AssignmentItem(nil), values...)
}

func (r *MemoryRepository) PutActivity(personID string, values ...ActivityItem) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.activity[personID] = append([]ActivityItem(nil), values...)
}

func (r *MemoryRepository) Profile(_ context.Context, scope Scope) (Profile, error) {
	if r == nil {
		return Profile{}, ErrNotFound
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.profiles[scope.PersonID]
	if !ok {
		return Profile{}, ErrNotFound
	}
	return value, nil
}

func (r *MemoryRepository) Work(_ context.Context, query PageQuery) (WorkPage, error) {
	r.mu.RLock()
	values := append([]WorkItem(nil), r.work[query.Scope.PersonID]...)
	r.mu.RUnlock()
	if query.State == "ACTIVE" {
		values = filterWork(values, true)
	}
	if query.State == "COMPLETED" {
		values = filterWork(values, false)
	}
	sort.SliceStable(values, func(i, j int) bool { return values[i].UpdatedAt.After(values[j].UpdatedAt) })
	return WorkPage{Items: pageWork(values, query), AsOf: time.Now().UTC()}, nil
}

func (r *MemoryRepository) Assignments(_ context.Context, query PageQuery) (AssignmentPage, error) {
	r.mu.RLock()
	values := append([]AssignmentItem(nil), r.assignments[query.Scope.PersonID]...)
	r.mu.RUnlock()
	sort.SliceStable(values, func(i, j int) bool { return values[i].OccurredAt.After(values[j].OccurredAt) })
	return AssignmentPage{Items: pageAssignments(values, query), AsOf: time.Now().UTC()}, nil
}

func (r *MemoryRepository) Activity(_ context.Context, query PageQuery) (ActivityPage, error) {
	r.mu.RLock()
	values := append([]ActivityItem(nil), r.activity[query.Scope.PersonID]...)
	r.mu.RUnlock()
	sort.SliceStable(values, func(i, j int) bool { return values[i].OccurredAt.After(values[j].OccurredAt) })
	return ActivityPage{Items: pageActivity(values, query), AsOf: time.Now().UTC()}, nil
}

func filterWork(values []WorkItem, active bool) []WorkItem {
	result := make([]WorkItem, 0, len(values))
	for _, value := range values {
		complete := value.Status == "IMPLEMENTED" || value.Status == "CANCELLED" || value.Status == "CLOSED"
		if complete != active {
			result = append(result, value)
		}
	}
	return result
}

func pageWork(values []WorkItem, query PageQuery) []WorkItem {
	start := 0
	if query.Cursor != "" {
		for i := range values {
			if values[i].ID == query.Cursor {
				start = i + 1
				break
			}
		}
	}
	end := minInt(start+query.Limit, len(values))
	return values[start:end]
}
func pageAssignments(values []AssignmentItem, query PageQuery) []AssignmentItem {
	start := 0
	if query.Cursor != "" {
		for i := range values {
			if values[i].ID == query.Cursor {
				start = i + 1
				break
			}
		}
	}
	end := minInt(start+query.Limit, len(values))
	return values[start:end]
}
func pageActivity(values []ActivityItem, query PageQuery) []ActivityItem {
	start := 0
	if query.Cursor != "" {
		for i := range values {
			if values[i].ID == query.Cursor {
				start = i + 1
				break
			}
		}
	}
	end := minInt(start+query.Limit, len(values))
	return values[start:end]
}
func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
