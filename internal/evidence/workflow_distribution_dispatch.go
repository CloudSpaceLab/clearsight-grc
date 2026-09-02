package evidence

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// WorkflowDistributionDispatchInput is the internal boundary used by domain
// workflows that send one governed form to one external recipient.
type WorkflowDistributionDispatchInput struct {
	Request        CreateRequestInput
	AccessPolicy   AccessPolicy
	RouteExpiresAt time.Time
	AudienceHint   string
}

type WorkflowDistributionDispatch struct {
	Distribution FormDistribution
	Request      Request
	Route        IssuedAccessRoute
}

type WorkflowDistributionDispatcher struct {
	distributions *DistributionService
	access        *DistributionAccessService
}

func NewWorkflowDistributionDispatcher(distributions *DistributionService, access *DistributionAccessService) *WorkflowDistributionDispatcher {
	return &WorkflowDistributionDispatcher{distributions: distributions, access: access}
}

func (service *WorkflowDistributionDispatcher) Dispatch(ctx context.Context, input WorkflowDistributionDispatchInput) (WorkflowDistributionDispatch, error) {
	if service == nil || service.distributions == nil || service.access == nil || service.access.store == nil {
		return WorkflowDistributionDispatch{}, ErrDistributionAccessUnavailable
	}
	requestInput := input.Request
	if requestInput.Recipient.Type != RecipientExternalAudience || strings.TrimSpace(requestInput.Recipient.Audience) == "" || requestInput.Origin.empty() {
		return WorkflowDistributionDispatch{}, fmt.Errorf("%w: external workflow request, recipient and origin are required", ErrDistributionInvalid)
	}
	hint := strings.TrimSpace(input.AudienceHint)
	if hint == "" {
		hint = audienceHint(normalizeAudience(requestInput.Recipient.Audience))
	}
	prepared, err := service.distributions.Prepare(ctx, CreateDistributionInput{
		TenantID: requestInput.TenantID, LegalEntityID: requestInput.LegalEntityID,
		FormTemplateID: requestInput.FormTemplateID, FormTemplateVersion: requestInput.FormTemplateVersion,
		SubjectType: requestInput.SubjectType, SubjectID: requestInput.SubjectID,
		Title: requestInput.Title, Purpose: requestInput.Purpose,
		AccessPolicy: input.AccessPolicy, EstimatedMinutes: requestInput.EstimatedMinutes,
		Deadline: requestInput.Deadline, RouteExpiresAt: input.RouteExpiresAt,
		CreatedBy: requestInput.CreatedBy,
		Recipients: []DistributionRecipientInput{{
			Role: RecipientTo, Type: RecipientExternalAudience, Address: requestInput.Recipient.Audience,
			AudienceHint: hint, ContactLabel: "Vendor contact",
		}},
		RequestInput: &requestInput,
	})
	if err != nil {
		return WorkflowDistributionDispatch{}, err
	}
	if len(prepared.Recipients) != 1 || prepared.Recipients[0].RequestID == "" {
		return WorkflowDistributionDispatch{}, ErrDistributionInvalid
	}
	request, err := service.access.store.GetRequest(ctx, prepared.Distribution.TenantID, prepared.Recipients[0].RequestID)
	if err != nil {
		return WorkflowDistributionDispatch{}, err
	}
	partial := WorkflowDistributionDispatch{Distribution: prepared.Distribution, Request: request}
	routes, err := service.access.EnsureDistributionAccessRoutes(ctx, prepared.Distribution.TenantID, prepared.Distribution.LegalEntityID, prepared.Distribution.ID, requestInput.CreatedBy)
	if err != nil || len(routes) != 1 {
		return partial, ErrDistributionAccessUnavailable
	}
	partial.Route = routes[0]
	opened, err := service.distributions.Open(ctx, prepared.Distribution.TenantID, prepared.Distribution.LegalEntityID, prepared.Distribution.ID, prepared.Distribution.Version, requestInput.CreatedBy)
	if err != nil {
		_ = service.access.RevokeDistributionAccessRoute(ctx, prepared.Distribution.TenantID, prepared.Distribution.LegalEntityID, prepared.Distribution.ID, routes[0].RouteID)
		return partial, err
	}
	if len(opened.Recipients) != 1 || opened.Recipients[0].RequestID == "" {
		return WorkflowDistributionDispatch{}, ErrDistributionInvalid
	}
	request, err = service.access.store.GetRequest(ctx, opened.Distribution.TenantID, opened.Recipients[0].RequestID)
	if err != nil {
		return WorkflowDistributionDispatch{}, err
	}
	return WorkflowDistributionDispatch{Distribution: opened.Distribution, Request: request, Route: routes[0]}, nil
}

func (service *WorkflowDistributionDispatcher) Revoke(ctx context.Context, tenantID, legalEntityID, distributionID, routeID string) error {
	if service == nil || service.access == nil {
		return ErrDistributionAccessUnavailable
	}
	return service.access.RevokeDistributionAccessRoute(ctx, tenantID, legalEntityID, distributionID, routeID)
}

func (service *WorkflowDistributionDispatcher) RevokeRequest(ctx context.Context, tenantID, legalEntityID, requestID, routeID string) error {
	if service == nil || service.distributions == nil {
		return ErrDistributionAccessUnavailable
	}
	bundle, err := service.distributions.GetForRequest(ctx, tenantID, legalEntityID, requestID)
	if err != nil {
		return err
	}
	return service.Revoke(ctx, bundle.Distribution.TenantID, bundle.Distribution.LegalEntityID, bundle.Distribution.ID, routeID)
}

// RevokeRequestCapabilities invalidates every active canonical route for the
// exact request. Existing sessions become unusable because each grant remains
// bound to its now-revoked route.
func (service *WorkflowDistributionDispatcher) RevokeRequestCapabilities(ctx context.Context, tenantID, requestID string) error {
	if service == nil || service.distributions == nil || service.access == nil || service.access.store == nil {
		return ErrDistributionAccessUnavailable
	}
	request, err := service.access.store.GetRequest(ctx, tenantID, requestID)
	if err != nil {
		return err
	}
	bundle, err := service.distributions.GetForRequest(ctx, request.TenantID, request.LegalEntityID, request.ID)
	if err != nil {
		return err
	}
	routes, err := service.access.store.ListActiveAccessRoutes(ctx, bundle.Distribution.TenantID, bundle.Distribution.LegalEntityID, bundle.Distribution.ID, service.access.currentTime())
	if err != nil {
		return ErrDistributionAccessUnavailable
	}
	for _, route := range routes {
		if err := service.access.RevokeDistributionAccessRoute(ctx, bundle.Distribution.TenantID, bundle.Distribution.LegalEntityID, bundle.Distribution.ID, route.ID); err != nil {
			return err
		}
	}
	return nil
}

func (service *WorkflowDistributionDispatcher) Resume(ctx context.Context, tenantID, legalEntityID, requestID, actorID string, routeExpiresAt time.Time) (WorkflowDistributionDispatch, error) {
	if service == nil || service.distributions == nil || service.access == nil || service.access.store == nil {
		return WorkflowDistributionDispatch{}, ErrDistributionAccessUnavailable
	}
	bundle, err := service.distributions.GetForRequest(ctx, tenantID, legalEntityID, requestID)
	if err != nil {
		return WorkflowDistributionDispatch{}, err
	}
	if !routeExpiresAt.IsZero() {
		routeExpiresAt = routeExpiresAt.UTC()
		if routeExpiresAt.After(bundle.Distribution.Deadline) {
			routeExpiresAt = bundle.Distribution.Deadline
		}
		if !routeExpiresAt.After(service.access.currentTime()) {
			return WorkflowDistributionDispatch{}, ErrDistributionInvalid
		}
		if !routeExpiresAt.Equal(bundle.Distribution.RouteExpiresAt) {
			amended, amendErr := service.distributions.Amend(ctx, bundle.Distribution.TenantID, bundle.Distribution.LegalEntityID, bundle.Distribution.ID, AmendDistributionInput{
				ExpectedVersion: bundle.Distribution.Version, RouteExpiresAt: &routeExpiresAt, ActorID: actorID,
			})
			if amendErr != nil {
				return WorkflowDistributionDispatch{}, amendErr
			}
			bundle = amended.Bundle
		}
	}
	request, err := service.access.store.GetRequest(ctx, bundle.Distribution.TenantID, requestID)
	if err != nil {
		return WorkflowDistributionDispatch{}, err
	}
	routes, err := service.access.EnsureDistributionAccessRoutes(ctx, bundle.Distribution.TenantID, bundle.Distribution.LegalEntityID, bundle.Distribution.ID, actorID)
	if err != nil {
		return WorkflowDistributionDispatch{Distribution: bundle.Distribution, Request: request}, err
	}
	var issued IssuedAccessRoute
	if len(routes) == 1 {
		issued = routes[0]
	} else if len(routes) == 0 {
		active, listErr := service.access.store.ListActiveAccessRoutes(ctx, bundle.Distribution.TenantID, bundle.Distribution.LegalEntityID, bundle.Distribution.ID, service.access.currentTime())
		if listErr != nil {
			return WorkflowDistributionDispatch{Distribution: bundle.Distribution, Request: request}, ErrDistributionAccessUnavailable
		}
		for _, route := range active {
			for _, recipient := range bundle.Recipients {
				if recipient.RequestID == requestID && route.RecipientID == recipient.ID {
					issued, err = service.access.RotateDistributionAccessRoute(ctx, bundle.Distribution.TenantID, bundle.Distribution.LegalEntityID, bundle.Distribution.ID, route.ID, actorID)
					break
				}
			}
		}
	}
	if err != nil || issued.RouteID == "" {
		return WorkflowDistributionDispatch{Distribution: bundle.Distribution, Request: request}, ErrDistributionAccessUnavailable
	}
	if bundle.Distribution.Status != DistributionOpen {
		bundle, err = service.distributions.Open(ctx, bundle.Distribution.TenantID, bundle.Distribution.LegalEntityID, bundle.Distribution.ID, bundle.Distribution.Version, actorID)
		if err != nil {
			_ = service.access.RevokeDistributionAccessRoute(ctx, bundle.Distribution.TenantID, bundle.Distribution.LegalEntityID, bundle.Distribution.ID, issued.RouteID)
			return WorkflowDistributionDispatch{Distribution: bundle.Distribution, Request: request}, err
		}
	}
	return WorkflowDistributionDispatch{Distribution: bundle.Distribution, Request: request, Route: issued}, nil
}
