package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) listEvidenceActiveSessions(w http.ResponseWriter, r *http.Request) {
	service, actor, ok := a.evidenceRequesterAdmin(w, r)
	if !ok {
		return
	}
	limit := 50
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed < 1 || parsed > 50 {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_limit", "Limit must be between 1 and 50.")
			return
		}
		limit = parsed
	}
	page, err := service.ListActiveSessionMetadata(r.Context(), evidence.ManageSessionsInput{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, RequestID: r.PathValue("id"), ActorPrincipalID: actor.PrincipalID, Limit: limit,
	})
	switch {
	case errors.Is(err, evidence.ErrNotFound), errors.Is(err, evidence.ErrRecipientManagerRequired), errors.Is(err, evidence.ErrSubjectAccessDenied), errors.Is(err, evidence.ErrSubjectScopeMismatch), errors.Is(err, evidence.ErrSubjectUnsupported):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Evidence request not found.")
	case err != nil:
		httpx.WriteError(w, http.StatusInternalServerError, "session_inventory_failed", "Active external sessions could not be loaded. Reload the evidence request before trying again.")
	default:
		httpx.WriteJSON(w, http.StatusOK, page)
	}
}
