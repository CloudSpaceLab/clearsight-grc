package aigovernance

import (
	"context"
	"sort"
	"strings"
)

func (r *MemoryRepository) CreateGatewayTransport(_ context.Context, value GatewayTransportRevision) (GatewayTransportRevision, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := memKey(value.TenantID, value.ID)
	if _, exists := r.gatewayTransports[key]; exists {
		return GatewayTransportRevision{}, ErrConflict
	}
	r.gatewayTransports[key] = value
	return value, nil
}

func (r *MemoryRepository) NextGatewayTransportVersion(_ context.Context, tenantID, environment string) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var maximum int64
	for _, value := range r.gatewayTransports {
		if value.TenantID == tenantID && value.Environment == environment && value.Version > maximum {
			maximum = value.Version
		}
	}
	return maximum + 1, nil
}

func (r *MemoryRepository) GatewayTransport(_ context.Context, tenantID, id string) (GatewayTransportRevision, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, exists := r.gatewayTransports[memKey(tenantID, id)]
	if !exists {
		return GatewayTransportRevision{}, ErrNotFound
	}
	return value, nil
}

func (r *MemoryRepository) ListGatewayTransports(_ context.Context, tenantID, environment string, limit int) ([]GatewayTransportRevision, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]GatewayTransportRevision, 0)
	for _, value := range r.gatewayTransports {
		if value.TenantID == tenantID && (environment == "" || strings.EqualFold(value.Environment, environment)) {
			items = append(items, value)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Environment != items[j].Environment {
			return items[i].Environment < items[j].Environment
		}
		return items[i].Version > items[j].Version
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (r *MemoryRepository) ActiveGatewayTransport(_ context.Context, tenantID, environment string) (GatewayTransportRevision, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, value := range r.gatewayTransports {
		if value.TenantID == tenantID && value.Environment == environment && value.Status == GatewayTransportActive {
			return value, nil
		}
	}
	return GatewayTransportRevision{}, ErrNotFound
}

func (r *MemoryRepository) UpdateGatewayTransport(_ context.Context, value GatewayTransportRevision, expected int64) (GatewayTransportRevision, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := memKey(value.TenantID, value.ID)
	current, exists := r.gatewayTransports[key]
	if !exists {
		return GatewayTransportRevision{}, ErrNotFound
	}
	if current.RecordVersion != expected {
		return GatewayTransportRevision{}, ErrConflict
	}
	r.gatewayTransports[key] = value
	return value, nil
}

func (r *MemoryRepository) ActivateGatewayTransport(_ context.Context, value GatewayTransportRevision, expected int64) (GatewayTransportRevision, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := memKey(value.TenantID, value.ID)
	current, exists := r.gatewayTransports[key]
	if !exists {
		return GatewayTransportRevision{}, ErrNotFound
	}
	if current.RecordVersion != expected {
		return GatewayTransportRevision{}, ErrConflict
	}
	for otherKey, other := range r.gatewayTransports {
		if otherKey == key || other.TenantID != value.TenantID || other.Environment != value.Environment || other.Status != GatewayTransportActive {
			continue
		}
		other.Status = GatewayTransportSuspended
		other.SuspendedAt = value.ActivatedAt
		other.UpdatedAt = value.UpdatedAt
		other.RecordVersion++
		r.gatewayTransports[otherKey] = other
	}
	r.gatewayTransports[key] = value
	return value, nil
}
