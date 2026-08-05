package httpapi

import (
	"net/http"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) actorContext(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"tenant": map[string]string{
			"id":   actor.TenantID,
			"name": "Connected organization",
		},
		"legal_entity": map[string]string{
			"id":   actor.LegalEntityID,
			"name": "Connected legal entity",
		},
		"actor": map[string]string{
			"id":                actor.PrincipalID,
			"name":              actor.PrincipalID,
			"kind":              actor.Kind,
			"assurance_level":   actor.AssuranceLevel,
			"authentication":    actor.AuthenticationMethod,
			"session_id":        actor.SessionID,
		},
		"mode": a.deps.Mode,
	})
}

func (a *API) actorToday(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	if a.deps.Today == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "today_unavailable", "Today's work is unavailable.")
		return
	}
	items, err := a.deps.Today.ListFor(r.Context(), actor)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "today_failed", "Today's work could not be loaded.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "generated_at": time.Now().UTC()})
}
