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
	ListAutomationPolicies(context.Context, string) ([]AutomationPolicy, error)
	GetAutomationPolicy(context.Context, string, string, int64) (AutomationPolicy, error)
}
