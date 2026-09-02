package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/onboarding"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
)

const maxWorkflowReadLimit = 200

func (a *API) live(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "live"})
}

func (a *API) ready(w http.ResponseWriter, _ *http.Request) {
	revision := strings.TrimSpace(a.deps.ReleaseSHA)
	if revision == "" {
		revision = "unknown"
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready", "mode": a.deps.Mode, "revision": revision})
}

func (a *API) resolveAuthority(w http.ResponseWriter, r *http.Request) {
	var input authority.ResolveInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	resolution, err := a.deps.Authority.Resolve(r.Context(), input)
	switch {
	case errors.Is(err, authority.ErrInvalidInput):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_authority_input", err.Error())
	case errors.Is(err, authority.ErrNoRoute):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "routing_failed", "No eligible route exists for the supplied scope and responsibility.")
	case err != nil:
		httpx.WriteError(w, http.StatusInternalServerError, "resolution_failed", "Authority could not be resolved.")
	default:
		httpx.WriteJSON(w, http.StatusOK, resolution)
	}
}

func (a *API) simulateAuthority(w http.ResponseWriter, r *http.Request) {
	var input authority.ResolveInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := a.deps.Authority.Simulate(r.Context(), input)
	if errors.Is(err, authority.ErrInvalidInput) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_authority_input", err.Error())
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "simulation_failed", "Routing could not be simulated.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

func (a *API) authorityIntegrity(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requiredQuery(w, r, "tenant_id")
	if !ok {
		return
	}
	findings, err := a.deps.Authority.Integrity(r.Context(), tenantID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "integrity_failed", "Routing integrity could not be evaluated.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"findings": findings, "checked_at": time.Now().UTC()})
}

func (a *API) authorityPolicies(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requiredQuery(w, r, "tenant_id")
	if !ok {
		return
	}
	values, err := a.deps.Authority.Policies(r.Context(), tenantID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "policies_failed", "Routing policies could not be loaded.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": values})
}

func (a *API) listWorkflowTasks(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	tenantID, ok := requiredQuery(w, r, "tenant_id")
	if !ok {
		return
	}
	if tenantID != actor.TenantID {
		httpx.WriteError(w, http.StatusForbidden, "tenant_not_allowed", "This request is outside your signed-in bank scope.")
		return
	}
	if requested := strings.TrimSpace(r.URL.Query().Get("principal_id")); requested != "" && requested != actor.PrincipalID {
		httpx.WriteError(w, http.StatusForbidden, "principal_not_allowed", "This request is outside your signed-in user scope.")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > maxWorkflowReadLimit {
		limit = 50
	}
	status := workflow.Status(strings.TrimSpace(r.URL.Query().Get("status")))
	values, err := a.deps.Workflow.List(r.Context(), workflow.ListFilter{
		TenantID: tenantID, PrincipalID: actor.PrincipalID, Status: status,
		ActiveOnly: status == "", VisibleMatterWorkOnly: true, Limit: limit,
	})
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "workflow_failed", "Tasks could not be loaded.")
		return
	}
	// Repository filtering happens before LIMIT. Recheck canonical access in Go
	// as defense-in-depth against future query changes.
	visible := values[:0]
	for _, task := range values {
		if workflow.MatterWorkVisibleTo(task, actor.PrincipalID) {
			visible = append(visible, task)
		}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": visible})
}

func (a *API) onboardingGuide(w http.ResponseWriter, r *http.Request) {
	guide, err := a.deps.Onboarding.Guide(r.URL.Query().Get("role"), r.URL.Query().Get("code"))
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Guide not found.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, guide)
}

func (a *API) onboardingState(w http.ResponseWriter, r *http.Request) {
	tenantID, tenantOK := requiredQuery(w, r, "tenant_id")
	principalID, principalOK := requiredQuery(w, r, "principal_id")
	guideCode, guideOK := requiredQuery(w, r, "guide_code")
	if !tenantOK || !principalOK || !guideOK {
		return
	}
	value, err := a.deps.Onboarding.State(r.Context(), tenantID, principalID, guideCode)
	if err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "state_failed", "Onboarding state could not be loaded.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) updateOnboardingState(w http.ResponseWriter, r *http.Request) {
	tenantID, tenantOK := requiredQuery(w, r, "tenant_id")
	principalID, principalOK := requiredQuery(w, r, "principal_id")
	guideCode, guideOK := requiredQuery(w, r, "guide_code")
	if !tenantOK || !principalOK || !guideOK {
		return
	}
	var input onboarding.UpdateInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := a.deps.Onboarding.Update(r.Context(), tenantID, principalID, guideCode, input)
	if errors.Is(err, onboarding.ErrVersionConflict) {
		httpx.WriteError(w, http.StatusConflict, "version_conflict", "Onboarding state changed. Reload before updating.")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "state_failed", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) readiness(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requiredQuery(w, r, "tenant_id")
	if !ok {
		return
	}
	value, err := a.deps.Autonomy.Readiness(r.Context(), tenantID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "readiness_failed", "Continuous readiness could not be calculated.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) ingestSignal(w http.ResponseWriter, r *http.Request) {
	var input autonomy.Signal
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	drift, inserted, err := a.deps.Autonomy.Ingest(r.Context(), input)
	if err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "signal_failed", err.Error())
		return
	}
	var driftValue *autonomy.Drift
	if inserted {
		driftValue = &drift
	}
	httpx.WriteJSON(w, http.StatusAccepted, struct {
		Inserted bool            `json:"inserted"`
		Drift    *autonomy.Drift `json:"drift"`
	}{Inserted: inserted, Drift: driftValue})
}

func requiredQuery(w http.ResponseWriter, r *http.Request, name string) (string, bool) {
	value := strings.TrimSpace(r.URL.Query().Get(name))
	if value == "" {
		httpx.WriteError(w, http.StatusBadRequest, "missing_query_parameter", name+" is required")
		return "", false
	}
	return value, true
}
