package formpolicy

import (
	"context"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

type Repository interface {
	CreatePolicy(context.Context, Policy) (Policy, error)
	GetPolicy(context.Context, string, string, string) (Policy, error)
	ListPolicies(context.Context, string, string, int) ([]Policy, error)
	NextPolicyVersion(context.Context, string, string, string) (int64, error)
	UpdatePolicy(context.Context, Policy, int64) (Policy, error)
	HasShadowHistory(context.Context, string, string, string, int64) (bool, error)
	SaveSimulation(context.Context, SimulationReceipt) (SimulationReceipt, error)
	GetSimulation(context.Context, string, string, string) (SimulationReceipt, error)
	CreateExecution(context.Context, ExecutionReceipt) (ExecutionReceipt, bool, error)
	OpenEpisode(context.Context, AdverseEpisode) (AdverseEpisode, bool, error)
}

type FormReader interface {
	GetDistributionFormRevision(context.Context, string, string, string, int64) (evidence.DistributionFormRevision, error)
}

type CompletedResponseReader interface {
	ListCompletedResponses(context.Context, evidence.CompletedResponseQuery) (evidence.CompletedResponsePage, error)
}

// ActivationAuthority revalidates external governance dependencies immediately
// before a policy becomes executable. Implementations must fail closed when the
// referenced Automation Policy, subject resolver, Matter route, service actor or
// current authorizer cannot be confirmed.
type ActivationAuthority interface {
	ValidatePolicyActivation(context.Context, Actor, Policy) error
}
