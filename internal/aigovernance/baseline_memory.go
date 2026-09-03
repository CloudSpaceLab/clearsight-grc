package aigovernance

import (
	"context"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/aigateway"
)

func (r *MemoryRepository) ActiveGatewayBaseline(_ context.Context, tenantID string) (Policy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	now := time.Now().UTC()
	var selected Policy
	found := false
	for _, policy := range r.policies {
		if policy.TenantID != tenantID || policy.Code != aigateway.GatewayBaselinePolicyCode || policy.ActionClass != aigateway.GatewayBaselineActionClass || policy.Status != "ACTIVE" {
			continue
		}
		if policy.EffectiveFrom != nil && policy.EffectiveFrom.After(now) {
			continue
		}
		if policy.EffectiveUntil != nil && !policy.EffectiveUntil.After(now) {
			continue
		}
		if !found || policy.Version > selected.Version {
			selected = policy
			found = true
		}
	}
	if !found {
		return Policy{}, ErrNotFound
	}
	return selected, nil
}
