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

func filterMatterAggregates(ctx context.Context, values []continuity.MatterAggregate) []continuity.MatterAggregate {
	visible := make([]continuity.MatterAggregate, 0, len(values))
	for _, value := range values {
		if canReadMatter(ctx, value.Matter) {
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
	if !ok || request.TenantID != actor.TenantID || !evidence.RequestAssignedTo(request, actor.PrincipalID) {
		return false
	}
	if !strings.EqualFold(request.SubjectType, "MATTER") {
		return true
	}
	if a.deps.Continuity == nil || strings.TrimSpace(request.SubjectID) == "" {
		return false
	}
	matter, err := a.deps.Continuity.GetMatter(ctx, request.TenantID, request.SubjectID)
	return err == nil && canReadMatter(ctx, matter.Matter)
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
