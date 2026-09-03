package httpapi

import (
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func (a *API) aiGatewayTransportRoutes() []routeSpec {
	base := "/api/v1/ai-governance/gateway-configs"
	return []routeSpec{
		withPermission(read(base, a.listAIGatewayTransports), identity.PermissionConfigRead),
		withPermission(read(base+"/active", a.getActiveAIGatewayTransport), identity.PermissionConfigRead),
		withPermission(read(base+"/{id}", a.getAIGatewayTransport), identity.PermissionConfigRead),
		withPermission(write(http.MethodPost, base, a.createAIGatewayTransport, bindJSONIdentity(false, "maker_id")), identity.PermissionConfigWrite),
		withPermission(write(http.MethodPost, base+"/{id}/submit", a.aiGatewayTransportAction("submit"), nil), identity.PermissionConfigWrite),
		withPermission(write(http.MethodPost, base+"/{id}/approve", a.aiGatewayTransportAction("approve"), nil), identity.PermissionConfigWrite),
		withPermission(write(http.MethodPost, base+"/{id}/activate", a.aiGatewayTransportAction("activate"), nil), identity.PermissionConfigWrite),
		withPermission(write(http.MethodPost, base+"/{id}/suspend", a.aiGatewayTransportAction("suspend"), nil), identity.PermissionConfigWrite),
		withPermission(write(http.MethodPost, base+"/{id}/retire", a.aiGatewayTransportAction("retire"), nil), identity.PermissionConfigWrite),
	}
}
