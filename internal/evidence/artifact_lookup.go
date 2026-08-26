package evidence

import (
	"context"
	"strings"
)

// GetArtifact returns one exact request-scoped artifact without exposing its
// object-store contents. Callers must still project away StorageKey.
func (s *Service) GetArtifact(ctx context.Context, tenant, requestID, artifactID string) (Artifact, error) {
	tenant, requestID, artifactID = strings.TrimSpace(tenant), strings.TrimSpace(requestID), strings.TrimSpace(artifactID)
	if tenant == "" || requestID == "" || artifactID == "" {
		return Artifact{}, ErrNotFound
	}
	return s.repo.GetArtifact(ctx, tenant, requestID, artifactID)
}

func (r *MemoryRepository) GetArtifact(_ context.Context, tenant, requestID, artifactID string) (Artifact, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.artifacts[artifactID]
	if !ok || value.TenantID != tenant || value.RequestID != requestID {
		return Artifact{}, ErrNotFound
	}
	return value, nil
}
