package autonomy

import "context"

type Repository interface {
	Ingest(context.Context, Signal, Drift) (bool, error)
	ListDrifts(context.Context, string) ([]Drift, error)
	ListAutomationPolicies(context.Context, string) ([]AutomationPolicy, error)
}
