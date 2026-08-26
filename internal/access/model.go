package access

import (
	"context"
	"errors"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

var (
	ErrIdentityNotProvisioned = errors.New("enterprise identity is not provisioned")
	ErrPrincipalUnavailable   = errors.New("principal is unavailable")
	ErrPrincipalBatchTooLarge = errors.New("principal resolution batch exceeds the supported limit")
)

const MaxPrincipalBatchSize = 500

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

// PrincipalResolveOutcome preserves input order while keeping an unavailable
// principal isolated from other display-name resolutions in the same batch.
type PrincipalResolveOutcome struct {
	Resolution Resolution
	Err        error
}

// BatchPrincipalResolver is an optional display-identity capability. Command
// authorization continues to use current authority routes; this interface
// avoids one directory query per stored responsibility on record workspaces.
type BatchPrincipalResolver interface {
	ResolvePrincipals(context.Context, string, string, []string) ([]PrincipalResolveOutcome, error)
}
