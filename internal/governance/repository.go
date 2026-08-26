package governance

import (
	"context"
	"time"
)

type Repository interface {
	ListPolicies(context.Context, string) ([]RoutingPolicy, error)
	ListPoliciesForEntity(context.Context, string, string, int) ([]RoutingPolicy, error)
	GetPolicy(context.Context, string, string) (RoutingPolicy, error)
	GetPolicyForEntity(context.Context, string, string, string) (RoutingPolicy, error)
	CreatePolicy(context.Context, RoutingPolicy) (RoutingPolicy, error)
	TransitionPolicy(context.Context, string, string, string, int64, PolicyState, PolicyState, string, string, time.Time) (RoutingPolicy, error)
	ActivatePolicy(context.Context, string, string, string, int64, string, string, time.Time) (RoutingPolicy, error)
	CreatePolicyRevision(context.Context, string, string, int64, string, []byte, string, time.Time) (RoutingPolicyRevision, error)
	PendingPolicyRevision(context.Context, string, string) (RoutingPolicyRevision, error)
	ActivatePolicyRevision(context.Context, string, string, int64, int, string, string, time.Time) (RoutingPolicy, error)
	PolicyConflicts(context.Context, RoutingPolicy) ([]ConflictFinding, error)
	EscalationReferenceConflicts(context.Context, string, []byte) ([]ConflictFinding, error)
	ListDelegations(context.Context, string) ([]Delegation, error)
	ListDelegationsForEntity(context.Context, string, string, int) ([]Delegation, error)
	GetDelegation(context.Context, string, string) (Delegation, error)
	GetDelegationForEntity(context.Context, string, string, string) (Delegation, error)
	CreateDelegation(context.Context, Delegation) (Delegation, error)
	TransitionDelegation(context.Context, string, string, string, int64, DelegationState, DelegationState, string, string, time.Time) (Delegation, error)
	HasDelegationCycle(context.Context, Delegation) (bool, error)
	DelegationConflicts(context.Context, string, string, string) ([]ConflictFinding, error)
	ActivateDelegation(context.Context, string, string, string, int64, string, string, time.Time) (Delegation, error)
	ActivateDueDelegations(context.Context, time.Time, int) (int, error)
	ExpireDueDelegations(context.Context, time.Time, int) (int, error)
}
