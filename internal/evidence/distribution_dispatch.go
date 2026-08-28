package evidence

import "context"

// Prepare persists a recoverable DRAFT without making any external capability
// usable. The API dispatch boundary prepares routes first, then calls Open.
func (service *DistributionService) Prepare(ctx context.Context, input CreateDistributionInput) (DistributionBundle, error) {
	if service == nil || service.store == nil {
		return DistributionBundle{}, ErrDistributionInvalid
	}
	bundle, err := service.store.CreateDistribution(ctx, input)
	if err != nil {
		return DistributionBundle{}, normalizeDistributionError(err)
	}
	return bundle, nil
}

// Open activates a prepared or previously locked distribution after all
// external delivery prerequisites are ready.
func (service *DistributionService) Open(ctx context.Context, tenantID, legalEntityID, distributionID string, expectedVersion int64, actorID string) (DistributionBundle, error) {
	return service.transition(ctx, tenantID, legalEntityID, distributionID, expectedVersion, DistributionOpen, actorID)
}
