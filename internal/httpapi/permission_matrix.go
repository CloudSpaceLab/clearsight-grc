package httpapi

import (
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

type permissionRule struct {
	Method     string
	Path       string
	Permission string
}

var administrativePermissions = []permissionRule{
	{http.MethodGet, "/api/v1/authority/integrity", identity.PermissionConfigRead},
	{http.MethodGet, "/api/v1/authority/policies", identity.PermissionConfigRead},
	{http.MethodGet, "/api/v1/governance/policies", identity.PermissionConfigRead},
	{http.MethodPost, "/api/v1/governance/policies", identity.PermissionConfigWrite},
	{http.MethodPost, "/api/v1/governance/policies/{id}/submit", identity.PermissionConfigWrite},
	{http.MethodPost, "/api/v1/governance/policies/{id}/approve", identity.PermissionConfigWrite},
	{http.MethodPost, "/api/v1/governance/policies/{id}/reject", identity.PermissionConfigWrite},
	{http.MethodPost, "/api/v1/governance/policies/{id}/retire", identity.PermissionConfigWrite},
	{http.MethodGet, "/api/v1/governance/delegations", identity.PermissionConfigRead},
	{http.MethodPost, "/api/v1/governance/delegations", identity.PermissionConfigWrite},
	{http.MethodPost, "/api/v1/governance/delegations/{id}/submit", identity.PermissionConfigWrite},
	{http.MethodPost, "/api/v1/governance/delegations/{id}/approve", identity.PermissionConfigWrite},
	{http.MethodPost, "/api/v1/governance/delegations/{id}/revoke", identity.PermissionConfigWrite},
	{http.MethodGet, "/api/v1/compliance/automation-policies", identity.PermissionConfigRead},
	{http.MethodGet, "/api/v1/operations/projections", identity.PermissionPlatformOperationsRead},
	{http.MethodPost, "/api/v1/operations/projections/reconcile", identity.PermissionPlatformOperationsWrite},
	{http.MethodPost, "/api/v1/operations/projections/rebuild", identity.PermissionPlatformOperationsWrite},
	{http.MethodGet, "/api/v1/operations/background-jobs", identity.PermissionPlatformJobsRead},
}

func permissionGate(next http.Handler) http.Handler {
	matrix := http.NewServeMux()
	for _, rule := range administrativePermissions {
		permission := rule.Permission
		matrix.HandleFunc(rule.Method+" "+rule.Path, func(http.ResponseWriter, *http.Request) {})
		_ = permission
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, pattern := matrix.Handler(r)
		if pattern == "" {
			next.ServeHTTP(w, r)
			return
		}
		permission := permissionForPattern(pattern)
		actor, err := identity.Require(r.Context())
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to continue.")
			return
		}
		if !identity.HasPermission(actor, permission) {
			httpx.WriteError(w, http.StatusForbidden, "permission_required", "You do not have permission to use this administrative function.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func permissionForPattern(pattern string) string {
	for _, rule := range administrativePermissions {
		if rule.Method+" "+rule.Path == pattern {
			return rule.Permission
		}
	}
	return ""
}
