package runtimecontext

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrInvalid  = errors.New("runtime context scope is required")
	ErrNotFound = errors.New("runtime context is unavailable")
)

type Scope struct {
	TenantID      string
	LegalEntityID string
	PrincipalID   string
}

type DisplayContext struct {
	TenantName      string
	LegalEntityName string
	PrincipalName   string
}

type Resolver interface {
	Resolve(context.Context, Scope) (DisplayContext, error)
}

// IdentifierResolver is the empty development adapter. It exposes only the
// exact identifiers supplied by verified request identity and never invents
// organization, legal-entity, role or person labels.
type IdentifierResolver struct{}

func (IdentifierResolver) Resolve(_ context.Context, scope Scope) (DisplayContext, error) {
	scope.TenantID = strings.TrimSpace(scope.TenantID)
	scope.LegalEntityID = strings.TrimSpace(scope.LegalEntityID)
	scope.PrincipalID = strings.TrimSpace(scope.PrincipalID)
	if scope.TenantID == "" || scope.LegalEntityID == "" || scope.PrincipalID == "" {
		return DisplayContext{}, ErrInvalid
	}
	return DisplayContext{TenantName: scope.TenantID, LegalEntityName: scope.LegalEntityID, PrincipalName: scope.PrincipalID}, nil
}

var _ Resolver = IdentifierResolver{}
