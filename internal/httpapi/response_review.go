package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

type responseReviewRead struct {
	evidence.ResponseReview
	ReviewerDisplayName string `json:"reviewer_display_name,omitempty"`
}
type responseReviewPageRead struct {
	evidence.ResponseReviewPage
	Items []responseReviewRead `json:"items"`
}

func (a *API) responseReviewsWithLabels(ctx context.Context, actor identity.Actor, entity string, page evidence.ResponseReviewPage) responseReviewPageRead {
	ids := make([]string, 0, len(page.Items))
	for _, item := range page.Items {
		ids = append(ids, item.ReviewerID)
	}
	labels := a.exactAssessmentLabels(ctx, actor, entity, ids)
	result := responseReviewPageRead{ResponseReviewPage: page, Items: make([]responseReviewRead, 0, len(page.Items))}
	for _, item := range page.Items {
		result.Items = append(result.Items, responseReviewRead{ResponseReview: item, ReviewerDisplayName: labels[item.ReviewerID]})
	}
	return result
}

func (a *API) getResponseReviews(w http.ResponseWriter, r *http.Request) {
	documentProtection(w)
	service, ok := a.formDistributionService(w)
	if !ok {
		return
	}
	actor, ok := distributionActor(w, r)
	if !ok {
		return
	}
	requested := ""
	if actor.LegalEntityID == "*" {
		requested = r.URL.Query().Get("legal_entity_id")
	}
	entity, ok := distributionLegalEntity(w, r, actor, requested)
	if !ok {
		return
	}
	var before int64
	if value := r.URL.Query().Get("before_version"); value != "" {
		var err error
		before, err = strconv.ParseInt(value, 10, 64)
		if err != nil || before < 1 {
			writeResponseReviewError(w, evidence.ErrAssessmentInvalid)
			return
		}
	}
	result, err := service.GetResponseReviews(r.Context(), actor.TenantID, entity, actor.PrincipalID, r.PathValue("revision_id"), before, 20)
	if err != nil {
		writeResponseReviewError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, a.responseReviewsWithLabels(r.Context(), actor, entity, result))
}

func (a *API) recordResponseReview(w http.ResponseWriter, r *http.Request) {
	documentProtection(w)
	service, ok := a.formDistributionService(w)
	if !ok {
		return
	}
	actor, ok := distributionActor(w, r)
	if !ok {
		return
	}
	var body struct {
		ExpectedVersion *int64 `json:"expected_version"`
		Decision        string `json:"decision"`
		Comment         string `json:"comment"`
		LegalEntityID   string `json:"legal_entity_id,omitempty"`
		ReviewerID      string `json:"reviewer_id,omitempty"`
		ActorID         string `json:"actor_id,omitempty"`
	}
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "response_review_request_invalid", "Check the review and try again.")
		return
	}
	if body.ExpectedVersion == nil {
		writeResponseReviewError(w, evidence.ErrAssessmentInvalid)
		return
	}
	entity, ok := distributionLegalEntity(w, r, actor, body.LegalEntityID)
	if !ok {
		return
	}
	result, err := service.RecordResponseReview(r.Context(), actor.TenantID, entity, r.PathValue("revision_id"), evidence.RecordResponseReviewInput{ExpectedVersion: *body.ExpectedVersion, Decision: body.Decision, Comment: body.Comment})
	if err != nil {
		writeResponseReviewError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

func writeResponseReviewError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, evidence.ErrAssessmentConflict):
		httpx.WriteError(w, http.StatusConflict, "response_review_changed", "The response or review has changed. Reload before saving.")
	case errors.Is(err, evidence.ErrAssessmentForbidden):
		httpx.WriteError(w, http.StatusForbidden, "response_review_not_permitted", "You cannot review this response under the current review route.")
	case errors.Is(err, evidence.ErrAssessmentInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "response_review_invalid", "Choose a review decision and explain any changes requested.")
	case errors.Is(err, evidence.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "response_not_found", "The submitted response was not found in your permitted work.")
	default:
		httpx.WriteError(w, http.StatusServiceUnavailable, "response_review_unavailable", "Review unavailable. Try again.")
	}
}
