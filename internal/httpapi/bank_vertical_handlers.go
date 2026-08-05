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
	tenant, ok := requiredQuery(w, r, "tenant_id")
	if !ok {
		return
	}
	values, err := a.deps.BankVerticals.List(r.Context(), tenant)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "journeys_failed", "Bank journeys could not be loaded.")
		return
	}
	actor, actorPresent := identity.FromContext(r.Context())
	visible := make([]bankverticals.Journey, 0, len(values))
	for _, value := range values {
		if value.Sensitive && (!actorPresent || !value.VisibleTo(actor.PrincipalID, actor.LegalEntityID == "*")) {
			continue
		}
		visible = append(visible, value)
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
