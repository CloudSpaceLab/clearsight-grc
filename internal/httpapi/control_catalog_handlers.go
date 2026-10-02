package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/controlcatalog"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/risk"
)

func (a *API) controlCatalogService(w http.ResponseWriter) (*controlcatalog.Service, bool) {
	if a == nil || a.deps.ControlCatalog == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "control_catalog_unavailable", "Control catalog information is unavailable. Try again.")
		return nil, false
	}
	return a.deps.ControlCatalog, true
}

type catalogProgramControlInput struct {
	ExpectedVersion int64  `json:"expected_version"`
	Category        string `json:"category,omitempty"`
}

func (a *API) promoteProgramControlToCatalog(w http.ResponseWriter, r *http.Request) {
	catalog, ok := a.controlCatalogService(w)
	if !ok {
		return
	}
	var input catalogProgramControlInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		writeControlCatalogError(w, controlcatalog.ErrInvalid)
		return
	}
	actor, aggregate, implementation, objective, ok := a.catalogProgramControl(w, r, input.ExpectedVersion)
	if !ok {
		return
	}
	definition, link, err := catalog.Promote(r.Context(), controlcatalog.PromoteInput{
		TenantID:         actor.TenantID,
		LegalEntityID:    aggregate.Program.LegalEntityID,
		Code:             objective.Code,
		Name:             objective.Name,
		Objective:        objective.Outcome,
		Category:         strings.TrimSpace(input.Category),
		ProgramID:        aggregate.Program.ID,
		ImplementationID: implementation.ID,
	})
	if err != nil {
		writeControlCatalogError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"definition": definition, "implementation_link": link})
}

func (a *API) reuseProgramControlDefinition(w http.ResponseWriter, r *http.Request) {
	catalog, ok := a.controlCatalogService(w)
	if !ok {
		return
	}
	var input catalogProgramControlInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		writeControlCatalogError(w, controlcatalog.ErrInvalid)
		return
	}
	actor, aggregate, implementation, _, ok := a.catalogProgramControl(w, r, input.ExpectedVersion)
	if !ok {
		return
	}
	link, err := catalog.LinkImplementation(r.Context(), controlcatalog.LinkImplementationInput{
		TenantID:         actor.TenantID,
		LegalEntityID:    aggregate.Program.LegalEntityID,
		DefinitionID:     r.PathValue("definition_id"),
		ProgramID:        aggregate.Program.ID,
		ImplementationID: implementation.ID,
	})
	if err != nil {
		writeControlCatalogError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, link)
}

func (a *API) catalogProgramControl(w http.ResponseWriter, r *http.Request, expectedVersion int64) (identity.Actor, continuity.ProgramAggregate, continuity.ControlImplementation, continuity.ControlObjective, bool) {
	programs, ok := a.continuityService(w)
	if !ok {
		return identity.Actor{}, continuity.ProgramAggregate{}, continuity.ControlImplementation{}, continuity.ControlObjective{}, false
	}
	actor, err := identity.Require(r.Context())
	if err != nil || strings.TrimSpace(actor.TenantID) == "" || strings.TrimSpace(actor.LegalEntityID) == "" || actor.LegalEntityID == "*" {
		httpx.WriteError(w, http.StatusForbidden, "control_catalog_scope_unavailable", "Choose an eligible legal entity before changing the control catalog.")
		return identity.Actor{}, continuity.ProgramAggregate{}, continuity.ControlImplementation{}, continuity.ControlObjective{}, false
	}
	aggregate, err := programs.GetProgram(r.Context(), actor.TenantID, r.PathValue("id"))
	if err != nil || aggregate.Program.LegalEntityID != actor.LegalEntityID {
		writeContinuityError(w, continuity.ErrNotFound)
		return identity.Actor{}, continuity.ProgramAggregate{}, continuity.ControlImplementation{}, continuity.ControlObjective{}, false
	}
	if expectedVersion <= 0 || aggregate.Program.Version != expectedVersion {
		writeContinuityError(w, continuity.ErrVersionConflict)
		return identity.Actor{}, continuity.ProgramAggregate{}, continuity.ControlImplementation{}, continuity.ControlObjective{}, false
	}
	implementationID := strings.TrimSpace(r.PathValue("implementation_id"))
	var implementation *continuity.ControlImplementation
	for index := range aggregate.ControlImplementations {
		if aggregate.ControlImplementations[index].ID == implementationID {
			implementation = &aggregate.ControlImplementations[index]
			break
		}
	}
	if implementation == nil {
		writeContinuityError(w, continuity.ErrNotFound)
		return identity.Actor{}, continuity.ProgramAggregate{}, continuity.ControlImplementation{}, continuity.ControlObjective{}, false
	}
	var objective *continuity.ControlObjective
	for index := range aggregate.ControlObjectives {
		if aggregate.ControlObjectives[index].ID == implementation.ObjectiveID {
			objective = &aggregate.ControlObjectives[index]
			break
		}
	}
	if objective == nil {
		writeContinuityError(w, continuity.ErrNotFound)
		return identity.Actor{}, continuity.ProgramAggregate{}, continuity.ControlImplementation{}, continuity.ControlObjective{}, false
	}
	if objective.Status != continuity.ObjectiveActive || implementation.Status == continuity.ImplementationRetired {
		writeControlCatalogError(w, controlcatalog.ErrInvalid)
		return identity.Actor{}, continuity.ProgramAggregate{}, continuity.ControlImplementation{}, continuity.ControlObjective{}, false
	}
	return actor, aggregate, *implementation, *objective, true
}

func (a *API) linkRiskControl(w http.ResponseWriter, r *http.Request) {
	risks, ok := a.riskService(w)
	if !ok {
		return
	}
	catalog, ok := a.controlCatalogService(w)
	if !ok {
		return
	}
	actor, _, ok := a.riskActorScope(w, r)
	if !ok {
		return
	}
	var input risk.LinkControlInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		writeRiskError(w, risk.ErrInvalid)
		return
	}
	catalogLinkID := strings.TrimSpace(input.CatalogLinkID)
	if _, err := catalog.GetImplementationLink(r.Context(), actor.TenantID, actor.LegalEntityID, catalogLinkID); err != nil {
		if errors.Is(err, controlcatalog.ErrNotFound) {
			writeRiskError(w, risk.ErrNotFound)
			return
		}
		writeControlCatalogError(w, err)
		return
	}
	input.TenantID = actor.TenantID
	input.LegalEntityID = actor.LegalEntityID
	input.RiskID = r.PathValue("id")
	input.CatalogLinkID = catalogLinkID
	input.ActorID = actor.PrincipalID
	current, linked, err := risks.LinkControl(r.Context(), input)
	if err != nil {
		if errors.Is(err, risk.ErrDuplicate) {
			httpx.WriteError(w, http.StatusConflict, "risk_control_duplicate", "This control is already linked to the risk.")
			return
		}
		writeRiskError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"risk": current, "control": linked})
}

func writeControlCatalogError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, controlcatalog.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "control_catalog_not_found", "This control reference is not available in the current scope.")
	case errors.Is(err, controlcatalog.ErrDuplicate):
		httpx.WriteError(w, http.StatusConflict, "control_catalog_duplicate", "This Program control is already in the reusable catalog.")
	case errors.Is(err, controlcatalog.ErrInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "control_catalog_invalid", "The control catalog request is not valid. Review the current Program and try again.")
	default:
		httpx.WriteError(w, http.StatusServiceUnavailable, "control_catalog_unavailable", "Control catalog information is unavailable. No change was made; try again.")
	}
}
