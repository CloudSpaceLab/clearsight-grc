package autonomy

import (
	"context"
	"time"
)

type Repository interface {
	Ingest(context.Context, Signal, Drift) (bool, error)
	Resolve(context.Context, Signal, string, time.Time) (bool, error)
	ListDrifts(context.Context, string) ([]Drift, error)
	ListAutomationPolicies(context.Context, string) ([]AutomationPolicy, error)
}
