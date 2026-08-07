package httpapi

import (
	"net/http"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) actorOnboardingGuide(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	resolved, err := a.deps.Onboarding.ResolveRoles(actor.RoleCodes)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Guide not found.")
		return
	}
	requestedCode := strings.TrimSpace(r.URL.Query().Get("code"))
	if requestedCode != "" && requestedCode != resolved.Code {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Guide not found.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, resolved)
}
