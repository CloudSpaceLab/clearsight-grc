package httpapi

import (
	"errors"
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) oversightScopeForRequest(w http.ResponseWriter, r *http.Request, actor identity.Actor) (oversight.Scope, bool) {
	scope := oversight.Scope{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID}
	selection, err := a.resolveOrganizationScopeSelection(r.Context(), actor, r.URL.Query().Get("organization_scope_id"), true)
	if err == nil {
		scope.OrganizationScopeID = selection.ID
		scope.OrganizationScopeIDs = selection.IDs
		return scope, true
	}
	if errors.Is(err, errOrganizationScopeUnavailable) {
		httpx.WriteError(w, http.StatusServiceUnavailable, "organization_scope_unavailable", "Organization scope could not be verified. Try again.")
		return oversight.Scope{}, false
	}
	httpx.WriteError(w, http.StatusForbidden, "organization_scope_forbidden", "This organization scope is not available for Home.")
	return oversight.Scope{}, false
}
