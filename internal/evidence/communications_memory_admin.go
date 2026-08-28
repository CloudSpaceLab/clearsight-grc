package evidence

import "context"

func (store *MemoryCommunicationStore) MarkProfileRollback(_ context.Context, tenantID, legalEntityID string, version, sourceVersion int64) (CommunicationProfile, error) {
	if store == nil {
		return CommunicationProfile{}, ErrCommunicationInvalid
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	key := communicationScopeKey(tenantID, legalEntityID)
	values := store.profiles[key]
	index := profileVersionIndex(values, version)
	if index < 0 || profileVersionIndex(values, sourceVersion) < 0 {
		return CommunicationProfile{}, ErrCommunicationNotFound
	}
	values[index].RollbackOriginVersion = sourceVersion
	store.profiles[key] = values
	return cloneCommunicationProfile(values[index]), nil
}
