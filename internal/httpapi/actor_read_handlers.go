package httpapi

import (
	"net/http"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) actorContext(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	roleCodes := identity.NormalizeRoleCodes(actor.RoleCodes)
	tenantName, legalEntityName, actorName := actor.TenantID, actor.LegalEntityID, actor.PrincipalID
	if a.deps.DemoMode && actor.TenantID == identity.DurableDemoTenantID {
		tenantName = "Clear Bank"
		legalEntityName = "Clear Bank Nigeria"
		actorName = demoActorName(actor.PrincipalID)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		// Directory-backed display names are enterprise work. Until they are available,
		// show the verified identifiers rather than ambiguous placeholder organization names.
		"tenant":       map[string]string{"id": actor.TenantID, "name": tenantName},
		"legal_entity": map[string]string{"id": actor.LegalEntityID, "name": legalEntityName},
		"actor": map[string]any{
			"id": actor.PrincipalID, "name": actorName, "kind": actor.Kind, "role_codes": roleCodes,
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
		},
	})
}

func demoActorName(principalID string) string {
	names := map[string]string{
		identity.DurableDemoPrincipalCRO: "Chief Risk Officer", identity.DurableDemoPrincipalCCO: "Chief Compliance Officer",
		identity.DurableDemoPrincipalCISO: "Chief Information Security Officer", identity.DurableDemoPrincipalGRCAdmin: "GRC Administrator",
		identity.DurableDemoPrincipalSystemAdmin: "System Administrator", identity.DurableDemoPrincipalAuditor: "Internal Auditor",
		identity.DurableDemoPrincipalProgramOwner: "Program Owner", identity.DurableDemoPrincipalEvidenceRespondent: "Evidence Respondent",
	}
	if value := names[principalID]; value != "" {
		return value
	}
	return principalID
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
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "generated_at": time.Now().UTC()})
}
