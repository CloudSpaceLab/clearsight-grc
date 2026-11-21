package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

// A listed response is not a decision capability.
func (a *API) listProgramResponseReviews(w http.ResponseWriter, r *http.Request) {
	documentProtection(w)
	service, ok := a.formDistributionService(w)
	if !ok {
		return
	}
	actor, ok := distributionActor(w, r)
	if !ok {
		return
	}
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		var err error
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 100 {
			httpx.WriteError(w, http.StatusBadRequest, "review_query_invalid", "Check the review list and try again.")
			return
		}
	}
	page, err := service.ListPendingProgramReviews(r.Context(), evidence.ProgramReviewQuery{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, Cursor: strings.TrimSpace(r.URL.Query().Get("cursor")), Limit: limit})
	if err != nil {
		if errors.Is(err, evidence.ErrDistributionInvalid) {
			httpx.WriteError(w, http.StatusBadRequest, "review_query_invalid", "Check the review list and try again.")
			return
		}
		httpx.WriteError(w, http.StatusServiceUnavailable, "program_reviews_unavailable", "Program reviews unavailable. Try again.")
		return
	}
	items := a.programReviewPageWithLabels(r.Context(), actor, page)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "next_cursor": page.NextCursor, "coverage": page.Coverage, "scope": map[string]string{"tenant_id": actor.TenantID, "legal_entity_id": actor.LegalEntityID, "principal_id": actor.PrincipalID}})
}

func (a *API) programReviewPageWithLabels(ctx context.Context, actor identity.Actor, page evidence.ProgramReviewPage) []map[string]any {
	ids := make([]string, 0, len(page.Items))
	for _, value := range page.Items {
		ids = append(ids, value.SubmittedBy)
	}
	labels := a.exactAssessmentLabels(ctx, actor, actor.LegalEntityID, ids)
	items := make([]map[string]any, 0, len(page.Items))
	for _, value := range page.Items {
		item := completedResponseSummaryJSON(value)
		if label := labels[value.SubmittedBy]; label != "" {
			item["submitted_by_display_name"] = label
		}
		items = append(items, item)
	}
	return items
}
