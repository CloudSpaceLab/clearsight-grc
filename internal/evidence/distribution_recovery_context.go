package evidence

import (
	"context"
	"strings"
	"time"
)

// DistributionRecoveryContext is the bounded public metadata required to bind
// encrypted browser recovery to the exact authorized distribution and form
// revision. It intentionally excludes tenant, recipient, route and session
// credentials.
type DistributionRecoveryContext struct {
	LegalEntityID  string    `json:"legal_entity_id"`
	DistributionID string    `json:"distribution_id"`
	SchemaVersion  int64     `json:"schema_version"`
	RouteExpiresAt time.Time `json:"route_expires_at"`
}

func (service *DistributionAccessService) ResponseRecoveryContext(ctx context.Context, session DistributionAccessSession) (DistributionRecoveryContext, error) {
	if service == nil || service.store == nil || strings.TrimSpace(session.TenantID) == "" || strings.TrimSpace(session.LegalEntityID) == "" || strings.TrimSpace(session.DistributionID) == "" {
		return DistributionRecoveryContext{}, ErrSessionInvalid
	}
	bundle, err := service.store.GetDistribution(ctx, session.TenantID, session.LegalEntityID, session.DistributionID)
	if err != nil || !distributionOpenForAccess(bundle.Distribution, service.currentTime()) || bundle.Distribution.ID != session.DistributionID || bundle.Distribution.LegalEntityID != session.LegalEntityID || bundle.Distribution.FormTemplateVersion < 1 || bundle.Distribution.RouteExpiresAt.IsZero() {
		return DistributionRecoveryContext{}, ErrSessionInvalid
	}
	return DistributionRecoveryContext{
		LegalEntityID:  bundle.Distribution.LegalEntityID,
		DistributionID: bundle.Distribution.ID,
		SchemaVersion:  bundle.Distribution.FormTemplateVersion,
		RouteExpiresAt: bundle.Distribution.RouteExpiresAt,
	}, nil
}
