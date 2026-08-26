package httpapi

import (
	"errors"
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) declareEvidenceWrongRecipient(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	var input evidence.DeclareWrongRecipientInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.RequestID = r.PathValue("id")
	value, err := service.DeclareWrongRecipient(r.Context(), input)
	writeEvidenceRecipientLifecycleResult(w, value, err)
}

func (a *API) reassignEvidenceRecipient(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	var input evidence.ReassignRecipientInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.RequestID = r.PathValue("id")
	value, err := service.ReassignRecipient(r.Context(), input)
	writeEvidenceRecipientLifecycleResult(w, value, err)
}

func writeEvidenceRecipientLifecycleResult(w http.ResponseWriter, value evidence.Request, err error) {
	switch {
	case errors.Is(err, evidence.ErrNotFound), errors.Is(err, evidence.ErrSubjectScopeMismatch), errors.Is(err, evidence.ErrSubjectAccessDenied):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Evidence request not found.")
	case errors.Is(err, evidence.ErrVersionConflict):
		httpx.WriteError(w, http.StatusConflict, "version_conflict", "This request changed. Reload before changing its recipient.")
	case errors.Is(err, evidence.ErrRecipientMismatch), errors.Is(err, evidence.ErrRecipientManagerRequired):
		httpx.WriteError(w, http.StatusForbidden, "recipient_not_allowed", "You cannot change this request's recipient.")
	case errors.Is(err, evidence.ErrRequestClosed):
		httpx.WriteError(w, http.StatusConflict, "request_closed", "This request can no longer be reassigned.")
	case errors.Is(err, evidence.ErrRecipientInvalid), errors.Is(err, evidence.ErrRecipientRequired):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "recipient_invalid", "The recipient change is not valid for this request.")
	case err != nil:
		httpx.WriteError(w, http.StatusInternalServerError, "recipient_change_failed", "The recipient change could not be recorded.")
	default:
		httpx.WriteJSON(w, http.StatusOK, value)
	}
}
