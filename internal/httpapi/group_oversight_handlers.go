package httpapi

import (
	"errors"
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) groupOversightSnapshot(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	if a.deps.GroupOversight == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "group_oversight_unavailable", "Group oversight is unavailable.")
		return
	}
	value, err := a.deps.GroupOversight.Get(r.Context(), actor)
	switch {
	case errors.Is(err, oversight.ErrGroupForbidden):
		httpx.WriteError(w, http.StatusForbidden, "group_scope_forbidden", "Group oversight is not available for this sign-in.")
	case errors.Is(err, oversight.ErrGroupUnavailable):
		httpx.WriteError(w, http.StatusServiceUnavailable, "group_oversight_not_ready", "Group oversight is not ready.")
	case err != nil:
		httpx.WriteError(w, http.StatusServiceUnavailable, "group_oversight_unavailable", "Group oversight is unavailable.")
	default:
		httpx.WriteJSON(w, http.StatusOK, value)
	}
}
