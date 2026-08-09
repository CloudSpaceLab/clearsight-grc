package httpapi

import (
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

type acceptProgramReviewRequest struct {
	ExpectedProgramVersion    int64 `json:"expected_program_version"`
	ExpectedProjectionVersion int64 `json:"expected_projection_version"`
}

func (a *API) getProgramReviewDigest(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	value, err := service.ProgramReviewDigest(r.Context(), actor.TenantID, r.PathValue("id"), actor.PrincipalID)
	writeContinuityResult(w, value, err, http.StatusOK)
}

func (a *API) acceptProgramReview(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	var request acceptProgramReviewRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := service.AcceptProgramReview(r.Context(), continuity.AcceptProgramReviewInput{
		TenantID:                  actor.TenantID,
		ProgramID:                 r.PathValue("id"),
		PrincipalID:               actor.PrincipalID,
		ExpectedProgramVersion:    request.ExpectedProgramVersion,
		ExpectedProjectionVersion: request.ExpectedProjectionVersion,
	})
	writeContinuityResult(w, value, err, http.StatusOK)
}
