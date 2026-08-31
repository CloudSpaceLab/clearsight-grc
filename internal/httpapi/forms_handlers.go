package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

type createLibraryFormRequest struct {
	TenantID         string                     `json:"tenant_id,omitempty"`
	LegalEntityID    string                     `json:"legal_entity_id,omitempty"`
	Code             string                     `json:"code"`
	Name             string                     `json:"name"`
	Purpose          string                     `json:"purpose"`
	ProgramID        string                     `json:"program_id,omitempty"`
	OwnerPrincipalID string                     `json:"owner_principal_id,omitempty"`
	ResponsibleTeam  string                     `json:"responsible_team,omitempty"`
	ApprovedUses     []string                   `json:"approved_uses,omitempty"`
	Tags             []string                   `json:"tags,omitempty"`
	Jurisdiction     string                     `json:"jurisdiction,omitempty"`
	Industry         string                     `json:"industry,omitempty"`
	Sensitivity      string                     `json:"sensitivity,omitempty"`
	ScoringMode      monitoringFormScoringMode  `json:"scoring_mode,omitempty"`
	NextReviewAt     *monitoringFormTime        `json:"next_review_at,omitempty"`
	Presentation     monitoringFormPresentation `json:"presentation"`
	Sections         []monitoringFormSection    `json:"sections"`
	Fields           []monitoring.TemplateField `json:"fields"`
}

// Local aliases keep the request surface explicit while sharing the canonical
// validation and JSON representation owned by the form contract package.
type monitoringFormScoringMode = formcontract.ScoringMode
type monitoringFormTime = time.Time
type monitoringFormPresentation = formcontract.Presentation
type monitoringFormSection = formcontract.Section

type createLibraryFormRevisionRequest struct {
	TenantID        string                   `json:"tenant_id,omitempty"`
	ExpectedVersion int64                    `json:"expected_version"`
	Form            createLibraryFormRequest `json:"form"`
}

type transitionLibraryFormRequest struct {
	TenantID        string                     `json:"tenant_id,omitempty"`
	ExpectedVersion int64                      `json:"expected_version"`
	To              monitoring.LifecycleStatus `json:"to"`
}

type instantiateStarterFormRequest struct {
	TenantID         string     `json:"tenant_id,omitempty"`
	LegalEntityID    string     `json:"legal_entity_id,omitempty"`
	Code             string     `json:"code,omitempty"`
	Name             string     `json:"name,omitempty"`
	Purpose          string     `json:"purpose,omitempty"`
	ProgramID        string     `json:"program_id,omitempty"`
	OwnerPrincipalID string     `json:"owner_principal_id,omitempty"`
	ResponsibleTeam  string     `json:"responsible_team,omitempty"`
	Jurisdiction     string     `json:"jurisdiction,omitempty"`
	Industry         string     `json:"industry,omitempty"`
	NextReviewAt     *time.Time `json:"next_review_at,omitempty"`
}

type saveFormViewRequest struct {
	ID     string                       `json:"id,omitempty"`
	Name   string                       `json:"name"`
	Filter monitoring.FormLibraryFilter `json:"filter"`
}

func (request createLibraryFormRequest) input() monitoring.CreateFormInput {
	return monitoring.CreateFormInput{
		ProgramID: request.ProgramID, Code: request.Code, Name: request.Name, Purpose: request.Purpose,
		OwnerPrincipalID: request.OwnerPrincipalID, ResponsibleTeam: request.ResponsibleTeam,
		ApprovedUses: request.ApprovedUses, Tags: request.Tags, Jurisdiction: request.Jurisdiction, Industry: request.Industry,
		Sensitivity: request.Sensitivity, ScoringMode: request.ScoringMode, NextReviewAt: request.NextReviewAt,
		Presentation: request.Presentation, Sections: request.Sections, Fields: request.Fields,
	}
}

func (a *API) listLibraryForms(w http.ResponseWriter, r *http.Request) {
	service, ok := a.monitoringService(w)
	if !ok {
		return
	}
	filter, err := formLibraryFilterFromRequest(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_form_filter", err.Error())
		return
	}
	includeStatusFacets, err := formLibraryStatusFacetsFromRequest(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_form_filter", err.Error())
		return
	}
	var page monitoring.FormTemplatePage
	if filter.Expression != nil || includeStatusFacets {
		page, err = service.ListAdvancedFormLibrary(r.Context(), filter, includeStatusFacets)
	} else {
		page, err = service.ListFormLibrary(r.Context(), filter)
	}
	if err != nil {
		writeFormsError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, page)
}

func (a *API) createLibraryForm(w http.ResponseWriter, r *http.Request) {
	service, ok := a.monitoringService(w)
	if !ok {
		return
	}
	var request createLibraryFormRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := service.CreateLibraryForm(r.Context(), request.input())
	if err != nil {
		writeFormsError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, value)
}

func (a *API) getLibraryFormRevision(w http.ResponseWriter, r *http.Request) {
	service, ok := a.monitoringService(w)
	if !ok {
		return
	}
	version, err := strconv.ParseInt(r.PathValue("version"), 10, 64)
	if err != nil || version < 1 {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_form_version", "Choose a valid form revision.")
		return
	}
	value, err := service.GetLibraryForm(r.Context(), r.PathValue("id"), version)
	if err != nil {
		writeFormsError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) createLibraryFormRevision(w http.ResponseWriter, r *http.Request) {
	service, ok := a.monitoringService(w)
	if !ok {
		return
	}
	var request createLibraryFormRevisionRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := service.CreateFormRevision(r.Context(), r.PathValue("id"), monitoring.CreateFormRevisionInput{ExpectedVersion: request.ExpectedVersion, Form: request.Form.input()})
	if err != nil {
		writeFormsError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, value)
}

func (a *API) transitionLibraryForm(w http.ResponseWriter, r *http.Request) {
	service, ok := a.monitoringService(w)
	if !ok {
		return
	}
	var request transitionLibraryFormRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := service.TransitionLibraryForm(r.Context(), r.PathValue("id"), monitoring.TransitionInput{ExpectedVersion: request.ExpectedVersion, To: request.To})
	if err != nil {
		writeFormsError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) listStarterForms(w http.ResponseWriter, r *http.Request) {
	service, ok := a.monitoringService(w)
	if !ok {
		return
	}
	items, err := service.ListStarterTemplates(r.Context())
	if err != nil {
		writeFormsError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *API) instantiateStarterForm(w http.ResponseWriter, r *http.Request) {
	service, ok := a.monitoringService(w)
	if !ok {
		return
	}
	var request instantiateStarterFormRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := service.InstantiateStarterTemplate(r.Context(), r.PathValue("code"), monitoring.InstantiateStarterTemplateInput{
		Code: request.Code, Name: request.Name, Purpose: request.Purpose, ProgramID: request.ProgramID,
		OwnerPrincipalID: request.OwnerPrincipalID, ResponsibleTeam: request.ResponsibleTeam,
		Jurisdiction: request.Jurisdiction, Industry: request.Industry, NextReviewAt: request.NextReviewAt,
	})
	if err != nil {
		writeFormsError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, value)
}

func (a *API) listSavedFormViews(w http.ResponseWriter, r *http.Request) {
	service, ok := a.monitoringService(w)
	if !ok {
		return
	}
	items, err := service.ListSavedFormViews(r.Context())
	if err != nil {
		writeFormsError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *API) saveFormView(w http.ResponseWriter, r *http.Request) {
	service, ok := a.monitoringService(w)
	if !ok {
		return
	}
	var request saveFormViewRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := service.SaveFormView(r.Context(), monitoring.SavedFormView{ID: request.ID, Name: request.Name, Filter: request.Filter})
	if err != nil {
		writeFormsError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) deleteSavedFormView(w http.ResponseWriter, r *http.Request) {
	service, ok := a.monitoringService(w)
	if !ok {
		return
	}
	if err := service.DeleteSavedFormView(r.Context(), r.PathValue("id")); err != nil {
		writeFormsError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func formLibraryFilterFromRequest(r *http.Request) (monitoring.FormLibraryFilter, error) {
	allowed := map[string]bool{"tenant_id": true, "search": true, "status": true, "owner": true, "program": true, "use": true, "tag": true, "filter": true, "facets": true, "cursor": true, "limit": true}
	for key := range r.URL.Query() {
		if !allowed[key] {
			return monitoring.FormLibraryFilter{}, errors.New("Use only the supported form search and filter options.")
		}
	}
	limit := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 100 {
			return monitoring.FormLibraryFilter{}, errors.New("Choose a page size from 1 to 100 forms.")
		}
		limit = value
	}
	expression, err := formLibraryExpressionFromRequest(r)
	if err != nil {
		return monitoring.FormLibraryFilter{}, err
	}
	return monitoring.FormLibraryFilter{
		Search: r.URL.Query().Get("search"), Status: monitoring.LifecycleStatus(r.URL.Query().Get("status")),
		OwnerPrincipalID: r.URL.Query().Get("owner"), ProgramID: r.URL.Query().Get("program"), Use: r.URL.Query().Get("use"), Tag: r.URL.Query().Get("tag"),
		Expression: expression, Cursor: r.URL.Query().Get("cursor"), Limit: limit,
	}, nil
}

func writeFormsError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, identity.ErrMissingIdentity), errors.Is(err, identity.ErrInvalidIdentity), errors.Is(err, identity.ErrExpiredIdentity), errors.Is(err, commandauth.ErrIdentityRequired):
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to work with forms.")
	case errors.Is(err, commandauth.ErrNotAuthorized), errors.Is(err, commandauth.ErrTenantMismatch), errors.Is(err, commandauth.ErrLegalEntityMismatch):
		writeCommandAuthorizationError(w, err)
	case errors.Is(err, commandauth.ErrGuardUnavailable):
		httpx.WriteError(w, http.StatusServiceUnavailable, "form_authority_unavailable", "The current approval route could not be checked. No form was changed.")
	case errors.Is(err, monitoring.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "form_not_found", "The form template or saved view was not found in this legal entity.")
	case errors.Is(err, monitoring.ErrConflict), errors.Is(err, monitoring.ErrMakerChecker), errors.Is(err, monitoring.ErrInactive):
		httpx.WriteError(w, http.StatusConflict, "form_conflict", err.Error())
	case errors.Is(err, monitoring.ErrInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "form_invalid", err.Error())
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "form_failed", "The form change could not be completed.")
	}
}
