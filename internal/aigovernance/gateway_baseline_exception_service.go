package aigovernance

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/aigateway"
)

// CreateGovernedPolicy keeps the generic Automation Policy API while routing
// organization-baseline exceptions through their stricter scope validation.
func (s *Service) CreateGovernedPolicy(ctx context.Context, input CreatePolicyInput) (Policy, error) {
	if gatewayBaselineExceptionShape(input.Code, input.ActionClass) {
		candidate := Policy{
			TenantID:       strings.TrimSpace(input.TenantID),
			Code:           strings.TrimSpace(input.Code),
			Name:           strings.TrimSpace(input.Name),
			ActionClass:    strings.TrimSpace(input.ActionClass),
			Eligibility:    normalizedJSON(input.Eligibility, `{}`),
			Definition:     input.Definition,
			RolloutMode:    input.RolloutMode,
			MakerID:        strings.TrimSpace(input.MakerID),
			EffectiveFrom:  input.EffectiveFrom,
			EffectiveUntil: input.EffectiveUntil,
		}
		if candidate.RolloutMode == "" {
			candidate.RolloutMode = aigateway.RolloutShadow
		}
		if err := s.validateGatewayBaselineExceptionCandidate(ctx, candidate, s.now().UTC()); err != nil {
			return Policy{}, err
		}
	} else if gatewayBaselineExceptionPartialShape(input.Code, input.ActionClass) {
		return Policy{}, errors.Join(ErrInvalid, fmt.Errorf("gateway baseline exception code and action class must be used together"))
	}
	return s.CreatePolicy(ctx, input)
}

// TransitionGovernedPolicy revalidates exception scope at every material
// lifecycle boundary so an expired or drifted exception cannot become approved
// or active merely because the draft was valid when first created.
func (s *Service) TransitionGovernedPolicy(ctx context.Context, action string, input TransitionInput) (Policy, error) {
	if s == nil || s.repo == nil {
		return Policy{}, ErrInvalid
	}
	policy, err := s.repo.Policy(ctx, input.TenantID, input.ID)
	if err != nil {
		return Policy{}, err
	}
	if IsGatewayBaselineExceptionPolicy(policy) {
		switch strings.ToLower(strings.TrimSpace(action)) {
		case "submit", "approve", "activate":
			if err := s.validateGatewayBaselineExceptionCandidate(ctx, policy, s.now().UTC()); err != nil {
				return Policy{}, errors.Join(ErrInvalidTransition, err)
			}
		}
	}
	return s.TransitionPolicy(ctx, action, input)
}

func (s *Service) validateGatewayBaselineExceptionCandidate(ctx context.Context, policy Policy, now time.Time) error {
	if s == nil || s.repo == nil || strings.TrimSpace(policy.TenantID) == "" {
		return ErrInvalid
	}
	if !gatewayBaselineExceptionShape(policy.Code, policy.ActionClass) {
		return errors.Join(ErrInvalid, fmt.Errorf("gateway baseline exception code and action class must be used together"))
	}
	scope, err := gatewayBaselineExceptionScope(policy, now)
	if err != nil {
		return err
	}
	target, err := s.repo.Policy(ctx, policy.TenantID, scope.TargetBaselineID)
	if err != nil {
		return errors.Join(ErrInvalid, fmt.Errorf("gateway baseline exception target is unavailable"))
	}
	workloads := make([]Workload, 0, len(scope.WorkloadRecordIDs))
	for _, workloadID := range scope.WorkloadRecordIDs {
		workload, err := s.repo.Workload(ctx, policy.TenantID, workloadID)
		if err != nil {
			return errors.Join(ErrInvalid, fmt.Errorf("gateway baseline exception workload is unavailable"))
		}
		workloads = append(workloads, workload)
	}
	return validateGatewayBaselineExceptionPolicy(policy, target, workloads, now)
}

func gatewayBaselineExceptionShape(code, actionClass string) bool {
	return strings.HasPrefix(strings.TrimSpace(code), aigateway.GatewayBaselineExceptionCodeRoot+":") &&
		strings.TrimSpace(actionClass) == aigateway.GatewayBaselineExceptionActionClass
}

func gatewayBaselineExceptionPartialShape(code, actionClass string) bool {
	codeMatch := strings.HasPrefix(strings.TrimSpace(code), aigateway.GatewayBaselineExceptionCodeRoot+":")
	classMatch := strings.TrimSpace(actionClass) == aigateway.GatewayBaselineExceptionActionClass
	return codeMatch != classMatch
}
