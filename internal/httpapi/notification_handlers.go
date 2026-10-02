package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
)

func (a *API) listNotifications(w http.ResponseWriter, r *http.Request) {
	if a.deps.Workflow == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "notifications_unavailable", "Notifications are unavailable. Try again.")
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	limit := 25
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		value, parseErr := strconv.Atoi(raw)
		if parseErr != nil || value < 1 || value > 100 {
			httpx.WriteError(w, http.StatusBadRequest, "notification_filter_invalid", "Notification limit must be between 1 and 100.")
			return
		}
		limit = value
	}
	unreadOnly := false
	if raw := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("unread_only"))); raw != "" {
		switch raw {
		case "true":
			unreadOnly = true
		case "false":
		default:
			httpx.WriteError(w, http.StatusBadRequest, "notification_filter_invalid", "Unread filter must be true or false.")
			return
		}
	}
	page, err := a.deps.Workflow.ListNotifications(r.Context(), workflow.NotificationFilter{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PrincipalID: actor.PrincipalID,
		Cursor: r.URL.Query().Get("cursor"), UnreadOnly: unreadOnly, Limit: limit,
	})
	switch {
	case errors.Is(err, workflow.ErrNotificationInvalidCursor):
		httpx.WriteError(w, http.StatusBadRequest, "notification_cursor_invalid", "The notification cursor is invalid. Reload the first page.")
	case errors.Is(err, workflow.ErrNotificationUnavailable):
		httpx.WriteError(w, http.StatusServiceUnavailable, "notifications_unavailable", "Notifications are unavailable. Try again.")
	case err != nil:
		httpx.WriteError(w, http.StatusServiceUnavailable, "notifications_failed", "Notifications could not be loaded. Try again.")
	default:
		httpx.WriteJSON(w, http.StatusOK, page)
	}
}

func (a *API) markNotificationRead(w http.ResponseWriter, r *http.Request) {
	if a.deps.Workflow == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "notifications_unavailable", "Notifications are unavailable. Try again.")
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	value, err := a.deps.Workflow.MarkNotificationRead(r.Context(), workflow.NotificationFilter{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PrincipalID: actor.PrincipalID,
	}, r.PathValue("id"), time.Time{})
	switch {
	case errors.Is(err, workflow.ErrNotificationNotFound):
		httpx.WriteError(w, http.StatusNotFound, "notification_not_found", "The notification was not found.")
	case errors.Is(err, workflow.ErrNotificationUnavailable):
		httpx.WriteError(w, http.StatusServiceUnavailable, "notifications_unavailable", "Notifications are unavailable. Try again.")
	case err != nil:
		httpx.WriteError(w, http.StatusServiceUnavailable, "notification_update_failed", "The notification could not be updated. Try again.")
	default:
		httpx.WriteJSON(w, http.StatusOK, value)
	}
}
