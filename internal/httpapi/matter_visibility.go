package httpapi

import (
	"context"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func canReadMatter(ctx context.Context, matter continuity.Matter) bool {
	actor, ok := identity.FromContext(ctx)
	if !ok || strings.TrimSpace(actor.TenantID) == "" || matter.TenantID != actor.TenantID {
		return false
	}
	return continuity.MatterVisibleTo(matter, actor.PrincipalID)
}

func canReadMatterAggregate(ctx context.Context, aggregate continuity.MatterAggregate) bool {
	actor, ok := identity.FromContext(ctx)
	if !ok || strings.TrimSpace(actor.TenantID) == "" || aggregate.Matter.TenantID != actor.TenantID {
		return false
	}
	return continuity.MatterAggregateVisibleTo(aggregate, actor.PrincipalID)
}

func filterMatterAggregates(ctx context.Context, values []continuity.MatterAggregate) []continuity.MatterAggregate {
	visible := make([]continuity.MatterAggregate, 0, len(values))
	for _, value := range values {
		if canReadMatterAggregate(ctx, value) {
			visible = append(visible, value)
		}
	}
	return visible
}

func filterMatterSummaries(ctx context.Context, values []continuity.MatterSummary) []continuity.MatterSummary {
	visible := make([]continuity.MatterSummary, 0, len(values))
	for _, value := range values {
		if canReadMatter(ctx, value.Matter) {
			visible = append(visible, value)
		}
	}
	return visible
}

func (a *API) canReadEvidenceRequest(ctx context.Context, request evidence.Request) bool {
	actor, ok := identity.FromContext(ctx)
	if !ok || request.TenantID != actor.TenantID || !evidence.RequestManageableBy(request, actor.PrincipalID) {
		return false
	}
	if !strings.EqualFold(request.SubjectType, "MATTER") {
		return true
	}
	if a.deps.Continuity == nil || strings.TrimSpace(request.SubjectID) == "" {
		return false
	}
	matter, err := a.deps.Continuity.GetMatter(ctx, request.TenantID, request.SubjectID)
	if err != nil || !canReadMatterAggregate(ctx, matter) {
		return false
	}
	if request.Origin.Type != "THIRD_PARTY_ADDRESS_VERIFICATION" || request.CreatedBy == actor.PrincipalID {
		return true
	}
	// Action assignment is the security boundary for internal address
	// verification. Even before the durable reassignment worker has rotated the
	// request recipient and sessions, the superseded assignee must lose access.
	for _, action := range matter.Actions {
		if action.ID == request.Origin.ID {
			return action.OwnerPrincipalID == actor.PrincipalID && action.Version == request.Origin.Version
		}
	}
	return false
}

func (a *API) filterEvidenceRequests(ctx context.Context, values []evidence.Request) []evidence.Request {
	visible := make([]evidence.Request, 0, len(values))
	for _, value := range values {
		if a.canReadEvidenceRequest(ctx, value) {
			visible = append(visible, value)
		}
	}
	return visible
}
