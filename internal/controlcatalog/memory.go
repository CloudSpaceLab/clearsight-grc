package controlcatalog

import (
	"context"
	"strings"
	"sync"
)

type MemoryRepository struct {
	mu          sync.RWMutex
	definitions map[string]Definition
	codes       map[string]string
	links       map[string]ImplementationLink
	byImpl      map[string]string
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		definitions: map[string]Definition{},
		codes: map[string]string{},
		links: map[string]ImplementationLink{},
		byImpl: map[string]string{},
	}
}

func (r *MemoryRepository) CreateWithImplementationLink(ctx context.Context, definition Definition, link ImplementationLink) (Definition, ImplementationLink, error) {
	if err := ctx.Err(); err != nil {
		return Definition{}, ImplementationLink{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	codeKey := definition.TenantID + "\x00" + strings.ToUpper(definition.Code)
	if _, ok := r.codes[codeKey]; ok {
		return Definition{}, ImplementationLink{}, ErrDuplicate
	}
	if _, ok := r.definitions[definitionKey(definition.TenantID, definition.ID)]; ok {
		return Definition{}, ImplementationLink{}, ErrDuplicate
	}
	implKey := implementationKey(link.TenantID, link.LegalEntityID, link.ImplementationID)
	if _, ok := r.byImpl[implKey]; ok {
		return Definition{}, ImplementationLink{}, ErrDuplicate
	}
	r.definitions[definitionKey(definition.TenantID, definition.ID)] = definition
	r.codes[codeKey] = definition.ID
	r.links[linkKey(link.TenantID, link.LegalEntityID, link.ID)] = link
	r.byImpl[implKey] = link.ID
	return definition, link, nil
}

func (r *MemoryRepository) GetDefinition(ctx context.Context, tenant, id string) (Definition, error) {
	if err := ctx.Err(); err != nil {
		return Definition{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.definitions[definitionKey(strings.TrimSpace(tenant), strings.TrimSpace(id))]
	if !ok {
		return Definition{}, ErrNotFound
	}
	return value, nil
}

func (r *MemoryRepository) LinkImplementation(ctx context.Context, link ImplementationLink) (ImplementationLink, error) {
	if err := ctx.Err(); err != nil {
		return ImplementationLink{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.definitions[definitionKey(link.TenantID, link.DefinitionID)]; !ok {
		return ImplementationLink{}, ErrNotFound
	}
	implKey := implementationKey(link.TenantID, link.LegalEntityID, link.ImplementationID)
	if _, ok := r.byImpl[implKey]; ok {
		return ImplementationLink{}, ErrDuplicate
	}
	r.links[linkKey(link.TenantID, link.LegalEntityID, link.ID)] = link
	r.byImpl[implKey] = link.ID
	return link, nil
}

func (r *MemoryRepository) GetImplementationLink(ctx context.Context, tenant, entity, id string) (ImplementationLink, error) {
	if err := ctx.Err(); err != nil {
		return ImplementationLink{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.links[linkKey(strings.TrimSpace(tenant), strings.TrimSpace(entity), strings.TrimSpace(id))]
	if !ok {
		return ImplementationLink{}, ErrNotFound
	}
	return value, nil
}

func (r *MemoryRepository) ListImplementationLinks(ctx context.Context, tenant, definitionID string, limit int) ([]ImplementationLink, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := make([]ImplementationLink, 0)
	for _, value := range r.links {
		if value.TenantID == tenant && value.DefinitionID == definitionID {
			values = append(values, value)
		}
	}
	if len(values) > limit {
		values = values[:limit]
	}
	return values, nil
}

func definitionKey(tenant, id string) string { return tenant + "\x00" + id }
func linkKey(tenant, entity, id string) string { return tenant + "\x00" + entity + "\x00" + id }
func implementationKey(tenant, entity, id string) string { return tenant + "\x00" + entity + "\x00" + id }
