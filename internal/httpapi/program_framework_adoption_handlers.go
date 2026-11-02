package httpapi

import (
	"errors"
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

// Program command authorization (Owner/Authorizer) is enforced by the material
// route registry before these handlers run. Tenant and actor always come from
// the verified request identity, never the JSON body.
func (a *API) proposeProgramFrameworkAdoption(w http.ResponseWriter, r *http.Request) {
	programs, ok := a.continuityService(w)
	if !ok {
		return
	}
	var request struct {
		TenantID        string `json:"tenant_id,omitempty"`
		ExpectedVersion int64  `json:"expected_version"`
		EditionID       string `json:"edition_id"`
		Rationale       string `json:"rationale"`
	}
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Check the standard and Program revision.")
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in to add a standard.")
		return
	}
	if request.TenantID != "" && request.TenantID != actor.TenantID {
		writeContinuityError(w, continuity.ErrNotFound)
		return
	}
	value, err := programs.ProposeFrameworkAdoption(r.Context(), continuity.ProposeFrameworkAdoptionInput{
		TenantID: actor.TenantID, ProgramID: r.PathValue("id"),
		EditionID: request.EditionID, ExpectedVersion: request.ExpectedVersion,
		ActorID: actor.PrincipalID, Rationale: request.Rationale,
	})
	a.writeFrameworkAdoptionResult(w, r, value, err)
}

func (a *API) decideProgramFrameworkAdoption(approve bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		programs, ok := a.continuityService(w)
		if !ok {
			return
		}
		var request struct {
			TenantID        string `json:"tenant_id,omitempty"`
			ExpectedVersion int64  `json:"expected_version"`
			Rationale       string `json:"rationale"`
		}
		if err := httpx.DecodeJSON(w, r, &request); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Check the approval details and Program revision.")
			return
		}
		actor, err := identity.Require(r.Context())
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in to review this standard.")
			return
		}
		if request.TenantID != "" && request.TenantID != actor.TenantID {
			writeContinuityError(w, continuity.ErrNotFound)
			return
		}
		input := continuity.DecideFrameworkAdoptionInput{
			TenantID: actor.TenantID, ProgramID: r.PathValue("id"),
			AdoptionID: r.PathValue("adoption_id"), ExpectedVersion: request.ExpectedVersion,
			ActorID: actor.PrincipalID, Rationale: request.Rationale,
		}
		var value continuity.ProgramAggregate
		if approve {
			value, err = programs.ApproveFrameworkAdoption(r.Context(), input)
		} else {
			value, err = programs.RejectFrameworkAdoption(r.Context(), input)
		}
		a.writeFrameworkAdoptionResult(w, r, value, err)
	}
}

func (a *API) writeFrameworkAdoptionResult(w http.ResponseWriter, r *http.Request, value continuity.ProgramAggregate, err error) {
	if errors.Is(err, continuity.ErrFrameworkEditionUnavailable) {
		httpx.WriteError(w, http.StatusServiceUnavailable, "framework_catalog_unavailable", "Standards are temporarily unavailable.")
		return
	}
	a.writeProgramResult(w, r, value, err, http.StatusOK, nil)
}
