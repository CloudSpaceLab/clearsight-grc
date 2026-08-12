package httpapi

import (
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) sessionStatus(w http.ResponseWriter, r *http.Request) {
	_, authenticated := identity.FromContext(r.Context())
	_, demoAuthenticator := a.deps.Identity.(identity.DemoSessionAuthenticator)
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{
		"authenticated":        authenticated,
		"demo_login_available": a.deps.DemoMode && demoAuthenticator,
	})
}
