package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
)

// actorWorkDueCounts reports complete totals for the active workflow/proposal
// source only. Other Work/exception populations are not silently treated as 0.
func (a *API) actorWorkDueCounts(w http.ResponseWriter, r *http.Request) {
	actor, ok := verifiedWorkActor(w, r)
	if !ok {
		return
	}
	if a.deps.Workflow == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "work_unavailable", "Assigned work is unavailable.")
		return
	}
	for name, values := range r.URL.Query() {
		var expected string
		switch name {
		case "tenant_id":
			expected = actor.TenantID
		case "legal_entity_id":
			expected = actor.LegalEntityID
		case "principal_id":
			expected = actor.PrincipalID
		default:
			httpx.WriteError(w, http.StatusBadRequest, "work_filter_unsupported", "This Work count filter is unavailable.")
			return
		}
		if len(values) != 1 {
			httpx.WriteError(w, http.StatusBadRequest, "work_filter_invalid", "Specify each work count scope once.")
			return
		}
		if supplied := strings.TrimSpace(values[0]); supplied != expected {
			httpx.WriteError(w, http.StatusForbidden, "work_scope_denied", "This request is outside your assigned work scope.")
			return
		}
	}
	counts, err := a.deps.Workflow.ReadActorWorkDueCounts(r.Context(), actor.TenantID, actor.LegalEntityID, actor.PrincipalID)
	if err != nil {
		if errors.Is(err, workflow.ErrActorWorkCountsUnavailable) {
			httpx.WriteError(w, http.StatusServiceUnavailable, "work_counts_unavailable", "Authorized Work counts are unavailable.")
			return
		}
		httpx.WriteError(w, http.StatusServiceUnavailable, "work_counts_failed", "Authorized Work counts could not be loaded.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"total": counts.Total, "overdue": counts.Overdue,
		"due_soon": counts.DueSoon, "undated": counts.Undated,
		"as_of": counts.AsOf, "coverage": counts.Coverage, "count_basis": counts.CountBasis,
		"scope": map[string]string{
			"tenant_id":       actor.TenantID,
			"legal_entity_id": actor.LegalEntityID,
			"principal_id":    actor.PrincipalID,
		},
	})
}
