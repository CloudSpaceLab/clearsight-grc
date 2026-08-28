package evidence

import (
	"context"
	"fmt"
	"strings"
)

type distributionResponseRevisionStore interface {
	ListDistributionResponseRevisions(context.Context, string, string, string, int) ([]ResponseRevision, error)
}

// ListResponseRevisions returns immutable sender-visible response metadata for a
// distribution. The distribution is resolved first so scope checks remain
// identical to the ordinary authenticated detail path.
func (service *DistributionService) ListResponseRevisions(ctx context.Context, tenantID, legalEntityID, distributionID string, limit int) ([]ResponseRevision, error) {
	if service == nil || service.store == nil || limit < 1 || limit > 100 || strings.TrimSpace(distributionID) == "" {
		return nil, fmt.Errorf("%w: response revision query is invalid", ErrDistributionInvalid)
	}
	bundle, err := service.Get(ctx, tenantID, legalEntityID, distributionID)
	if err != nil {
		return nil, err
	}
	reader, ok := service.store.(distributionResponseRevisionStore)
	if !ok {
		return []ResponseRevision{}, nil
	}
	values, err := reader.ListDistributionResponseRevisions(ctx, bundle.Distribution.TenantID, bundle.Distribution.LegalEntityID, bundle.Distribution.ID, limit)
	if err != nil {
		return nil, normalizeDistributionError(err)
	}
	return values, nil
}
