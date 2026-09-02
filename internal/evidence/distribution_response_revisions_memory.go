package evidence

import "context"

func (s *MemoryDistributionStore) ListDistributionResponseRevisions(_ context.Context, tenantID, legalEntityID, distributionID string, limit int) ([]ResponseRevision, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	distribution, ok := s.distributions[distributionID]
	if !ok || distribution.TenantID != tenantID || distribution.LegalEntityID != legalEntityID {
		return nil, ErrNotFound
	}
	values := s.responseRevisions[distributionID]
	result := make([]ResponseRevision, 0, min(limit, len(values)))
	for index := len(values) - 1; index >= 0 && len(result) < limit; index-- {
		result = append(result, cloneResponseRevision(values[index]))
	}
	return result, nil
}
