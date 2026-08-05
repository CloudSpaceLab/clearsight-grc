package httpapi

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type matterAccess struct {
	Access              string   `json:"access"`
	AllowedPrincipalIDs []string `json:"allowed_principal_ids"`
}

func canReadMatter(ctx context.Context, matter continuity.Matter) bool {
	var access matterAccess
	if json.Unmarshal(matter.Scope, &access) != nil || !strings.EqualFold(access.Access, "RESTRICTED") {
		return true
	}
	actor, ok := identity.FromContext(ctx)
	if !ok {
		return false
	}
	if actor.LegalEntityID == "*" {
		return true
	}
	for _, principalID := range access.AllowedPrincipalIDs {
		if principalID == actor.PrincipalID {
			return true
		}
	}
	return false
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
	if !strings.EqualFold(request.Sensitivity, "RESTRICTED") || !strings.EqualFold(request.SubjectType, "MATTER") {
		return true
	}
	if a.deps.Continuity == nil {
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
