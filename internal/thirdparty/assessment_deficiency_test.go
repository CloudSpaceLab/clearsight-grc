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
	if err != nil || len(links) == 0 {
		t.Fatalf("matter links = (%#v, %v)", links, err)
	}
	var deficiencyLink AssessmentMatterLink
	for _, link := range links {
		if link.Kind == AssessmentMatterDeficiency {
			deficiencyLink = link
		}
	}
	if deficiencyLink.MatterID != outcome.Matter.Matter.ID || deficiencyLink.RelationshipLinkID == "" {
		t.Fatalf("deficiency link = %#v", deficiencyLink)
	}
	canonical, err := repo.ListRelationshipLinks(context.Background(), scopeFromVerified(), RelationshipLinkListInput{RelationshipID: assessment.RelationshipID, TargetType: LinkTargetMatter, Limit: 10})
	if err != nil || len(canonical.Items) == 0 {
		t.Fatalf("canonical relationship links = (%#v, %v)", canonical, err)
	}
	for _, link := range links {
		found := false
		for _, relationshipLink := range canonical.Items {
			if relationshipLink.ID == link.RelationshipLinkID && relationshipLink.TargetID == link.MatterID && relationshipLink.RelationshipID == assessment.RelationshipID {
				found = true
			}
		}
		if !found {
			t.Fatalf("assessment link does not reference its canonical vendor relationship link: %#v", link)
		}
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

type ambiguousDeficiencyRepository struct {
	AssessmentRepository
	failBeforeCommit bool
	afterCommit      func()
}

type ambiguousDeficiencyListUnavailableRepository struct {
	*ambiguousDeficiencyRepository
}

func (r *ambiguousDeficiencyListUnavailableRepository) ListAssessmentMatterLinks(context.Context, Scope, string, int) ([]AssessmentMatterLink, error) {
	return nil, errors.New("assessment Matter list unavailable")
}

func (r *ambiguousDeficiencyRepository) LinkAssessmentDeficiency(ctx context.Context, record LinkAssessmentDeficiencyRecord) (AssessmentMatterLink, Assessment, error) {
	if r.failBeforeCommit {
		return AssessmentMatterLink{}, Assessment{}, errors.New("link unavailable")
	}
	_, _, err := r.AssessmentRepository.LinkAssessmentDeficiency(ctx, record)
	if err != nil {
		return AssessmentMatterLink{}, Assessment{}, err
	}
	if r.afterCommit != nil {
		r.afterCommit()
	}
	return AssessmentMatterLink{}, Assessment{}, errors.New("commit result unavailable")
}

type recordingCompensatingMatters struct {
	service *continuity.Service
	created continuity.MatterAggregate
}

func (m *recordingCompensatingMatters) MatterByTriggerKey(ctx context.Context, tenant, triggerKey string) (continuity.MatterAggregate, error) {
	return m.service.MatterByTriggerKey(ctx, tenant, triggerKey)
}

func (m *recordingCompensatingMatters) CreateMatter(ctx context.Context, input continuity.CreateMatterInput) (continuity.MatterAggregate, error) {
	value, err := m.service.CreateMatter(ctx, input)
	if err == nil {
		m.created = value
	}
	return value, err
}

func (m *recordingCompensatingMatters) TransitionMatter(ctx context.Context, input continuity.TransitionInput) (continuity.MatterAggregate, error) {
	return m.service.TransitionMatter(ctx, input)
}

func TestCreateAssessmentDeficiencyReconcilesCommittedLinkAfterAmbiguousFailure(t *testing.T) {
	assessmentService, repo, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	assessment := assessmentUnderReviewFixture(t, assessmentService, repo, relationship)
	matters := continuity.NewService(continuity.NewMemoryRepository())
	service := NewAssessmentDeficiencyService(assessmentService, &ambiguousDeficiencyRepository{AssessmentRepository: repo}, matters)
	input := CreateAssessmentDeficiencyInput{ExpectedVersion: assessment.Version, TriggerKey: "access-review", Title: "Resolve the access review gap", Summary: "The submitted access review does not cover the assessed service."}

	outcome, err := service.CreateDeficiency(assessmentContext(), assessmentActor(), assessment.ID, input)
	if err != nil {
		t.Fatalf("ambiguous committed link returned an error: %v", err)
	}
	if outcome.Assessment.Version != assessment.Version+1 || outcome.Matter.Matter.ID == "" {
		t.Fatalf("reconciled outcome = %#v", outcome)
	}
}

func TestCreateAssessmentDeficiencyCanonicalLinkWinsAfterLaterAssessmentMutation(t *testing.T) {
	assessmentService, repo, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	assessment := assessmentUnderReviewFixture(t, assessmentService, repo, relationship)
	matters := &recordingCompensatingMatters{service: continuity.NewService(continuity.NewMemoryRepository())}
	ambiguous := &ambiguousDeficiencyRepository{AssessmentRepository: repo}
	ambiguous.afterCommit = func() {
		repo.assessmentMu.Lock()
		current := repo.assessments[assessment.ID]
		current.Version++
		current.UpdatedAt = current.UpdatedAt.Add(time.Second)
		repo.assessments[assessment.ID] = current
		repo.assessmentMu.Unlock()
	}
	service := NewAssessmentDeficiencyService(assessmentService, ambiguous, matters)
	input := CreateAssessmentDeficiencyInput{ExpectedVersion: assessment.Version, TriggerKey: "access-review", Title: "Resolve the access review gap", Summary: "The submitted access review does not cover the assessed service."}

	outcome, err := service.CreateDeficiency(assessmentContext(), assessmentActor(), assessment.ID, input)
	if err != nil {
		t.Fatalf("canonical committed link returned an error after a later mutation: %v", err)
	}
	if outcome.Assessment.Version != assessment.Version+2 {
		t.Fatalf("reconciled current assessment = %#v", outcome.Assessment)
	}
	stored, err := matters.service.GetMatter(context.Background(), assessment.TenantID, matters.created.Matter.ID)
	if err != nil || stored.Matter.Status == continuity.MatterCancelled {
		t.Fatalf("linked Matter was cancelled: (%#v, %v)", stored.Matter, err)
	}
}

func TestCreateAssessmentDeficiencyReconcilesByExactCanonicalAssociation(t *testing.T) {
	assessmentService, repo, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	assessment := assessmentUnderReviewFixture(t, assessmentService, repo, relationship)
	matters := &recordingCompensatingMatters{service: continuity.NewService(continuity.NewMemoryRepository())}
	ambiguous := &ambiguousDeficiencyListUnavailableRepository{ambiguousDeficiencyRepository: &ambiguousDeficiencyRepository{AssessmentRepository: repo}}
	service := NewAssessmentDeficiencyService(assessmentService, ambiguous, matters)
	input := CreateAssessmentDeficiencyInput{ExpectedVersion: assessment.Version, TriggerKey: "access-review", Title: "Resolve the access review gap", Summary: "The submitted access review does not cover the assessed service."}

	outcome, err := service.CreateDeficiency(assessmentContext(), assessmentActor(), assessment.ID, input)
	if err != nil || outcome.Matter.Matter.ID == "" {
		t.Fatalf("exact canonical reconciliation = (%#v, %v)", outcome, err)
	}
	stored, err := matters.service.GetMatter(context.Background(), assessment.TenantID, matters.created.Matter.ID)
	if err != nil || stored.Matter.Status == continuity.MatterCancelled {
		t.Fatalf("linked Matter was cancelled: (%#v, %v)", stored.Matter, err)
	}
}

func TestCreateAssessmentDeficiencyCancelsNewMatterWhenLinkDidNotCommit(t *testing.T) {
	assessmentService, repo, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	assessment := assessmentUnderReviewFixture(t, assessmentService, repo, relationship)
	matters := &recordingCompensatingMatters{service: continuity.NewService(continuity.NewMemoryRepository())}
	service := NewAssessmentDeficiencyService(assessmentService, &ambiguousDeficiencyRepository{AssessmentRepository: repo, failBeforeCommit: true}, matters)
	input := CreateAssessmentDeficiencyInput{ExpectedVersion: assessment.Version, TriggerKey: "access-review", Title: "Resolve the access review gap", Summary: "The submitted access review does not cover the assessed service."}

	if _, err := service.CreateDeficiency(assessmentContext(), assessmentActor(), assessment.ID, input); err == nil {
		t.Fatal("link failure returned success")
	}
	stored, err := matters.service.GetMatter(context.Background(), assessment.TenantID, matters.created.Matter.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Matter.Status != continuity.MatterCancelled || !strings.Contains(stored.Matter.ClosureReason, "not linked") {
		t.Fatalf("compensated matter = %#v", stored.Matter)
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
