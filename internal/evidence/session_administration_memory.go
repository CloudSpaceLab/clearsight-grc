package evidence

import (
	"context"
	"sort"
	"time"
)

func (r *MemoryRepository) ListActiveSessionMetadata(_ context.Context, tenant, requestID string, now time.Time, limit int) ([]ActiveSessionMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	request, ok := r.requests[requestID]
	if !ok || request.TenantID != tenant {
		return nil, ErrNotFound
	}
	if !requestOpenAt(request, now) || !externalRecipientRequest(request) {
		return []ActiveSessionMetadata{}, nil
	}
	values := make([]ActiveSessionMetadata, 0, limit)
	for _, session := range r.sessions {
		if session.TenantID != tenant || session.RequestID != requestID || session.RevokedAt != nil || !now.Before(session.ExpiresAt) {
			continue
		}
		values = append(values, ActiveSessionMetadata{
			ID: session.ID, AudienceHint: session.AudienceHint, ExpiresAt: session.ExpiresAt, CreatedAt: session.CreatedAt,
		})
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].CreatedAt.Equal(values[j].CreatedAt) {
			return values[i].ID > values[j].ID
		}
		return values[i].CreatedAt.After(values[j].CreatedAt)
	})
	if len(values) > limit {
		values = values[:limit]
	}
	return values, nil
}

var _ activeSessionAdministrationStore = (*MemoryRepository)(nil)
