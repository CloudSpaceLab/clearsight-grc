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
		httpx.WriteError(w, http.StatusServiceUnavailable, "oversight_unavailable", "Oversight unavailable. Try again.")
		return
	}
	value, err := a.deps.Oversight.Get(r.Context(), oversight.Scope{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID})
	if errors.Is(err, oversight.ErrNotFound) {
		httpx.WriteError(w, http.StatusServiceUnavailable, "oversight_not_ready", "Oversight has not been calculated for this legal entity.")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "oversight_unavailable", "Oversight unavailable. Try again.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}
