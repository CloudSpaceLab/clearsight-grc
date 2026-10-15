package httpapi

import (
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

// Material command authorization is enforced by the existing route lifecycle:
// the current Program owner proposes and an independent authorizer decides.
// The request body cannot supply a trusted tenant, actor, or Program identity.
func (a *API) proposeProgramCompliancePolicy(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var request struct {
		TenantID        string                                        `json:"tenant_id,omitempty"`
		ExpectedVersion int64                                         `json:"expected_version"`
		Definition      continuity.ProgramComplianceCalculationPolicy `json:"definition"`
		Rationale       string                                        `json:"rationale"`
	}
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Check the calculation rules and try again.")
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in to change Program calculation rules.")
		return
	}
	if request.TenantID != "" && request.TenantID != actor.TenantID {
		writeContinuityError(w, continuity.ErrNotFound)
		return
	}
	value, err := service.ProposeProgramCompliancePolicy(r.Context(), continuity.ProposeProgramCompliancePolicyInput{
		TenantID: actor.TenantID, ProgramID: r.PathValue("id"),
		ExpectedVersion: request.ExpectedVersion, Definition: request.Definition,
		ActorID: actor.PrincipalID, Rationale: request.Rationale,
	})
	a.writeProgramResult(w, r, value, err, http.StatusOK, nil)
}

func (a *API) decideProgramCompliancePolicy(approve bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		service, ok := a.continuityService(w)
		if !ok {
			return
		}
		var request struct {
			TenantID        string `json:"tenant_id,omitempty"`
			ExpectedVersion int64  `json:"expected_version"`
			Rationale       string `json:"rationale"`
		}
		if err := httpx.DecodeJSON(w, r, &request); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Check the approval details and try again.")
			return
		}
		actor, err := identity.Require(r.Context())
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in to review calculation rules.")
			return
		}
		if request.TenantID != "" && request.TenantID != actor.TenantID {
			writeContinuityError(w, continuity.ErrNotFound)
			return
		}
		input := continuity.DecideProgramCompliancePolicyInput{
			TenantID: actor.TenantID, ProgramID: r.PathValue("id"),
			ExpectedVersion: request.ExpectedVersion, ActorID: actor.PrincipalID,
			Rationale: request.Rationale,
		}
		var value continuity.ProgramAggregate
		if approve {
			value, err = service.ApproveProgramCompliancePolicy(r.Context(), input)
		} else {
			value, err = service.RejectProgramCompliancePolicy(r.Context(), input)
		}
		a.writeProgramResult(w, r, value, err, http.StatusOK, nil)
	}
}
