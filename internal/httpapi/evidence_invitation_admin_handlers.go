package httpapi

import (
	"errors"
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) listEvidenceInvitationMetadata(w http.ResponseWriter, r *http.Request) {
	service, actor, ok := a.evidenceRequesterAdmin(w, r)
	if !ok {
		return
	}
	values, err := service.ListInvitationMetadata(r.Context(), evidence.ManageInvitationsInput{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, RequestID: r.PathValue("id"), ActorPrincipalID: actor.PrincipalID,
	})
	if writeEvidenceInvitationAdminError(w, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": values})
}

func (a *API) replaceEvidenceInvitation(w http.ResponseWriter, r *http.Request) {
	service, actor, ok := a.evidenceRequesterAdmin(w, r)
	if !ok {
		return
	}
	var input evidence.ReplaceInvitationInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.TenantID, input.LegalEntityID, input.RequestID, input.InvitationID, input.ActorPrincipalID = actor.TenantID, actor.LegalEntityID, r.PathValue("id"), r.PathValue("invitation_id"), actor.PrincipalID
	issued, err := service.ReplaceInvitation(r.Context(), input)
	if writeEvidenceInvitationAdminError(w, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, issued)
}

func (a *API) revokeEvidenceInvitationAsRequester(w http.ResponseWriter, r *http.Request) {
	service, actor, ok := a.evidenceRequesterAdmin(w, r)
	if !ok {
		return
	}
	err := service.RevokeInvitationAsRequester(r.Context(), evidence.RevokeInvitationAsRequesterInput{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, RequestID: r.PathValue("id"), InvitationID: r.PathValue("invitation_id"), ActorPrincipalID: actor.PrincipalID,
	})
	if writeEvidenceInvitationAdminError(w, err) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) revokeEvidenceSessionAsRequester(w http.ResponseWriter, r *http.Request) {
	service, actor, ok := a.evidenceRequesterAdmin(w, r)
	if !ok {
		return
	}
	err := service.RevokeSessionAsRequester(r.Context(), evidence.RevokeSessionAsRequesterInput{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, RequestID: r.PathValue("id"), SessionID: r.PathValue("session_id"), ActorPrincipalID: actor.PrincipalID,
	})
	if writeEvidenceInvitationAdminError(w, err) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) evidenceRequesterAdmin(w http.ResponseWriter, r *http.Request) (*evidence.Service, identity.Actor, bool) {
	service, ok := a.evidenceService(w)
	if !ok {
		return nil, identity.Actor{}, false
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return nil, identity.Actor{}, false
	}
	return service, actor, true
}

func writeEvidenceInvitationAdminError(w http.ResponseWriter, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, evidence.ErrNotFound), errors.Is(err, evidence.ErrRecipientManagerRequired), errors.Is(err, evidence.ErrSubjectAccessDenied), errors.Is(err, evidence.ErrSubjectScopeMismatch):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Evidence request not found.")
	case errors.Is(err, evidence.ErrRequestClosed):
		httpx.WriteError(w, http.StatusConflict, "request_closed", "The request is no longer open.")
	case errors.Is(err, evidence.ErrRecipientMismatch), errors.Is(err, evidence.ErrInvitationInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "invitation_invalid", "The invitation could not be changed. Check the current recipient and expiry, then try again.")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "invitation_admin_failed", "The invitation could not be changed. Reload the request before trying again.")
	}
	return true
}
