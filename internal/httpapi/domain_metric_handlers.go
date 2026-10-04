package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/metricview"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) domainMetrics(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	if a == nil || a.deps.DomainMetrics == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "domain_metrics_unavailable", "Risk metrics are unavailable. Try again.")
		return
	}
	if strings.TrimSpace(actor.TenantID) == "" || strings.TrimSpace(actor.LegalEntityID) == "" || actor.LegalEntityID == "*" {
		httpx.WriteError(w, http.StatusForbidden, "domain_metrics_scope_unavailable", "Choose an eligible legal entity before viewing risk metrics.")
		return
	}
	if strings.TrimSpace(r.URL.Query().Get("organization_scope_id")) != "" {
		httpx.WriteError(w, http.StatusBadRequest, "domain_metrics_scope_invalid", "This metric family is currently available at legal-entity scope.")
		return
	}
	bundle, err := a.deps.DomainMetrics.LatestDomainMetrics(r.Context(), actor.TenantID, actor.LegalEntityID)
	switch {
	case errors.Is(err, metricview.ErrDomainMetricsNotFound):
		httpx.WriteError(w, http.StatusServiceUnavailable, "domain_metrics_not_ready", "Risk metrics have not been calculated for this legal entity.")
	case errors.Is(err, metricview.ErrDomainMetricsInvalid):
		httpx.WriteError(w, http.StatusBadRequest, "domain_metrics_scope_invalid", "Choose an eligible legal entity before viewing risk metrics.")
	case err != nil:
		httpx.WriteError(w, http.StatusServiceUnavailable, "domain_metrics_unavailable", "Risk metrics are unavailable. Try again.")
	default:
		httpx.WriteJSON(w, http.StatusOK, bundle)
	}
}
