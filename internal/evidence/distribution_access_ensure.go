package evidence

import (
	"context"
	"strings"
)

// EnsureDistributionAccessRoutes creates only the currently missing routes for
// external TO recipients. It is safe to call after a retry or recipient
// amendment: existing active routes remain unchanged and their selectors are
// never reconstructed or re-exposed.
func (service *DistributionAccessService) EnsureDistributionAccessRoutes(ctx context.Context, tenantID, legalEntityID, distributionID, createdBy string) ([]IssuedAccessRoute, error) {
	if service == nil || service.store == nil || strings.TrimSpace(createdBy) == "" {
		return nil, ErrDistributionAccessUnavailable
	}
	now := service.currentTime()
	bundle, err := service.store.GetDistribution(ctx, tenantID, legalEntityID, distributionID)
	if err != nil || !distributionMayIssueAccess(bundle.Distribution, now) {
		return nil, ErrDistributionAccessUnavailable
	}
	external := externalTORecipients(bundle.Recipients)
	if len(external) == 0 {
		return []IssuedAccessRoute{}, nil
	}
	active, err := service.store.ListActiveAccessRoutes(ctx, bundle.Distribution.TenantID, bundle.Distribution.LegalEntityID, bundle.Distribution.ID, now)
	if err != nil {
		return nil, ErrDistributionAccessUnavailable
	}

	covered := make(map[string]struct{}, len(active))
	sharedExists := false
	for _, route := range active {
		if route.Policy != bundle.Distribution.AccessPolicy {
			continue
		}
		if route.Policy == AccessSharedEmailOTP {
			sharedExists = true
			continue
		}
		if route.RecipientID != "" {
			covered[route.RecipientID] = struct{}{}
		}
	}

	routes := make([]AccessRoute, 0, len(external))
	issued := make([]IssuedAccessRoute, 0, len(external))
	appendRoute := func(recipientID, hint string) error {
		route, secret, routeErr := service.engine.IssueRoute(AccessRouteInput{
			TenantID: bundle.Distribution.TenantID, LegalEntityID: bundle.Distribution.LegalEntityID,
			DistributionID: bundle.Distribution.ID, RecipientID: recipientID,
			Policy: bundle.Distribution.AccessPolicy, AudienceHint: hint,
			RouteExpiresAt: bundle.Distribution.RouteExpiresAt, Deadline: bundle.Distribution.Deadline,
			CreatedBy: createdBy,
		})
		if routeErr != nil {
			return routeErr
		}
		routes = append(routes, route)
		issued = append(issued, secret)
		return nil
	}

	if bundle.Distribution.AccessPolicy == AccessSharedEmailOTP {
		if !sharedExists {
			if err := appendRoute("", ""); err != nil {
				return nil, ErrDistributionAccessUnavailable
			}
		}
	} else {
		for _, recipient := range external {
			if _, exists := covered[recipient.ID]; exists {
				continue
			}
			if err := appendRoute(recipient.ID, recipient.AudienceHint); err != nil {
				return nil, ErrDistributionAccessUnavailable
			}
		}
	}
	if len(routes) == 0 {
		return []IssuedAccessRoute{}, nil
	}
	if err := service.store.CreateAccessRoutes(ctx, routes); err != nil {
		return nil, ErrDistributionAccessUnavailable
	}
	return issued, nil
}
