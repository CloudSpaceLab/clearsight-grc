package autonomy

import "context"

type Repository interface {
	InsertSignal(context.Context, Signal) (bool, error)
	UpsertDrift(context.Context, Drift) error
	ListDrifts(context.Context, string) ([]Drift, error)
}
