package httpapi

import (
	"errors"
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) createDocumentFormProposal(w http.ResponseWriter, r *http.Request) {
	service, ok := a.formProposalService(w)
	if !ok {
		return
	}
	var input monitoring.RequestDocumentFormProposalInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := service.RequestFromDocument(r.Context(), r.PathValue("id"), input)
	if err != nil {
		writeFormProposalError(w, err)
		return
	}
	// PostgreSQL generation is intentionally asynchronous through the shared
	// outbox. Memory/demo may already have produced REVIEW_REQUIRED by the time
	// this response is written, but the create contract remains uniformly 202.
	httpx.WriteJSON(w, http.StatusAccepted, value)
}

func (a *API) getFormProposal(w http.ResponseWriter, r *http.Request) {
	service, ok := a.formProposalService(w)
	if !ok {
		return
	}
	value, err := service.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeFormProposalError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) acceptFormProposal(w http.ResponseWriter, r *http.Request) {
	service, ok := a.formProposalService(w)
	if !ok {
		return
	}
	var input monitoring.AcceptFormProposalInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := service.Accept(r.Context(), r.PathValue("id"), input)
	if err != nil {
		writeFormProposalError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) rejectFormProposal(w http.ResponseWriter, r *http.Request) {
	service, ok := a.formProposalService(w)
	if !ok {
		return
	}
	var input monitoring.RejectFormProposalInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := service.Reject(r.Context(), r.PathValue("id"), input)
	if err != nil {
		writeFormProposalError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) formProposalService(w http.ResponseWriter) (*monitoring.FormProposalService, bool) {
	if a.deps.FormProposals == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "form_proposals_unavailable", "Form proposal generation is not configured.")
		return nil, false
	}
	return a.deps.FormProposals, true
}

func writeFormProposalError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, identity.ErrMissingIdentity), errors.Is(err, identity.ErrInvalidIdentity), errors.Is(err, identity.ErrExpiredIdentity), errors.Is(err, commandauth.ErrIdentityRequired):
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to work with form proposals.")
	case errors.Is(err, commandauth.ErrNotAuthorized), errors.Is(err, commandauth.ErrTenantMismatch), errors.Is(err, commandauth.ErrLegalEntityMismatch):
		writeCommandAuthorizationError(w, err)
	case errors.Is(err, commandauth.ErrGuardUnavailable):
		httpx.WriteError(w, http.StatusServiceUnavailable, "form_authority_unavailable", "The current form authority could not be checked. No draft was changed.")
	case errors.Is(err, monitoring.ErrNotFound), errors.Is(err, documentimport.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "form_proposal_not_found", "The form proposal or its exact source document was not found in this legal entity.")
	case errors.Is(err, monitoring.ErrConflict), errors.Is(err, documentimport.ErrVersionConflict), errors.Is(err, monitoring.ErrFormProposalSourceChanged), errors.Is(err, monitoring.ErrFormProposalState):
		httpx.WriteError(w, http.StatusConflict, "form_proposal_conflict", err.Error())
	case errors.Is(err, monitoring.ErrFormProposalSelection), errors.Is(err, monitoring.ErrInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "form_proposal_invalid", err.Error())
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "form_proposal_failed", "The form proposal operation could not be completed.")
	}
}
