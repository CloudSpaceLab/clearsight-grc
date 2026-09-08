package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) vendorFormsQuery(w http.ResponseWriter, r *http.Request) (*evidence.DistributionService, evidence.VendorFormsQuery, bool) {
	service, ok := a.formDistributionService(w)
	if !ok {
		return nil, evidence.VendorFormsQuery{}, false
	}
	actor, ok := distributionActor(w, r)
	if !ok {
		return nil, evidence.VendorFormsQuery{}, false
	}
	entity, ok := distributionLegalEntity(w, r, actor, r.URL.Query().Get("legal_entity_id"))
	if !ok {
		return nil, evidence.VendorFormsQuery{}, false
	}
	q := evidence.VendorFormsQuery{TenantID: actor.TenantID, LegalEntityID: entity, PrincipalID: actor.PrincipalID, RelationshipIDs: strings.Split(r.URL.Query().Get("relationship_ids"), ","), FormTemplateID: r.URL.Query().Get("form_template_id"), Filter: r.URL.Query().Get("filter"), Cursor: r.URL.Query().Get("cursor"), Limit: 25}
	if id := r.PathValue("id"); id != "" {
		q.RelationshipIDs = []string{id}
	}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil {
			writeCompletedResponseError(w, evidence.ErrDistributionInvalid)
			return nil, q, false
		}
		q.Limit = limit
	}
	return service, q, true
}
func (a *API) listVendorForms(w http.ResponseWriter, r *http.Request) {
	documentProtection(w)
	service, q, ok := a.vendorFormsQuery(w, r)
	if !ok {
		return
	}
	page, err := service.ListVendorForms(r.Context(), q)
	if err != nil {
		writeCompletedResponseError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, page)
}
func (a *API) summarizeVendorForms(w http.ResponseWriter, r *http.Request) {
	documentProtection(w)
	service, q, ok := a.vendorFormsQuery(w, r)
	if !ok {
		return
	}
	items, err := service.VendorFormSummaries(r.Context(), q)
	if err != nil {
		writeCompletedResponseError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}
