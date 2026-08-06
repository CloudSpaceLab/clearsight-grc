package httpapi

import (
	"net/http"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) listBankJourneys(w http.ResponseWriter, r *http.Request) {
	if a.deps.BankVerticals == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "journeys_unavailable", "Bank journeys are unavailable.")
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	values, err := a.deps.BankVerticals.List(r.Context(), actor.TenantID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "journeys_failed", "Bank journeys could not be loaded.")
		return
	}
	visible := make([]bankverticals.Journey, 0, len(values))
	for _, value := range values {
		if value.VisibleTo(actor.PrincipalID) {
			visible = append(visible, value)
		}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": visible, "generated_at": time.Now().UTC(), "sample": sampleJourneys(visible)})
}

func sampleJourneys(values []bankverticals.Journey) bool {
	for _, value := range values {
		if value.Sample {
			return true
		}
	}
	return false
}
