package aigovernance

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/aigateway"
)

type gatewayBaselineRepository interface {
	ActiveGatewayBaseline(context.Context, string) (Policy, error)
}

type baselineCacheEntry struct {
	policy    aigateway.PolicySnapshot
	found     bool
	expiresAt time.Time
}

type baselineCacheSlot struct {
	mu    sync.Mutex
	entry baselineCacheEntry
}

func (p *RuntimeProvider) activeGatewayBaseline(ctx context.Context, tenantID string) (aigateway.PolicySnapshot, bool, error) {
	if p == nil || p.repo == nil {
		return aigateway.PolicySnapshot{}, false, ErrNotFound
	}
	resolver, ok := p.repo.(gatewayBaselineRepository)
	if !ok {
		return aigateway.PolicySnapshot{}, false, nil
	}
	raw, _ := p.baselineCache.LoadOrStore(tenantID, &baselineCacheSlot{})
	slot := raw.(*baselineCacheSlot)
	slot.mu.Lock()
	defer slot.mu.Unlock()

	now := p.now().UTC()
	if now.Before(slot.entry.expiresAt) {
		return clonePolicySnapshot(slot.entry.policy), slot.entry.found, nil
	}
	policy, err := resolver.ActiveGatewayBaseline(ctx, tenantID)
	if errors.Is(err, ErrNotFound) {
		slot.entry = baselineCacheEntry{expiresAt: now.Add(p.baselineTTL)}
		return aigateway.PolicySnapshot{}, false, nil
	}
	if err != nil {
		return aigateway.PolicySnapshot{}, false, err
	}
	snapshot := aigateway.PolicySnapshot{
		ID: policy.ID, Code: policy.Code, Version: policy.Version,
		RolloutMode: policy.RolloutMode, Definition: policy.Definition,
	}
	slot.entry = baselineCacheEntry{policy: clonePolicySnapshot(snapshot), found: true, expiresAt: now.Add(p.baselineTTL)}
	return snapshot, true, nil
}

func clonePolicySnapshot(value aigateway.PolicySnapshot) aigateway.PolicySnapshot {
	value.Definition.Bindings = append([]aigateway.BindingRequirement(nil), value.Definition.Bindings...)
	value.Definition.Rules = append([]aigateway.PolicyRule(nil), value.Definition.Rules...)
	value.Definition.ResponseControl.DenyPatterns = append([]string(nil), value.Definition.ResponseControl.DenyPatterns...)
	value.Definition.ResponseControl.RedactPatterns = append([]string(nil), value.Definition.ResponseControl.RedactPatterns...)
	value.Baseline = nil
	return value
}
