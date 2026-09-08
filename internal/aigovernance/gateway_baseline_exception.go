package aigovernance

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/aigateway"
)

const maxGatewayBaselineExceptionTTL = 30 * 24 * time.Hour

type GatewayBaselineExceptionScope struct {
	TargetBaselineID      string   `json:"target_baseline_id"`
	TargetBaselineVersion int64    `json:"target_baseline_version"`
	WorkloadRecordIDs     []string `json:"workload_record_ids"`
	Environments          []string `json:"environments"`
	WaivedRuleIDs         []string `json:"waived_rule_ids"`
	Justification         string   `json:"justification"`
}

func IsGatewayBaselineExceptionPolicy(policy Policy) bool {
	return strings.HasPrefix(policy.Code, aigateway.GatewayBaselineExceptionCodeRoot+":") && policy.ActionClass == aigateway.GatewayBaselineExceptionActionClass
}

func gatewayBaselineExceptionScope(policy Policy, now time.Time) (GatewayBaselineExceptionScope, error) {
	if !IsGatewayBaselineExceptionPolicy(policy) || policy.EffectiveUntil == nil || !policy.EffectiveUntil.After(now) {
		return GatewayBaselineExceptionScope{}, ErrInvalid
	}
	var scope GatewayBaselineExceptionScope
	if err := json.Unmarshal(policy.Eligibility, &scope); err != nil {
		return GatewayBaselineExceptionScope{}, errors.Join(ErrInvalid, fmt.Errorf("decode baseline exception scope: %w", err))
	}
	scope.TargetBaselineID = strings.TrimSpace(scope.TargetBaselineID)
	scope.Justification = strings.TrimSpace(scope.Justification)
	scope.WorkloadRecordIDs = normalizedExceptionIdentifiers(scope.WorkloadRecordIDs)
	scope.Environments = normalizedExceptionEnvironments(scope.Environments)
	scope.WaivedRuleIDs = normalizedExceptionIdentifiers(scope.WaivedRuleIDs)
	if scope.TargetBaselineID == "" || scope.TargetBaselineVersion < 1 || scope.Justification == "" || len(scope.Justification) > 1000 ||
		len(scope.WorkloadRecordIDs) == 0 || len(scope.WorkloadRecordIDs) > aigateway.MaxBaselineExceptionWorkloads ||
		len(scope.Environments) == 0 || len(scope.Environments) > 3 ||
		len(scope.WaivedRuleIDs) == 0 || len(scope.WaivedRuleIDs) > aigateway.MaxBaselineExceptionRules {
		return GatewayBaselineExceptionScope{}, ErrInvalid
	}
	return scope, nil
}

func validateGatewayBaselineExceptionPolicy(policy Policy, target Policy, workloads []Workload, now time.Time) error {
	scope, err := gatewayBaselineExceptionScope(policy, now)
	if err != nil {
		return err
	}
	if target.ID != scope.TargetBaselineID || target.Version != scope.TargetBaselineVersion || target.Code != aigateway.GatewayBaselinePolicyCode || target.ActionClass != aigateway.GatewayBaselineActionClass {
		return errors.Join(ErrInvalid, fmt.Errorf("baseline exception must target one exact organization baseline revision"))
	}
	if policy.EffectiveUntil == nil || policy.EffectiveUntil.After(now.Add(maxGatewayBaselineExceptionTTL)) {
		return errors.Join(ErrInvalid, fmt.Errorf("baseline exception expiry must be within 30 days"))
	}
	if policy.EffectiveFrom != nil && !policy.EffectiveFrom.Before(*policy.EffectiveUntil) {
		return errors.Join(ErrInvalid, fmt.Errorf("baseline exception validity window is invalid"))
	}
	ruleIDs := make(map[string]struct{}, len(target.Definition.Rules))
	for _, rule := range target.Definition.Rules {
		ruleIDs[strings.TrimSpace(rule.ID)] = struct{}{}
	}
	for _, ruleID := range scope.WaivedRuleIDs {
		if _, ok := ruleIDs[ruleID]; !ok {
			return errors.Join(ErrInvalid, fmt.Errorf("baseline exception references unknown rule %q", ruleID))
		}
	}
	workloadByID := make(map[string]Workload, len(workloads))
	for _, workload := range workloads {
		workloadByID[workload.ID] = workload
	}
	for _, workloadID := range scope.WorkloadRecordIDs {
		workload, ok := workloadByID[workloadID]
		if !ok {
			return errors.Join(ErrInvalid, fmt.Errorf("baseline exception workload scope is unknown"))
		}
		if !containsFold(scope.Environments, workload.Environment) {
			return errors.Join(ErrInvalid, fmt.Errorf("baseline exception environment does not cover its workload scope"))
		}
	}
	if len(policy.Definition.Bindings) != 0 || len(policy.Definition.Rules) != 0 || policy.Definition.ResponseControl.MaxBytes != 0 || len(policy.Definition.ResponseControl.DenyPatterns) != 0 || len(policy.Definition.ResponseControl.RedactPatterns) != 0 {
		return errors.Join(ErrInvalid, fmt.Errorf("baseline exception policies cannot define independent gateway rules"))
	}
	return nil
}

func projectGatewayBaselineException(policy Policy, now time.Time) (aigateway.BaselineException, error) {
	scope, err := gatewayBaselineExceptionScope(policy, now)
	if err != nil {
		return aigateway.BaselineException{}, err
	}
	return aigateway.BaselineException{
		PolicyRevisionRef:     aigateway.PolicyRevisionRef{ID: policy.ID, Code: policy.Code, Version: policy.Version},
		TargetBaselineID:      scope.TargetBaselineID,
		TargetBaselineVersion: scope.TargetBaselineVersion,
		WorkloadRecordIDs:     append([]string(nil), scope.WorkloadRecordIDs...),
		Environments:          append([]string(nil), scope.Environments...),
		WaivedRuleIDs:         append([]string(nil), scope.WaivedRuleIDs...),
		ExpiresAt:             *policy.EffectiveUntil,
	}, nil
}

func normalizedExceptionIdentifiers(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && len(value) <= 256 {
			set[value] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func normalizedExceptionEnvironments(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.ToUpper(strings.TrimSpace(value))
		if value == "PRODUCTION" || value == "TEST" || value == "DEVELOPMENT" {
			set[value] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func containsFold(values []string, value string) bool {
	for _, candidate := range values {
		if strings.EqualFold(candidate, value) {
			return true
		}
	}
	return false
}
