package httpapi

import (
	"context"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/access"
	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

const unavailableResponsiblePartyName = "Recorded assignee name unavailable"

type principalLabelCacheKey struct{}

type principalLabelCache struct {
	values map[string]*authority.Principal
}

func (a *API) withPrincipalLabels(ctx context.Context, actor identity.Actor, principalIDs []string) (context.Context, bool) {
	unique := make([]string, 0, len(principalIDs))
	seen := make(map[string]struct{}, len(principalIDs))
	complete := true
	for _, principalID := range principalIDs {
		principalID = strings.TrimSpace(principalID)
		if principalID == "" {
			continue
		}
		if _, exists := seen[principalID]; exists {
			continue
		}
		seen[principalID] = struct{}{}
		if len(unique) >= access.MaxPrincipalBatchSize {
			complete = false
			continue
		}
		unique = append(unique, principalID)
	}
	cache := &principalLabelCache{values: make(map[string]*authority.Principal, len(seen))}
	for principalID := range seen {
		cache.values[principalID] = nil
	}
	if len(unique) == 0 {
		return context.WithValue(ctx, principalLabelCacheKey{}, cache), complete
	}
	if a == nil || a.deps.Access == nil {
		return context.WithValue(ctx, principalLabelCacheKey{}, cache), false
	}
	if batch, ok := a.deps.Access.(access.BatchPrincipalResolver); ok {
		outcomes, err := batch.ResolvePrincipals(ctx, actor.TenantID, actor.LegalEntityID, unique)
		if err != nil || len(outcomes) != len(unique) {
			return context.WithValue(ctx, principalLabelCacheKey{}, cache), false
		}
		for index, outcome := range outcomes {
			if outcome.Err != nil || strings.TrimSpace(outcome.Resolution.DisplayName) == "" {
				complete = false
				continue
			}
			cache.values[unique[index]] = &authority.Principal{
				ID: outcome.Resolution.PrincipalID, DisplayName: outcome.Resolution.DisplayName, Kind: outcome.Resolution.Kind,
			}
		}
		return context.WithValue(ctx, principalLabelCacheKey{}, cache), complete
	}
	for _, principalID := range unique {
		resolved, err := a.deps.Access.ResolvePrincipal(ctx, actor.TenantID, principalID, actor.LegalEntityID)
		if err != nil || strings.TrimSpace(resolved.DisplayName) == "" {
			complete = false
			continue
		}
		cache.values[principalID] = &authority.Principal{ID: resolved.PrincipalID, DisplayName: resolved.DisplayName, Kind: resolved.Kind}
	}
	return context.WithValue(ctx, principalLabelCacheKey{}, cache), complete
}

func cachedPrincipalLabel(ctx context.Context, principalID string) (*authority.Principal, bool) {
	cache, ok := ctx.Value(principalLabelCacheKey{}).(*principalLabelCache)
	if !ok || cache == nil {
		return nil, false
	}
	value, exists := cache.values[strings.TrimSpace(principalID)]
	if !exists || value == nil {
		return nil, true
	}
	copy := *value
	return &copy, true
}
