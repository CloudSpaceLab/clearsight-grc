package httpapi

import (
	"context"
	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/runtimecontext"
	"net/http"
	"regexp"
	"strings"
)

type starterWorkflowRepository interface {
	StarterRequirements(context.Context, string, string, string) ([]continuity.StarterRequirement, error)
	UpdateStarterRequirement(context.Context, string, continuity.StarterRequirement, string) (int64, error)
}

func (a *API) listFrameworkStarterRequirements(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, 401, "sign_in_required", "Sign in to review starter requirements.")
		return
	}
	repo, ok := a.deps.FrameworkCatalog.(starterWorkflowRepository)
	if !ok {
		httpx.WriteError(w, 503, "starter_unavailable", "Starter requirements are unavailable.")
		return
	}
	if actor.LegalEntityID == "" || actor.LegalEntityID == "*" {
		httpx.WriteError(w, 403, "entity_required", "Select a legal entity.")
		return
	}
	items, err := repo.StarterRequirements(r.Context(), actor.TenantID, actor.LegalEntityID, r.PathValue("code"))
	if err != nil {
		writeContinuityError(w, err)
		return
	}
	canManage := false
	reason := "Template maintenance authority is unavailable."
	if a.deps.CommandGuard != nil {
		decision, e := a.deps.CommandGuard.Authorize(r.Context(), commandauth.Request{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ObjectType: "LEGAL_ENTITY", ObjectID: actor.LegalEntityID, Responsibility: authority.ResponsibilityOwner, DecisionType: "framework.starter.requirement.update", Materiality: 3})
		canManage = e == nil && decision.Allowed
		if canManage {
			reason = "You can maintain starter requirements for this legal entity."
		} else {
			reason = "Template changes require the current legal-entity owner."
		}
	}
	httpx.WriteJSON(w, 200, map[string]any{"items": items, "can_manage": canManage, "management_reason": reason})
}
func (a *API) updateFrameworkStarterRequirement(w http.ResponseWriter, r *http.Request) {
	repo, ok := a.deps.FrameworkCatalog.(starterWorkflowRepository)
	if !ok {
		httpx.WriteError(w, 503, "starter_unavailable", "Starter requirements are unavailable.")
		return
	}
	var input struct {
		TenantID      string                        `json:"tenant_id,omitempty"`
		LegalEntityID string                        `json:"legal_entity_id,omitempty"`
		Requirement   continuity.StarterRequirement `json:"requirement"`
		Rationale     string                        `json:"rationale"`
	}
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, 400, "invalid_requirement", "Check the requirement and questionnaire group.")
		return
	}
	v := input.Requirement
	if !regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,63}$`).MatchString(v.Code) || v.Code != r.PathValue("requirement_code") || strings.TrimSpace(v.Title) == "" || len(v.Title) > 200 || strings.TrimSpace(v.Statement) == "" || len(v.Statement) > 2000 || strings.TrimSpace(input.Rationale) == "" || len(input.Rationale) > 2000 || v.Version < 0 || v.FormVersion < 1 || len(v.FormID) > 40 || len(v.GroupID) > 80 {
		writeContinuityError(w, continuity.ErrInvalidState)
		return
	}
	version, err := repo.UpdateStarterRequirement(r.Context(), r.PathValue("code"), v, input.Rationale)
	if err != nil {
		writeContinuityError(w, err)
		return
	}
	httpx.WriteJSON(w, 200, map[string]any{"version": version})
}

func (a *API) listStarterRespondentUnits(w http.ResponseWriter, r *http.Request) {
	actor, aggregate, err := a.bindMonitoringProgram(r, r.PathValue("id"))
	if err != nil {
		writeMonitoringScopeError(w, err)
		return
	}
	if actor.PrincipalID != aggregate.Program.OwnerPrincipalID && actor.PrincipalID != aggregate.Program.AuthorityPrincipalID {
		httpx.WriteError(w, 403, "respondent_selection_denied", "The Program Owner or approval authority can review respondent units.")
		return
	}
	if _, err = a.programForActor(r.Context(), aggregate, nil); err != nil {
		writeContinuityError(w, continuity.ErrNotFound)
		return
	}
	repo, ok := a.deps.FrameworkCatalog.(interface {
		StarterRespondentUnits(context.Context, string, string, string) ([]runtimecontext.ScopeNode, error)
	})
	if !ok {
		httpx.WriteError(w, 503, "respondent_units_unavailable", "Respondent units could not be loaded.")
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(query) > 120 {
		httpx.WriteError(w, 400, "invalid_search", "Use at most 120 characters to search respondent units.")
		return
	}
	items, err := repo.StarterRespondentUnits(r.Context(), actor.TenantID, aggregate.Program.LegalEntityID, query)
	if err != nil {
		writeContinuityError(w, err)
		return
	}
	httpx.WriteJSON(w, 200, map[string]any{"items": items, "limit": 50})
}
