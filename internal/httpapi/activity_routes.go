package httpapi

import (
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func (a *API) activityRoutes() []routeSpec {
	return []routeSpec{
		withPermission(read("/api/v1/system-activity", a.listSystemActivity), identity.PermissionPlatformOperationsRead),
		withPermission(read("/api/v1/system-activity/{event_id}", a.getSystemActivity), identity.PermissionPlatformOperationsRead),
		withPermission(write(http.MethodPost, "/api/v1/audit-exports", a.createAuditExport, nil), identity.PermissionAuditExport),
		withPermission(read("/api/v1/audit-exports/{id}", a.getAuditExport), identity.PermissionAuditExport),
		withPermission(read("/api/v1/audit-exports/{id}/download", a.downloadAuditExport), identity.PermissionAuditExport),
	}
}
