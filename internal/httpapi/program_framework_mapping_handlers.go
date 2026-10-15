package httpapi

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

// Clause mapping commands share the Program material-command authority,
// version and event/outbox path. Verified actor identity owns tenant scope.
func (a *API) proposeProgramFrameworkMapping(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var body struct {
		TenantID        string                          `json:"tenant_id,omitempty"`
		ExpectedVersion int64                           `json:"expected_version"`
		ClauseID        string                          `json:"clause_id"`
		RequirementID   string                          `json:"requirement_id"`
		MappingKind     continuity.FrameworkMappingKind `json:"mapping_kind"`
		Rationale       string                          `json:"rationale"`
	}
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Check the clause mapping details.")
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in to link a requirement.")
		return
	}
	if body.TenantID != "" && body.TenantID != actor.TenantID {
		writeContinuityError(w, continuity.ErrNotFound)
		return
	}
	mapping, updated, err := service.ProposeFrameworkMapping(r.Context(), continuity.ProposeFrameworkMappingInput{
		TenantID: actor.TenantID, ProgramID: r.PathValue("id"), AdoptionID: r.PathValue("adoption_id"),
		ClauseID: body.ClauseID, RequirementID: body.RequirementID,
		MappingKind: body.MappingKind, ExpectedVersion: body.ExpectedVersion,
		ActorID: actor.PrincipalID, Rationale: body.Rationale,
	})
	a.writeFrameworkMappingResult(w, r, mapping, updated, err)
}

func (a *API) decideProgramFrameworkMapping(approve bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		service, ok := a.continuityService(w)
		if !ok {
			return
		}
		var body struct {
			TenantID        string `json:"tenant_id,omitempty"`
			ExpectedVersion int64  `json:"expected_version"`
			Rationale       string `json:"rationale"`
		}
		if err := httpx.DecodeJSON(w, r, &body); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Check the mapping review details.")
			return
		}
		actor, err := identity.Require(r.Context())
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in to review the mapping.")
			return
		}
		if body.TenantID != "" && body.TenantID != actor.TenantID {
			writeContinuityError(w, continuity.ErrNotFound)
			return
		}
		input := continuity.DecideFrameworkMappingInput{
			TenantID: actor.TenantID, ProgramID: r.PathValue("id"),
			MappingID: r.PathValue("mapping_id"), AdoptionID: r.PathValue("adoption_id"), ExpectedVersion: body.ExpectedVersion,
			ActorID: actor.PrincipalID, Rationale: body.Rationale,
		}
		var mapping continuity.FrameworkClauseMapping
		var program continuity.ProgramAggregate
		if approve {
			mapping, program, err = service.ApproveFrameworkMapping(r.Context(), input)
		} else {
			mapping, program, err = service.RejectFrameworkMapping(r.Context(), input)
		}
		a.writeFrameworkMappingResult(w, r, mapping, program, err)
	}
}

func (a *API) writeFrameworkMappingResult(
	w http.ResponseWriter, r *http.Request, mapping continuity.FrameworkClauseMapping,
	program continuity.ProgramAggregate, err error,
) {
	if errors.Is(err, continuity.ErrFrameworkEditionUnavailable) {
		httpx.WriteError(w, http.StatusServiceUnavailable, "framework_mapping_unavailable", "Standard mapping service is unavailable.")
		return
	}
	if err != nil {
		writeContinuityError(w, err)
		return
	}
	visible, err := a.programForActor(r.Context(), program, nil)
	if err != nil {
		writeContinuityError(w, continuity.ErrNotFound)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"mapping": mapping,
		"program": visible,
	})
}

var frameworkMappingCursor = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func (a *API) listProgramFrameworkMappings(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in to review standard mappings.")
		return
	}
	if actor.LegalEntityID == "" || actor.LegalEntityID == "*" {
		httpx.WriteError(w, http.StatusForbidden, "legal_entity_required", "Select a legal entity.")
		return
	}
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 100 {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_limit", "Choose a page size from 1 to 100.")
			return
		}
	}
	afterID := r.URL.Query().Get("after_id")
	if afterID != "" && !frameworkMappingCursor.MatchString(afterID) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_cursor", "Check the mapping cursor.")
		return
	}
	program, err := service.GetProgram(r.Context(), actor.TenantID, r.PathValue("id"))
	if err != nil {
		writeContinuityError(w, err)
		return
	}
	if program.Program.LegalEntityID != actor.LegalEntityID {
		writeContinuityError(w, continuity.ErrNotFound)
		return
	}
	visible, err := a.programForActor(r.Context(), program, nil)
	if err != nil {
		writeContinuityError(w, continuity.ErrNotFound)
		return
	}
	adoptionID := r.PathValue("adoption_id")
	active := false
	for _, adoption := range visible.FrameworkAdoptions {
		if adoption.ID == adoptionID && adoption.Status == continuity.FrameworkAdoptionActive {
			active = true
			break
		}
	}
	if !active {
		writeContinuityError(w, continuity.ErrNotFound)
		return
	}
	items, err := service.ListFrameworkMappings(r.Context(), actor.TenantID,
		visible.Program.ID, adoptionID, afterID, limit)
	if errors.Is(err, continuity.ErrFrameworkEditionUnavailable) {
		httpx.WriteError(w, http.StatusServiceUnavailable, "framework_mapping_unavailable", "Mapping history is unavailable.")
		return
	}
	if err != nil {
		writeContinuityError(w, err)
		return
	}
	next := ""
	if len(items) == limit {
		next = items[len(items)-1].ID
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"next_after_id": next,
		"program_version": visible.Program.Version,
		"adoption_id": adoptionID,
	})
}
