package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

type createVendorRelationshipRequest struct {
	thirdparty.CreateRelationshipInput
	TenantID      string `json:"tenant_id,omitempty"`
	LegalEntityID string `json:"legal_entity_id,omitempty"`
	ActorID       string `json:"actor_id,omitempty"`
}

type updateVendorRelationshipRequest struct {
	thirdparty.UpdateRelationshipInput
	TenantID      string `json:"tenant_id,omitempty"`
	LegalEntityID string `json:"legal_entity_id,omitempty"`
	ActorID       string `json:"actor_id,omitempty"`
}

func (a *API) thirdPartyService(w http.ResponseWriter) (*thirdparty.Service, bool) {
	if a.deps.ThirdParty == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "vendors_unavailable", "Vendor records are temporarily unavailable. No change was made.")
		return nil, false
	}
	return a.deps.ThirdParty, true
}

func thirdPartyActor(r *http.Request) (thirdparty.Actor, error) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		return thirdparty.Actor{}, err
	}
	return thirdparty.Actor{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PrincipalID: actor.PrincipalID}, nil
}

func (a *API) listVendorRelationships(w http.ResponseWriter, r *http.Request) {
	service, ok := a.thirdPartyService(w)
	if !ok {
		return
	}
	actor, err := thirdPartyActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to view vendor records.")
		return
	}
	if requested := strings.TrimSpace(r.URL.Query().Get("legal_entity_id")); requested != "" && requested != actor.LegalEntityID {
		httpx.WriteError(w, http.StatusForbidden, "legal_entity_not_allowed", "Vendor records for that legal entity are outside your signed-in scope.")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	page, err := service.ListRelationships(r.Context(), actor, thirdparty.ListInput{Search: r.URL.Query().Get("search"), Cursor: r.URL.Query().Get("cursor"), Limit: limit})
	if err != nil {
		writeThirdPartyError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, page)
}

func (a *API) getVendorRelationship(w http.ResponseWriter, r *http.Request) {
	service, ok := a.thirdPartyService(w)
	if !ok {
		return
	}
	actor, err := thirdPartyActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to view this vendor record.")
		return
	}
	value, err := service.GetRelationship(r.Context(), actor, r.PathValue("id"))
	if err != nil {
		writeThirdPartyError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) createVendorRelationship(w http.ResponseWriter, r *http.Request) {
	service, ok := a.thirdPartyService(w)
	if !ok {
		return
	}
	actor, err := thirdPartyActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to add a vendor relationship.")
		return
	}
	var request createVendorRelationshipRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Check the vendor fields and submit one JSON object.")
		return
	}
	value, err := service.CreateRelationship(r.Context(), actor, request.CreateRelationshipInput)
	if err != nil {
		writeThirdPartyError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, value)
}

func (a *API) updateVendorRelationship(w http.ResponseWriter, r *http.Request) {
	service, ok := a.thirdPartyService(w)
	if !ok {
		return
	}
	actor, err := thirdPartyActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to update this vendor relationship.")
		return
	}
	var request updateVendorRelationshipRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Check the vendor fields and submit one JSON object.")
		return
	}
	value, err := service.UpdateRelationship(r.Context(), actor, r.PathValue("id"), request.UpdateRelationshipInput)
	if err != nil {
		writeThirdPartyError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func writeThirdPartyError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, thirdparty.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "vendor_not_found", "This vendor relationship was not found in your current legal-entity scope.")
	case errors.Is(err, thirdparty.ErrVersionConflict):
		httpx.WriteError(w, http.StatusConflict, "vendor_version_conflict", "This vendor relationship changed. Reload the record before saving again.")
	case errors.Is(err, thirdparty.ErrInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "vendor_invalid", "Add the vendor legal name, service, criticality and privacy role, then try again.")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "vendor_change_failed", "The vendor change could not be completed. No change was made.")
	}
}
