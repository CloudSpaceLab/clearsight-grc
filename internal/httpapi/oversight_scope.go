package httpapi

import (
	"net/http"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/runtimecontext"
)

func (a *API) oversightScopeForRequest(w http.ResponseWriter, r *http.Request, actor identity.Actor) (oversight.Scope, bool) {
	scope := oversight.Scope{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID}
	requested := strings.TrimSpace(r.URL.Query().Get("organization_scope_id"))
	if requested == "" {
		return scope, true
	}
	resolver, ok := a.deps.RuntimeContext.(runtimecontext.HierarchyResolver)
	if !ok {
		httpx.WriteError(w, http.StatusServiceUnavailable, "organization_scope_unavailable", "Organization scope could not be verified. Try again.")
		return oversight.Scope{}, false
	}
	hierarchy, err := resolver.ResolveHierarchy(r.Context(), runtimecontext.Scope{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PrincipalID: actor.PrincipalID,
	})
	if err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "organization_scope_unavailable", "Organization scope could not be verified. Try again.")
		return oversight.Scope{}, false
	}
	for _, node := range hierarchy.OrganizationScopes {
		if node.ID != requested || !node.Filterable {
			continue
		}
		scope.OrganizationScopeID = requested
		for _, candidate := range hierarchy.OrganizationScopes {
			if candidate.Filterable && organizationScopeDescendant(candidate.DepartmentPath, node.DepartmentPath) {
				scope.OrganizationScopeIDs = append(scope.OrganizationScopeIDs, candidate.ID)
			}
		}
		if len(scope.OrganizationScopeIDs) == 0 {
			scope.OrganizationScopeIDs = []string{requested}
		}
		return scope, true
	}
	httpx.WriteError(w, http.StatusForbidden, "organization_scope_forbidden", "This organization scope is not available for Home.")
	return oversight.Scope{}, false
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
