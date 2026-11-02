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

type ScopeKind string

const (
	ScopeKindOrganization     ScopeKind = "ORGANIZATION"
	ScopeKindLegalEntity      ScopeKind = "LEGAL_ENTITY"
	ScopeKindOrganizationUnit ScopeKind = "ORGANIZATION_UNIT"
	ScopeKindBranch           ScopeKind = "BRANCH"
	ScopeKindDepartment       ScopeKind = "DEPARTMENT"
	ScopeKindFunction         ScopeKind = "FUNCTION"
	ScopeKindBusinessUnit     ScopeKind = "BUSINESS_UNIT"
	ScopeKindCriticalService  ScopeKind = "CRITICAL_SERVICE"
)

type HierarchyState string

const (
	HierarchyComplete    HierarchyState = "COMPLETE"
	HierarchyCurrentOnly HierarchyState = "CURRENT_ONLY"
	HierarchyTruncated   HierarchyState = "TRUNCATED"
	HierarchyUnavailable HierarchyState = "UNAVAILABLE"
)

type ScopeNode struct {
	ID             string    `json:"id"`
	Code           string    `json:"code,omitempty"`
	Name           string    `json:"name"`
	Kind           ScopeKind `json:"kind"`
	ParentID       string    `json:"parent_id,omitempty"`
	Jurisdiction   string    `json:"jurisdiction,omitempty"`
	DepartmentPath []string  `json:"department_path,omitempty"`
	Current        bool      `json:"current,omitempty"`
	Filterable     bool      `json:"filterable,omitempty"`
}

type ScopeHierarchy struct {
	State                       HierarchyState `json:"state"`
	Root                        ScopeNode      `json:"root"`
	Current                     ScopeNode      `json:"current"`
	LegalEntities               []ScopeNode    `json:"legal_entities"`
	OrganizationScopes          []ScopeNode    `json:"organization_scopes,omitempty"`
	OrganizationScopesTruncated bool           `json:"organization_scopes_truncated,omitempty"`
}

type OrganizationScopeSearchPage struct {
	Items   []ScopeNode `json:"items"`
	HasMore bool        `json:"has_more"`
}

type OrganizationScopeSelection struct {
	Node ScopeNode
	IDs  []string
}

type OrganizationScopeSelectionResolver interface {
	ResolveOrganizationScopeSelection(context.Context, Scope, string, bool) (OrganizationScopeSelection, error)
}

type Resolver interface {
	Resolve(context.Context, Scope) (DisplayContext, error)
}

// HierarchyResolver is an optional additive capability on a runtime-context
// resolver. It returns only organization scopes the verified principal is
// currently eligible to operate in. It does not grant authority or switch the
// request actor's legal-entity context.
type HierarchyResolver interface {
	ResolveHierarchy(context.Context, Scope) (ScopeHierarchy, error)
}

// OrganizationScopeSearcher is an optional bounded discovery capability for
// authorized areas that are not carried in the compact runtime hierarchy.
type OrganizationScopeSearcher interface {
	SearchOrganizationScopes(context.Context, Scope, string, int) (OrganizationScopeSearchPage, error)
}

func CurrentHierarchy(scope Scope, display DisplayContext, state HierarchyState) ScopeHierarchy {
	root := ScopeNode{
		ID:   strings.TrimSpace(scope.TenantID),
		Code: strings.TrimSpace(scope.TenantID),
		Name: strings.TrimSpace(display.TenantName),
		Kind: ScopeKindOrganization,
	}
	current := ScopeNode{
		ID:       strings.TrimSpace(scope.LegalEntityID),
		Code:     strings.TrimSpace(scope.LegalEntityID),
		Name:     strings.TrimSpace(display.LegalEntityName),
		Kind:     ScopeKindLegalEntity,
		ParentID: root.ID,
		Current:  true,
	}
	return ScopeHierarchy{
		State:         state,
		Root:          root,
		Current:       current,
		LegalEntities: []ScopeNode{current},
	}
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

func (r IdentifierResolver) ResolveHierarchy(ctx context.Context, scope Scope) (ScopeHierarchy, error) {
	display, err := r.Resolve(ctx, scope)
	if err != nil {
		return ScopeHierarchy{}, err
	}
	return CurrentHierarchy(scope, display, HierarchyCurrentOnly), nil
}

var _ Resolver = IdentifierResolver{}
var _ HierarchyResolver = IdentifierResolver{}
