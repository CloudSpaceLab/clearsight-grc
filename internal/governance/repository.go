package governance

import (
	"context"
	"time"
)

type Repository interface {
	ListPolicies(context.Context, string) ([]RoutingPolicy, error)
	GetPolicy(context.Context, string, string) (RoutingPolicy, error)
	CreatePolicy(context.Context, RoutingPolicy) (RoutingPolicy, error)
	TransitionPolicy(context.Context, string, string, int64, PolicyState, PolicyState, string, string, time.Time) (RoutingPolicy, error)
	PolicyConflicts(context.Context, RoutingPolicy) ([]ConflictFinding, error)
	ListDelegations(context.Context, string) ([]Delegation, error)
	GetDelegation(context.Context, string, string) (Delegation, error)
	CreateDelegation(context.Context, Delegation) (Delegation, error)
	TransitionDelegation(context.Context, string, string, int64, DelegationState, DelegationState, string, string, time.Time) (Delegation, error)
	HasDelegationCycle(context.Context, Delegation) (bool, error)
	DelegationConflicts(context.Context, string, string, string) ([]ConflictFinding, error)
	ActivateDueDelegations(context.Context, time.Time, int) (int, error)
	ExpireDueDelegations(context.Context, time.Time, int) (int, error)
}
