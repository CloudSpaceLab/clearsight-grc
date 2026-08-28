package evidence

import "sync"

type memoryWorkspaceTestRegistry struct {
	mu         sync.Mutex
	workspaces map[string]*memoryWorkspaceState
}

// memoryWorkspaceRegistryFor preserves the older white-box test shape while
// taking a snapshot from the store-scoped workspace map. Production state is
// not registered globally.
func memoryWorkspaceRegistryFor(store *MemoryDistributionAccessStore) *memoryWorkspaceTestRegistry {
	store.workspaceMu.Lock()
	defer store.workspaceMu.Unlock()
	workspaces := make(map[string]*memoryWorkspaceState, len(store.workspaceStates))
	for distributionID, state := range store.workspaceStates {
		workspaces[distributionID] = state
	}
	return &memoryWorkspaceTestRegistry{workspaces: workspaces}
}
