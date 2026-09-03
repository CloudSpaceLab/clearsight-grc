package aigovernance

import (
	"context"
	"time"
)

type Repository interface {
	CreatePolicy(context.Context, Policy) (Policy, error)
	NextPolicyVersion(context.Context, string, string) (int64, error)
	HasShadowHistory(context.Context, string, string, int64) (bool, error)
	Policy(context.Context, string, string) (Policy, error)
	ListPolicies(context.Context, string, int) ([]Policy, error)
	UpdatePolicy(context.Context, Policy, int64) (Policy, error)
	CreateWorkload(context.Context, Workload) (Workload, error)
	NextWorkloadVersion(context.Context, string, string) (int64, error)
	Workload(context.Context, string, string) (Workload, error)
	WorkloadByCredential(context.Context, [32]byte) (Workload, Policy, error)
	ListWorkloads(context.Context, string, int) ([]Workload, error)
	UpdateWorkload(context.Context, Workload, int64) (Workload, error)
	CreateGatewayTransport(context.Context, GatewayTransportRevision) (GatewayTransportRevision, error)
	NextGatewayTransportVersion(context.Context, string, string) (int64, error)
	GatewayTransport(context.Context, string, string) (GatewayTransportRevision, error)
	ListGatewayTransports(context.Context, string, string, int) ([]GatewayTransportRevision, error)
	ActiveGatewayTransport(context.Context, string, string) (GatewayTransportRevision, error)
	UpdateGatewayTransport(context.Context, GatewayTransportRevision, int64) (GatewayTransportRevision, error)
	ActivateGatewayTransport(context.Context, GatewayTransportRevision, int64) (GatewayTransportRevision, error)
	IngestReceipt(context.Context, DecisionReceipt) (bool, error)
	CreateGrant(context.Context, ExecutionGrant, [32]byte) (ExecutionGrant, error)
	ConsumeGrant(context.Context, string, string, string, [32]byte, time.Time) (ExecutionGrant, error)
	MaintainRetention(context.Context, time.Time, int) (int, error)
}
