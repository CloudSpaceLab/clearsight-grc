package httpapi

import (
	"errors"
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/notificationprefs"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) notificationPreferences(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	if a == nil || a.deps.NotificationPreferences == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "notification_preferences_unavailable", "Notification preferences are unavailable. Try again.")
		return
	}
	value, err := a.deps.NotificationPreferences.Get(r.Context(), actor.TenantID, actor.PrincipalID)
	if err != nil {
		writeNotificationPreferenceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) updateNotificationPreferences(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	if a == nil || a.deps.NotificationPreferences == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "notification_preferences_unavailable", "Notification preferences are unavailable. Try again.")
		return
	}
	var input notificationprefs.UpdateInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		writeNotificationPreferenceError(w, notificationprefs.ErrInvalid)
		return
	}
	value, err := a.deps.NotificationPreferences.Update(r.Context(), actor.TenantID, actor.PrincipalID, input)
	if err != nil {
		writeNotificationPreferenceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func writeNotificationPreferenceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, notificationprefs.ErrInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "notification_preferences_invalid", "Check the digest time, timezone and quiet hours.")
	case errors.Is(err, notificationprefs.ErrVersionConflict):
		httpx.WriteError(w, http.StatusConflict, "notification_preferences_changed", "Notification preferences changed. Reload and try again.")
	default:
		httpx.WriteError(w, http.StatusServiceUnavailable, "notification_preferences_unavailable", "Notification preferences are unavailable. Try again.")
	}
}
