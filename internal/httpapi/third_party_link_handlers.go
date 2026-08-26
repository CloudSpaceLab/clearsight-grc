package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

type linkVendorRelationshipRequest struct {
	thirdparty.LinkRelationshipInput
	TenantID      string `json:"tenant_id,omitempty"`
	LegalEntityID string `json:"legal_entity_id,omitempty"`
	ActorID       string `json:"actor_id,omitempty"`
}

type endVendorRelationshipLinkRequest struct {
	thirdparty.EndRelationshipLinkInput
	TenantID      string `json:"tenant_id,omitempty"`
	LegalEntityID string `json:"legal_entity_id,omitempty"`
	ActorID       string `json:"actor_id,omitempty"`
}

func (a *API) thirdPartyLinkService(w http.ResponseWriter) (*thirdparty.RelationshipLinkService, bool) {
	if a.deps.ThirdPartyRelationshipLinks == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "vendor_links_unavailable", "Vendor links are temporarily unavailable. No change was made.")
		return nil, false
	}
	return a.deps.ThirdPartyRelationshipLinks, true
}

func (a *API) linkVendorRelationship(w http.ResponseWriter, r *http.Request) {
	service, ok := a.thirdPartyLinkService(w)
	if !ok {
		return
	}
	actor, err := thirdPartyActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to link this vendor.")
		return
	}
	var input linkVendorRelationshipRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Check the vendor link details and try again.")
		return
	}
	value, err := service.Link(r.Context(), actor, r.PathValue("id"), input.LinkRelationshipInput)
	if err != nil {
		writeThirdPartyLinkError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, value)
}

func (a *API) endVendorRelationshipLink(w http.ResponseWriter, r *http.Request) {
	service, ok := a.thirdPartyLinkService(w)
	if !ok {
		return
	}
	actor, err := thirdPartyActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to end this vendor link.")
		return
	}
	var input endVendorRelationshipLinkRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Add a reason for ending this vendor link.")
		return
	}
	value, err := service.End(r.Context(), actor, r.PathValue("link_id"), input.EndRelationshipLinkInput)
	if err != nil {
		writeThirdPartyLinkError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) listVendorRelationshipLinks(w http.ResponseWriter, r *http.Request) {
	service, ok := a.thirdPartyLinkService(w)
	if !ok {
		return
	}
	actor, err := thirdPartyActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to view vendor links.")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	input := thirdparty.RelationshipLinkListInput{
		RelationshipID: strings.TrimSpace(r.PathValue("id")), TargetType: thirdparty.LinkTargetType(strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("target_type")))),
		TargetID: r.URL.Query().Get("target_id"), IncludeEnded: strings.EqualFold(r.URL.Query().Get("include_ended"), "true"), Cursor: r.URL.Query().Get("cursor"), Limit: limit,
	}
	page, err := service.List(r.Context(), actor, input)
	if err != nil {
		writeThirdPartyLinkError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, page)
}

func writeThirdPartyLinkError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, commandauth.ErrNotAuthorized), errors.Is(err, thirdparty.ErrRelationshipLinkIdentityMismatch):
		httpx.WriteError(w, http.StatusForbidden, "vendor_link_not_allowed", "Your current authority does not allow this vendor link change.")
	case errors.Is(err, thirdparty.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "vendor_link_not_found", "The vendor or related record was not found in your current scope.")
	case errors.Is(err, thirdparty.ErrVersionConflict):
		httpx.WriteError(w, http.StatusConflict, "vendor_link_conflict", "This vendor link changed. Reload the record before trying again.")
	case errors.Is(err, thirdparty.ErrInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "vendor_link_invalid", "Check the related record, purpose and current version, then try again.")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "vendor_link_failed", "The vendor link could not be changed. No change was made.")
	}
}
