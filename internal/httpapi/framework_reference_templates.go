package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/frameworkcatalog"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

type frameworkReferenceReader interface {
	ListReferenceTemplates(context.Context, string) ([]frameworkcatalog.ReferenceTemplate, error)
}

func (a *API) listFrameworkReferenceTemplates(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in to view framework templates.")
		return
	}
	if strings.TrimSpace(actor.LegalEntityID) == "" || actor.LegalEntityID == "*" {
		httpx.WriteError(w, http.StatusForbidden, "legal_entity_required", "Select a legal entity.")
		return
	}
	reader, ok := a.deps.FrameworkCatalog.(frameworkReferenceReader)
	if !ok {
		httpx.WriteError(w, http.StatusServiceUnavailable, "framework_templates_unavailable", "Framework templates are unavailable.")
		return
	}
	jurisdiction := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("jurisdiction")))
	if jurisdiction == "" {
		jurisdiction = "ALL"
	}
	items, err := reader.ListReferenceTemplates(r.Context(), jurisdiction)
	if err != nil {
		if errors.Is(err, frameworkcatalog.ErrInvalidPublication) {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_framework_filter", "Check the jurisdiction filter.")
			return
		}
		httpx.WriteError(w, http.StatusServiceUnavailable, "framework_templates_unavailable", "Framework templates could not be loaded.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}
