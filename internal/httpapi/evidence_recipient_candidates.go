package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) listEvidenceRecipientCandidates(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	limit := 50
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		limit, err = strconv.Atoi(rawLimit)
		if err != nil || limit < 1 || limit > 50 {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_limit", "Limit must be between 1 and 50.")
			return
		}
	}
	search := strings.TrimSpace(r.URL.Query().Get("q"))
	if utf8.RuneCountInString(search) > 100 {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_search", "Search must be 100 characters or fewer.")
		return
	}
	page, err := service.SearchRecipientCandidates(r.Context(), evidence.ActorRequestScope{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ActorPrincipalID: actor.PrincipalID,
	}, r.PathValue("id"), evidence.RecipientCandidateSearch{Query: search, Limit: limit})
	switch {
	case errors.Is(err, evidence.ErrNotFound), errors.Is(err, evidence.ErrSubjectUnsupported), errors.Is(err, evidence.ErrSubjectScopeMismatch), errors.Is(err, evidence.ErrSubjectAccessDenied), errors.Is(err, evidence.ErrRecipientManagerRequired):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Evidence request not found.")
	case errors.Is(err, evidence.ErrRecipientCandidateSearchInvalid):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_search", "Search must be 100 characters or fewer.")
	case err != nil:
		httpx.WriteError(w, http.StatusInternalServerError, "recipient_candidates_failed", "Recipient candidates could not be loaded.")
	default:
		httpx.WriteJSON(w, http.StatusOK, page)
	}
}
