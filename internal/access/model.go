package access

import (
	"context"
	"errors"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

var (
	ErrIdentityNotProvisioned = errors.New("enterprise identity is not provisioned")
	ErrPrincipalUnavailable   = errors.New("principal is unavailable")
)

type Resolution struct {
	TenantID         string
	PrincipalID      string
	LegalEntityID    string
	DisplayName      string
	Kind             string
	RoleCodes        []string
	PermissionCodes  []string
	DepartmentGrants []identity.DepartmentGrant
}

type Resolver interface {
	ResolveOIDC(context.Context, string, string, string, string) (Resolution, error)
	ResolvePrincipal(context.Context, string, string, string) (Resolution, error)
}
