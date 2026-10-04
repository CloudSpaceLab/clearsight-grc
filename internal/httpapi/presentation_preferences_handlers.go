package httpapi

import (
	"errors"
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/presentationprefs"
)

func (a *API) presentationPreferences(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	if a == nil || a.deps.PresentationPreferences == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "presentation_preferences_unavailable", "Presentation preferences are unavailable. Try again.")
		return
	}
	value, err := a.deps.PresentationPreferences.Get(r.Context(), actor.TenantID, actor.PrincipalID, actor.RoleCodes)
	if err != nil {
		writePresentationPreferenceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) updatePresentationPreferences(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	if a == nil || a.deps.PresentationPreferences == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "presentation_preferences_unavailable", "Presentation preferences are unavailable. Try again.")
		return
	}
	var input presentationprefs.UpdateInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		writePresentationPreferenceError(w, presentationprefs.ErrInvalid)
		return
	}
	value, err := a.deps.PresentationPreferences.Update(r.Context(), actor.TenantID, actor.PrincipalID, actor.RoleCodes, input)
	if err != nil {
		writePresentationPreferenceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func writePresentationPreferenceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, presentationprefs.ErrInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "presentation_preferences_invalid", "Choose one of the available presentation defaults.")
	case errors.Is(err, presentationprefs.ErrVersionConflict):
		httpx.WriteError(w, http.StatusConflict, "presentation_preferences_changed", "Presentation preferences changed. Reload and try again.")
	default:
		httpx.WriteError(w, http.StatusServiceUnavailable, "presentation_preferences_unavailable", "Presentation preferences are unavailable. Try again.")
	}
}
