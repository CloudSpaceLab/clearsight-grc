package workflow

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

type MemoryRepository struct {
	mu            sync.RWMutex
	tasks         map[string]Task
	notifications map[string]InAppNotification
}

func NewMemoryRepository(seed []Task) *MemoryRepository {
	tasks := make(map[string]Task, len(seed))
	for _, task := range seed {
		tasks[task.ID] = cloneTask(task)
	}
	return &MemoryRepository{tasks: tasks, notifications: map[string]InAppNotification{}}
}

func NewMemoryRepositoryWithNotifications(seed []Task, notifications []InAppNotification) *MemoryRepository {
	repository := NewMemoryRepository(seed)
	for _, item := range notifications {
		key := item.TenantID + "\x00" + item.OutboxEventID + "\x00" + item.PrincipalID + "\x00" + item.Kind
		repository.notifications[key] = cloneInAppNotification(item)
	}
	return repository
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


func (r *MemoryRepository) StoreInAppNotification(_ context.Context, record inAppNotificationRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.notifications == nil {
		r.notifications = map[string]InAppNotification{}
	}
	key := record.TenantID + "\x00" + record.OutboxEventID + "\x00" + record.PrincipalID + "\x00" + record.Kind
	for _, existing := range r.notifications {
		if existing.TenantID+"\x00"+existing.OutboxEventID+"\x00"+existing.PrincipalID+"\x00"+existing.Kind == key {
			return nil
		}
	}
	r.notifications[key] = InAppNotification{
		ID: record.OutboxEventID, Kind: record.Kind, Title: record.Title, Summary: record.Summary,
		SubjectType: record.SubjectType, SubjectID: record.SubjectID, ActionPath: record.ActionPath,
		OccurredAt: record.OccurredAt.UTC(), TenantID: record.TenantID, LegalEntityID: record.LegalEntityID,
		PrincipalID: record.PrincipalID, OutboxEventID: record.OutboxEventID,
	}
	return nil
}

func (r *MemoryRepository) ListInAppNotifications(_ context.Context, filter NotificationFilter, cursor *notificationCursor) (NotificationPage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]InAppNotification, 0, len(r.notifications))
	unread := 0
	for _, item := range r.notifications {
		if item.TenantID != filter.TenantID || item.LegalEntityID != filter.LegalEntityID || item.PrincipalID != filter.PrincipalID {
			continue
		}
		if item.ReadAt == nil {
			unread++
		}
		if filter.UnreadOnly && item.ReadAt != nil {
			continue
		}
		if cursor != nil && !(item.OccurredAt.Before(cursor.OccurredAt) || (item.OccurredAt.Equal(cursor.OccurredAt) && item.ID < cursor.ID)) {
			continue
		}
		items = append(items, cloneInAppNotification(item))
	}
	sort.Slice(items, func(i, j int) bool {
		if !items[i].OccurredAt.Equal(items[j].OccurredAt) {
			return items[i].OccurredAt.After(items[j].OccurredAt)
		}
		return items[i].ID > items[j].ID
	})
	page := NotificationPage{UnreadCount: unread}
	if len(items) > filter.Limit {
		page.Items = items[:filter.Limit]
		last := page.Items[len(page.Items)-1]
		page.NextCursor = encodeNotificationCursor(notificationCursor{OccurredAt: last.OccurredAt, ID: last.ID})
	} else {
		page.Items = items
	}
	return page, nil
}

func (r *MemoryRepository) MarkInAppNotificationRead(_ context.Context, filter NotificationFilter, notificationID string, at time.Time) (InAppNotification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, item := range r.notifications {
		if item.ID != notificationID || item.TenantID != filter.TenantID || item.LegalEntityID != filter.LegalEntityID || item.PrincipalID != filter.PrincipalID {
			continue
		}
		if item.ReadAt == nil {
			value := at.UTC()
			item.ReadAt = &value
			r.notifications[key] = item
		}
		return cloneInAppNotification(item), nil
	}
	return InAppNotification{}, ErrNotificationNotFound
}

func cloneInAppNotification(value InAppNotification) InAppNotification {
	if value.ReadAt != nil {
		readAt := *value.ReadAt
		value.ReadAt = &readAt
	}
	return value
}
