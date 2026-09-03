package activity

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
)

type MemoryRepository struct {
	mu     sync.RWMutex
	events []Event
}

func NewMemoryRepository(events ...Event) *MemoryRepository {
	copyEvents := append([]Event(nil), events...)
	return &MemoryRepository{events: copyEvents}
}

func (r *MemoryRepository) Put(event Event) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
}

func (r *MemoryRepository) List(_ context.Context, query Query) (Page, error) {
	if r == nil {
		return Page{}, ErrInvalid
	}
	r.mu.RLock()
	values := append([]Event(nil), r.events...)
	r.mu.RUnlock()

	sort.SliceStable(values, func(i, j int) bool {
		if values[i].OccurredAt.Equal(values[j].OccurredAt) {
			return values[i].ID > values[j].ID
		}
		return values[i].OccurredAt.After(values[j].OccurredAt)
	})

	items := make([]Event, 0, query.Limit+1)
	pastCursor := query.Cursor == ""
	for _, value := range values {
		if value.ID == query.Cursor {
			pastCursor = true
			continue
		}
		if !pastCursor || !memoryMatches(value, query) {
			continue
		}
		items = append(items, value)
		if len(items) > query.Limit {
			break
		}
	}
	page := Page{Items: items, AsOf: time.Now().UTC()}
	if len(items) > query.Limit {
		page.Items = items[:query.Limit]
		page.NextCursor = page.Items[len(page.Items)-1].ID
	}
	return page, nil
}

func (r *MemoryRepository) Get(_ context.Context, tenantID, eventID string) (Event, error) {
	if r == nil {
		return Event{}, ErrInvalid
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, value := range r.events {
		if value.ID == eventID && value.SourceTenantID() == tenantID {
			return value, nil
		}
	}
	return Event{}, ErrNotFound
}

func memoryMatches(value Event, query Query) bool {
	if tenant := value.SourceTenantID(); tenant != "" && tenant != query.TenantID {
		return false
	}
	if query.From != nil && value.OccurredAt.Before(*query.From) {
		return false
	}
	if query.To != nil && value.OccurredAt.After(*query.To) {
		return false
	}
	if query.Category != "" && categoryFor(value.ObjectType) != query.Category {
		return false
	}
	if query.EventType != "" && !strings.EqualFold(value.EventType, query.EventType) {
		return false
	}
	if query.ObjectType != "" && !strings.EqualFold(value.ObjectType, query.ObjectType) {
		return false
	}
	if query.ObjectID != "" && value.ObjectID != query.ObjectID {
		return false
	}
	if query.ActorID != "" && value.ActorID != query.ActorID {
		return false
	}
	return query.LegalEntityID == "" || value.LegalEntityID == query.LegalEntityID
}

// SourceTenantID is intentionally not serialized. Memory fixtures may encode a
// tenant prefix in Source as TENANT:<id>; production events obtain tenant scope
// from their repository query and never expose it as event metadata.
func (e Event) SourceTenantID() string {
	const prefix = "TENANT:"
	if strings.HasPrefix(e.Source, prefix) {
		return strings.TrimPrefix(e.Source, prefix)
	}
	return ""
}
