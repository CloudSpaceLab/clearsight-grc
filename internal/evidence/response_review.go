package evidence

import (
	"context"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

type RecordResponseReviewInput struct {
	ExpectedVersion int64  `json:"expected_version"`
	Decision        string `json:"decision"`
	Comment         string `json:"comment"`
}

type ResponseReview struct {
	ID                     string    `json:"id"`
	ResponseID             string    `json:"response_id"`
	ResponseRevision       int64     `json:"response_revision"`
	Version                int64     `json:"version"`
	Decision               string    `json:"decision"`
	Comment                string    `json:"comment"`
	ReviewerID             string    `json:"reviewer_id"`
	AuthorityRoute         string    `json:"authority_route"`
	AuthoritySourceType    string    `json:"authority_source_type,omitempty"`
	AuthoritySourceID      string    `json:"authority_source_id,omitempty"`
	AuthorityRuleID        string    `json:"authority_rule_id,omitempty"`
	AuthorityPolicyVersion string    `json:"authority_policy_version,omitempty"`
	ReviewedAt             time.Time `json:"reviewed_at"`
	SupersedesID           string    `json:"supersedes_id,omitempty"`
}

type responseReviewStore interface {
	WriteResponseReview(context.Context, string, string, string, string, assessmentReadAuthorizer, func(context.Context, assessmentMaterial, *ResponseReview) (ResponseReview, error)) (ResponseReview, error)
	ReadResponseReviews(context.Context, string, string, string, string, assessmentReadAuthorizer, int64, int) (assessmentMaterial, []ResponseReview, int64, error)
}

// ConfigureDemoResponseReviewDecisions is for explicitly configured,
// non-production in-memory journeys only. It does not bypass reviewer routes.
func (s *DistributionService) ConfigureDemoResponseReviewDecisions(enabled bool) {
	if s != nil {
		_, memoryOnly := s.store.(*MemoryDistributionStore)
		s.demoResponseReviewDecisions = enabled && memoryOnly
	}
}

type ResponseReviewPage struct {
	Version           int64            `json:"version"`
	MayReview         bool             `json:"may_review"`
	Items             []ResponseReview `json:"items"`
	NextBeforeVersion int64            `json:"next_before_version,omitempty"`
}

// GetResponseReviews returns a bounded history for one authorized response.
func (s *DistributionService) GetResponseReviews(ctx context.Context, tenant, entity, principal, response string, before int64, limit int) (ResponseReviewPage, error) {
	actor, ok := identity.FromContext(ctx)
	if s == nil || !ok || actor.Valid(s.currentTime()) != nil || actor.TenantID != tenant || actor.PrincipalID != principal || (actor.LegalEntityID != entity && actor.LegalEntityID != "*") {
		return ResponseReviewPage{}, ErrAssessmentForbidden
	}
	if tenant == "" || entity == "" || principal == "" || response == "" || before < 0 || limit < 1 || limit > 100 {
		return ResponseReviewPage{}, ErrAssessmentInvalid
	}
	store, ok := s.store.(responseReviewStore)
	if !ok {
		return ResponseReviewPage{}, ErrAssessmentInvalid
	}
	m, items, version, err := store.ReadResponseReviews(ctx, tenant, entity, principal, response, s.assessmentReadAuthority(ctx, tenant, entity, principal), before, limit+1)
	if err != nil {
		return ResponseReviewPage{}, err
	}
	page := ResponseReviewPage{Version: version, Items: items}
	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		page.NextBeforeVersion = page.Items[len(page.Items)-1].Version
	}
	if s.demoResponseReviewDecisions && m.Revision.Current && !assessmentRespondentConflict(m, principal) && s.responseReviewAuthorizer != nil {
		route, routeErr := s.responseReviewAuthorizer(ctx, actor, m.Summary)
		page.MayReview = routeErr == nil && strings.TrimSpace(route) != ""
	}
	return page, nil
}

// RecordResponseReview records a decision on one immutable response revision;
// it neither changes the submitted answers nor produces a compliance score.
func (s *DistributionService) RecordResponseReview(ctx context.Context, tenant, entity, response string, input RecordResponseReviewInput) (ResponseReview, error) {
	actor, ok := identity.FromContext(ctx)
	if s == nil || !ok || actor.Valid(s.currentTime()) != nil || actor.TenantID != tenant || (actor.LegalEntityID != entity && actor.LegalEntityID != "*") || s.responseReviewAuthorizer == nil {
		return ResponseReview{}, ErrAssessmentForbidden
	}
	if !s.demoResponseReviewDecisions {
		return ResponseReview{}, ErrProgramReviewUnavailable
	}
	input.Comment = strings.TrimSpace(input.Comment)
	if input.ExpectedVersion < 0 || (input.Decision != "REVIEWED" && input.Decision != "CHANGES_REQUESTED") || len(input.Comment) > 4000 || (input.Decision == "CHANGES_REQUESTED" && input.Comment == "") {
		return ResponseReview{}, ErrAssessmentInvalid
	}
	store, ok := s.store.(responseReviewStore)
	if !ok {
		return ResponseReview{}, ErrAssessmentInvalid
	}
	return store.WriteResponseReview(ctx, tenant, entity, actor.PrincipalID, response, s.assessmentReadAuthority(ctx, tenant, entity, actor.PrincipalID), func(ctx context.Context, m assessmentMaterial, previous *ResponseReview) (ResponseReview, error) {
		if assessmentRespondentConflict(m, actor.PrincipalID) {
			return ResponseReview{}, ErrAssessmentForbidden
		}
		version := int64(0)
		if previous != nil {
			version = previous.Version
		}
		if !m.Revision.Current || version != input.ExpectedVersion {
			return ResponseReview{}, ErrAssessmentConflict
		}
		route, err := s.responseReviewAuthorizer(ctx, actor, m.Summary)
		if err != nil || strings.TrimSpace(route) == "" {
			return ResponseReview{}, ErrAssessmentForbidden
		}
		// Authority resolution can wait on the database. Recheck the verified
		// session after it returns, and record the same decision time.
		decisionAt := s.currentTime()
		if actor.Valid(decisionAt) != nil {
			return ResponseReview{}, ErrAssessmentForbidden
		}
		identifier, err := id.NewUUIDv7()
		if err != nil {
			return ResponseReview{}, err
		}
		review := ResponseReview{ID: identifier, ResponseID: response, ResponseRevision: m.Revision.Revision, Version: version + 1, Decision: input.Decision, Comment: input.Comment, ReviewerID: actor.PrincipalID, AuthorityRoute: route, ReviewedAt: decisionAt}
		if previous != nil {
			review.SupersedesID = previous.ID
		}
		return review, nil
	})
}

func responseReviewEvent(m assessmentMaterial, review ResponseReview) distributionEvent {
	payload := map[string]any{"response_revision_id": review.ResponseID, "review_id": review.ID, "version": review.Version, "decision": review.Decision}
	if review.AuthoritySourceID != "" {
		payload["authority_source_type"] = review.AuthoritySourceType
		payload["authority_source_id"] = review.AuthoritySourceID
		payload["authority_rule_id"] = review.AuthorityRuleID
		payload["authority_policy_version"] = review.AuthorityPolicyVersion
	}
	return distributionEvent{DistributionID: m.Revision.DistributionID, Version: review.Version, EventType: "FORM_RESPONSE_REVIEWED", ActorID: review.ReviewerID, OccurredAt: review.ReviewedAt, Payload: payload}
}
