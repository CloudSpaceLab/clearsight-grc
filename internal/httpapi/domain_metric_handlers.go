package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

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
	organizationScopeID := strings.TrimSpace(r.URL.Query().Get("organization_scope_id"))
	var bundle metricview.DomainBundle
	if organizationScopeID == "" {
		bundle, err = a.deps.DomainMetrics.LatestDomainMetrics(r.Context(), actor.TenantID, actor.LegalEntityID)
	} else {
		selection, scopeErr := a.resolveOrganizationScopeSelection(r.Context(), actor, organizationScopeID, true)
		if scopeErr != nil {
			writeOrganizationScopeRequestError(w, scopeErr, "This organization scope is not available for risk metrics.")
			return
		}
		scoped, ok := a.deps.DomainMetrics.(metricview.ScopedDomainReader)
		if !ok {
			httpx.WriteError(w, http.StatusServiceUnavailable, "domain_metrics_unavailable", "Risk metrics are unavailable for this organization scope. Try again.")
			return
		}
		bundle, err = scoped.CurrentDomainMetrics(
			r.Context(), actor.TenantID, actor.LegalEntityID,
			selection.ID, selection.IDs, time.Now().UTC(),
		)
	}
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
