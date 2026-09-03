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
	now := service.currentTime()
	bundle, err := service.store.GetDistribution(ctx, session.TenantID, session.LegalEntityID, session.DistributionID)
	if err != nil || !distributionAcceptsAccess(bundle.Distribution) || bundle.Distribution.ID != session.DistributionID || bundle.Distribution.LegalEntityID != session.LegalEntityID || bundle.Distribution.FormTemplateVersion < 1 {
		return DistributionRecoveryContext{}, ErrSessionInvalid
	}
	route, err := service.store.AccessRouteByID(ctx, session.TenantID, session.LegalEntityID, session.DistributionID, session.RouteID)
	if err != nil || !AccessGrantUsable(route, AccessGrant{
		RouteID: route.ID, TenantID: session.TenantID, DistributionID: session.DistributionID,
		RecipientID: session.RecipientID, Assurance: session.Assurance, ExpiresAt: session.ExpiresAt,
	}, now) {
		return DistributionRecoveryContext{}, ErrSessionInvalid
	}
	return DistributionRecoveryContext{
		LegalEntityID:  bundle.Distribution.LegalEntityID,
		DistributionID: bundle.Distribution.ID,
		SchemaVersion:  bundle.Distribution.FormTemplateVersion,
		RouteExpiresAt: route.ExpiresAt,
	}, nil
}
