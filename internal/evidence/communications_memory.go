package evidence

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
)

type MemoryCommunicationStore struct {
	mu        sync.RWMutex
	profiles  map[string][]CommunicationProfile
	templates map[string][]CommunicationTemplate
}

func NewMemoryCommunicationStore() *MemoryCommunicationStore {
	return &MemoryCommunicationStore{
		profiles:  map[string][]CommunicationProfile{},
		templates: map[string][]CommunicationTemplate{},
	}
}

func (store *MemoryCommunicationStore) CreateProfileRevision(_ context.Context, input CreateCommunicationProfileInput, now time.Time) (CommunicationProfile, error) {
	if store == nil {
		return CommunicationProfile{}, ErrCommunicationInvalid
	}
	id, err := nextCommunicationID()
	if err != nil {
		return CommunicationProfile{}, err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	key := communicationScopeKey(input.TenantID, input.LegalEntityID)
	version := int64(len(store.profiles[key]) + 1)
	value := CommunicationProfile{
		ID: id, TenantID: input.TenantID, LegalEntityID: input.LegalEntityID, Version: version,
		DefaultLocale: input.DefaultLocale, BankName: input.BankName, SupportContact: input.SupportContact,
		BrandAssetID: input.BrandAssetID, Status: CommunicationDraft, EffectiveFrom: input.EffectiveFrom.UTC(),
		EffectiveUntil: cloneOptionalTime(input.EffectiveUntil), MakerID: input.MakerID,
		CreatedAt: now.UTC(), UpdatedAt: now.UTC(),
	}
	store.profiles[key] = append(store.profiles[key], value)
	return cloneCommunicationProfile(value), nil
}

func (store *MemoryCommunicationStore) GetProfile(_ context.Context, tenantID, legalEntityID string, version int64) (CommunicationProfile, error) {
	if store == nil {
		return CommunicationProfile{}, ErrCommunicationNotFound
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	for _, value := range store.profiles[communicationScopeKey(tenantID, legalEntityID)] {
		if value.Version == version {
			return cloneCommunicationProfile(value), nil
		}
	}
	return CommunicationProfile{}, ErrCommunicationNotFound
}

func (store *MemoryCommunicationStore) ListProfiles(_ context.Context, tenantID, legalEntityID string) ([]CommunicationProfile, error) {
	if store == nil {
		return nil, ErrCommunicationNotFound
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	values := store.profiles[communicationScopeKey(tenantID, legalEntityID)]
	result := make([]CommunicationProfile, len(values))
	for index, value := range values {
		result[index] = cloneCommunicationProfile(value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Version > result[j].Version })
	return result, nil
}

func (store *MemoryCommunicationStore) TransitionProfile(_ context.Context, tenantID, legalEntityID string, version int64, input CommunicationTransitionInput, now time.Time) (CommunicationProfile, error) {
	if store == nil {
		return CommunicationProfile{}, ErrCommunicationInvalid
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	key := communicationScopeKey(tenantID, legalEntityID)
	values := store.profiles[key]
	index := profileVersionIndex(values, version)
	if index < 0 {
		return CommunicationProfile{}, ErrCommunicationNotFound
	}
	current := values[index]
	status, checkerID, effectiveFrom, effectiveUntil, err := applyCommunicationTransition(current.Status, current.MakerID, input, current.EffectiveFrom, current.EffectiveUntil, now)
	if err != nil {
		return CommunicationProfile{}, err
	}
	if status == CommunicationActive {
		retireOverlappingProfiles(values, index, effectiveFrom, effectiveUntil, now)
	}
	current.Status = status
	if checkerID != "" {
		current.CheckerID = checkerID
	}
	current.EffectiveFrom = effectiveFrom
	current.EffectiveUntil = cloneOptionalTime(effectiveUntil)
	current.UpdatedAt = now.UTC()
	values[index] = current
	store.profiles[key] = values
	return cloneCommunicationProfile(current), nil
}

func (store *MemoryCommunicationStore) CreateTemplateRevision(_ context.Context, input CreateCommunicationTemplateInput, now time.Time) (CommunicationTemplate, error) {
	if store == nil {
		return CommunicationTemplate{}, ErrCommunicationInvalid
	}
	id, err := nextCommunicationID()
	if err != nil {
		return CommunicationTemplate{}, err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	key := communicationTemplateKey(input.TenantID, input.LegalEntityID, input.Action, input.Locale)
	version := int64(len(store.templates[key]) + 1)
	value := CommunicationTemplate{
		ID: id, TenantID: input.TenantID, LegalEntityID: input.LegalEntityID, Action: input.Action,
		Locale: input.Locale, Version: version, SubjectTemplate: input.SubjectTemplate,
		Document: cloneCommunicationNodes(input.Document), Status: CommunicationDraft,
		EffectiveFrom: input.EffectiveFrom.UTC(), EffectiveUntil: cloneOptionalTime(input.EffectiveUntil),
		MakerID: input.MakerID, CreatedAt: now.UTC(), UpdatedAt: now.UTC(),
	}
	store.templates[key] = append(store.templates[key], value)
	return cloneCommunicationTemplate(value), nil
}

func (store *MemoryCommunicationStore) GetTemplate(_ context.Context, tenantID, legalEntityID string, action CommunicationAction, locale string, version int64) (CommunicationTemplate, error) {
	if store == nil {
		return CommunicationTemplate{}, ErrCommunicationNotFound
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	for _, value := range store.templates[communicationTemplateKey(tenantID, legalEntityID, action, locale)] {
		if value.Version == version {
			return cloneCommunicationTemplate(value), nil
		}
	}
	return CommunicationTemplate{}, ErrCommunicationNotFound
}

func (store *MemoryCommunicationStore) ListTemplates(_ context.Context, query CommunicationTemplateQuery) ([]CommunicationTemplate, error) {
	if store == nil {
		return nil, ErrCommunicationNotFound
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	result := make([]CommunicationTemplate, 0)
	for _, values := range store.templates {
		for _, value := range values {
			if query.TenantID != "" && value.TenantID != query.TenantID || query.LegalEntityID != "" && value.LegalEntityID != query.LegalEntityID || query.Action != "" && value.Action != query.Action || query.Locale != "" && value.Locale != query.Locale || query.Status != "" && value.Status != query.Status {
				continue
			}
			result = append(result, cloneCommunicationTemplate(value))
		}
	}
	sortCommunicationTemplates(result)
	return result, nil
}

func (store *MemoryCommunicationStore) TransitionTemplate(_ context.Context, tenantID, legalEntityID string, action CommunicationAction, locale string, version int64, input CommunicationTransitionInput, now time.Time) (CommunicationTemplate, error) {
	if store == nil {
		return CommunicationTemplate{}, ErrCommunicationInvalid
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	key := communicationTemplateKey(tenantID, legalEntityID, action, locale)
	values := store.templates[key]
	index := templateVersionIndex(values, version)
	if index < 0 {
		return CommunicationTemplate{}, ErrCommunicationNotFound
	}
	current := values[index]
	if input.To == CommunicationActive {
		if err := ValidateCommunicationTemplate(current); err != nil {
			return CommunicationTemplate{}, err
		}
	}
	status, checkerID, effectiveFrom, effectiveUntil, err := applyCommunicationTransition(current.Status, current.MakerID, input, current.EffectiveFrom, current.EffectiveUntil, now)
	if err != nil {
		return CommunicationTemplate{}, err
	}
	if status == CommunicationActive {
		retireOverlappingTemplates(values, index, effectiveFrom, effectiveUntil, now)
	}
	current.Status = status
	if checkerID != "" {
		current.CheckerID = checkerID
	}
	current.EffectiveFrom = effectiveFrom
	current.EffectiveUntil = cloneOptionalTime(effectiveUntil)
	current.UpdatedAt = now.UTC()
	values[index] = current
	store.templates[key] = values
	return cloneCommunicationTemplate(current), nil
}

func (store *MemoryCommunicationStore) MarkTemplateRollback(_ context.Context, tenantID, legalEntityID string, action CommunicationAction, locale string, version, sourceVersion int64) (CommunicationTemplate, error) {
	if store == nil {
		return CommunicationTemplate{}, ErrCommunicationInvalid
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	key := communicationTemplateKey(tenantID, legalEntityID, action, locale)
	values := store.templates[key]
	index := templateVersionIndex(values, version)
	if index < 0 || templateVersionIndex(values, sourceVersion) < 0 {
		return CommunicationTemplate{}, ErrCommunicationNotFound
	}
	values[index].RollbackOriginVersion = sourceVersion
	store.templates[key] = values
	return cloneCommunicationTemplate(values[index]), nil
}

func retireOverlappingProfiles(values []CommunicationProfile, except int, from time.Time, until *time.Time, now time.Time) {
	for index := range values {
		if index == except || values[index].Status != CommunicationActive || !communicationWindowsOverlap(values[index].EffectiveFrom, values[index].EffectiveUntil, from, until) {
			continue
		}
		values[index].Status = CommunicationRetired
		values[index].UpdatedAt = now.UTC()
	}
}

func retireOverlappingTemplates(values []CommunicationTemplate, except int, from time.Time, until *time.Time, now time.Time) {
	for index := range values {
		if index == except || values[index].Status != CommunicationActive || !communicationWindowsOverlap(values[index].EffectiveFrom, values[index].EffectiveUntil, from, until) {
			continue
		}
		values[index].Status = CommunicationRetired
		values[index].UpdatedAt = now.UTC()
	}
}

func communicationWindowsOverlap(leftFrom time.Time, leftUntil *time.Time, rightFrom time.Time, rightUntil *time.Time) bool {
	leftEndsAfterRightStarts := leftUntil == nil || leftUntil.After(rightFrom)
	rightEndsAfterLeftStarts := rightUntil == nil || rightUntil.After(leftFrom)
	return leftEndsAfterRightStarts && rightEndsAfterLeftStarts
}

func communicationScopeKey(tenantID, legalEntityID string) string {
	return strings.TrimSpace(tenantID) + "\x00" + strings.TrimSpace(legalEntityID)
}

func communicationTemplateKey(tenantID, legalEntityID string, action CommunicationAction, locale string) string {
	return communicationScopeKey(tenantID, legalEntityID) + "\x00" + string(action) + "\x00" + normalizeCommunicationLocale(locale)
}

func profileVersionIndex(values []CommunicationProfile, version int64) int {
	for index := range values {
		if values[index].Version == version {
			return index
		}
	}
	return -1
}

func templateVersionIndex(values []CommunicationTemplate, version int64) int {
	for index := range values {
		if values[index].Version == version {
			return index
		}
	}
	return -1
}

func cloneCommunicationProfile(value CommunicationProfile) CommunicationProfile {
	value.EffectiveUntil = cloneOptionalTime(value.EffectiveUntil)
	return value
}

func cloneCommunicationTemplate(value CommunicationTemplate) CommunicationTemplate {
	value.Document = cloneCommunicationNodes(value.Document)
	value.EffectiveUntil = cloneOptionalTime(value.EffectiveUntil)
	return value
}

func cloneOptionalTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := value.UTC()
	return &cloned
}

var _ communicationStore = (*MemoryCommunicationStore)(nil)
