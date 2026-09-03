package aigovernance

import (
	"fmt"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/aigateway"
)

func runtimeBindingRequirements(workload aigateway.Workload, workloadRequirements []aigateway.BindingRequirement) ([]aigateway.BindingRequirement, error) {
	baselineRequirements := []aigateway.BindingRequirement(nil)
	if workload.Policy.Baseline != nil {
		baselineRequirements = workload.Policy.Baseline.Definition.Bindings
	}
	if len(baselineRequirements) == 0 {
		return append([]aigateway.BindingRequirement(nil), workloadRequirements...), nil
	}

	out := make([]aigateway.BindingRequirement, 0, len(baselineRequirements)+len(workloadRequirements))
	byFact := make(map[string]aigateway.BindingRequirement, len(baselineRequirements)+len(workloadRequirements))
	appendRequirement := func(requirement aigateway.BindingRequirement) error {
		factKey := strings.TrimSpace(requirement.FactKey)
		if existing, ok := byFact[factKey]; ok {
			if existing != requirement {
				return fmt.Errorf("gateway baseline and workload policy define conflicting source fact %q", factKey)
			}
			return nil
		}
		byFact[factKey] = requirement
		out = append(out, requirement)
		return nil
	}
	// Baseline first makes the non-bypassable layer explicit while conflicts fail
	// closed instead of allowing either layer to replace the other's fact source.
	for _, requirement := range baselineRequirements {
		if err := appendRequirement(requirement); err != nil {
			return nil, err
		}
	}
	for _, requirement := range workloadRequirements {
		if err := appendRequirement(requirement); err != nil {
			return nil, err
		}
	}
	return out, nil
}
