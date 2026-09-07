package aigovernance

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/aigateway"
)

type gatewayBaselineExceptionRepository interface {
	ActiveGatewayBaselineExceptions(context.Context, string, string, int64, string, string, time.Time) ([]aigateway.BaselineException, error)
}

type baselineExceptionCacheEntry struct {
	exceptions []aigateway.BaselineException
	expiresAt  time.Time
}

type baselineExceptionCacheSlot struct {
	mu    sync.Mutex
	entry baselineExceptionCacheEntry
}

func (p *RuntimeProvider) activeGatewayBaselineExceptions(ctx context.Context, tenantID string, baseline aigateway.PolicySnapshot, workloadRecordID, environment string) ([]aigateway.BaselineException, error) {
	resolver, ok := p.repo.(gatewayBaselineExceptionRepository)
	if !ok || baseline.ID == "" {
		return nil, nil
	}
	key := fmt.Sprintf("%s|%s|%d|%s|%s", tenantID, baseline.ID, baseline.Version, workloadRecordID, environment)
	raw, _ := p.exceptionCache.LoadOrStore(key, &baselineExceptionCacheSlot{})
	slot := raw.(*baselineExceptionCacheSlot)
	slot.mu.Lock()
	defer slot.mu.Unlock()

	now := p.now().UTC()
	if now.Before(slot.entry.expiresAt) {
		return cloneBaselineExceptions(slot.entry.exceptions), nil
	}
	exceptions, err := resolver.ActiveGatewayBaselineExceptions(ctx, tenantID, baseline.ID, baseline.Version, workloadRecordID, environment, now)
	if err != nil {
		return nil, err
	}
	expiresAt := now.Add(p.baselineTTL)
	for _, exception := range exceptions {
		if exception.ExpiresAt.Before(expiresAt) {
			expiresAt = exception.ExpiresAt
		}
	}
	slot.entry = baselineExceptionCacheEntry{exceptions: cloneBaselineExceptions(exceptions), expiresAt: expiresAt}
	return exceptions, nil
}

func applyGatewayBaselineExceptions(baseline aigateway.PolicySnapshot, exceptions []aigateway.BaselineException) (aigateway.PolicySnapshot, error) {
	if len(exceptions) == 0 {
		return baseline, nil
	}
	if len(exceptions) > 8 {
		return aigateway.PolicySnapshot{}, fmt.Errorf("too many applicable gateway baseline exceptions")
	}
	ruleIDs := make(map[string]struct{}, len(baseline.Definition.Rules))
	for _, rule := range baseline.Definition.Rules {
		ruleIDs[rule.ID] = struct{}{}
	}
	waived := make(map[string]struct{})
	attribution := make([]aigateway.Obligation, 0, len(exceptions))
	for _, exception := range exceptions {
		if exception.TargetBaselineID != baseline.ID || exception.TargetBaselineVersion != baseline.Version || exception.ID == "" || exception.Version < 1 {
			return aigateway.PolicySnapshot{}, fmt.Errorf("gateway baseline exception target is invalid")
		}
		for _, ruleID := range exception.WaivedRuleIDs {
			if _, ok := ruleIDs[ruleID]; !ok {
				return aigateway.PolicySnapshot{}, fmt.Errorf("gateway baseline exception references unknown rule %q", ruleID)
			}
			waived[ruleID] = struct{}{}
		}
		attribution = append(attribution, aigateway.Obligation{Code: fmt.Sprintf("BASELINE_EXCEPTION_REVISION/%s/%d", exception.ID, exception.Version)})
	}
	sort.Slice(attribution, func(i, j int) bool { return attribution[i].Code < attribution[j].Code })

	rules := make([]aigateway.PolicyRule, 0, len(baseline.Definition.Rules)+1)
	for _, rule := range baseline.Definition.Rules {
		if _, skip := waived[rule.ID]; skip {
			continue
		}
		if len(rule.Obligations)+len(attribution) > 32 {
			return aigateway.PolicySnapshot{}, fmt.Errorf("gateway baseline exception attribution exceeds obligation limit")
		}
		rule.Obligations = append(append([]aigateway.Obligation(nil), rule.Obligations...), attribution...)
		rules = append(rules, rule)
	}
	defaultAction := baseline.Definition.DefaultAction
	if defaultAction == "" {
		defaultAction = aigateway.DecisionAllow
	}
	rules = append(rules, aigateway.PolicyRule{
		ID: "baseline-exception-default-attribution", Priority: 0,
		FactKey: aigateway.FactPromptInjectionRisk, Operator: "EXISTS",
		Action: defaultAction, ReasonCode: "POLICY_DEFAULT", Obligations: append([]aigateway.Obligation(nil), attribution...),
	})
	baseline.Definition.Rules = rules
	return baseline, nil
}

func cloneBaselineExceptions(values []aigateway.BaselineException) []aigateway.BaselineException {
	out := make([]aigateway.BaselineException, len(values))
	for index, value := range values {
		out[index] = value
		out[index].WorkloadRecordIDs = append([]string(nil), value.WorkloadRecordIDs...)
		out[index].Environments = append([]string(nil), value.Environments...)
		out[index].WaivedRuleIDs = append([]string(nil), value.WaivedRuleIDs...)
	}
	return out
}
