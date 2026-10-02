package httpapi

import (
	"errors"
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/metricview"
	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) homeMetrics(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	if a.deps.Oversight == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "metrics_unavailable", "Risk metrics are unavailable. Try again.")
		return
	}
	snapshot, err := a.deps.Oversight.GetForPeriod(r.Context(), oversight.Scope{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID,
	}, oversightPeriodRequest(r))
	if writeOversightPeriodError(w, err) {
		return
	}
	if errors.Is(err, oversight.ErrNotFound) {
		httpx.WriteError(w, http.StatusServiceUnavailable, "metrics_not_ready", "Risk metrics have not been calculated for this legal entity.")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "metrics_unavailable", "Risk metrics are unavailable. Try again.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, metricview.FromOversight(snapshot))
}
