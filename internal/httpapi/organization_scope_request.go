package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/runtimecontext"
)

var (
	errOrganizationScopeUnavailable = errors.New("organization scope unavailable")
	errOrganizationScopeForbidden   = errors.New("organization scope forbidden")
)

type organizationScopeSelection struct {
	ID  string
	IDs []string
}

func (a *API) resolveOrganizationScopeSelection(ctx context.Context, actor identity.Actor, requested string, includeDescendants bool) (organizationScopeSelection, error) {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		return organizationScopeSelection{}, nil
	}
	if resolver, ok := a.deps.RuntimeContext.(runtimecontext.OrganizationScopeSelectionResolver); ok {
		resolved, err := resolver.ResolveOrganizationScopeSelection(ctx, runtimecontext.Scope{
			TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PrincipalID: actor.PrincipalID,
		}, requested, includeDescendants)
		if err != nil {
			if errors.Is(err, runtimecontext.ErrNotFound) {
				return organizationScopeSelection{}, errOrganizationScopeForbidden
			}
			return organizationScopeSelection{}, errOrganizationScopeUnavailable
		}
		return organizationScopeSelection{ID: resolved.Node.ID, IDs: resolved.IDs}, nil
	}
	resolver, ok := a.deps.RuntimeContext.(runtimecontext.HierarchyResolver)
	if !ok {
		return organizationScopeSelection{}, errOrganizationScopeUnavailable
	}
	hierarchy, err := resolver.ResolveHierarchy(ctx, runtimecontext.Scope{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PrincipalID: actor.PrincipalID,
	})
	if err != nil {
		return organizationScopeSelection{}, errOrganizationScopeUnavailable
	}
	for _, node := range hierarchy.OrganizationScopes {
		if node.ID != requested || !node.Filterable {
			continue
		}
		selection := organizationScopeSelection{ID: requested, IDs: []string{requested}}
		if !includeDescendants {
			return selection, nil
		}
		selection.IDs = selection.IDs[:0]
		for _, candidate := range hierarchy.OrganizationScopes {
			if candidate.Filterable && organizationScopeDescendant(candidate.DepartmentPath, node.DepartmentPath) {
				selection.IDs = append(selection.IDs, candidate.ID)
			}
		}
		if len(selection.IDs) == 0 {
			selection.IDs = []string{requested}
		}
		return selection, nil
	}
	return organizationScopeSelection{}, errOrganizationScopeForbidden
}

func writeOrganizationScopeRequestError(w http.ResponseWriter, err error, forbiddenMessage string) {
	if errors.Is(err, errOrganizationScopeUnavailable) {
		httpx.WriteError(w, http.StatusServiceUnavailable, "organization_scope_unavailable", "Organization scope could not be verified. Try again.")
		return
	}
	httpx.WriteError(w, http.StatusForbidden, "organization_scope_forbidden", forbiddenMessage)
}

func organizationScopeDescendant(candidate, ancestor []string) bool {
	if len(ancestor) == 0 || len(candidate) < len(ancestor) {
		return false
	}
	for index := range ancestor {
		if !strings.EqualFold(strings.TrimSpace(candidate[index]), strings.TrimSpace(ancestor[index])) {
			return false
		}
	}
	return true
}
