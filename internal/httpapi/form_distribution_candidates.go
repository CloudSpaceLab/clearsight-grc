package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) listFormDistributionRecipientCandidates(w http.ResponseWriter, r *http.Request) {
	if a.deps.Evidence == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "form_recipient_directory_unavailable", "Internal recipient search is unavailable.")
		return
	}
	actor, ok := distributionActor(w, r)
	if !ok {
		return
	}
	legalEntityID, ok := distributionLegalEntity(w, r, actor, r.URL.Query().Get("legal_entity_id"))
	if !ok {
		return
	}
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 50 {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_recipient_search", "Choose a recipient page size from 1 to 50.")
			return
		}
		limit = value
	}
	page, err := a.deps.Evidence.SearchDistributionRecipientCandidates(r.Context(), actor.TenantID, legalEntityID, evidence.RecipientCandidateSearch{
		Query: strings.TrimSpace(r.URL.Query().Get("search")), Limit: limit,
	})
	if err != nil {
		switch {
		case errors.Is(err, evidence.ErrRecipientCandidateSearchInvalid):
			httpx.WriteError(w, http.StatusBadRequest, "invalid_recipient_search", "Recipient search text is too long.")
		case errors.Is(err, evidence.ErrRecipientCandidatesUnavailable):
			httpx.WriteError(w, http.StatusServiceUnavailable, "form_recipient_directory_unavailable", "Internal recipient search is unavailable.")
		default:
			httpx.WriteError(w, http.StatusNotFound, "recipient_directory_not_found", "The recipient directory is unavailable in this legal entity.")
		}
		return
	}
	httpx.WriteJSON(w, http.StatusOK, page)
}
