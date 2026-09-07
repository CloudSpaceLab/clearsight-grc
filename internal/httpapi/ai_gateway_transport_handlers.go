package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/aigateway"
	"github.com/CloudSpaceLab/clearsight-grc/internal/aigovernance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

type aiGatewayRuntimeStatus struct {
	Configured         bool   `json:"configured"`
	Available          bool   `json:"available"`
	TenantID           string `json:"tenant_id"`
	Environment        string `json:"environment"`
	DesiredRevision    int64  `json:"desired_revision"`
	DesiredChecksum    string `json:"desired_checksum,omitempty"`
	AppliedRevision    int64  `json:"applied_revision"`
	AppliedChecksum    string `json:"applied_checksum,omitempty"`
	EmergencySupported bool   `json:"emergency_supported"`
	EmergencyRevision  int64  `json:"emergency_revision"`
	OutboundFrozen     bool   `json:"outbound_frozen"`
	Degraded           bool   `json:"degraded"`
	ErrorCode          string `json:"error_code,omitempty"`
}

type aiGatewayIngressRoute struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

type aiGatewayProxyInfo struct {
	Configured bool                    `json:"configured"`
	BaseURL    string                  `json:"base_url,omitempty"`
	Ingress    []aiGatewayIngressRoute `json:"ingress"`
}

func (a *API) listAIGatewayTransports(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to continue.")
		return
	}
	environment := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("environment")))
	items, err := a.deps.AIGovernance.ListGatewayTransports(r.Context(), actor.TenantID, environment, queryLimit(r, 50))
	if err != nil {
		writeAIGovernanceResult(w, map[string]any{}, err, http.StatusOK)
		return
	}
	emergency := aigovernance.GatewayEmergencyControl{TenantID: actor.TenantID, Environment: environment}
	if environment != "" {
		emergency, err = a.deps.AIGovernance.GatewayEmergencyControl(r.Context(), actor.TenantID, environment)
		if err != nil {
			writeAIGovernanceResult(w, map[string]any{}, err, http.StatusOK)
			return
		}
	}
	response := map[string]any{
		"items":             items,
		"runtime_status":    a.aiGatewayRuntimeStatus(r.Context(), actor.TenantID, environment),
		"proxy":             a.aiGatewayProxyInfo(),
		"emergency_control": emergency,
	}
	if fixture := strings.TrimSpace(r.URL.Query().Get("simulate_fixture")); fixture != "" {
		simulation, simErr := a.deps.AIGovernance.SimulateGateway(r.Context(), aigovernance.GatewaySimulationInput{
			TenantID:         actor.TenantID,
			Environment:      environment,
			Fixture:          fixture,
			WorkloadRecordID: strings.TrimSpace(r.URL.Query().Get("simulate_workload_id")),
			BaselinePolicyID: strings.TrimSpace(r.URL.Query().Get("simulate_baseline_policy_id")),
			TransportID:      strings.TrimSpace(r.URL.Query().Get("simulate_transport_id")),
			ModelAlias:       strings.TrimSpace(r.URL.Query().Get("simulate_model_alias")),
		})
		if simErr != nil {
			writeAIGovernanceResult(w, map[string]any{}, simErr, http.StatusOK)
			return
		}
		response["simulation"] = simulation
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, response)
}

func (a *API) aiGatewayProxyInfo() aiGatewayProxyInfo {
	ingress := make([]aiGatewayIngressRoute, 0, 3)
	for _, route := range aigateway.GatewayRoutes() {
		if route.Access != aigateway.RouteWorkload {
			continue
		}
		ingress = append(ingress, aiGatewayIngressRoute{Method: route.Method, Path: route.Path})
	}
	return aiGatewayProxyInfo{
		Configured: strings.TrimSpace(a.deps.AIGatewayPublicBaseURL) != "",
		BaseURL:    strings.TrimRight(strings.TrimSpace(a.deps.AIGatewayPublicBaseURL), "/"),
		Ingress:    ingress,
	}
}

func (a *API) aiGatewayRuntimeStatus(ctx context.Context, tenantID, environment string) aiGatewayRuntimeStatus {
	status := aiGatewayRuntimeStatus{TenantID: tenantID, Environment: environment}
	if environment == "" || a.deps.AIGatewayOperations == nil {
		return status
	}
	status.Configured = true
	actual, err := a.deps.AIGatewayOperations.TransportStatus(ctx, tenantID, environment)
	if err != nil {
		status.Degraded = true
		status.ErrorCode = "GATEWAY_STATUS_UNAVAILABLE"
		return status
	}
	status.Available = true
	status.DesiredRevision = actual.DesiredRevision
	status.DesiredChecksum = actual.DesiredChecksum
	status.AppliedRevision = actual.AppliedRevision
	status.AppliedChecksum = actual.AppliedChecksum
	status.EmergencySupported = actual.EmergencySupported
	status.EmergencyRevision = actual.EmergencyRevision
	status.OutboundFrozen = actual.OutboundFrozen
	status.Degraded = actual.Degraded
	status.ErrorCode = actual.ErrorCode
	return status
}

func (a *API) setAIGatewayEmergencyControl(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to continue.")
		return
	}
	var input aigovernance.SetGatewayEmergencyControlInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.TenantID = actor.TenantID
	input.ActorID = actor.PrincipalID
	value, err := a.deps.AIGovernance.SetGatewayEmergencyControl(r.Context(), input)
	writeAIGovernanceResult(w, value, err, http.StatusOK)
}

func (a *API) getAIGatewayTransport(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to continue.")
		return
	}
	value, err := a.deps.AIGovernance.GetGatewayTransport(r.Context(), actor.TenantID, r.PathValue("id"))
	writeAIGovernanceResult(w, value, err, http.StatusOK)
}

func (a *API) getActiveAIGatewayTransport(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to continue.")
		return
	}
	environment := strings.TrimSpace(r.URL.Query().Get("environment"))
	if environment == "" {
		environment = "PRODUCTION"
	}
	value, err := a.deps.AIGovernance.ActiveGatewayTransport(r.Context(), actor.TenantID, environment)
	writeAIGovernanceResult(w, value, err, http.StatusOK)
}

func (a *API) createAIGatewayTransport(w http.ResponseWriter, r *http.Request) {
	var input aigovernance.CreateGatewayTransportInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := a.deps.AIGovernance.CreateGatewayTransport(r.Context(), input)
	writeAIGovernanceResult(w, value, err, http.StatusCreated)
}

func (a *API) aiGatewayTransportAction(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, err := identity.Require(r.Context())
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to continue.")
			return
		}
		var input aigovernance.GatewayTransportTransitionInput
		if err := httpx.DecodeJSON(w, r, &input); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		input.ID = r.PathValue("id")
		input.TenantID = actor.TenantID
		input.ActorID = actor.PrincipalID
		value, err := a.deps.AIGovernance.TransitionGatewayTransport(r.Context(), action, input)
		writeAIGovernanceResult(w, value, err, http.StatusOK)
	}
}
