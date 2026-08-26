package thirdparty

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
)

type MemoryRelationshipLinkRepository struct {
	mu            sync.RWMutex
	links         map[string]RelationshipLink
	relationships map[string]bool
	targets       map[string]bool
}

func NewMemoryRelationshipLinkRepository() *MemoryRelationshipLinkRepository {
	return &MemoryRelationshipLinkRepository{links: map[string]RelationshipLink{}, relationships: map[string]bool{}, targets: map[string]bool{}}
}

func linkScopeKey(tenant, entity, id string) string { return tenant + "\x00" + entity + "\x00" + id }
func targetScopeKey(tenant, entity string, kind LinkTargetType, id string) string {
	return linkScopeKey(tenant, entity, string(kind)+"\x00"+id)
}

func (r *MemoryRelationshipLinkRepository) AllowRelationship(tenant, entity, id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.relationships[linkScopeKey(tenant, entity, id)] = true
}
func (r *MemoryRelationshipLinkRepository) AllowTarget(tenant, entity string, kind LinkTargetType, id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.targets[targetScopeKey(tenant, entity, kind, id)] = true
}
func (r *MemoryRelationshipLinkRepository) RelationshipExists(_ context.Context, scope Scope, id string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.relationships[linkScopeKey(scope.TenantID, scope.LegalEntityID, id)], nil
}
func (r *MemoryRelationshipLinkRepository) TargetAvailable(_ context.Context, scope Scope, kind LinkTargetType, id string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.targets[targetScopeKey(scope.TenantID, scope.LegalEntityID, kind, id)], nil
}
func (r *MemoryRelationshipLinkRepository) CreateRelationshipLink(_ context.Context, value RelationshipLink) (RelationshipLink, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.links {
		if existing.TenantID == value.TenantID && existing.LegalEntityID == value.LegalEntityID && existing.RelationshipID == value.RelationshipID && existing.TargetType == value.TargetType && existing.TargetID == value.TargetID && existing.State == RelationshipLinkActive {
			return RelationshipLink{}, ErrVersionConflict
		}
	}
	r.links[value.ID] = value
	return value, nil
}
func (r *MemoryRelationshipLinkRepository) GetRelationshipLink(_ context.Context, scope Scope, id string) (RelationshipLink, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.links[id]
	if !ok || value.TenantID != scope.TenantID || value.LegalEntityID != scope.LegalEntityID {
		return RelationshipLink{}, ErrNotFound
	}
	return value, nil
}
func (r *MemoryRelationshipLinkRepository) EndRelationshipLink(_ context.Context, scope Scope, id string, expected int64, actor, reason string, now time.Time) (RelationshipLink, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.links[id]
	if !ok || value.TenantID != scope.TenantID || value.LegalEntityID != scope.LegalEntityID {
		return RelationshipLink{}, ErrNotFound
	}
	if value.Version != expected {
		return RelationshipLink{}, ErrVersionConflict
	}
	if value.State != RelationshipLinkActive {
		return RelationshipLink{}, ErrInvalid
	}
	value.State, value.EndedBy, value.EndReason, value.EndedAt, value.UpdatedAt, value.Version = RelationshipLinkEnded, actor, reason, &now, now, value.Version+1
	r.links[id] = value
	return value, nil
}
func (r *MemoryRelationshipLinkRepository) ListRelationshipLinks(_ context.Context, scope Scope, input RelationshipLinkListInput) (RelationshipLinkPage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := []RelationshipLink{}
	for _, value := range r.links {
		if value.TenantID != scope.TenantID || value.LegalEntityID != scope.LegalEntityID || (!input.IncludeEnded && value.State == RelationshipLinkEnded) || (input.RelationshipID != "" && value.RelationshipID != input.RelationshipID) || (input.TargetType != "" && value.TargetType != input.TargetType) || (input.TargetID != "" && value.TargetID != input.TargetID) {
			continue
		}
		values = append(values, value)
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].UpdatedAt.Equal(values[j].UpdatedAt) {
			return strings.Compare(values[i].ID, values[j].ID) > 0
		}
		return values[i].UpdatedAt.After(values[j].UpdatedAt)
	})
	if len(values) > input.Limit {
		values = values[:input.Limit]
	}
	return RelationshipLinkPage{Items: values}, nil
}

var _ RelationshipLinkRepository = (*MemoryRelationshipLinkRepository)(nil)
