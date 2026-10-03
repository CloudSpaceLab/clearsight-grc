package aigovernance

import (
	"context"
	"errors"
	"sort"

	"github.com/CloudSpaceLab/clearsight-grc/internal/aigateway"
)

func (s *Service) simulationBaselineSnapshot(ctx context.Context, input GatewaySimulationInput, baseline *Policy, workload *Workload) (*aigateway.PolicySnapshot, []aigateway.PolicyRevisionRef, error) {
	if baseline == nil {
		return nil, nil, nil
	}
	snapshot := policySnapshot(*baseline)
	if workload == nil {
		return &snapshot, nil, nil
	}
	resolver, ok := s.repo.(gatewayBaselineExceptionRepository)
	if !ok {
		return &snapshot, nil, nil
	}
	exceptions, err := resolver.ActiveGatewayBaselineExceptions(
		ctx,
		input.TenantID,
		baseline.ID,
		baseline.Version,
		workload.ID,
		input.Environment,
		s.now().UTC(),
	)
	if err != nil {
		return nil, nil, err
	}
	if len(exceptions) == 0 {
		return &snapshot, nil, nil
	}
	snapshot, err = applyGatewayBaselineExceptions(snapshot, exceptions)
	if err != nil {
		return nil, nil, errors.Join(ErrInvalid, err)
	}
	refs := make([]aigateway.PolicyRevisionRef, 0, len(exceptions))
	for _, exception := range exceptions {
		refs = append(refs, exception.PolicyRevisionRef)
	}
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].Code != refs[j].Code {
			return refs[i].Code < refs[j].Code
		}
		if refs[i].Version != refs[j].Version {
			return refs[i].Version < refs[j].Version
		}
		return refs[i].ID < refs[j].ID
	})
	return &snapshot, refs, nil
}
