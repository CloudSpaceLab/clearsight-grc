package aigateway

import "net/http"

type RouteAccess string

const (
	RoutePublicHealth RouteAccess = "PUBLIC_HEALTH"
	RouteWorkload     RouteAccess = "WORKLOAD_AUTHENTICATED"
	RouteMetrics      RouteAccess = "METRICS_AUTHENTICATED"
)

// GatewayRoute is the executable route/access inventory for the isolated
// ai-gateway process. It does not grant access to the main ClearSight API.
type GatewayRoute struct {
	Method string
	Path   string
	Access RouteAccess
}

type gatewayRouteSpec struct {
	GatewayRoute
	handler func(*HTTPHandler, http.ResponseWriter, *http.Request)
}

var gatewayRouteSpecs = []gatewayRouteSpec{
	{GatewayRoute: GatewayRoute{Method: http.MethodGet, Path: "/health/live", Access: RoutePublicHealth}, handler: (*HTTPHandler).live},
	{GatewayRoute: GatewayRoute{Method: http.MethodGet, Path: "/health/ready", Access: RoutePublicHealth}, handler: (*HTTPHandler).ready},
	{GatewayRoute: GatewayRoute{Method: http.MethodGet, Path: "/health/config", Access: RouteMetrics}, handler: (*HTTPHandler).transportStatus},
	{GatewayRoute: GatewayRoute{Method: http.MethodGet, Path: "/metrics", Access: RouteMetrics}, handler: (*HTTPHandler).metrics},
	{GatewayRoute: GatewayRoute{Method: http.MethodGet, Path: "/v1/models", Access: RouteWorkload}, handler: (*HTTPHandler).models},
	{GatewayRoute: GatewayRoute{Method: http.MethodPost, Path: "/v1/chat/completions", Access: RouteWorkload}, handler: (*HTTPHandler).chatCompletions},
	{GatewayRoute: GatewayRoute{Method: http.MethodPost, Path: "/v1/responses", Access: RouteWorkload}, handler: (*HTTPHandler).responses},
}

func GatewayRoutes() []GatewayRoute {
	routes := make([]GatewayRoute, len(gatewayRouteSpecs))
	for index, spec := range gatewayRouteSpecs {
		routes[index] = spec.GatewayRoute
	}
	return routes
}

func (h *HTTPHandler) registerRoutes() {
	for _, spec := range gatewayRouteSpecs {
		spec := spec
		h.mux.HandleFunc(spec.Method+" "+spec.Path, func(writer http.ResponseWriter, request *http.Request) {
			spec.handler(h, writer, request)
		})
	}
}
