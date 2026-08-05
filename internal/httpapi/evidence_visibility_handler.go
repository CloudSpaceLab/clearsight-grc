package httpapi

import (
	"net/http"
	"strconv"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) listVisibleEvidenceRequests(w http.ResponseWriter, r *http.Request) {
	if a.deps.Evidence == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "requests_unavailable", "Evidence requests are unavailable.")
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	values, err := a.deps.Evidence.ListVisibleRequests(r.Context(), actor.TenantID, actor.PrincipalID, limit, func(requestValue evidence.Request) bool {
		return a.canReadEvidenceRequest(r.Context(), requestValue)
	})
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "requests_failed", "Evidence requests could not be loaded.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": values})
}
