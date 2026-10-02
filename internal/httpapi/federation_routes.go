package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/federation"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
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
		write(http.MethodPost, "/auth/scope", a.switchFederationScope, nil),
	}
	if err := validateRoutes(routes); err != nil {
		panic(err)
	}
	for _, spec := range routes {
		handler := a.routeAccess(spec, spec.Handler)
		mux.HandleFunc(spec.Method+" "+spec.Path, handler)
	}
}


type scopeSwitchRequest struct {
	LegalEntityID string `json:"legal_entity_id"`
}

func (a *API) switchFederationScope(w http.ResponseWriter, r *http.Request) {
	if a.deps.Federation == nil {
		httpx.WriteError(w, http.StatusNotFound, "scope_switch_unavailable", "Scope switching is unavailable.")
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	var input scopeSwitchRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil || strings.TrimSpace(input.LegalEntityID) == "" {
		httpx.WriteError(w, http.StatusBadRequest, "scope_switch_invalid", "Choose an available legal entity.")
		return
	}
	if _, err := a.deps.Federation.SwitchScope(r.Context(), actor, input.LegalEntityID); err != nil {
		switch {
		case errors.Is(err, federation.ErrScopeInvalid):
			httpx.WriteError(w, http.StatusBadRequest, "scope_switch_invalid", "Choose an available legal entity.")
		case errors.Is(err, federation.ErrScopeUnavailable):
			httpx.WriteError(w, http.StatusNotFound, "scope_switch_unavailable", "That legal entity is not available to your current identity.")
		default:
			httpx.WriteError(w, http.StatusServiceUnavailable, "scope_switch_failed", "The active legal entity could not be changed. Try again.")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
