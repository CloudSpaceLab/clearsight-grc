package sourceaccess

import (
	"context"
	"sort"
	"strconv"
	"sync"
)

type MemoryCatalogRepository struct {
	mu          sync.RWMutex
	sources     map[string]struct{}
	connections map[string]ConnectionRevision
	views       map[string]ViewRevision
	bindings    map[string]BindingRevision
}

func NewMemoryCatalogRepository(sources []SourceScope) *MemoryCatalogRepository {
	repository := &MemoryCatalogRepository{
		sources:     make(map[string]struct{}, len(sources)),
		connections: map[string]ConnectionRevision{},
		views:       map[string]ViewRevision{},
		bindings:    map[string]BindingRevision{},
	}
	for _, source := range sources {
		repository.sources[catalogScopeKey(source.TenantID, source.SourceID)] = struct{}{}
	}
	return repository
}

func (r *MemoryCatalogRepository) CreateConnectionRevision(_ context.Context, value ConnectionRevision) (ConnectionRevision, error) {
	value, err := normalizeConnectionRevision(value)
	if err != nil {
		return ConnectionRevision{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.sources[catalogScopeKey(value.TenantID, value.SourceID)]; !exists {
		return ConnectionRevision{}, ErrCatalogNotFound
	}
	if r.connectionScopeConflict(value) {
		return ConnectionRevision{}, ErrCatalogInvalid
	}
	key := catalogRevisionKey(value.TenantID, value.ConnectionID, value.Version)
	if _, exists := r.connections[key]; exists || r.connectionConflict(value) {
		return ConnectionRevision{}, ErrCatalogConflict
	}
	r.connections[key] = cloneConnectionRevision(value)
	return cloneConnectionRevision(value), nil
}

func (r *MemoryCatalogRepository) ConnectionRevision(_ context.Context, tenantID, connectionID string, version int64) (ConnectionRevision, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, exists := r.connections[catalogRevisionKey(tenantID, connectionID, version)]
	if !exists {
		return ConnectionRevision{}, ErrCatalogNotFound
	}
	return cloneConnectionRevision(value), nil
}

func (r *MemoryCatalogRepository) CurrentConnection(_ context.Context, tenantID, connectionID string) (ConnectionRevision, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, value := range r.connections {
		if value.TenantID == tenantID && value.ConnectionID == connectionID && value.IsCurrent {
			return cloneConnectionRevision(value), nil
		}
	}
	return ConnectionRevision{}, ErrCatalogNotFound
}

func (r *MemoryCatalogRepository) ListCurrentConnections(_ context.Context, tenantID, sourceID string, limit int) ([]ConnectionRevision, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := make([]ConnectionRevision, 0)
	for _, value := range r.connections {
		if value.TenantID == tenantID && value.SourceID == sourceID && value.IsCurrent {
			values = append(values, cloneConnectionRevision(value))
		}
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].Code == values[j].Code {
			return values[i].ConnectionID < values[j].ConnectionID
		}
		return values[i].Code < values[j].Code
	})
	return limitConnections(values, catalogListLimit(limit)), nil
}

func (r *MemoryCatalogRepository) ListConnectionRevisions(_ context.Context, tenantID, sourceID string, limit int) ([]ConnectionRevision, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := make([]ConnectionRevision, 0)
	for _, value := range r.connections {
		if value.TenantID == tenantID && value.SourceID == sourceID {
			values = append(values, cloneConnectionRevision(value))
		}
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].Code != values[j].Code {
			return values[i].Code < values[j].Code
		}
		if values[i].ConnectionID != values[j].ConnectionID {
			return values[i].ConnectionID < values[j].ConnectionID
		}
		return values[i].Version > values[j].Version
	})
	return limitConnections(values, catalogListLimit(limit)), nil
}

func (r *MemoryCatalogRepository) CreateViewRevision(_ context.Context, value ViewRevision) (ViewRevision, error) {
	value, err := normalizeViewRevision(value)
	if err != nil {
		return ViewRevision{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	parent, exists := r.connections[catalogRevisionKey(value.TenantID, value.ConnectionID, value.ConnectionVersion)]
	if !exists || parent.SourceID != value.SourceID || (value.IsCurrent && !parent.IsCurrent) {
		return ViewRevision{}, ErrCatalogNotFound
	}
	if parent.AdapterKind == AdapterReference {
		return ViewRevision{}, ErrCatalogInvalid
	}
	value, err = validateViewAgainstConnection(value, parent)
	if err != nil {
		return ViewRevision{}, err
	}
	if r.viewScopeConflict(value) {
		return ViewRevision{}, ErrCatalogInvalid
	}
	key := catalogRevisionKey(value.TenantID, value.ViewID, value.Version)
	if _, exists := r.views[key]; exists || r.viewConflict(value) {
		return ViewRevision{}, ErrCatalogConflict
	}
	r.views[key] = cloneViewRevision(value)
	return cloneViewRevision(value), nil
}

func (r *MemoryCatalogRepository) ViewRevision(_ context.Context, tenantID, viewID string, version int64) (ViewRevision, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, exists := r.views[catalogRevisionKey(tenantID, viewID, version)]
	if !exists {
		return ViewRevision{}, ErrCatalogNotFound
	}
	return cloneViewRevision(value), nil
}

func (r *MemoryCatalogRepository) CurrentView(_ context.Context, tenantID, viewID string) (ViewRevision, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, value := range r.views {
		if value.TenantID == tenantID && value.ViewID == viewID && value.IsCurrent {
			return cloneViewRevision(value), nil
		}
	}
	return ViewRevision{}, ErrCatalogNotFound
}

func (r *MemoryCatalogRepository) ListCurrentViews(_ context.Context, tenantID, connectionID string, limit int) ([]ViewRevision, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := make([]ViewRevision, 0)
	for _, value := range r.views {
		if value.TenantID == tenantID && value.ConnectionID == connectionID && value.IsCurrent {
			values = append(values, cloneViewRevision(value))
		}
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].Code == values[j].Code {
			return values[i].ViewID < values[j].ViewID
		}
		return values[i].Code < values[j].Code
	})
	return limitViews(values, catalogListLimit(limit)), nil
}

func (r *MemoryCatalogRepository) ListViewRevisions(_ context.Context, tenantID, connectionID string, limit int) ([]ViewRevision, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := make([]ViewRevision, 0)
	for _, value := range r.views {
		if value.TenantID == tenantID && value.ConnectionID == connectionID {
			values = append(values, cloneViewRevision(value))
		}
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].Code != values[j].Code {
			return values[i].Code < values[j].Code
		}
		if values[i].ViewID != values[j].ViewID {
			return values[i].ViewID < values[j].ViewID
		}
		return values[i].Version > values[j].Version
	})
	return limitViews(values, catalogListLimit(limit)), nil
}

func (r *MemoryCatalogRepository) CreateBindingRevision(_ context.Context, value BindingRevision) (BindingRevision, error) {
	value, err := normalizeBindingRevision(value)
	if err != nil {
		return BindingRevision{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	parent, exists := r.views[catalogRevisionKey(value.TenantID, value.ViewID, value.ViewVersion)]
	if !exists || parent.SourceID != value.SourceID || (value.IsCurrent && !parent.IsCurrent) {
		return BindingRevision{}, ErrCatalogNotFound
	}
	value, err = validateBindingAgainstView(value, parent)
	if err != nil {
		return BindingRevision{}, err
	}
	if r.bindingScopeConflict(value) {
		return BindingRevision{}, ErrCatalogInvalid
	}
	key := catalogRevisionKey(value.TenantID, value.BindingID, value.Version)
	if _, exists := r.bindings[key]; exists || r.bindingConflict(value) {
		return BindingRevision{}, ErrCatalogConflict
	}
	r.bindings[key] = cloneBindingRevision(value)
	return cloneBindingRevision(value), nil
}

func (r *MemoryCatalogRepository) BindingRevision(_ context.Context, tenantID, bindingID string, version int64) (BindingRevision, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, exists := r.bindings[catalogRevisionKey(tenantID, bindingID, version)]
	if !exists {
		return BindingRevision{}, ErrCatalogNotFound
	}
	return cloneBindingRevision(value), nil
}

func (r *MemoryCatalogRepository) CurrentBinding(_ context.Context, tenantID, bindingID string) (BindingRevision, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, value := range r.bindings {
		if value.TenantID == tenantID && value.BindingID == bindingID && value.IsCurrent {
			return cloneBindingRevision(value), nil
		}
	}
	return BindingRevision{}, ErrCatalogNotFound
}

func (r *MemoryCatalogRepository) ListCurrentBindings(_ context.Context, tenantID, viewID string, limit int) ([]BindingRevision, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := make([]BindingRevision, 0)
	for _, value := range r.bindings {
		if value.TenantID == tenantID && value.ViewID == viewID && value.IsCurrent {
			values = append(values, cloneBindingRevision(value))
		}
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].Code == values[j].Code {
			return values[i].BindingID < values[j].BindingID
		}
		return values[i].Code < values[j].Code
	})
	return limitBindings(values, catalogListLimit(limit)), nil
}

func (r *MemoryCatalogRepository) ListBindingRevisions(_ context.Context, tenantID, viewID string, limit int) ([]BindingRevision, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := make([]BindingRevision, 0)
	for _, value := range r.bindings {
		if value.TenantID == tenantID && value.ViewID == viewID {
			values = append(values, cloneBindingRevision(value))
		}
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].Code != values[j].Code {
			return values[i].Code < values[j].Code
		}
		if values[i].BindingID != values[j].BindingID {
			return values[i].BindingID < values[j].BindingID
		}
		return values[i].Version > values[j].Version
	})
	return limitBindings(values, catalogListLimit(limit)), nil
}

func (r *MemoryCatalogRepository) connectionScopeConflict(candidate ConnectionRevision) bool {
	for _, value := range r.connections {
		if value.ConnectionID == candidate.ConnectionID && (value.TenantID != candidate.TenantID || value.SourceID != candidate.SourceID) {
			return true
		}
	}
	return false
}

func (r *MemoryCatalogRepository) connectionConflict(candidate ConnectionRevision) bool {
	for _, value := range r.connections {
		if value.ConnectionID == candidate.ConnectionID && (value.Version == candidate.Version || (candidate.IsCurrent && value.IsCurrent)) {
			return true
		}
		if candidate.IsCurrent && value.IsCurrent && value.TenantID == candidate.TenantID && value.SourceID == candidate.SourceID && value.Code == candidate.Code {
			return true
		}
	}
	return false
}

func (r *MemoryCatalogRepository) viewScopeConflict(candidate ViewRevision) bool {
	for _, value := range r.views {
		if value.ViewID == candidate.ViewID && (value.TenantID != candidate.TenantID || value.SourceID != candidate.SourceID || value.ConnectionID != candidate.ConnectionID) {
			return true
		}
	}
	return false
}

func (r *MemoryCatalogRepository) viewConflict(candidate ViewRevision) bool {
	for _, value := range r.views {
		if value.ViewID == candidate.ViewID && (value.Version == candidate.Version || (candidate.IsCurrent && value.IsCurrent)) {
			return true
		}
		if candidate.IsCurrent && value.IsCurrent && value.TenantID == candidate.TenantID && value.ConnectionID == candidate.ConnectionID && value.Code == candidate.Code {
			return true
		}
	}
	return false
}

func (r *MemoryCatalogRepository) bindingScopeConflict(candidate BindingRevision) bool {
	for _, value := range r.bindings {
		if value.BindingID == candidate.BindingID && (value.TenantID != candidate.TenantID || value.SourceID != candidate.SourceID || value.ViewID != candidate.ViewID) {
			return true
		}
	}
	return false
}

func (r *MemoryCatalogRepository) bindingConflict(candidate BindingRevision) bool {
	for _, value := range r.bindings {
		if value.BindingID == candidate.BindingID && (value.Version == candidate.Version || (candidate.IsCurrent && value.IsCurrent)) {
			return true
		}
		if candidate.IsCurrent && value.IsCurrent && value.TenantID == candidate.TenantID && value.ViewID == candidate.ViewID && value.Code == candidate.Code {
			return true
		}
	}
	return false
}

func catalogScopeKey(tenantID, resourceID string) string {
	return tenantID + "\x1f" + resourceID
}

func catalogRevisionKey(tenantID, resourceID string, version int64) string {
	return catalogScopeKey(tenantID, resourceID) + "\x1f" + strconv.FormatInt(version, 10)
}

func cloneConnectionRevision(value ConnectionRevision) ConnectionRevision {
	value.Definition = cloneRawMessage(value.Definition)
	value.DeclaredCapabilities = append([]Capability(nil), value.DeclaredCapabilities...)
	value.VerifiedCapabilities = append([]Capability(nil), value.VerifiedCapabilities...)
	value.EffectiveFrom = cloneTime(value.EffectiveFrom)
	value.EffectiveUntil = cloneTime(value.EffectiveUntil)
	return value
}

func cloneViewRevision(value ViewRevision) ViewRevision {
	value.Definition = cloneRawMessage(value.Definition)
	value.StableKeys = append([]string(nil), value.StableKeys...)
	value.NativeSchema = append([]NativeField(nil), value.NativeSchema...)
	value.EffectiveFrom = cloneTime(value.EffectiveFrom)
	value.EffectiveUntil = cloneTime(value.EffectiveUntil)
	return value
}

func cloneBindingRevision(value BindingRevision) BindingRevision {
	value.Operations = append([]Operation(nil), value.Operations...)
	value.SelectedFields = append([]string(nil), value.SelectedFields...)
	value.KeyFields = append([]string(nil), value.KeyFields...)
	value.Mapping = cloneRawMessage(value.Mapping)
	value.ParameterSchema = cloneRawMessage(value.ParameterSchema)
	value.OutputSchema = cloneRawMessage(value.OutputSchema)
	value.SensitivityHandling = cloneRawMessage(value.SensitivityHandling)
	value.EffectiveFrom = cloneTime(value.EffectiveFrom)
	value.EffectiveUntil = cloneTime(value.EffectiveUntil)
	return value
}

func limitConnections(values []ConnectionRevision, limit int) []ConnectionRevision {
	if len(values) > limit {
		return values[:limit]
	}
	return values
}

func limitViews(values []ViewRevision, limit int) []ViewRevision {
	if len(values) > limit {
		return values[:limit]
	}
	return values
}

func limitBindings(values []BindingRevision, limit int) []BindingRevision {
	if len(values) > limit {
		return values[:limit]
	}
	return values
}
