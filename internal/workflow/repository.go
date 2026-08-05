package workflow

import (
	"context"
	"errors"
)

var (
	ErrTaskNotFound      = errors.New("workflow task not found")
	ErrVersionConflict   = errors.New("workflow task version conflict")
	ErrInvalidTransition = errors.New("invalid workflow transition")
)

type Repository interface {
	Create(context.Context, CreateInput) (Task, error)
	Get(context.Context, string) (Task, error)
	List(context.Context, ListFilter) ([]Task, error)
	Transition(context.Context, string, TransitionInput) (Task, error)
}
