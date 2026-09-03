package httpapi

import (
	"errors"
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) getEvidenceReviewSubmission(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	request, err := service.GetRequestForEntity(r.Context(), actor.TenantID, actor.LegalEntityID, r.PathValue("id"))
	if evidenceRequestUnavailable(err) || !a.canReadEvidenceRequest(r.Context(), request) || !evidence.RequestReviewableBy(request, actor.PrincipalID) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Submitted evidence response not found.")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "submission_failed", "The submitted evidence response could not be loaded.")
		return
	}
	submission, err := service.GetSubmissionForRequest(r.Context(), actor.TenantID, request.ID)
	if errors.Is(err, evidence.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Submitted evidence response not found.")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "submission_failed", "The submitted evidence response could not be loaded.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, submission)
}
