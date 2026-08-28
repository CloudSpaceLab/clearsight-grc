package httpapi

import "net/http"

// productionRoutes is the single executable route catalog. Feature modules may
// own their route specs, but runtime registration and the OpenAPI parity test
// must consume this same aggregate so no route can bypass contract validation.
func (a *API) productionRoutes() []routeSpec {
	base := a.routes()
	distributions := a.formDistributionRoutes()
	routes := make([]routeSpec, 0, len(base)+len(distributions))
	routes = append(routes, base...)
	routes = append(routes, distributions...)
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
