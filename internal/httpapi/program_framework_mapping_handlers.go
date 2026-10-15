package httpapi

import (
	"errors"
	"net/http"

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
			MappingID: r.PathValue("mapping_id"), ExpectedVersion: body.ExpectedVersion,
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
