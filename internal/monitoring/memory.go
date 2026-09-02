package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
)

type MemoryRepository struct {
	mu         sync.RWMutex
	forms      map[string]FormTemplate
	starters   map[string]StarterTemplate
	savedViews map[string]SavedFormView
	checks     map[string]MonitoringCheck
	results    map[string]MonitoringResult
	events     []MonitoringEvent
	outbox     []MonitoringEvent
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{forms: map[string]FormTemplate{}, starters: map[string]StarterTemplate{}, savedViews: map[string]SavedFormView{}, checks: map[string]MonitoringCheck{}, results: map[string]MonitoringResult{}}
}

func (r *MemoryRepository) SeedStarterTemplates(values ...StarterTemplate) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, value := range values {
		r.starters[strings.ToUpper(strings.TrimSpace(value.Code))] = value
	}
}

func (r *MemoryRepository) ListStarterTemplates(_ context.Context) ([]StarterTemplate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := make([]StarterTemplate, 0, len(r.starters))
	for _, value := range r.starters {
		values = append(values, value)
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Code < values[j].Code })
	return values, nil
}

func (r *MemoryRepository) StarterTemplateByCode(_ context.Context, code string) (StarterTemplate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.starters[strings.ToUpper(strings.TrimSpace(code))]
	if !ok {
		return StarterTemplate{}, ErrNotFound
	}
	return value, nil
}

func (r *MemoryRepository) CreateFormRevision(_ context.Context, value FormTemplate) (FormTemplate, error) {
	if value.TenantID == "" || value.LegalEntityID == "" || value.ID == "" || value.Version < 1 {
		return FormTemplate{}, ErrInvalid
	}
	applyFormMetadataDefaults(&value)
	event, err := newMonitoringEvent(value.TenantID, AggregateMonitoringForm, value.ID, value.Version, EventMonitoringFormCreated, value, value.CreatedBy, value.CreatedAt)
	if err != nil {
		return FormTemplate{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := revisionKey(value.TenantID, value.ID, value.Version)
	if _, exists := r.forms[key]; exists {
		return FormTemplate{}, ErrConflict
	}
	stored := cloneValue(value)
	r.forms[key] = stored
	r.appendEvent(event)
	return cloneValue(stored), nil
}

func (r *MemoryRepository) FormRevision(_ context.Context, tenant, legalEntityID, programID, id string, version int64) (FormTemplate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.forms[revisionKey(tenant, id, version)]
	if !ok || value.LegalEntityID != legalEntityID || value.ProgramID != programID {
		return FormTemplate{}, ErrNotFound
	}
	return cloneValue(value), nil
}

func (r *MemoryRepository) ReusableFormRevision(_ context.Context, tenant, legalEntityID, id string, version int64) (FormTemplate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.forms[revisionKey(tenant, id, version)]
	if !ok || value.LegalEntityID != legalEntityID {
		return FormTemplate{}, ErrNotFound
	}
	return cloneValue(value), nil
}

func (r *MemoryRepository) ListReusableFormRevisions(_ context.Context, tenant, legalEntityID string, limit int) ([]FormTemplate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := make([]FormTemplate, 0)
	for _, value := range r.forms {
		if value.TenantID == tenant && value.LegalEntityID == legalEntityID && value.Status == LifecycleActive && value.IsCurrent {
			values = append(values, cloneValue(value))
		}
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].Code == values[j].Code {
			if values[i].Version == values[j].Version {
				return values[i].ID < values[j].ID
			}
			return values[i].Version > values[j].Version
		}
		return values[i].Code < values[j].Code
	})
	return boundedForms(values, limit), nil
}

func (r *MemoryRepository) ListFormLibrary(_ context.Context, filter FormLibraryFilter) (FormTemplatePage, error) {
	if filter.TenantID == "" || filter.LegalEntityID == "" {
		return FormTemplatePage{}, ErrInvalid
	}
	cursor, err := decodeFormLibraryCursor(filter.Cursor)
	if err != nil {
		return FormTemplatePage{}, err
	}
	order, err := normalizedFormLibrarySort(filter.Sort)
	if err != nil {
		return FormTemplatePage{}, err
	}
	limit := boundedFormLibraryLimit(filter.Limit)
	search := strings.ToLower(strings.TrimSpace(filter.Search))
	use := strings.ToUpper(strings.TrimSpace(filter.Use))
	tag := strings.ToLower(strings.TrimSpace(filter.Tag))

	r.mu.RLock()
	defer r.mu.RUnlock()
	latest := make(map[string]FormTemplate)
	activeVersions := make(map[string]FormTemplate)
	for _, value := range r.forms {
		if value.TenantID != filter.TenantID || value.LegalEntityID != filter.LegalEntityID {
			continue
		}
		if prior, exists := latest[value.ID]; !exists || value.Version > prior.Version {
			latest[value.ID] = value
		}
		if value.IsCurrent && (value.Status == LifecycleActive || value.Status == LifecyclePaused) {
			activeVersions[value.ID] = value
		}
	}
	items := make([]FormLibraryItem, 0, len(latest))
	for _, value := range latest {
		if search != "" && !strings.Contains(strings.ToLower(value.Code+" "+value.Name+" "+value.Purpose), search) {
			continue
		}
		if filter.ProgramID != "" && value.ProgramID != filter.ProgramID || filter.OwnerPrincipalID != "" && value.OwnerPrincipalID != filter.OwnerPrincipalID || filter.Status != "" && value.Status != filter.Status {
			continue
		}
		if use != "" && !containsFold(value.ApprovedUses, use) || tag != "" && !containsFold(value.Tags, tag) {
			continue
		}
		if !formLibraryItemBeyondCursor(value, cursor, order) {
			continue
		}
		item := FormLibraryItem{Template: cloneValue(value)}
		if active, exists := activeVersions[value.ID]; exists {
			item.ActiveVersion = active.Version
			item.ActiveStatus = active.Status
		}
		items = append(items, item)
	}
	sortFormLibraryItems(items, order)
	page := FormTemplatePage{Items: items}
	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		last := page.Items[len(page.Items)-1].Template
		page.NextCursor = encodeFormLibraryCursor(formLibraryCursor{UpdatedAt: last.UpdatedAt, ID: last.ID})
	}
	return page, nil
}

func (r *MemoryRepository) ListSavedFormViews(_ context.Context, tenantID, legalEntityID, principalID string) ([]SavedFormView, error) {
	if tenantID == "" || legalEntityID == "" || principalID == "" {
		return nil, ErrInvalid
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	views := make([]SavedFormView, 0)
	for _, view := range r.savedViews {
		if view.TenantID == tenantID && view.LegalEntityID == legalEntityID && view.PrincipalID == principalID {
			views = append(views, cloneSavedFormView(view))
		}
	}
	sort.Slice(views, func(i, j int) bool {
		if views[i].UpdatedAt.Equal(views[j].UpdatedAt) {
			return views[i].Name < views[j].Name
		}
		return views[i].UpdatedAt.After(views[j].UpdatedAt)
	})
	return views, nil
}

func (r *MemoryRepository) SaveFormView(_ context.Context, view SavedFormView) (SavedFormView, error) {
	view.Name = strings.TrimSpace(view.Name)
	if view.ID == "" || view.TenantID == "" || view.LegalEntityID == "" || view.PrincipalID == "" || view.Name == "" || len(view.Name) > 120 || view.CreatedAt.IsZero() || view.UpdatedAt.Before(view.CreatedAt) || view.Filter.TenantID != "" || view.Filter.LegalEntityID != "" || view.Filter.Cursor != "" {
		return SavedFormView{}, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, existing := range r.savedViews {
		if key != savedViewKey(view.TenantID, view.LegalEntityID, view.PrincipalID, view.ID) && existing.TenantID == view.TenantID && existing.LegalEntityID == view.LegalEntityID && existing.PrincipalID == view.PrincipalID && strings.EqualFold(existing.Name, view.Name) {
			return SavedFormView{}, ErrConflict
		}
	}
	r.savedViews[savedViewKey(view.TenantID, view.LegalEntityID, view.PrincipalID, view.ID)] = cloneSavedFormView(view)
	return cloneSavedFormView(view), nil
}

func (r *MemoryRepository) DeleteSavedFormView(_ context.Context, tenantID, legalEntityID, principalID, id string) error {
	if tenantID == "" || legalEntityID == "" || principalID == "" || id == "" {
		return ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := savedViewKey(tenantID, legalEntityID, principalID, id)
	if _, exists := r.savedViews[key]; !exists {
		return ErrNotFound
	}
	delete(r.savedViews, key)
	return nil
}

func containsFold(values []string, expected string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), expected) {
			return true
		}
	}
	return false
}

func (r *MemoryRepository) ListFormRevisions(_ context.Context, tenant, legalEntityID, programID string, limit int) ([]FormTemplate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := make([]FormTemplate, 0)
	for _, value := range r.forms {
		if value.TenantID == tenant && value.LegalEntityID == legalEntityID && value.ProgramID == programID {
			values = append(values, cloneValue(value))
		}
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].Code == values[j].Code {
			return values[i].Version > values[j].Version
		}
		return values[i].Code < values[j].Code
	})
	return boundedForms(values, limit), nil
}

func (r *MemoryRepository) TransitionForm(_ context.Context, input LifecycleTransition) (FormTemplate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := revisionKey(input.TenantID, input.ID, input.ExpectedVersion)
	current, ok := r.forms[key]
	if !ok {
		return FormTemplate{}, ErrNotFound
	}
	if current.LegalEntityID != input.LegalEntityID || current.ProgramID != input.ProgramID {
		return FormTemplate{}, ErrNotFound
	}
	nextLifecycle, err := transitionLifecycle(current.Lifecycle, input)
	if err != nil {
		return FormTemplate{}, err
	}
	next := cloneValue(current)
	next.Lifecycle = nextLifecycle
	event, err := newMonitoringEvent(input.TenantID, AggregateMonitoringForm, input.ID, next.Version, EventMonitoringFormStateChanged, next, input.ActorID, input.At)
	if err != nil {
		return FormTemplate{}, err
	}
	nextKey := revisionKey(input.TenantID, input.ID, next.Version)
	if _, exists := r.forms[nextKey]; exists {
		return FormTemplate{}, ErrConflict
	}
	if next.IsCurrent || current.IsCurrent {
		for storedKey, stored := range r.forms {
			if stored.TenantID == input.TenantID && stored.ID == input.ID && stored.IsCurrent {
				stored.IsCurrent = false
				stored.Status = LifecycleRetired
				until := input.At.UTC()
				stored.EffectiveUntil = &until
				stored.UpdatedAt = until
				r.forms[storedKey] = stored
			}
		}
	}
	r.forms[nextKey] = cloneValue(next)
	r.appendEvent(event)
	return cloneValue(next), nil
}

func (r *MemoryRepository) CreateCheckRevision(_ context.Context, value MonitoringCheck) (MonitoringCheck, error) {
	if value.TenantID == "" || value.ID == "" || value.ProgramID == "" || value.Version < 1 {
		return MonitoringCheck{}, ErrInvalid
	}
	event, err := newMonitoringEvent(value.TenantID, AggregateMonitoringCheck, value.ID, value.Version, EventMonitoringCheckCreated, value, value.CreatedBy, value.CreatedAt)
	if err != nil {
		return MonitoringCheck{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := revisionKey(value.TenantID, value.ID, value.Version)
	if _, exists := r.checks[key]; exists {
		return MonitoringCheck{}, ErrConflict
	}
	stored := cloneValue(value)
	r.checks[key] = stored
	r.appendEvent(event)
	return cloneValue(stored), nil
}

func (r *MemoryRepository) CheckRevision(_ context.Context, tenant, id string, version int64) (MonitoringCheck, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.checks[revisionKey(tenant, id, version)]
	if !ok {
		return MonitoringCheck{}, ErrNotFound
	}
	return cloneValue(value), nil
}

func (r *MemoryRepository) LatestCheckRevision(_ context.Context, tenant, id string) (MonitoringCheck, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var latest MonitoringCheck
	found := false
	for _, value := range r.checks {
		if value.TenantID == tenant && value.ID == id && (!found || value.Version > latest.Version) {
			latest = value
			found = true
		}
	}
	if !found {
		return MonitoringCheck{}, ErrNotFound
	}
	return cloneValue(latest), nil
}

func (r *MemoryRepository) ListCheckRevisions(_ context.Context, tenant, programID string, limit int) ([]MonitoringCheck, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := make([]MonitoringCheck, 0)
	for _, value := range r.checks {
		if value.TenantID == tenant && value.ProgramID == programID {
			values = append(values, cloneValue(value))
		}
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].Code == values[j].Code {
			return values[i].Version > values[j].Version
		}
		return values[i].Code < values[j].Code
	})
	return boundedChecks(values, limit), nil
}

func (r *MemoryRepository) TransitionCheck(_ context.Context, input LifecycleTransition) (MonitoringCheck, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := revisionKey(input.TenantID, input.ID, input.ExpectedVersion)
	current, ok := r.checks[key]
	if !ok {
		return MonitoringCheck{}, ErrNotFound
	}
	nextLifecycle, err := transitionLifecycle(current.Lifecycle, input)
	if err != nil {
		return MonitoringCheck{}, err
	}
	next := cloneValue(current)
	next.Lifecycle = nextLifecycle
	event, err := newMonitoringEvent(input.TenantID, AggregateMonitoringCheck, input.ID, next.Version, EventMonitoringCheckStateChanged, next, input.ActorID, input.At)
	if err != nil {
		return MonitoringCheck{}, err
	}
	nextKey := revisionKey(input.TenantID, input.ID, next.Version)
	if _, exists := r.checks[nextKey]; exists {
		return MonitoringCheck{}, ErrConflict
	}
	if next.IsCurrent || current.IsCurrent {
		for storedKey, stored := range r.checks {
			if stored.TenantID == input.TenantID && stored.ProgramID == current.ProgramID && stored.Code == current.Code && stored.IsCurrent {
				stored.IsCurrent = false
				stored.Status = LifecycleRetired
				until := input.At.UTC()
				stored.EffectiveUntil = &until
				stored.UpdatedAt = until
				r.checks[storedKey] = stored
			}
		}
	}
	r.checks[nextKey] = cloneValue(next)
	r.appendEvent(event)
	return cloneValue(next), nil
}

func transitionLifecycle(current Lifecycle, input LifecycleTransition) (Lifecycle, error) {
	if input.TenantID == "" || input.ID == "" || input.ActorID == "" || input.ExpectedVersion < 1 || input.At.IsZero() {
		return Lifecycle{}, ErrInvalid
	}
	allowed := false
	switch current.Status {
	case LifecycleDraft:
		allowed = input.To == LifecyclePendingApproval
	case LifecyclePendingApproval:
		allowed = input.To == LifecycleActive || input.To == LifecycleRejected
	case LifecycleActive:
		allowed = input.To == LifecyclePaused || input.To == LifecycleRetired
	case LifecyclePaused:
		allowed = input.To == LifecycleActive || input.To == LifecycleRetired
	}
	if !allowed {
		return Lifecycle{}, ErrInvalid
	}
	now := input.At.UTC()
	next := current
	next.Status = input.To
	next.Version++
	next.UpdatedAt = now
	next.IsCurrent = input.To == LifecycleActive || input.To == LifecyclePaused
	switch input.To {
	case LifecyclePendingApproval:
		next.SubmittedBy = input.ActorID
		next.EffectiveFrom = nil
		next.EffectiveUntil = nil
	case LifecycleActive:
		next.ApprovedBy = input.ActorID
		if next.EffectiveFrom == nil {
			next.EffectiveFrom = &now
		}
		next.EffectiveUntil = nil
	case LifecycleRejected:
		next.RejectedBy = input.ActorID
		next.EffectiveFrom = nil
		next.EffectiveUntil = nil
	case LifecyclePaused:
		next.EffectiveUntil = nil
	case LifecycleRetired:
		next.IsCurrent = false
		if next.EffectiveFrom == nil {
			next.EffectiveFrom = &now
		}
		next.EffectiveUntil = &now
	}
	return next, nil
}

func (r *MemoryRepository) AppendResult(_ context.Context, value MonitoringResult) (MonitoringResult, error) {
	if value.TenantID == "" || value.ID == "" || value.MonitoringCheckID == "" || value.MonitoringCheckVersion < 1 || value.InputReferenceID == "" || value.EvaluatorVersion == "" {
		return MonitoringResult{}, ErrInvalid
	}
	event, err := newMonitoringEvent(value.TenantID, AggregateMonitoringResult, value.ID, 1, EventMonitoringResultRecorded, value, "", value.CreatedAt)
	if err != nil {
		return MonitoringResult{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.results {
		if existing.TenantID == value.TenantID && existing.MonitoringCheckID == value.MonitoringCheckID && existing.InputReferenceID == value.InputReferenceID && existing.EvaluatorVersion == value.EvaluatorVersion {
			return MonitoringResult{}, ErrConflict
		}
	}
	if _, exists := r.results[value.ID]; exists {
		return MonitoringResult{}, ErrConflict
	}
	stored := cloneValue(value)
	r.results[value.ID] = stored
	r.appendEvent(event)
	return cloneValue(stored), nil
}

func (r *MemoryRepository) appendEvent(event MonitoringEvent) {
	r.events = append(r.events, cloneValue(event))
	r.outbox = append(r.outbox, cloneValue(event))
}

func (r *MemoryRepository) MonitoringEvents(tenant, aggregateType, aggregateID string) []MonitoringEvent {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return filterMonitoringEvents(r.events, tenant, aggregateType, aggregateID)
}

func (r *MemoryRepository) MonitoringOutbox(tenant, aggregateType, aggregateID string) []MonitoringEvent {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return filterMonitoringEvents(r.outbox, tenant, aggregateType, aggregateID)
}

func filterMonitoringEvents(values []MonitoringEvent, tenant, aggregateType, aggregateID string) []MonitoringEvent {
	result := make([]MonitoringEvent, 0)
	for _, event := range values {
		if event.TenantID == tenant && event.AggregateType == aggregateType && event.AggregateID == aggregateID {
			result = append(result, cloneValue(event))
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].AggregateVersion < result[j].AggregateVersion })
	return result
}

func (r *MemoryRepository) Result(_ context.Context, tenant, id string) (MonitoringResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.results[id]
	if !ok || value.TenantID != tenant {
		return MonitoringResult{}, ErrNotFound
	}
	return cloneValue(value), nil
}

func (r *MemoryRepository) ListResults(_ context.Context, tenant, checkID string, limit int) ([]MonitoringResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := make([]MonitoringResult, 0)
	for _, value := range r.results {
		if value.TenantID == tenant && value.MonitoringCheckID == checkID {
			values = append(values, cloneValue(value))
		}
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].EvaluatedAt.Equal(values[j].EvaluatedAt) {
			return values[i].ID < values[j].ID
		}
		return values[i].EvaluatedAt.After(values[j].EvaluatedAt)
	})
	if limit < 1 || limit > 500 {
		limit = 100
	}
	if len(values) > limit {
		values = values[:limit]
	}
	return values, nil
}

func revisionKey(tenant, id string, version int64) string {
	return fmt.Sprintf("%s\x00%s\x00%d", tenant, id, version)
}

func cloneValue[T any](value T) T {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	var cloned T
	if err := json.Unmarshal(encoded, &cloned); err != nil {
		panic(err)
	}
	return cloned
}

func boundedForms(values []FormTemplate, limit int) []FormTemplate {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	if len(values) > limit {
		return values[:limit]
	}
	return values
}

func boundedChecks(values []MonitoringCheck, limit int) []MonitoringCheck {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	if len(values) > limit {
		return values[:limit]
	}
	return values
}

var _ Repository = (*MemoryRepository)(nil)
