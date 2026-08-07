package evidence

import "context"

func (r *MemoryRepository) GetArtifact(_ context.Context, tenant, requestID, artifactID string) (Artifact, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.artifacts[artifactID]
	if !ok || value.TenantID != tenant || value.RequestID != requestID {
		return Artifact{}, ErrNotFound
	}
	return value, nil
}
