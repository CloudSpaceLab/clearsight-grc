package httpapi

import (
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

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
	_, ok := a.continuityService(w)
	if !ok {
		return
	}
	_, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	httpx.WriteError(w, http.StatusServiceUnavailable, "program_review_unavailable", "Program review unavailable. Try again later.")
}
