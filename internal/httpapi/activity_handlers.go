package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/activity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) listSystemActivity(w http.ResponseWriter, r *http.Request) {
	if a.deps.Activity == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "system_activity_unavailable", "System activity is unavailable.")
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, parseErr := strconv.Atoi(raw)
		if parseErr != nil || parsed <= 0 {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_activity_filter", "Activity limit must be a positive number.")
			return
		}
		limit = parsed
	}
	from, err := parseActivityTime(r.URL.Query().Get("from"), false)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_activity_filter", "Activity start time must be an RFC3339 timestamp or YYYY-MM-DD date.")
		return
	}
	to, err := parseActivityTime(r.URL.Query().Get("to"), true)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_activity_filter", "Activity end time must be an RFC3339 timestamp or YYYY-MM-DD date.")
		return
	}
	page, err := a.deps.Activity.List(r.Context(), activity.Query{
		TenantID:      actor.TenantID,
		Limit:         limit,
		Cursor:        r.URL.Query().Get("cursor"),
		From:          from,
		To:            to,
		Category:      r.URL.Query().Get("category"),
		EventType:     r.URL.Query().Get("event_type"),
		ObjectType:    r.URL.Query().Get("object_type"),
		ObjectID:      r.URL.Query().Get("object_id"),
		ActorID:       r.URL.Query().Get("actor_id"),
		ActorQuery:    r.URL.Query().Get("actor"),
		ActorKind:     r.URL.Query().Get("actor_kind"),
		LegalEntityID: r.URL.Query().Get("legal_entity_id"),
	})
	if errors.Is(err, activity.ErrInvalid) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_activity_filter", "The activity filter is invalid.")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "system_activity_failed", "System activity could not be loaded.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, page)
}

func (a *API) getSystemActivity(w http.ResponseWriter, r *http.Request) {
	if a.deps.Activity == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "system_activity_unavailable", "System activity is unavailable.")
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	value, err := a.deps.Activity.Get(r.Context(), actor.TenantID, r.PathValue("event_id"))
	if errors.Is(err, activity.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "system_activity_not_found", "The activity event was not found.")
		return
	}
	if errors.Is(err, activity.ErrInvalid) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_activity_event", "The activity event identifier is invalid.")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "system_activity_failed", "The activity event could not be loaded.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func parseActivityTime(raw string, endOfDay bool) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if value, err := time.Parse(time.RFC3339, raw); err == nil {
		value = value.UTC()
		return &value, nil
	}
	value, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, err
	}
	value = value.UTC()
	if endOfDay {
		value = value.Add(24*time.Hour - time.Nanosecond)
	}
	return &value, nil
}
