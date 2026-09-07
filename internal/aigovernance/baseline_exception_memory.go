package aigovernance

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/aigateway"
)

func (r *MemoryRepository) ActiveGatewayBaselineExceptions(_ context.Context, tenantID, baselineID string, baselineVersion int64, workloadRecordID, environment string, now time.Time) ([]aigateway.BaselineException, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]aigateway.BaselineException, 0, 4)
	for _, policy := range r.policies {
		if policy.TenantID != tenantID || policy.Status != "ACTIVE" || policy.RolloutMode != aigateway.RolloutEnforce || !IsGatewayBaselineExceptionPolicy(policy) {
			continue
		}
		if policy.EffectiveFrom != nil && policy.EffectiveFrom.After(now) {
			continue
		}
		projected, err := projectGatewayBaselineException(policy, now)
		if err != nil {
			return nil, err
		}
		if projected.TargetBaselineID != baselineID || projected.TargetBaselineVersion != baselineVersion || !containsFold(projected.WorkloadRecordIDs, workloadRecordID) || !containsFold(projected.Environments, environment) {
			continue
		}
		out = append(out, projected)
		if len(out) > 8 {
			return nil, fmt.Errorf("too many applicable gateway baseline exceptions")
		}
	}
	return out, nil
}

func containsFold(values []string, value string) bool {
	for _, candidate := range values {
		if strings.EqualFold(candidate, value) {
			return true
		}
	}
	return false
}
