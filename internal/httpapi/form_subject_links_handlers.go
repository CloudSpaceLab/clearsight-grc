package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

// Every form-link route is bound to an existing, currently readable Matter
// before the Forms service can read or change subject-associated records.
func (a *API) readableFormLinkMatter(w http.ResponseWriter, r *http.Request) (string, bool) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required.")
		return "", false
	}
	service, ok := a.continuityService(w)
	if !ok {
		return "", false
	}
	matterID := r.PathValue("id")
	matter, err := service.GetMatter(r.Context(), actor.TenantID, matterID)
	if err != nil || matter.Matter.LegalEntityID != actor.LegalEntityID ||
		!canReadMatterAggregate(r.Context(), matter) {
		writeContinuityError(w, continuity.ErrNotFound)
		return "", false
	}
	return matterID, true
}

func (a *API) listMatterFormLinks(w http.ResponseWriter, r *http.Request) {
	subjectID, ok := a.readableFormLinkMatter(w, r)
	if !ok {
		return
	}
	service, ok := a.monitoringService(w)
	if !ok {
		return
	}
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 50 {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_form_link_limit", "Choose 1 to 50 linked forms.")
			return
		}
		limit = value
	}
	page, err := service.ListLinkedForms(r.Context(), monitoring.FormSubjectMatter, subjectID, r.URL.Query().Get("cursor"), limit)
	if err != nil {
		writeFormsError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, page)
}

func (a *API) createMatterFormLink(w http.ResponseWriter, r *http.Request) {
	subjectID, ok := a.readableFormLinkMatter(w, r)
	if !ok {
		return
	}
	service, ok := a.monitoringService(w)
	if !ok {
		return
	}
	var input struct {
		FormTemplateID      string `json:"form_template_id"`
		FormTemplateVersion int64  `json:"form_template_version"`
	}
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_form_link", err.Error())
		return
	}
	link, err := service.LinkExistingForm(r.Context(), monitoring.FormSubjectMatter, subjectID, input.FormTemplateID, input.FormTemplateVersion)
	if err != nil {
		writeFormsError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, link)
}

func (a *API) retireMatterFormLink(w http.ResponseWriter, r *http.Request) {
	subjectID, ok := a.readableFormLinkMatter(w, r)
	if !ok {
		return
	}
	service, ok := a.monitoringService(w)
	if !ok {
		return
	}
	var input struct {
		ExpectedVersion int64  `json:"expected_version"`
		Reason          string `json:"reason"`
	}
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_form_link", err.Error())
		return
	}
	link, err := service.UnlinkForm(r.Context(), monitoring.FormSubjectMatter, subjectID, r.PathValue("link_id"), input.ExpectedVersion, input.Reason)
	if err != nil {
		writeFormsError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, link)
}
