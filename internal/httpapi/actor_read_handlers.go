package httpapi

import (
	"net/http"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/runtimecontext"
	"github.com/CloudSpaceLab/clearsight-grc/internal/today"
)

func (a *API) actorContext(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	roleCodes := identity.NormalizeRoleCodes(actor.RoleCodes)
	if a.deps.RuntimeContext == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "directory_context_unavailable", "Your organization and role details could not be loaded. Refresh the workspace; no task data was changed.")
		return
	}
	display, err := a.deps.RuntimeContext.Resolve(r.Context(), runtimecontext.Scope{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PrincipalID: actor.PrincipalID,
	})
	if err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "directory_context_unavailable", "Your organization and role details could not be loaded. Refresh the workspace; no task data was changed.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"tenant":       map[string]string{"id": actor.TenantID, "name": display.TenantName},
		"legal_entity": map[string]string{"id": actor.LegalEntityID, "name": display.LegalEntityName},
		"actor": map[string]any{
			"id": actor.PrincipalID, "name": display.PrincipalName, "kind": actor.Kind, "role_codes": roleCodes,
			"department_grants": actor.DepartmentGrants,
			"assurance_level":   actor.AssuranceLevel, "authentication": actor.AuthenticationMethod, "session_id": actor.SessionID,
		},
		"mode":      a.deps.Mode,
		"demo_mode": a.deps.DemoMode,
		"capabilities": map[string]bool{
			"document_import":           a.deps.DocumentImports != nil,
			"reference_journeys":        a.deps.DemoMode && a.deps.BankVerticals != nil,
			"config_read":               identity.HasPermission(actor, identity.PermissionConfigRead),
			"config_write":              identity.HasPermission(actor, identity.PermissionConfigWrite),
			"identity_read":             identity.HasPermission(actor, identity.PermissionIdentityRead),
			"identity_configure":        identity.HasPermission(actor, identity.PermissionIdentityConfigure),
			"platform_operations_read":  identity.HasPermission(actor, identity.PermissionPlatformOperationsRead),
			"platform_operations_write": identity.HasPermission(actor, identity.PermissionPlatformOperationsWrite),
			"audit_export":              identity.HasPermission(actor, identity.PermissionAuditExport),
			"oversight_read":            identity.HasPermission(actor, identity.PermissionOversightRead),
		},
	})
}

func (a *API) actorToday(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	if a.deps.Today == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "today_unavailable", "Today's work is unavailable.")
		return
	}
	items, err := a.deps.Today.ListFor(r.Context(), actor)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "today_failed", "Today's work could not be loaded.")
		return
	}
	if items == nil {
		items = []today.AttentionItem{}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "generated_at": time.Now().UTC()})
}
