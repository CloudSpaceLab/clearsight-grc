package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/frameworkcatalog"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

// listFrameworkEditions presents bounded, licensed publication metadata.
// It never exposes source document bodies or tenant-specific interpretations.
func (a *API) listFrameworkEditions(w http.ResponseWriter, r *http.Request) {
	if a.deps.FrameworkCatalog == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "framework_catalog_unavailable", "Standards are unavailable.")
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in to view standards.")
		return
	}
	if strings.TrimSpace(actor.LegalEntityID) == "" || actor.LegalEntityID == "*" {
		httpx.WriteError(w, http.StatusForbidden, "legal_entity_required", "Select a legal entity.")
		return
	}
	jurisdiction := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("jurisdiction")))
	if jurisdiction == "" {
		jurisdiction = "GLOBAL"
	}
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 100 {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_limit", "Choose a page size from 1 to 100.")
			return
		}
	}
	items, err := a.deps.FrameworkCatalog.ListEditions(r.Context(), jurisdiction, r.URL.Query().Get("after_id"), limit)
	if err != nil {
		if errors.Is(err, frameworkcatalog.ErrInvalidPublication) {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_framework_filter", "Check the standard filter.")
			return
		}
		httpx.WriteError(w, http.StatusServiceUnavailable, "framework_catalog_unavailable", "Standards could not be loaded.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}
