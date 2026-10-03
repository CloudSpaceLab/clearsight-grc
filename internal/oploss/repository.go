package oploss

import (
	"context"
	"errors"
)

var (
	ErrInvalid         = errors.New("operational loss input is invalid")
	ErrNotFound        = errors.New("operational loss not found")
	ErrDuplicate       = errors.New("operational loss already exists")
	ErrVersionConflict = errors.New("operational loss version conflict")
	ErrRecoveryLimit   = errors.New("operational loss recovery exceeds gross loss")
)

type Repository interface {
	Create(context.Context, Loss, Event) (Loss, error)
	Update(context.Context, Scope, Loss, int64, Event) (Loss, error)
	AddRecovery(context.Context, Scope, string, int64, Recovery, Event) (Loss, Recovery, error)
	Get(context.Context, Scope, string) (Aggregate, error)
	ResolveLegalEntity(context.Context, string, string) (string, error)
}
