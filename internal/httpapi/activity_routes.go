package httpapi

import "github.com/CloudSpaceLab/clearsight-grc/internal/identity"

func (a *API) activityRoutes() []routeSpec {
	return []routeSpec{
		withPermission(read("/api/v1/system-activity", a.listSystemActivity), identity.PermissionPlatformOperationsRead),
		withPermission(read("/api/v1/system-activity/{event_id}", a.getSystemActivity), identity.PermissionPlatformOperationsRead),
	}
}
