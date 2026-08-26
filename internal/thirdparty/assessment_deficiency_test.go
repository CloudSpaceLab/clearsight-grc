package thirdparty

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

func TestCreateAssessmentDeficiencyCreatesCanonicalRestrictedMatterAndSafeLink(t *testing.T) {
	guard := newAssessmentGuard()
	assessmentService, repo, relationship := newAssessmentServiceFixture(t, guard)
	assessment := assessmentUnderReviewFixture(t, assessmentService, repo, relationship)
	guard.requests = nil
	matters := continuity.NewService(continuity.NewMemoryRepository())
	service := NewAssessmentDeficiencyService(assessmentService, repo, matters)
	due := assessment.ReviewDueAt.Add(7 * 24 * time.Hour)
	input := CreateAssessmentDeficiencyInput{ExpectedVersion: assessment.Version, TriggerKey: "security-test-report", Title: "Provide current security test evidence", Summary: "The submitted report is no longer current for this review.", DueAt: &due}

	outcome, err := service.CreateDeficiency(assessmentContext(), Actor{TenantID: "forged", LegalEntityID: "forged", PrincipalID: "forged"}, assessment.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Assessment.Version != assessment.Version+1 || outcome.Assessment.Status != AssessmentUnderReview || outcome.Matter.Matter.Type != continuity.MatterVendorDeficiency {
		t.Fatalf("deficiency outcome = %#v", outcome)
	}
	policy, valid := continuity.ParseMatterAccessPolicy(outcome.Matter.Matter.Scope)
	if !valid || policy.Access != continuity.MatterAccessRestricted || !containsString(policy.AllowedPrincipalIDs, "verified-owner") {
		t.Fatalf("deficiency access = %#v", policy)
	}
	links, err := repo.ListAssessmentMatterLinks(context.Background(), scopeFromVerified(), assessment.ID, assessmentReviewMaxMatters+1)
	if err != nil || len(links) != 2 || links[1].Kind != AssessmentMatterDeficiency || links[1].MatterID != outcome.Matter.Matter.ID {
		t.Fatalf("matter links = (%#v, %v)", links, err)
	}
	replayed, err := service.CreateDeficiency(assessmentContext(), assessmentActor(), assessment.ID, input)
	if err != nil || replayed.Matter.Matter.ID != outcome.Matter.Matter.ID || replayed.Assessment.Version != outcome.Assessment.Version {
		t.Fatalf("replay = (%#v, %v)", replayed, err)
	}
	if len(guard.requests) != 2 || guard.requests[0].DecisionType != AssessmentDeficiencyCommand || guard.requests[0].Responsibility != authority.ResponsibilityReviewer {
		t.Fatalf("authority = %#v", guard.requests)
	}
	stored, _ := json.Marshal([]any{links, repo.assessmentEvents, repo.assessmentOutbox})
	for _, protected := range []string{"recipient", "invitation", "token", "vendor_answers"} {
		if strings.Contains(strings.ToLower(string(stored)), protected) {
			t.Fatalf("protected field persisted: %s", stored)
		}
	}
}

func TestCreateAssessmentDeficiencySeparatesDistinctStableTriggersAndAssessments(t *testing.T) {
	assessmentService, repo, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	first := assessmentUnderReviewFixture(t, assessmentService, repo, relationship)
	matters := continuity.NewService(continuity.NewMemoryRepository())
	service := NewAssessmentDeficiencyService(assessmentService, repo, matters)
	input := CreateAssessmentDeficiencyInput{ExpectedVersion: first.Version, TriggerKey: "security-test-report", Title: "Provide current security test evidence", Summary: "The submitted report is no longer current for this review."}
	one, err := service.CreateDeficiency(assessmentContext(), assessmentActor(), first.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	input.ExpectedVersion = one.Assessment.Version
	input.TriggerKey = "data-retention-gap"
	two, err := service.CreateDeficiency(assessmentContext(), assessmentActor(), first.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	if one.Matter.Matter.ID == two.Matter.Matter.ID || one.Matter.Matter.TriggerKey == two.Matter.Matter.TriggerKey {
		t.Fatal("distinct deficiency triggers collided")
	}

	second := first
	second.ID = "assessment-second"
	second.StableEpisodeKey = "episode-second"
	second.Version = 5
	repo.assessmentMu.Lock()
	repo.assessments[second.ID] = second
	repo.assessmentMu.Unlock()
	input.ExpectedVersion, input.TriggerKey = second.Version, "security-test-report"
	three, err := service.CreateDeficiency(assessmentContext(), assessmentActor(), second.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	if one.Matter.Matter.ID == three.Matter.Matter.ID || one.Matter.Matter.TriggerKey == three.Matter.Matter.TriggerKey {
		t.Fatal("assessment-scoped deficiency triggers collided")
	}
}

func TestCreateAssessmentDeficiencyRejectsStaleWrongStateScopeAndInvalidTrigger(t *testing.T) {
	assessmentService, repo, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	assessment := assessmentUnderReviewFixture(t, assessmentService, repo, relationship)
	service := NewAssessmentDeficiencyService(assessmentService, repo, continuity.NewService(continuity.NewMemoryRepository()))
	base := CreateAssessmentDeficiencyInput{ExpectedVersion: assessment.Version, TriggerKey: "security-test-report", Title: "Provide current security test evidence", Summary: "The submitted report is no longer current for this review."}
	stale := base
	stale.ExpectedVersion--
	if _, err := service.CreateDeficiency(assessmentContext(), assessmentActor(), assessment.ID, stale); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale = %v", err)
	}
	bad := base
	bad.TriggerKey = "not allowed spaces"
	if _, err := service.CreateDeficiency(assessmentContext(), assessmentActor(), assessment.ID, bad); !errors.Is(err, ErrInvalid) {
		t.Fatalf("trigger = %v", err)
	}
	if _, err := service.CreateDeficiency(assessmentContextFor("bank", "other-entity", "verified-owner"), assessmentActor(), assessment.ID, base); !errors.Is(err, ErrNotFound) {
		t.Fatalf("scope = %v", err)
	}
	repo.assessmentMu.Lock()
	current := repo.assessments[assessment.ID]
	current.Status = AssessmentCompleted
	repo.assessments[assessment.ID] = current
	repo.assessmentMu.Unlock()
	if _, err := service.CreateDeficiency(assessmentContext(), assessmentActor(), assessment.ID, base); !errors.Is(err, ErrInvalidAssessmentTransition) {
		t.Fatalf("state = %v", err)
	}
}

func TestConcurrentDeficiencyLinkCreatesOneMaterialMutation(t *testing.T) {
	assessmentService, repo, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	assessment := assessmentUnderReviewFixture(t, assessmentService, repo, relationship)
	record := LinkAssessmentDeficiencyRecord{Scope: scopeFromVerified(), AssessmentID: assessment.ID, ExpectedVersion: assessment.Version, ActorPrincipalID: "verified-owner", MatterID: "matter-deficiency", MatterTriggerKey: "vendor-deficiency:stable", LinkedAt: assessment.UpdatedAt.Add(time.Minute)}
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		go func() { _, _, err := repo.LinkAssessmentDeficiency(context.Background(), record); errs <- err }()
	}
	for i := 0; i < 8; i++ {
		if err := <-errs; err != nil {
			t.Fatalf("concurrent deficiency link = %v", err)
		}
	}
	current, _ := repo.GetAssessment(context.Background(), scopeFromVerified(), assessment.ID)
	if current.Version != assessment.Version+1 || len(repo.matterLinks[assessment.ID]) != 1 {
		t.Fatalf("concurrent deficiency state = %#v links=%#v", current, repo.matterLinks[assessment.ID])
	}
	count := 0
	for _, event := range repo.assessmentEvents {
		if event.Type == "AssessmentDeficiencyLinked" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("deficiency events=%d", count)
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
