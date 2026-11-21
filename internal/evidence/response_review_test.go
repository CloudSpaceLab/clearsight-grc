package evidence

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestResponseReviewDecisionsDefaultUnavailableWhileHistoryRemainsReadable(t *testing.T) {
	s, store, ctx := assessmentFixture(t)
	s.WithResponseReviewAuthorizer(func(context.Context, identity.Actor, CompletedResponseSummary) (string, error) {
		return "policy:rule", nil
	})
	page, err := s.GetResponseReviews(ctx, "tenant", "entity", "reviewer", "response", 0, 10)
	if err != nil || page.MayReview {
		t.Fatalf("history or unavailable decision: %+v %v", page, err)
	}
	if _, err = s.RecordResponseReview(ctx, "tenant", "entity", "response", RecordResponseReviewInput{Decision: "REVIEWED"}); !errors.Is(err, ErrProgramReviewUnavailable) {
		t.Fatalf("decision should be unavailable: %v", err)
	}
	if len(store.events) != 0 || len(store.outbox) != 0 || len(store.responseReviews["response"]) != 0 {
		t.Fatal("unavailable review wrote records")
	}
	s.ConfigureDemoResponseReviewDecisions(true)
	if _, err = s.RecordResponseReview(ctx, "tenant", "entity", "response", RecordResponseReviewInput{Decision: "REVIEWED"}); err != nil {
		t.Fatal(err)
	}
	s.ConfigureDemoResponseReviewDecisions(false)
	page, err = s.GetResponseReviews(ctx, "tenant", "entity", "reviewer", "response", 0, 10)
	if err != nil || page.MayReview || len(page.Items) != 1 {
		t.Fatalf("existing history must remain readable: %+v %v", page, err)
	}
}

func TestResponseReviewRejectsSessionExpiryDuringReview(t *testing.T) {
	s, store, _ := assessmentFixture(t)
	s.ConfigureDemoResponseReviewDecisions(true)
	now := time.Now().UTC()
	actor := identity.Actor{TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "reviewer", ExpiresAt: now.Add(time.Second)}
	ctx := identity.WithActor(context.Background(), actor)
	s.now = func() time.Time { return now }
	calls := 0
	s.WithResponseReviewAuthorizer(func(context.Context, identity.Actor, CompletedResponseSummary) (string, error) {
		calls++
		if calls == 1 {
			now = now.Add(2 * time.Second)
		}
		return "policy:rule", nil
	})
	_, err := s.RecordResponseReview(ctx, "tenant", "entity", "response", RecordResponseReviewInput{Decision: "REVIEWED"})
	if !errors.Is(err, ErrAssessmentForbidden) || calls != 1 || len(store.events) != 0 || len(store.outbox) != 0 || len(store.responseReviews["response"]) != 0 {
		t.Fatalf("expired actor recorded a review: err=%v calls=%d events=%d outbox=%d", err, calls, len(store.events), len(store.outbox))
	}
}

func TestWholeResponseReviewRecordsIndependentVersionedDecision(t *testing.T) {
	s, store, ctx := assessmentFixture(t)
	s.ConfigureDemoResponseReviewDecisions(true)
	s.WithResponseReviewAuthorizer(func(context.Context, identity.Actor, CompletedResponseSummary) (string, error) {
		return "policy:rule", nil
	})
	input := RecordResponseReviewInput{ExpectedVersion: 0, Decision: "REVIEWED", Comment: "Reviewed the report."}
	first, err := s.RecordResponseReview(ctx, "tenant", "entity", "response", input)
	if err != nil || first.Version != 1 || first.ReviewerID != "reviewer" || first.AuthorityRoute != "policy:rule" || first.ResponseRevision != 1 {
		t.Fatalf("review=%+v err=%v", first, err)
	}
	if _, err := s.RecordResponseReview(ctx, "tenant", "entity", "response", input); !errors.Is(err, ErrAssessmentConflict) {
		t.Fatalf("stale review: %v", err)
	}
	input.ExpectedVersion = 1
	input.Decision = "CHANGES_REQUESTED"
	input.Comment = "Please add the missing period."
	second, err := s.RecordResponseReview(ctx, "tenant", "entity", "response", input)
	if err != nil || second.Version != 2 || second.SupersedesID != first.ID {
		t.Fatalf("correction=%+v err=%v", second, err)
	}
	if len(store.events) != 2 || len(store.outbox) != 2 {
		t.Fatal("review event or outbox missing")
	}
	sub := store.repo.submissions["submission"]
	sub.SubmittedBy = "reviewer"
	store.repo.submissions["submission"] = sub
	input.ExpectedVersion = 2
	if _, err := s.RecordResponseReview(ctx, "tenant", "entity", "response", input); !errors.Is(err, ErrAssessmentForbidden) {
		t.Fatalf("self review: %v", err)
	}
	if len(store.events) != 2 {
		t.Fatal("denied review created an event")
	}
}

func TestResponseReviewHistoryRequiresScopedReadAndReportsPermission(t *testing.T) {
	s, _, ctx := assessmentFixture(t)
	s.ConfigureDemoResponseReviewDecisions(true)
	allowed := true
	s.WithResponseReviewAuthorizer(func(context.Context, identity.Actor, CompletedResponseSummary) (string, error) {
		if !allowed {
			return "", ErrAssessmentForbidden
		}
		return "policy:rule", nil
	})
	initial, err := s.GetResponseReviews(ctx, "tenant", "entity", "reviewer", "response", 0, 1)
	if err != nil || initial.Version != 0 || !initial.MayReview || len(initial.Items) != 0 {
		t.Fatalf("initial=%+v err=%v", initial, err)
	}
	for i := int64(0); i < 2; i++ {
		if _, err = s.RecordResponseReview(ctx, "tenant", "entity", "response", RecordResponseReviewInput{ExpectedVersion: i, Decision: "REVIEWED"}); err != nil {
			t.Fatal(err)
		}
	}
	page, err := s.GetResponseReviews(ctx, "tenant", "entity", "reviewer", "response", 0, 1)
	if err != nil || page.Version != 2 || len(page.Items) != 1 || page.Items[0].Version != 2 || page.NextBeforeVersion != 2 {
		t.Fatalf("page=%+v err=%v", page, err)
	}
	next, err := s.GetResponseReviews(ctx, "tenant", "entity", "reviewer", "response", page.NextBeforeVersion, 1)
	if err != nil || next.Version != 2 || len(next.Items) != 1 || next.Items[0].Version != 1 || next.NextBeforeVersion != 0 {
		t.Fatalf("next=%+v err=%v", next, err)
	}
	allowed = false
	page, err = s.GetResponseReviews(ctx, "tenant", "entity", "reviewer", "response", 0, 10)
	if err != nil || page.MayReview {
		t.Fatalf("revoked permission=%+v err=%v", page, err)
	}
	if _, err = s.GetResponseReviews(ctx, "tenant", "other", "reviewer", "response", 0, 10); err == nil {
		t.Fatal("cross-entity read allowed")
	}
	if _, err = s.GetResponseReviews(context.Background(), "tenant", "entity", "reviewer", "response", 0, 10); err == nil {
		t.Fatal("unverified read allowed")
	}
}
