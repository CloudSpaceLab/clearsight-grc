package documentimport

import "context"

func (r *MemoryRepository) ListScoped(_ context.Context, tenant, legalEntityID string, limit int) ([]DocumentSummary, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := make([]DocumentSummary, 0, limit)
	for _, value := range r.items {
		if value.TenantID != tenant || value.LegalEntityID != legalEntityID {
			continue
		}
		values = append(values, summarizeDocument(value))
		if len(values) == limit {
			break
		}
	}
	return values, nil
}

func (r *MemoryRepository) GetScoped(_ context.Context, tenant, legalEntityID, id string) (Document, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.items[id]
	if !ok || value.TenantID != tenant || value.LegalEntityID != legalEntityID {
		return Document{}, ErrNotFound
	}
	return cloneDocument(value), nil
}

var _ scopedReadRepository = (*MemoryRepository)(nil)
