package httpapi

import (
	"fmt"
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func (a *API) registerIdentityAccessRoutes(mux *http.ServeMux) {
	routes := []routeSpec{
		withPermission(read("/api/v1/access/overview", a.identityAccessOverview), identity.PermissionIdentityRead),
		withPermission(write(http.MethodPost, "/api/v1/access/scim-sources", a.createSCIMSource, nil), identity.PermissionIdentityConfigure),
		withPermission(write(http.MethodPost, "/api/v1/access/scim-sources/{id}/rotate-token", a.rotateSCIMSourceToken, nil), identity.PermissionIdentityConfigure),
		withPermission(write(http.MethodPost, "/api/v1/access/scim-sources/{id}/revoke", a.revokeSCIMSource, nil), identity.PermissionIdentityConfigure),
		withPermission(write(http.MethodPost, "/api/v1/access/group-role-bindings", a.createDirectoryGroupRoleBinding, nil), identity.PermissionIdentityConfigure),
		withPermission(write(http.MethodPost, "/api/v1/access/group-role-bindings/{id}/retire", a.retireDirectoryGroupRoleBinding, nil), identity.PermissionIdentityConfigure),
		withPermission(operation("/api/v1/access/escalations/preview", a.previewEscalation, nil), identity.PermissionIdentityRead),
	}
	if err := validateRoutes(routes); err != nil {
		panic(fmt.Errorf("identity access routes: %w", err))
	}
	for _, spec := range routes {
		mux.HandleFunc(spec.Method+" "+spec.Path, a.routeAccess(spec, spec.Handler))
	}
}
