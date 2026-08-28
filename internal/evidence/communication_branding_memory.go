package evidence

import (
	"context"
	"sort"
	"strings"
	"sync"
)

type MemoryCommunicationBrandStore struct {
	mu     sync.RWMutex
	assets map[string]BrandAsset
}

func NewMemoryCommunicationBrandStore() *MemoryCommunicationBrandStore {
	return &MemoryCommunicationBrandStore{assets: map[string]BrandAsset{}}
}

func (store *MemoryCommunicationBrandStore) CreateCommunicationBrandAsset(_ context.Context, value BrandAsset) (BrandAsset, error) {
	if store == nil || strings.TrimSpace(value.ID) == "" {
		return BrandAsset{}, ErrCommunicationInvalid
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	key := communicationBrandKey(value.TenantID, value.LegalEntityID, value.ID)
	if _, exists := store.assets[key]; exists {
		return BrandAsset{}, ErrCommunicationConflict
	}
	store.assets[key] = value
	return value, nil
}

func (store *MemoryCommunicationBrandStore) GetCommunicationBrandAsset(_ context.Context, tenantID, legalEntityID, assetID string) (BrandAsset, error) {
	if store == nil {
		return BrandAsset{}, ErrCommunicationNotFound
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	value, ok := store.assets[communicationBrandKey(tenantID, legalEntityID, assetID)]
	if !ok {
		return BrandAsset{}, ErrCommunicationNotFound
	}
	return value, nil
}

func (store *MemoryCommunicationBrandStore) ListCommunicationBrandAssets(_ context.Context, tenantID, legalEntityID string) ([]BrandAsset, error) {
	if store == nil {
		return nil, ErrCommunicationNotFound
	}
	prefix := communicationScopeKey(tenantID, legalEntityID) + "\x00"
	store.mu.RLock()
	defer store.mu.RUnlock()
	result := make([]BrandAsset, 0)
	for key, value := range store.assets {
		if strings.HasPrefix(key, prefix) {
			result = append(result, value)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if !result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].CreatedAt.After(result[j].CreatedAt)
		}
		return result[i].ID > result[j].ID
	})
	return result, nil
}

func communicationBrandKey(tenantID, legalEntityID, assetID string) string {
	return communicationScopeKey(tenantID, legalEntityID) + "\x00" + strings.TrimSpace(assetID)
}

var _ communicationBrandStore = (*MemoryCommunicationBrandStore)(nil)
