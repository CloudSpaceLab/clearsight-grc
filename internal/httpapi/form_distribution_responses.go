package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) listFormDistributionResponses(w http.ResponseWriter, r *http.Request) {
	service, ok := a.formDistributionService(w)
	if !ok {
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
	limit := 100
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 100 {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_distribution_filter", "Choose a response revision limit from 1 to 100.")
			return
		}
		limit = value
	}
	values, err := service.ListResponseRevisions(r.Context(), actor.TenantID, legalEntityID, r.PathValue("id"), limit)
	if err != nil {
		writeFormDistributionError(w, err)
		return
	}
	items := make([]responseRevisionResponse, 0, len(values))
	for _, value := range values {
		items = append(items, responseRevisionJSON(value))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}
