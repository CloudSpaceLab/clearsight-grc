package rcsa

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrInvalid   = errors.New("RCSA input is invalid")
	ErrNotFound  = errors.New("RCSA record not found")
	ErrDuplicate = errors.New("RCSA record already exists")
	ErrConflict  = errors.New("RCSA record changed")
)

type Repository interface {
	CreateCycle(context.Context, Cycle, []Item, []ItemControl) (Aggregate, error)
	GetCycle(context.Context, Scope, string) (Aggregate, error)
	AttachDistribution(context.Context, Scope, DistributionLink) (DistributionLink, error)
}

func normalizeScope(scope Scope) (Scope, error) {
	scope.TenantID = strings.TrimSpace(scope.TenantID)
	scope.LegalEntityID = strings.TrimSpace(scope.LegalEntityID)
	if scope.TenantID == "" || scope.LegalEntityID == "" || scope.LegalEntityID == "*" {
		return Scope{}, ErrInvalid
	}
	return scope, nil
}
