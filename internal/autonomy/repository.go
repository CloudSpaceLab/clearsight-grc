package autonomy

import (
	"context"
	"errors"
	"time"
)

var ErrAutomationPolicyNotFound = errors.New("automation policy not found")

type Repository interface {
	Ingest(context.Context, Signal, Drift) (bool, error)
	Resolve(context.Context, Signal, string, time.Time) (bool, error)
	ListDrifts(context.Context, string) ([]Drift, error)
	// ListAutomationPolicies returns the latest revision per code in code order,
	// bounded to AutomationPolicyListLimit+1 so the service can report truncation.
	ListAutomationPolicies(context.Context, string) ([]AutomationPolicy, error)
	GetAutomationPolicy(context.Context, string, string, int64) (AutomationPolicy, error)
}
