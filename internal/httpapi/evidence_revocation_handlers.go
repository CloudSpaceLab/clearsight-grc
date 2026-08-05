package httpapi

import (
	"errors"
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) revokeEvidenceInvitation(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	tenant, ok := requiredQuery(w, r, "tenant_id")
	if !ok {
		return
	}
	err := service.RevokeInvitation(r.Context(), tenant, r.PathValue("id"))
	if errors.Is(err, evidence.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Invitation not found.")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "revocation_failed", "The invitation could not be revoked.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) revokeEvidenceSession(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	tenant, ok := requiredQuery(w, r, "tenant_id")
	if !ok {
		return
	}
	err := service.RevokeSession(r.Context(), tenant, r.PathValue("id"))
	if errors.Is(err, evidence.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Session not found.")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "revocation_failed", "The session could not be revoked.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
