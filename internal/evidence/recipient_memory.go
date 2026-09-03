package evidence

import (
	"context"
	"sort"
	"strings"
)

func (r *MemoryRepository) InternalRecipientEligible(_ context.Context, tenant, principalID string) (bool, error) {
	return strings.TrimSpace(tenant) != "" && strings.TrimSpace(principalID) != "", nil
}

func (r *MemoryRepository) CanReadSubject(_ context.Context, tenant, principalID, subjectType, subjectID string) (bool, error) {
	return strings.TrimSpace(tenant) != "" && strings.TrimSpace(principalID) != "" && strings.TrimSpace(subjectType) != "" && strings.TrimSpace(subjectID) != "", nil
}

func (r *MemoryRepository) CreateRequestWithRecipient(_ context.Context, value Request) (Request, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.createRequestLocked(value)
}

func (r *MemoryRepository) GetRequestRecipient(_ context.Context, tenant, requestID string) (Recipient, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.requests[requestID]
	if !ok || value.TenantID != tenant {
		return Recipient{}, ErrNotFound
	}
	return cloneRecipient(value.Recipient), nil
}

func (r *MemoryRepository) ListRecipientRequests(_ context.Context, tenant, principalID string, limit int) ([]Request, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := make([]Request, 0, limit)
	for _, value := range r.requests {
		if value.TenantID != tenant || !RequestAssignedTo(value, principalID) {
			continue
		}
		values = append(values, cloneRequest(value))
	}
	sortRequests(values)
	if len(values) > limit {
		values = values[:limit]
	}
	return values, nil
}

func (r *MemoryRepository) ListManageableRequests(_ context.Context, tenant, principalID string, limit int) ([]Request, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := make([]Request, 0, limit)
	for _, value := range r.requests {
		if value.TenantID != tenant || !RequestManageableBy(value, principalID) {
			continue
		}
		values = append(values, cloneRequest(value))
	}
	sortRequests(values)
	if len(values) > limit {
		values = values[:limit]
	}
	return values, nil
}

func sortRecipientRequests(values []Request) {
	sort.Slice(values, func(i, j int) bool { return values[i].Deadline.Before(values[j].Deadline) })
}

var _ recipientStore = (*MemoryRepository)(nil)
var _ internalRecipientDirectory = (*MemoryRepository)(nil)
var _ SubjectAccessChecker = (*MemoryRepository)(nil)
var _ manageableRequestRepository = (*MemoryRepository)(nil)
