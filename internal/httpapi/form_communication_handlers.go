package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

type createCommunicationProfileRequest struct {
	TenantID       string     `json:"tenant_id,omitempty"`
	LegalEntityID  string     `json:"legal_entity_id,omitempty"`
	DefaultLocale  string     `json:"default_locale"`
	BankName       string     `json:"bank_name"`
	SupportContact string     `json:"support_contact,omitempty"`
	BrandAssetID   string     `json:"brand_asset_id,omitempty"`
	EffectiveFrom  time.Time  `json:"effective_from"`
	EffectiveUntil *time.Time `json:"effective_until,omitempty"`
}

type createCommunicationTemplateRequest struct {
	TenantID        string                       `json:"tenant_id,omitempty"`
	LegalEntityID   string                       `json:"legal_entity_id,omitempty"`
	Action          evidence.CommunicationAction `json:"action"`
	Locale          string                       `json:"locale"`
	SubjectTemplate string                       `json:"subject_template"`
	Document        []evidence.CommunicationNode `json:"document"`
	EffectiveFrom   time.Time                    `json:"effective_from"`
	EffectiveUntil  *time.Time                   `json:"effective_until,omitempty"`
}

type communicationTransitionRequest struct {
	TenantID        string                       `json:"tenant_id,omitempty"`
	LegalEntityID   string                       `json:"legal_entity_id,omitempty"`
	ExpectedVersion int64                        `json:"expected_version"`
	To              evidence.CommunicationStatus `json:"to"`
	EffectiveFrom   *time.Time                   `json:"effective_from,omitempty"`
	EffectiveUntil  *time.Time                   `json:"effective_until,omitempty"`
}

type communicationTestSendRequest struct {
	TenantID      string `json:"tenant_id,omitempty"`
	LegalEntityID string `json:"legal_entity_id,omitempty"`
	Address       string `json:"address"`
}

func (a *API) formCommunicationService(w http.ResponseWriter) (*evidence.CommunicationService, bool) {
	if a.deps.FormCommunications == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "form_communications_unavailable", "Form communication configuration is unavailable.")
		return nil, false
	}
	return a.deps.FormCommunications, true
}

func (a *API) listCommunicationProfiles(w http.ResponseWriter, r *http.Request) {
	service, actor, legalEntityID, ok := a.communicationContext(w, r, r.URL.Query().Get("legal_entity_id"))
	if !ok {
		return
	}
	values, err := service.ListProfiles(r.Context(), actor.TenantID, legalEntityID)
	if err != nil {
		writeCommunicationError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": values})
}

func (a *API) createCommunicationProfile(w http.ResponseWriter, r *http.Request) {
	service, ok := a.formCommunicationService(w)
	if !ok {
		return
	}
	actor, ok := distributionActor(w, r)
	if !ok {
		return
	}
	var request createCommunicationProfileRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "communication_invalid", "The communication profile request must be valid JSON.")
		return
	}
	legalEntityID, ok := distributionLegalEntity(w, r, actor, request.LegalEntityID)
	if !ok {
		return
	}
	value, err := service.CreateProfileRevision(r.Context(), evidence.CreateCommunicationProfileInput{
		TenantID: actor.TenantID, LegalEntityID: legalEntityID, DefaultLocale: request.DefaultLocale,
		BankName: request.BankName, SupportContact: request.SupportContact, BrandAssetID: request.BrandAssetID,
		EffectiveFrom: request.EffectiveFrom, EffectiveUntil: request.EffectiveUntil, MakerID: actor.PrincipalID,
	})
	if err != nil {
		writeCommunicationError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, value)
}

func (a *API) transitionCommunicationProfile(w http.ResponseWriter, r *http.Request) {
	service, actor, legalEntityID, ok := a.communicationContext(w, r, r.URL.Query().Get("legal_entity_id"))
	if !ok {
		return
	}
	version, ok := communicationVersion(w, r.PathValue("version"))
	if !ok {
		return
	}
	var request communicationTransitionRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil || request.ExpectedVersion != version {
		httpx.WriteError(w, http.StatusBadRequest, "communication_invalid", "The current communication revision version is required.")
		return
	}
	value, err := service.TransitionProfile(r.Context(), actor.TenantID, legalEntityID, version, evidence.CommunicationTransitionInput{
		ExpectedVersion: version, To: request.To, ActorID: actor.PrincipalID,
		EffectiveFrom: request.EffectiveFrom, EffectiveUntil: request.EffectiveUntil,
	})
	if err != nil {
		writeCommunicationError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) rollbackCommunicationProfile(w http.ResponseWriter, r *http.Request) {
	service, actor, legalEntityID, ok := a.communicationContext(w, r, r.URL.Query().Get("legal_entity_id"))
	if !ok {
		return
	}
	version, ok := communicationVersion(w, r.PathValue("version"))
	if !ok {
		return
	}
	value, err := service.RollbackProfile(r.Context(), actor.TenantID, legalEntityID, version, actor.PrincipalID)
	if err != nil {
		writeCommunicationError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, value)
}

func (a *API) listCommunicationTemplates(w http.ResponseWriter, r *http.Request) {
	service, actor, legalEntityID, ok := a.communicationContext(w, r, r.URL.Query().Get("legal_entity_id"))
	if !ok {
		return
	}
	query := evidence.CommunicationTemplateQuery{
		TenantID: actor.TenantID, LegalEntityID: legalEntityID,
		Action: evidence.CommunicationAction(strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("action")))),
		Locale: strings.TrimSpace(r.URL.Query().Get("locale")),
		Status: evidence.CommunicationStatus(strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))),
	}
	values, err := service.ListTemplates(r.Context(), query)
	if err != nil {
		writeCommunicationError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": values})
}

func (a *API) createCommunicationTemplate(w http.ResponseWriter, r *http.Request) {
	service, ok := a.formCommunicationService(w)
	if !ok {
		return
	}
	actor, ok := distributionActor(w, r)
	if !ok {
		return
	}
	var request createCommunicationTemplateRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "communication_invalid", "The communication template request must be valid JSON.")
		return
	}
	legalEntityID, ok := distributionLegalEntity(w, r, actor, request.LegalEntityID)
	if !ok {
		return
	}
	value, err := service.CreateTemplateRevision(r.Context(), evidence.CreateCommunicationTemplateInput{
		TenantID: actor.TenantID, LegalEntityID: legalEntityID, Action: request.Action, Locale: request.Locale,
		SubjectTemplate: request.SubjectTemplate, Document: request.Document,
		EffectiveFrom: request.EffectiveFrom, EffectiveUntil: request.EffectiveUntil, MakerID: actor.PrincipalID,
	})
	if err != nil {
		writeCommunicationError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, value)
}

func (a *API) getCommunicationTemplate(w http.ResponseWriter, r *http.Request) {
	service, actor, legalEntityID, action, locale, version, ok := a.communicationTemplateContext(w, r)
	if !ok {
		return
	}
	value, err := service.GetTemplate(r.Context(), actor.TenantID, legalEntityID, action, locale, version)
	if err != nil {
		writeCommunicationError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) previewCommunicationTemplate(w http.ResponseWriter, r *http.Request) {
	service, actor, legalEntityID, action, locale, version, ok := a.communicationTemplateContext(w, r)
	if !ok {
		return
	}
	value, err := service.Preview(r.Context(), actor.TenantID, legalEntityID, action, locale, version)
	if err != nil {
		writeCommunicationError(w, err)
		return
	}
	secureNoStore(w)
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) impactCommunicationTemplate(w http.ResponseWriter, r *http.Request) {
	service, actor, legalEntityID, action, locale, version, ok := a.communicationTemplateContext(w, r)
	if !ok {
		return
	}
	value, err := service.Impact(r.Context(), actor.TenantID, legalEntityID, action, locale, version)
	if err != nil {
		writeCommunicationError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) transitionCommunicationTemplate(w http.ResponseWriter, r *http.Request) {
	service, actor, legalEntityID, action, locale, version, ok := a.communicationTemplateContext(w, r)
	if !ok {
		return
	}
	var request communicationTransitionRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil || request.ExpectedVersion != version {
		httpx.WriteError(w, http.StatusBadRequest, "communication_invalid", "The current communication revision version is required.")
		return
	}
	value, err := service.TransitionTemplate(r.Context(), actor.TenantID, legalEntityID, action, locale, version, evidence.CommunicationTransitionInput{
		ExpectedVersion: version, To: request.To, ActorID: actor.PrincipalID,
		EffectiveFrom: request.EffectiveFrom, EffectiveUntil: request.EffectiveUntil,
	})
	if err != nil {
		writeCommunicationError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) rollbackCommunicationTemplate(w http.ResponseWriter, r *http.Request) {
	service, actor, legalEntityID, action, locale, version, ok := a.communicationTemplateContext(w, r)
	if !ok {
		return
	}
	value, err := service.RollbackTemplate(r.Context(), actor.TenantID, legalEntityID, action, locale, version, actor.PrincipalID)
	if err != nil {
		writeCommunicationError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, value)
}

func (a *API) testSendCommunicationTemplate(w http.ResponseWriter, r *http.Request) {
	service, actor, legalEntityID, action, locale, version, ok := a.communicationTemplateContext(w, r)
	if !ok {
		return
	}
	if a.deps.FormCommunicationTestDelivery == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "communication_delivery_unavailable", "SMTP delivery is not configured.")
		return
	}
	var request communicationTestSendRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "communication_invalid", "A test recipient address is required.")
		return
	}
	receipt, err := service.TestSend(r.Context(), actor.TenantID, legalEntityID, action, locale, version, request.Address, a.deps.FormCommunicationTestDelivery)
	if err != nil {
		writeCommunicationError(w, err)
		return
	}
	secureNoStore(w)
	httpx.WriteJSON(w, http.StatusOK, receipt)
}

func (a *API) listCommunicationBrandAssets(w http.ResponseWriter, r *http.Request) {
	if a.deps.FormCommunicationBrands == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "communication_branding_unavailable", "Communication branding is unavailable.")
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
	values, err := a.deps.FormCommunicationBrands.ListLogos(r.Context(), actor.TenantID, legalEntityID)
	if err != nil {
		writeCommunicationError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": values})
}

func (a *API) uploadCommunicationBrandAsset(w http.ResponseWriter, r *http.Request) {
	if a.deps.FormCommunicationBrands == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "communication_branding_unavailable", "Communication branding is unavailable.")
		return
	}
	actor, ok := distributionActor(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 768<<10)
	if err := r.ParseMultipartForm(768 << 10); err != nil {
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, "communication_branding_invalid", "The logo must be a small PNG file.")
		return
	}
	legalEntityID, ok := distributionLegalEntity(w, r, actor, r.FormValue("legal_entity_id"))
	if !ok {
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "communication_branding_invalid", "Choose a PNG logo to upload.")
		return
	}
	defer file.Close()
	value, err := a.deps.FormCommunicationBrands.StoreLogo(r.Context(), evidence.CommunicationLogoUploadInput{
		TenantID: actor.TenantID, LegalEntityID: legalEntityID, FileName: header.Filename,
		MediaType: header.Header.Get("Content-Type"), AltText: r.FormValue("alt_text"), CreatedBy: actor.PrincipalID,
	}, file)
	if err != nil {
		writeCommunicationError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, value)
}

func (a *API) communicationContext(w http.ResponseWriter, r *http.Request, requestedEntity string) (*evidence.CommunicationService, identity.Actor, string, bool) {
	service, ok := a.formCommunicationService(w)
	if !ok {
		return nil, identity.Actor{}, "", false
	}
	actor, ok := distributionActor(w, r)
	if !ok {
		return nil, identity.Actor{}, "", false
	}
	legalEntityID, ok := distributionLegalEntity(w, r, actor, requestedEntity)
	if !ok {
		return nil, identity.Actor{}, "", false
	}
	return service, actor, legalEntityID, true
}

func (a *API) communicationTemplateContext(w http.ResponseWriter, r *http.Request) (*evidence.CommunicationService, identity.Actor, string, evidence.CommunicationAction, string, int64, bool) {
	service, actor, legalEntityID, ok := a.communicationContext(w, r, r.URL.Query().Get("legal_entity_id"))
	if !ok {
		return nil, identity.Actor{}, "", "", "", 0, false
	}
	version, ok := communicationVersion(w, r.PathValue("version"))
	if !ok {
		return nil, identity.Actor{}, "", "", "", 0, false
	}
	action := evidence.CommunicationAction(strings.ToUpper(strings.TrimSpace(r.PathValue("action"))))
	locale := strings.TrimSpace(r.PathValue("locale"))
	return service, actor, legalEntityID, action, locale, version, true
}

func communicationVersion(w http.ResponseWriter, raw string) (int64, bool) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || value < 1 {
		httpx.WriteError(w, http.StatusBadRequest, "communication_invalid", "A valid communication revision version is required.")
		return 0, false
	}
	return value, true
}

func writeCommunicationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, evidence.ErrCommunicationNotFound):
		httpx.WriteError(w, http.StatusNotFound, "communication_not_found", "The communication configuration was not found in this legal entity.")
	case errors.Is(err, evidence.ErrCommunicationConflict):
		httpx.WriteError(w, http.StatusConflict, "communication_conflict", "The communication revision changed. Refresh before trying again.")
	case errors.Is(err, evidence.ErrCommunicationUnavailable):
		httpx.WriteError(w, http.StatusConflict, "communication_unavailable", "Configure an active profile and message for this action and locale.")
	case errors.Is(err, evidence.ErrCommunicationInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "communication_invalid", "The communication configuration is invalid.")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "communication_failed", "The communication operation could not be completed.")
	}
}
