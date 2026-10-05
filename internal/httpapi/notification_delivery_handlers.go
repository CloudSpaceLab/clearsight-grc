package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/attention"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) notificationDeliveryHealth(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	if a == nil || a.deps.NotificationDelivery == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "notification_delivery_unavailable", "Notification delivery status is unavailable.")
		return
	}
	value, err := a.deps.NotificationDelivery.Health(r.Context(), actor.TenantID, time.Now().UTC(), 24*time.Hour)
	if err != nil {
		writeNotificationDeliveryReadError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) riskNotificationHistory(w http.ResponseWriter, r *http.Request) {
	service, ok := a.riskService(w)
	if !ok {
		return
	}
	actor, scope, ok := a.riskActorScope(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	if _, err := service.Get(r.Context(), scope, id); err != nil {
		writeRiskError(w, err)
		return
	}
	a.writeRecordNotificationHistory(w, r, actor.TenantID, actor.LegalEntityID, "RISK", id)
}

func (a *API) operationalLossNotificationHistory(w http.ResponseWriter, r *http.Request) {
	service, ok := a.operationalLossService(w)
	if !ok {
		return
	}
	actor, scope, ok := a.operationalLossActorScope(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	if _, err := service.Get(r.Context(), scope, id); err != nil {
		writeOperationalLossError(w, err)
		return
	}
	a.writeRecordNotificationHistory(w, r, actor.TenantID, actor.LegalEntityID, "LOSS", id)
}

func (a *API) writeRecordNotificationHistory(w http.ResponseWriter, r *http.Request, tenantID, legalEntityID, subjectType, subjectID string) {
	if a == nil || a.deps.NotificationDelivery == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "notification_history_unavailable", "Notification history is unavailable.")
		return
	}
	value, err := a.deps.NotificationDelivery.RecordHistory(r.Context(), tenantID, legalEntityID, subjectType, subjectID, 50)
	if err != nil {
		writeNotificationDeliveryReadError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func writeNotificationDeliveryReadError(w http.ResponseWriter, err error) {
	if errors.Is(err, attention.ErrDeliveryReadUnavailable) {
		httpx.WriteError(w, http.StatusServiceUnavailable, "notification_delivery_unavailable", "Notification delivery status is unavailable.")
		return
	}
	httpx.WriteError(w, http.StatusInternalServerError, "notification_delivery_failed", "Notification delivery status could not be loaded.")
}
