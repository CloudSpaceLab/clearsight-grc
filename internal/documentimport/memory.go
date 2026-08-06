package documentimport

import (
	"context"
	"sync"
)

type MemoryRepository struct {
	mu    sync.RWMutex
	items map[string]Document
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{items: map[string]Document{}}
}

func (r *MemoryRepository) Create(_ context.Context, value Document) (Document, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if value.Version == 0 {
		value.Version = 1
	}
	r.items[value.ID] = cloneDocument(value)
	return cloneDocument(value), nil
}

func (r *MemoryRepository) List(_ context.Context, tenant string, limit int) ([]Document, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := make([]Document, 0, limit)
	for _, value := range r.items {
		if value.TenantID != tenant {
			continue
		}
		values = append(values, cloneDocument(value))
		if len(values) == limit {
			break
		}
	}
	return values, nil
}

func (r *MemoryRepository) Get(_ context.Context, tenant, id string) (Document, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.items[id]
	if !ok || value.TenantID != tenant {
		return Document{}, ErrNotFound
	}
	return cloneDocument(value), nil
}

func (r *MemoryRepository) SaveReview(_ context.Context, value Document, expected int64) (Document, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.items[value.ID]
	if !ok || current.TenantID != value.TenantID {
		return Document{}, ErrNotFound
	}
	if current.Version != expected {
		return Document{}, ErrVersionConflict
	}
	value.Version = current.Version + 1
	r.items[value.ID] = cloneDocument(value)
	return cloneDocument(value), nil
}

func cloneDocument(value Document) Document {
	value.Limitations = append([]string(nil), value.Limitations...)
	value.Sections = append([]Section(nil), value.Sections...)
	value.Proposals = append([]Proposal(nil), value.Proposals...)
	return value
}
