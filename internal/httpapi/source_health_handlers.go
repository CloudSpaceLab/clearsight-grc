package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) sourceScopeHealth(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	values, err := service.ListSourceScopeHealth(r.Context(), actor.TenantID, r.PathValue("source_id"), limit)
	switch {
	case errors.Is(err, evidence.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Source not found.")
	case err != nil:
		httpx.WriteError(w, http.StatusInternalServerError, "source_health_failed", "Source health could not be loaded.")
	default:
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": values})
	}
}
