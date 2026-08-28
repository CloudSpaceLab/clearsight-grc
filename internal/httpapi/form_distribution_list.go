package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) listFilteredFormDistributions(w http.ResponseWriter, r *http.Request) {
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
	allowed := map[string]bool{
		"tenant_id": true, "legal_entity_id": true, "status": true, "due_state": true,
		"subject_type": true, "subject_id": true, "owner": true, "cursor": true, "limit": true,
	}
	for key := range r.URL.Query() {
		if !allowed[key] {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_distribution_filter", "Use only the supported distribution filters.")
			return
		}
	}
	limit := 25
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 100 {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_distribution_filter", "Choose a page size from 1 to 100 distributions.")
			return
		}
		limit = value
	}
	page, err := service.List(r.Context(), evidence.DistributionListQuery{
		TenantID: actor.TenantID, LegalEntityID: legalEntityID,
		Status: evidence.DistributionStatus(strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))),
		DueState: evidence.DistributionDueState(strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("due_state")))),
		SubjectType: r.URL.Query().Get("subject_type"), SubjectID: r.URL.Query().Get("subject_id"),
		OwnerPrincipalID: r.URL.Query().Get("owner"), Cursor: strings.TrimSpace(r.URL.Query().Get("cursor")), Limit: limit,
	})
	if err != nil {
		writeFormDistributionError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, page)
}
