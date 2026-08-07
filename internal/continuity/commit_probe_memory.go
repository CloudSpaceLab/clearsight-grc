package continuity

import "context"

func (r *MemoryRepository) CurrentProgramVersion(_ context.Context, tenant, id string) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	aggregate, ok := r.programs[tenant][id]
	if !ok {
		return 0, ErrNotFound
	}
	return aggregate.Program.Version, nil
}

func (r *MemoryRepository) CurrentMatterVersion(_ context.Context, tenant, id string) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	aggregate, ok := r.matters[tenant][id]
	if !ok {
		return 0, ErrNotFound
	}
	return aggregate.Matter.Version, nil
}
