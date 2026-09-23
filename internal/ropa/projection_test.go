package ropa_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/ropa"
)

var projectionNow = time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

func newProjectionService() (*ropa.Service, *ropa.MemoryRepository, *ropa.MemorySummaryRepository, *time.Time) {
	repository := ropa.NewMemoryRepository()
	summaries := ropa.NewMemorySummaryRepository()
	service := ropa.NewService(repository, summaries)
	clock := projectionNow.Add(-2 * time.Hour)
	service.Now = func() time.Time { return clock }
	return service, repository, summaries, &clock
}

func projectionInput(code string) ropa.CreateActivityInput {
	return ropa.CreateActivityInput{
		TenantID:              "tenant-1",
		LegalEntityID:         "entity-1",
		Code:                  code,
		Name:                  "Activity " + code,
		Controller:            "Fidelity Bank",
		LawfulBasis:           "Contract",
		DataSubjectCategories: "Customers",
		OwnerPrincipalID:      "owner-1",
		ActorID:               "projection-test",
	}
}

func seedProjectionActivity(t *testing.T, service *ropa.Service, code string, mutate func(*ropa.CreateActivityInput)) ropa.ProcessingActivity {
	t.Helper()
	input := projectionInput(code)
	if mutate != nil {
		mutate(&input)
	}
	activity, err := service.CreateActivity(context.Background(), input)
	if err != nil {
		t.Fatalf("create projection activity %s: %v", code, err)
	}
	return activity
}

func readProjectionSummary(t *testing.T, summaries *ropa.MemorySummaryRepository, scope ropa.Scope) ropa.RegisterSummary {
	t.Helper()
	summary, err := summaries.LatestSummary(context.Background(), scope.TenantID, scope.LegalEntityID)
	if err != nil {
		t.Fatalf("read projection summary: %v", err)
	}
	return summary
}

func TestMaintainProducesHonestCoverage(t *testing.T) {
	service, repository, summaries, clock := newProjectionService()

	*clock = projectionNow.Add(-2 * time.Hour)
	seedProjectionActivity(t, service, "PA-HONEST-001", nil)
	*clock = projectionNow.Add(-time.Hour)
	seedProjectionActivity(t, service, "PA-HONEST-002", func(input *ropa.CreateActivityInput) {
		input.LawfulBasis = ""
	})
	*clock = projectionNow

	maintainer := ropa.NewSummaryMaintainer(repository, summaries, service)
	if err := maintainer.Maintain(context.Background(), ropa.Scope{TenantID: "tenant-1", LegalEntityID: "entity-1"}); err != nil {
		t.Fatalf("maintain: %v", err)
	}

	summary := readProjectionSummary(t, summaries, ropa.Scope{TenantID: "tenant-1", LegalEntityID: "entity-1"})
	if summary.Counts.Total != 2 || summary.Counts.New != 2 || summary.Counts.MissingLawfulBasis != 1 {
		t.Fatalf("honest counts = %#v, want total/new=2/2 and missing lawful basis=1", summary.Counts)
	}
	if summary.Coverage.Population != 2 {
		t.Fatalf("coverage population = %d, want 2", summary.Coverage.Population)
	}
	if summary.Coverage.Excluded == nil || summary.Coverage.Unknown == nil {
		t.Fatal("coverage must contain non-nil excluded and unknown values")
	}
	if *summary.Coverage.Excluded != 0 || *summary.Coverage.Unknown != 0 {
		t.Fatalf("coverage exclusions = %d/%d, want 0/0", *summary.Coverage.Excluded, *summary.Coverage.Unknown)
	}
	if summary.ProjectionVersion != ropa.ProjectionVersion {
		t.Fatalf("projection version = %q, want %q", summary.ProjectionVersion, ropa.ProjectionVersion)
	}
	if !summary.GeneratedAt.Equal(projectionNow) {
		t.Fatalf("generated at = %s, want injected now %s", summary.GeneratedAt, projectionNow)
	}
	if !summary.SourceHighWater.Equal(projectionNow.Add(-time.Hour)) {
		t.Fatalf("source high water = %s, want greatest observed %s", summary.SourceHighWater, projectionNow.Add(-time.Hour))
	}
	if summary.SourceHighWater.IsZero() {
		t.Fatal("source high water must not be the zero time")
	}
}

func TestMaintainCountsEveryLifecycleStatus(t *testing.T) {
	service, repository, summaries, clock := newProjectionService()
	*clock = projectionNow

	newActivity := seedProjectionActivity(t, service, "PA-STATUS-NEW", nil)
	openActivity := seedProjectionActivity(t, service, "PA-STATUS-OPEN", nil)
	closedActivity := seedProjectionActivity(t, service, "PA-STATUS-CLOSED", nil)

	if _, err := service.TransitionActivity(context.Background(), ropa.TransitionActivityInput{
		TenantID:        openActivity.TenantID,
		ActivityID:      openActivity.ID,
		ExpectedVersion: openActivity.Version,
		To:              ropa.StatusOpen,
	}); err != nil {
		t.Fatalf("transition open activity: %v", err)
	}
	if _, err := service.TransitionActivity(context.Background(), ropa.TransitionActivityInput{
		TenantID:        closedActivity.TenantID,
		ActivityID:      closedActivity.ID,
		ExpectedVersion: closedActivity.Version,
		To:              ropa.StatusClosed,
	}); err != nil {
		t.Fatalf("transition closed activity: %v", err)
	}
	if newActivity.Status != ropa.StatusNew {
		t.Fatalf("new activity status = %s", newActivity.Status)
	}

	maintainer := ropa.NewSummaryMaintainer(repository, summaries, service)
	if err := maintainer.Maintain(context.Background(), ropa.Scope{TenantID: "tenant-1", LegalEntityID: "entity-1"}); err != nil {
		t.Fatalf("maintain: %v", err)
	}
	summary := readProjectionSummary(t, summaries, ropa.Scope{TenantID: "tenant-1", LegalEntityID: "entity-1"})
	if summary.Counts.New != 1 || summary.Counts.Open != 1 || summary.Counts.Closed != 1 || summary.Counts.Total != 3 {
		t.Fatalf("lifecycle counts = %#v, want one each of NEW/OPEN/CLOSED and total=3", summary.Counts)
	}
}

func TestMaintainCountsRetiredAndOnlyTheRequestedScope(t *testing.T) {
	service, repository, summaries, clock := newProjectionService()
	*clock = projectionNow.Add(-time.Hour)
	retiredAt := projectionNow.Add(-24 * time.Hour)
	seedProjectionActivity(t, service, "PA-SCOPE-RETIRED", func(input *ropa.CreateActivityInput) {
		input.EndDate = &retiredAt
	})
	seedProjectionActivity(t, service, "PA-SCOPE-OTHER-ENTITY", func(input *ropa.CreateActivityInput) {
		input.LegalEntityID = "entity-2"
	})
	seedProjectionActivity(t, service, "PA-SCOPE-OTHER-TENANT", func(input *ropa.CreateActivityInput) {
		input.TenantID = "tenant-2"
	})

	scope := ropa.Scope{TenantID: "tenant-1", LegalEntityID: "entity-1"}
	maintainer := ropa.NewSummaryMaintainer(repository, summaries, service)
	if err := maintainer.Maintain(context.Background(), scope); err != nil {
		t.Fatalf("maintain: %v", err)
	}
	summary := readProjectionSummary(t, summaries, scope)
	if summary.Counts.Total != 1 || summary.Counts.Retired != 1 || summary.Coverage.Population != 1 {
		t.Fatalf("scoped retired counts = %#v, population=%d, want one retired in the requested scope", summary.Counts, summary.Coverage.Population)
	}
}

func TestMaintainCountsReviewOverdueUsingTheSameNowAsGeneratedAt(t *testing.T) {
	service, repository, summaries, clock := newProjectionService()
	*clock = projectionNow.Add(-time.Hour)
	overdue := projectionNow.Add(-time.Minute)
	future := projectionNow.Add(time.Minute)
	seedProjectionActivity(t, service, "PA-REVIEW-OVERDUE", func(input *ropa.CreateActivityInput) {
		input.NextReviewDate = &overdue
	})
	seedProjectionActivity(t, service, "PA-REVIEW-FUTURE", func(input *ropa.CreateActivityInput) {
		input.NextReviewDate = &future
	})
	*clock = projectionNow

	maintainer := ropa.NewSummaryMaintainer(repository, summaries, service)
	scope := ropa.Scope{TenantID: "tenant-1", LegalEntityID: "entity-1"}
	if err := maintainer.Maintain(context.Background(), scope); err != nil {
		t.Fatalf("maintain: %v", err)
	}
	summary := readProjectionSummary(t, summaries, scope)
	if summary.Counts.ReviewOverdue != 1 {
		t.Fatalf("review overdue count = %d, want 1", summary.Counts.ReviewOverdue)
	}
	if !summary.GeneratedAt.Equal(projectionNow) {
		t.Fatalf("generated at = %s, want %s", summary.GeneratedAt, projectionNow)
	}
}

func TestMaintainCountsMissingOwnerAndDataSubjects(t *testing.T) {
	service, repository, summaries, clock := newProjectionService()
	*clock = projectionNow
	seedProjectionActivity(t, service, "PA-MISSING-OWNER", func(input *ropa.CreateActivityInput) {
		input.OwnerPrincipalID = ""
	})
	seedProjectionActivity(t, service, "PA-MISSING-SUBJECTS", func(input *ropa.CreateActivityInput) {
		input.DataSubjectCategories = ""
	})

	maintainer := ropa.NewSummaryMaintainer(repository, summaries, service)
	scope := ropa.Scope{TenantID: "tenant-1", LegalEntityID: "entity-1"}
	if err := maintainer.Maintain(context.Background(), scope); err != nil {
		t.Fatalf("maintain: %v", err)
	}
	summary := readProjectionSummary(t, summaries, scope)
	if summary.Counts.MissingOwner != 1 || summary.Counts.NoDataSubjects != 1 {
		t.Fatalf("missing-field counts = %#v, want one missing owner and one missing data-subject category", summary.Counts)
	}
}

func TestMaintainRejectsInvalidScope(t *testing.T) {
	service, repository, summaries, _ := newProjectionService()
	maintainer := ropa.NewSummaryMaintainer(repository, summaries, service)
	for _, scope := range []ropa.Scope{
		{TenantID: "", LegalEntityID: "entity-1"},
		{TenantID: "tenant-1", LegalEntityID: ""},
		{TenantID: " ", LegalEntityID: "entity-1"},
		{TenantID: "tenant-1", LegalEntityID: "\t"},
	} {
		if err := maintainer.Maintain(context.Background(), scope); !errors.Is(err, ropa.ErrInvalid) {
			t.Errorf("scope %#v: expected ErrInvalid, got %v", scope, err)
		}
	}
}

func TestMaintainWritesAnEmptySummary(t *testing.T) {
	service, repository, summaries, clock := newProjectionService()
	*clock = projectionNow
	scope := ropa.Scope{TenantID: "tenant-empty", LegalEntityID: "entity-empty"}
	maintainer := ropa.NewSummaryMaintainer(repository, summaries, service)
	if err := maintainer.Maintain(context.Background(), scope); err != nil {
		t.Fatalf("maintain empty scope: %v", err)
	}
	summary := readProjectionSummary(t, summaries, scope)
	if summary.Coverage.Population != 0 || summary.Counts != (ropa.RegisterCounts{}) {
		t.Fatalf("empty summary = %#v, want zero population and all counts zero", summary)
	}
	if summary.Coverage.Excluded == nil || summary.Coverage.Unknown == nil || *summary.Coverage.Excluded != 0 || *summary.Coverage.Unknown != 0 {
		t.Fatalf("empty coverage = %#v, want explicit zero excluded/unknown", summary.Coverage)
	}
	if !summary.GeneratedAt.Equal(projectionNow) || !summary.SourceHighWater.IsZero() {
		t.Fatalf("empty freshness stamps = generated %s, high water %s", summary.GeneratedAt, summary.SourceHighWater)
	}
}

func TestMaintainIsIdempotent(t *testing.T) {
	service, repository, summaries, clock := newProjectionService()
	*clock = projectionNow
	seedProjectionActivity(t, service, "PA-IDEMPOTENT-001", nil)
	seedProjectionActivity(t, service, "PA-IDEMPOTENT-002", func(input *ropa.CreateActivityInput) {
		input.LawfulBasis = ""
	})
	maintainer := ropa.NewSummaryMaintainer(repository, summaries, service)
	scope := ropa.Scope{TenantID: "tenant-1", LegalEntityID: "entity-1"}
	if err := maintainer.Maintain(context.Background(), scope); err != nil {
		t.Fatalf("first maintain: %v", err)
	}
	first := readProjectionSummary(t, summaries, scope)
	if err := maintainer.Maintain(context.Background(), scope); err != nil {
		t.Fatalf("second maintain: %v", err)
	}
	second := readProjectionSummary(t, summaries, scope)
	if !reflect.DeepEqual(first.Counts, second.Counts) || first.Coverage.Population != second.Coverage.Population {
		t.Fatalf("re-running maintain changed counts: first=%#v second=%#v", first.Counts, second.Counts)
	}
}

func TestMaintainPaginatesOneHundredRowsExactly(t *testing.T) {
	service, repository, summaries, clock := newProjectionService()
	*clock = projectionNow.Add(-time.Hour)
	retiredAt := projectionNow.Add(-24 * time.Hour)
	overdue := projectionNow.Add(-time.Minute)
	future := projectionNow.Add(time.Minute)

	for index := 0; index < 100; index++ {
		code := fmt.Sprintf("PA-PAGE-%03d", index)
		seedProjectionActivity(t, service, code, func(input *ropa.CreateActivityInput) {
			if index%4 == 0 {
				input.EndDate = &retiredAt
			}
			if index%5 == 0 {
				input.LawfulBasis = ""
			}
			if index%7 == 0 {
				input.OwnerPrincipalID = ""
			}
			if index%3 == 0 {
				input.DataSubjectCategories = ""
			}
			if index%11 == 0 {
				input.NextReviewDate = &overdue
			} else if index%11 == 1 {
				input.NextReviewDate = &future
			}
		})
	}
	*clock = projectionNow

	maintainer := ropa.NewSummaryMaintainer(repository, summaries, service)
	maintainer.BatchSize = 7
	scope := ropa.Scope{TenantID: "tenant-1", LegalEntityID: "entity-1"}
	if err := maintainer.Maintain(context.Background(), scope); err != nil {
		t.Fatalf("paginated maintain: %v", err)
	}
	summary := readProjectionSummary(t, summaries, scope)
	want := ropa.RegisterCounts{
		Total:              100,
		New:                100,
		Retired:            25,
		ReviewOverdue:      10,
		MissingLawfulBasis: 20,
		MissingOwner:       15,
		NoDataSubjects:     34,
	}
	if !reflect.DeepEqual(summary.Counts, want) || summary.Coverage.Population != 100 {
		t.Fatalf("paginated counts = %#v, population=%d, want %#v", summary.Counts, summary.Coverage.Population, want)
	}
	if summary.Counts.Total != 100 {
		t.Fatalf("pagination total = %d, want 100", summary.Counts.Total)
	}
	t.Logf("100-row projection counts: total=%d new=%d open=%d closed=%d retired=%d review_overdue=%d missing_lawful_basis=%d missing_owner=%d no_data_subjects=%d",
		summary.Counts.Total, summary.Counts.New, summary.Counts.Open, summary.Counts.Closed, summary.Counts.Retired,
		summary.Counts.ReviewOverdue, summary.Counts.MissingLawfulBasis, summary.Counts.MissingOwner, summary.Counts.NoDataSubjects)
}

func TestMaintainAllMaintainsEachScope(t *testing.T) {
	service, repository, summaries, clock := newProjectionService()
	*clock = projectionNow
	seedProjectionActivity(t, service, "PA-ALL-001", nil)
	seedProjectionActivity(t, service, "PA-ALL-002", func(input *ropa.CreateActivityInput) {
		input.LegalEntityID = "entity-2"
	})

	maintainer := ropa.NewSummaryMaintainer(repository, summaries, service)
	scopes := []ropa.Scope{
		{TenantID: "tenant-1", LegalEntityID: "entity-1"},
		{TenantID: "tenant-1", LegalEntityID: "entity-2"},
	}
	if err := maintainer.MaintainAll(context.Background(), scopes); err != nil {
		t.Fatalf("maintain all: %v", err)
	}
	for _, scope := range scopes {
		summary := readProjectionSummary(t, summaries, scope)
		if summary.Counts.Total != 1 {
			t.Fatalf("scope %#v total = %d, want 1", scope, summary.Counts.Total)
		}
	}
}

func TestMaintainAllReturnsTheFirstScopeError(t *testing.T) {
	service, repository, summaries, _ := newProjectionService()
	maintainer := ropa.NewSummaryMaintainer(repository, summaries, service)
	err := maintainer.MaintainAll(context.Background(), []ropa.Scope{
		{TenantID: "tenant-1", LegalEntityID: "entity-1"},
		{TenantID: "tenant-1", LegalEntityID: ""},
		{TenantID: "tenant-1", LegalEntityID: "entity-never-reached"},
	})
	if !errors.Is(err, ropa.ErrInvalid) {
		t.Fatalf("maintain all error = %v, want first ErrInvalid", err)
	}
	if _, err := summaries.LatestSummary(context.Background(), "tenant-1", "entity-never-reached"); !errors.Is(err, ropa.ErrNotFound) {
		t.Fatalf("later scope should not run after first error; got %v", err)
	}
}

type stalledProjectionRepository struct {
	*ropa.MemoryRepository
}

func (r *stalledProjectionRepository) ListActivities(context.Context, ropa.ListActivitiesFilter) (ropa.ActivityPage, error) {
	return ropa.ActivityPage{HasMore: true, NextCursor: "unchanged-cursor"}, nil
}

func TestMaintainRejectsANonAdvancingCursor(t *testing.T) {
	repository := &stalledProjectionRepository{MemoryRepository: ropa.NewMemoryRepository()}
	summaries := ropa.NewMemorySummaryRepository()
	service := ropa.NewService(repository, summaries)
	service.Now = func() time.Time { return projectionNow }
	maintainer := ropa.NewSummaryMaintainer(repository, summaries, service)
	if err := maintainer.Maintain(context.Background(), ropa.Scope{TenantID: "tenant-1", LegalEntityID: "entity-1"}); !errors.Is(err, ropa.ErrInvalid) {
		t.Fatalf("stalled cursor error = %v, want ErrInvalid", err)
	}
}
