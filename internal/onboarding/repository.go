package onboarding

import (
	"context"
	"errors"
)

var (
	ErrStateNotFound   = errors.New("onboarding state not found")
	ErrVersionConflict = errors.New("onboarding state version conflict")
)

type Repository interface {
	Get(context.Context, string, string, string) (State, error)
	Upsert(context.Context, State, int64) (State, error)
}
