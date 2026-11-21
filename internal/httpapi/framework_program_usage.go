package httpapi

import (
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"net/http"
)

func (a *API) listFrameworkProgramUsage(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, 401, "identity_required", "Sign in to view framework Program counts.")
		return
	}
	if actor.LegalEntityID == "" || actor.LegalEntityID == "*" {
		httpx.WriteError(w, 403, "legal_entity_required", "Select a legal entity.")
		return
	}
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	items, err := service.FrameworkProgramUsage(r.Context())
	if err != nil {
		httpx.WriteError(w, 503, "framework_usage_unavailable", "Framework Program counts could not be loaded.")
		return
	}
	httpx.WriteJSON(w, 200, map[string]any{"items": items})
}
