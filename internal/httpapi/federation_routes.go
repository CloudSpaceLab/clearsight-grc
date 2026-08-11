package httpapi

import (
	"net/http"
)

// Authentication transport routes are intentionally kept outside the versioned
// /api/v1 application contract. They still use the same typed route access
// classes and identity middleware; only the OIDC callback is public and it is
// transaction-bound by state, nonce and PKCE in the federation service.
func (a *API) registerFederationRoutes(mux *http.ServeMux) {
	if a.deps.Federation == nil {
		return
	}
	routes := []routeSpec{
		public(http.MethodGet, "/auth/oidc/login", a.deps.Federation.Begin),
		public(http.MethodGet, "/auth/oidc/callback", a.deps.Federation.Callback),
		write(http.MethodPost, "/auth/logout", a.deps.Federation.Logout, nil),
	}
	if err := validateRoutes(routes); err != nil {
		panic(err)
	}
	for _, spec := range routes {
		handler := a.routeAccess(spec, spec.Handler)
		mux.HandleFunc(spec.Method+" "+spec.Path, handler)
	}
}
