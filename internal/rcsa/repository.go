package rcsa

import (
	"context"
	"errors"
)

var (
	ErrInvalid         = errors.New("rcsa input is invalid")
	ErrNotFound        = errors.New("rcsa cycle not found")
	ErrDuplicate       = errors.New("rcsa cycle already exists")
	ErrVersionConflict = errors.New("rcsa cycle version conflict")
)

type Repository interface {
	Create(context.Context, Cycle, []RiskSnapshot, []ControlSnapshot, Event) (Aggregate, error)
	Get(context.Context, Scope, string) (Aggregate, error)
	ResolveLegalEntity(context.Context, string, string) (string, error)
}

type PopulationResolver interface {
	ResolvePopulation(context.Context, Scope, []string) (Population, error)
}
