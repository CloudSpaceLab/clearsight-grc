package httpapi

import "net/http"

// productionRoutes is the single executable route catalog. Feature modules may
// own their route specs, but runtime registration and the OpenAPI parity test
// must consume this same aggregate so no route can bypass contract validation.
func (a *API) productionRoutes() []routeSpec {
	base := a.routes()
	distributions := a.formDistributionRoutes()
	communications := a.formCommunicationRoutes()
	proposals := a.formProposalRoutes()
	policies := a.formPolicyRoutes()
	activity := a.activityRoutes()
	gatewayTransports := a.aiGatewayTransportRoutes()
	routes := make([]routeSpec, 0, len(base)+len(distributions)+len(communications)+len(proposals)+len(policies)+len(activity)+len(gatewayTransports))
	routes = append(routes, base...)
	routes = append(routes, distributions...)
	routes = append(routes, communications...)
	routes = append(routes, proposals...)
	routes = append(routes, policies...)
	routes = append(routes, activity...)
	routes = append(routes, gatewayTransports...)
	return routes
}

func (a *API) registerProductionRoutes(mux *http.ServeMux) {
	routes := a.productionRoutes()
	if err := validateRoutes(routes); err != nil {
		panic(err)
	}
	for _, spec := range routes {
		handler := spec.Handler
		if spec.Command != nil {
			if spec.RawCommand {
				handler = a.rawCommand(spec.Command.Name, spec.Command.Policy, handler)
			} else {
				handler = a.command(spec.Command.Name, spec.Command.Policy, handler)
			}
		}
		handler = a.routeAccess(spec, handler)
		mux.HandleFunc(spec.Method+" "+spec.Path, handler)
	}
}
