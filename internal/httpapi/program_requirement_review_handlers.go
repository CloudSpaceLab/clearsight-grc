package httpapi

import (
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"net/http"
	"regexp"
)

var requirementReviewUUID = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// The registry performs command authorization; identity is never taken from
// caller-supplied actor fields.
func (a *API) proposeRequirementReview(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var request struct {
		TenantID        string `json:"tenant_id,omitempty"`
		ExpectedVersion int64  `json:"expected_version"`
		Rationale       string `json:"rationale"`
	}
	if httpx.DecodeJSON(w, r, &request) != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Check the requirement and reason.")
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in to propose the requirement.")
		return
	}
	if request.TenantID != "" && request.TenantID != actor.TenantID {
		writeContinuityError(w, continuity.ErrNotFound)
		return
	}
	review, program, err := service.ProposeRequirementReview(r.Context(), continuity.ProposeRequirementReviewInput{TenantID: actor.TenantID, ProgramID: r.PathValue("id"), RequirementID: r.PathValue("requirement_id"), ExpectedVersion: request.ExpectedVersion, ActorID: actor.PrincipalID, Rationale: request.Rationale})
	if err != nil {
		writeContinuityError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"review": review, "program": program.Program})
}
func (a *API) decideRequirementReview(approve bool) http.HandlerFunc {
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
		if httpx.DecodeJSON(w, r, &request) != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Check the decision and reason.")
			return
		}
		actor, err := identity.Require(r.Context())
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in to review the requirement.")
			return
		}
		if request.TenantID != "" && request.TenantID != actor.TenantID {
			writeContinuityError(w, continuity.ErrNotFound)
			return
		}
		review, program, err := service.DecideRequirementReview(r.Context(), continuity.DecideRequirementReviewInput{TenantID: actor.TenantID, ProgramID: r.PathValue("id"), ReviewID: r.PathValue("review_id"), ExpectedVersion: request.ExpectedVersion, ActorID: actor.PrincipalID, Rationale: request.Rationale}, approve)
		if err != nil {
			writeContinuityError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"review": review, "program": program.Program})
	}
}

func (a *API) listRequirementReviews(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in to view requirement reviews.")
		return
	}
	if actor.LegalEntityID == "" || actor.LegalEntityID == "*" {
		httpx.WriteError(w, http.StatusForbidden, "legal_entity_required", "Select a legal entity.")
		return
	}
	tenant, ok := requiredQuery(w, r, "tenant_id")
	if !ok {
		return
	}
	if tenant != actor.TenantID {
		writeContinuityError(w, continuity.ErrNotFound)
		return
	}
	after := r.URL.Query().Get("after_id")
	if after != "" && !requirementReviewUUID.MatchString(after) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_cursor", "Check the review cursor.")
		return
	}
	program, err := service.GetProgram(r.Context(), tenant, r.PathValue("id"))
	if err != nil {
		writeContinuityError(w, err)
		return
	}
	if program.Program.LegalEntityID != actor.LegalEntityID {
		writeContinuityError(w, continuity.ErrNotFound)
		return
	}
	if _, err = a.programForActor(r.Context(), program, nil); err != nil {
		writeContinuityError(w, continuity.ErrNotFound)
		return
	}
	values, err := service.ListRequirementReviews(r.Context(), tenant, r.PathValue("id"), after, 100)
	if err != nil {
		writeContinuityError(w, err)
		return
	}
	next := ""
	if len(values) > 100 {
		next = values[99].ID
		values = values[:100]
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"program_id": program.Program.ID, "program_version": program.Program.Version, "items": values, "next_after_id": next})
}
