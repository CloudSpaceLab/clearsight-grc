package workflow

import (
	"context"
	"sort"
	"sync"
	"time"
)

type MemoryActorNotificationRepository struct {
	mu    sync.RWMutex
	items []memoryActorNotification
}

type memoryActorNotification struct {
	scope ActorNotificationScope
	value ActorNotification
}

func NewMemoryActorNotificationRepository(items ...struct {
	Scope ActorNotificationScope
	Value ActorNotification
}) *MemoryActorNotificationRepository {
	repo := &MemoryActorNotificationRepository{}
	for _, item := range items {
		repo.items = append(repo.items, memoryActorNotification{scope: item.Scope, value: item.Value})
	}
	return repo
}

func (r *MemoryActorNotificationRepository) ListActorNotifications(_ context.Context, scope ActorNotificationScope, filter ActorNotificationFilter, cursor *actorNotificationCursor) (ActorNotificationPage, error) {
	if r == nil || !validActorNotificationScope(scope) {
		return ActorNotificationPage{}, ErrNotificationInvalid
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := make([]ActorNotification, 0, len(r.items))
	unread := 0
	for _, item := range r.items {
		if item.scope != scope {
			continue
		}
		if item.value.ReadAt == nil {
			unread++
		}
		if cursor != nil && !notificationBefore(item.value, *cursor) {
			continue
		}
		values = append(values, item.value)
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].OccurredAt.Equal(values[j].OccurredAt) {
			return values[i].ID > values[j].ID
		}
		return values[i].OccurredAt.After(values[j].OccurredAt)
	})
	limit := filter.Limit + 1
	if limit < 2 {
		limit = 21
	}
	if len(values) > limit {
		values = values[:limit]
	}
	return ActorNotificationPage{Items: values, Unread: unread}, nil
}

func (r *MemoryActorNotificationRepository) MarkActorNotificationRead(_ context.Context, scope ActorNotificationScope, id string, at time.Time) (ActorNotification, error) {
	if r == nil || !validActorNotificationScope(scope) || id == "" {
		return ActorNotification{}, ErrNotificationInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for index := range r.items {
		item := &r.items[index]
		if item.scope != scope || item.value.ID != id {
			continue
		}
		if item.value.ReadAt == nil {
			readAt := at.UTC()
			item.value.ReadAt = &readAt
		}
		return item.value, nil
	}
	return ActorNotification{}, ErrNotificationNotFound
}

func notificationBefore(value ActorNotification, cursor actorNotificationCursor) bool {
	if value.OccurredAt.Before(cursor.OccurredAt) {
		return true
	}
	return value.OccurredAt.Equal(cursor.OccurredAt) && value.ID < cursor.ID
}
