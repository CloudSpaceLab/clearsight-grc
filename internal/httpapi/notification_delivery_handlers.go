package httpapi

import (
	"net/http"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) notificationDeliveryHealth(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.deps.AttentionDeliveries == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "notification_delivery_unavailable", "Notification delivery status is unavailable.")
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to view notification delivery status.")
		return
	}
	health, err := a.deps.AttentionDeliveries.Health(r.Context(), actor.TenantID, time.Now().UTC().Add(-24*time.Hour), 20)
	if err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "notification_delivery_unavailable", "Notification delivery status is unavailable.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, health)
}
