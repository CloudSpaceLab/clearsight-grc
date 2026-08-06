package httpapi

import (
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) actorAutomationPolicies(w http.ResponseWriter, r *http.Request) {
	if a.deps.Autonomy == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "automation_policies_unavailable", "Automation policies are unavailable.")
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	values, err := a.deps.Autonomy.ListAutomationPolicies(r.Context(), actor.TenantID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "automation_policies_failed", "Automation policies could not be loaded.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": values})
}
