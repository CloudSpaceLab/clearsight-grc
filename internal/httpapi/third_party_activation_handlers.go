package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

type activationPolicyTransitionRequest struct {
	ExpectedVersion int64  `json:"expected_version"`
	SimulationID    string `json:"simulation_id,omitempty"`
	Rationale       string `json:"rationale"`
}

func (a *API) activationService(w http.ResponseWriter) (*thirdparty.ActivationService, bool) {
	if a.deps.ThirdPartyActivation == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "vendor_activation_unavailable", "Vendor activation is temporarily unavailable. No relationship or policy was changed.")
		return nil, false
	}
	return a.deps.ThirdPartyActivation, true
}

func (a *API) proposeThirdPartyActivationPolicy(w http.ResponseWriter, r *http.Request) {
	service, ok := a.activationService(w)
	if !ok {
		return
	}
	var input thirdparty.ProposeActivationPolicyInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := service.ProposePolicy(r.Context(), input)
	writeThirdPartyActivationResult(w, value, err, http.StatusCreated)
}

func (a *API) getCurrentThirdPartyActivationPolicy(w http.ResponseWriter, r *http.Request) {
	service, ok := a.activationService(w)
	if !ok {
		return
	}
	actor, err := thirdPartyActor(r)
	if err != nil {
		writeThirdPartyActivationError(w, err)
		return
	}
	at := time.Now().UTC()
	if raw := strings.TrimSpace(r.URL.Query().Get("at")); raw != "" {
		parsed, parseErr := time.Parse(time.RFC3339, raw)
		if parseErr != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "The policy time must use RFC 3339 format.")
			return
		}
		at = parsed.UTC()
	}
	value, err := service.CurrentPolicy(r.Context(), actor.LegalEntityID, at)
	writeThirdPartyActivationResult(w, value, err, http.StatusOK)
}

func (a *API) simulateThirdPartyActivationPolicy(w http.ResponseWriter, r *http.Request) {
	service, ok := a.activationService(w)
	if !ok {
		return
	}
	value, err := service.SimulatePolicy(r.Context(), r.PathValue("id"))
	writeThirdPartyActivationResult(w, value, err, http.StatusOK)
}

func (a *API) submitThirdPartyActivationPolicy(w http.ResponseWriter, r *http.Request) {
	service, ok := a.activationService(w)
	if !ok {
		return
	}
	var input activationPolicyTransitionRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := service.SubmitPolicy(r.Context(), r.PathValue("id"), input.ExpectedVersion, input.Rationale)
	writeThirdPartyActivationResult(w, value, err, http.StatusOK)
}

func (a *API) approveThirdPartyActivationPolicy(w http.ResponseWriter, r *http.Request) {
	service, ok := a.activationService(w)
	if !ok {
		return
	}
	var input activationPolicyTransitionRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := service.ApprovePolicy(r.Context(), r.PathValue("id"), input.ExpectedVersion, input.SimulationID, input.Rationale)
	writeThirdPartyActivationResult(w, value, err, http.StatusOK)
}

func (a *API) rollbackThirdPartyActivationPolicy(w http.ResponseWriter, r *http.Request) {
	service, ok := a.activationService(w)
	if !ok {
		return
	}
	var input thirdparty.RollbackActivationPolicyInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := service.PrepareRollback(r.Context(), r.PathValue("id"), input)
	writeThirdPartyActivationResult(w, value, err, http.StatusCreated)
}

func (a *API) activateVendorRelationship(w http.ResponseWriter, r *http.Request) {
	service, ok := a.activationService(w)
	if !ok {
		return
	}
	var input thirdparty.ActivateRelationshipInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := service.ActivateRelationship(r.Context(), r.PathValue("id"), input)
	writeThirdPartyActivationResult(w, value, err, http.StatusOK)
}

func (a *API) getVendorRelationshipActivation(w http.ResponseWriter, r *http.Request) {
	service, ok := a.activationService(w)
	if !ok {
		return
	}
	value, err := service.RelationshipEligibility(r.Context(), r.PathValue("id"), time.Now().UTC())
	writeThirdPartyActivationResult(w, value, err, http.StatusOK)
}

func writeThirdPartyActivationResult(w http.ResponseWriter, value any, err error, status int) {
	if err != nil {
		writeThirdPartyActivationError(w, err)
		return
	}
	httpx.WriteJSON(w, status, value)
}

func writeThirdPartyActivationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, thirdparty.ErrActivationIneligible):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "activation_gates_incomplete", "This vendor relationship cannot be activated until every current policy gate is satisfied.")
	case errors.Is(err, thirdparty.ErrActivationPolicyUnavailable):
		httpx.WriteError(w, http.StatusConflict, "activation_policy_unavailable", "No single approved activation policy applies at the requested time.")
	case errors.Is(err, thirdparty.ErrActivationMakerChecker), errors.Is(err, commandauth.ErrNotAuthorized):
		httpx.WriteError(w, http.StatusForbidden, "activation_not_authorized", "The current authority route does not permit this activation decision.")
	case errors.Is(err, thirdparty.ErrActivationSimulationRequired):
		httpx.WriteError(w, http.StatusConflict, "activation_simulation_required", "Run a complete activation impact simulation for this policy revision before approving it.")
	case errors.Is(err, thirdparty.ErrVersionConflict):
		httpx.WriteError(w, http.StatusConflict, "version_conflict", "This vendor activation record changed. Reload before continuing.")
	case errors.Is(err, thirdparty.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "The vendor activation record was not found in the current legal entity.")
	case errors.Is(err, thirdparty.ErrInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "invalid_activation_request", "The activation request is incomplete or invalid.")
	case errors.Is(err, commandauth.ErrIdentityRequired), errors.Is(err, commandauth.ErrTenantMismatch), errors.Is(err, commandauth.ErrLegalEntityMismatch):
		httpx.WriteError(w, http.StatusUnauthorized, "verified_identity_required", "A current verified identity is required for vendor activation.")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "vendor_activation_failed", "Vendor activation could not be completed. No activation status was reported.")
	}
}
