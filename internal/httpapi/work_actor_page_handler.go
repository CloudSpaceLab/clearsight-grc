package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/today"
	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
)

func (a *API) actorWorkAssignments(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	if actor.TenantID == "" || actor.LegalEntityID == "" || actor.PrincipalID == "" {
		httpx.WriteError(w, http.StatusForbidden, "work_scope_required", "Your assigned work scope could not be verified.")
		return
	}
	if a.deps.Workflow == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "work_unavailable", "Assigned work is unavailable.")
		return
	}
	params := r.URL.Query()
	for name, values := range params {
		switch name {
		case "tenant_id", "legal_entity_id", "principal_id", "q", "cursor", "limit", "status", "due_from", "due_before":
			if len(values) != 1 {
				httpx.WriteError(w, http.StatusBadRequest, "work_filter_invalid", "Specify each work filter once.")
				return
			}
		default:
			httpx.WriteError(w, http.StatusBadRequest, "work_filter_unsupported", "This work filter is unavailable.")
			return
		}
	}
	for name, expected := range map[string]string{
		"tenant_id": actor.TenantID, "legal_entity_id": actor.LegalEntityID, "principal_id": actor.PrincipalID,
	} {
		if supplied := strings.TrimSpace(params.Get(name)); supplied != "" && supplied != expected {
			httpx.WriteError(w, http.StatusForbidden, "work_scope_denied", "This request is outside your assigned work scope.")
			return
		}
	}
	// Entity/group membership filtering is not available in this endpoint.
	// Do not silently ignore a requested organization unit and return a wider page.
	if params.Has("organization_scope_id") || params.Has("group_id") {
		httpx.WriteError(w, http.StatusBadRequest, "work_filter_unsupported", "Work filtering for this organization area is unavailable.")
		return
	}
	search := strings.TrimSpace(params.Get("q"))
	if strings.ContainsRune(search, 0) || utf8.RuneCountInString(search) > 120 {
		httpx.WriteError(w, http.StatusBadRequest, "work_search_invalid", "Search is limited to 120 characters.")
		return
	}
	limit := 25
	if raw := strings.TrimSpace(params.Get("limit")); raw != "" {
		parsed, parseErr := strconv.Atoi(raw)
		if parseErr != nil || parsed < 1 || parsed > 100 {
			httpx.WriteError(w, http.StatusBadRequest, "work_limit_invalid", "Page size must be between 1 and 100.")
			return
		}
		limit = parsed
	}
	status := workflow.Status(strings.ToUpper(strings.TrimSpace(params.Get("status"))))
	switch status {
	case "", workflow.StatusReady, workflow.StatusInProgress, workflow.StatusBlocked, workflow.StatusEscalated:
	default:
		httpx.WriteError(w, http.StatusBadRequest, "work_status_invalid", "Select an active work status.")
		return
	}
	dueFrom, valid := parseWorkBoundary(params.Get("due_from"))
	if !valid {
		httpx.WriteError(w, http.StatusBadRequest, "work_date_invalid", "Enter a valid start date and time.")
		return
	}
	dueBefore, valid := parseWorkBoundary(params.Get("due_before"))
	if !valid || dueFrom != nil && dueBefore != nil && !dueFrom.Before(*dueBefore) {
		httpx.WriteError(w, http.StatusBadRequest, "work_date_invalid", "The end must follow the start date and time.")
		return
	}
	page, err := a.deps.Workflow.ListActorWorkPage(r.Context(), workflow.ActorWorkPageQuery{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PrincipalID: actor.PrincipalID,
		Search: search, Status: status, DueFrom: dueFrom, DueBefore: dueBefore,
		Cursor: strings.TrimSpace(params.Get("cursor")), Limit: limit,
	})
	switch {
	case errors.Is(err, workflow.ErrInvalidWorkCursor):
		httpx.WriteError(w, http.StatusBadRequest, "work_cursor_invalid", "This work page expired or the filters changed. Reload assigned work.")
	case err != nil:
		httpx.WriteError(w, http.StatusServiceUnavailable, "work_unavailable", "Assigned work could not be loaded.")
	default:
		items := today.FromWorkflowTasksForActor(page.Items, actor.PrincipalID)
		if len(items) != len(page.Items) {
			httpx.WriteError(w, http.StatusServiceUnavailable, "work_scope_changed", "Access to assigned work changed. Reload the queue.")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"items": items, "next_cursor": page.NextCursor, "has_more": page.HasMore,
			"as_of": page.AsOf, "coverage": page.Coverage, "count_basis": page.CountBasis,
		})
	}
}

func parseWorkBoundary(raw string) (*time.Time, bool) {
	if strings.TrimSpace(raw) == "" {
		return nil, true
	}
	parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(raw))
	if err != nil {
		return nil, false
	}
	utc := parsed.UTC()
	return &utc, true
}
