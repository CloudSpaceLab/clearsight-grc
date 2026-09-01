package httpapi

import (
	"errors"
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) oversightSnapshot(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	if a.deps.Oversight == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "oversight_unavailable", "Oversight information is not available. Try again after the projection worker has completed a cycle.")
		return
	}
	value, err := a.deps.Oversight.Get(r.Context(), oversight.Scope{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID})
	if errors.Is(err, oversight.ErrNotFound) {
		httpx.WriteError(w, http.StatusServiceUnavailable, "oversight_not_ready", "No oversight snapshot is available for this legal entity. Check projection operations and retry after the next cycle.")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "oversight_unavailable", "Oversight information could not be loaded. Retry or check projection operations.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}
