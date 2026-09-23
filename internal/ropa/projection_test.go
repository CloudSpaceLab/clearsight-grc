package ropa_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
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

	recordingLister := &recordingProjectionLister{delegate: repository}
	service.SetLister(recordingLister)
	maintainer := ropa.NewSummaryMaintainer(repository, summaries, service)
	maintainer.BatchSize = 7
	scope := ropa.Scope{TenantID: "tenant-1", LegalEntityID: "entity-1"}
	if err := maintainer.Maintain(context.Background(), scope); err != nil {
		t.Fatalf("paginated maintain: %v", err)
	}
	filters := recordingLister.snapshot()
	if len(filters) != 15 {
		t.Fatalf("paginated lister calls = %d, want 15", len(filters))
	}
	for index, filter := range filters {
		if filter.Limit != 7 {
			t.Fatalf("page %d requested limit = %d, want 7", index+1, filter.Limit)
		}
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
	if summary.Counts.Total != summary.Counts.New+summary.Counts.Open+summary.Counts.Closed {
		t.Fatalf("pagination counts do not reconcile: total=%d new=%d open=%d closed=%d", summary.Counts.Total, summary.Counts.New, summary.Counts.Open, summary.Counts.Closed)
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

const expectedProjectionPageLimit = 1000
const expectedProjectionBatchLimit = 1000

func projectionTestActivity(id string) ropa.ProcessingActivity {
	return ropa.ProcessingActivity{
		ID:                    id,
		TenantID:              "tenant-1",
		LegalEntityID:         "entity-1",
		Status:                ropa.StatusNew,
		LawfulBasis:           "Contract",
		DataSubjectCategories: "Customers",
		OwnerPrincipalID:      "owner-1",
		UpdatedAt:             projectionNow,
	}
}

type scriptedProjectionLister struct {
	pages  []ropa.ActivityPage
	err    error
	calls  int
	before func(int)
}

func (l *scriptedProjectionLister) ListActivities(ctx context.Context, _ ropa.ListActivitiesFilter) (ropa.ActivityPage, error) {
	if err := ctx.Err(); err != nil {
		return ropa.ActivityPage{}, err
	}
	l.calls++
	if l.before != nil {
		l.before(l.calls)
	}
	if l.err != nil {
		return ropa.ActivityPage{}, l.err
	}
	if len(l.pages) == 0 {
		return ropa.ActivityPage{}, nil
	}
	if l.calls > len(l.pages) {
		return ropa.ActivityPage{}, nil
	}
	return l.pages[l.calls-1], nil
}

// advancingProjectionLister models an unbounded stream of unique cursors. The
// test harness cap is deliberately above the production page bound so the
// maintainer, rather than the fake, must stop the traversal.
type advancingProjectionLister struct {
	calls    int
	maxCalls int
}

func (l *advancingProjectionLister) ListActivities(ctx context.Context, _ ropa.ListActivitiesFilter) (ropa.ActivityPage, error) {
	if err := ctx.Err(); err != nil {
		return ropa.ActivityPage{}, err
	}
	l.calls++
	if l.maxCalls > 0 && l.calls > l.maxCalls {
		return ropa.ActivityPage{}, nil
	}
	return ropa.ActivityPage{
		Rows:       []ropa.ProcessingActivity{projectionTestActivity(fmt.Sprintf("advancing-%d", l.calls))},
		HasMore:    true,
		NextCursor: fmt.Sprintf("cursor-%d", l.calls),
	}, nil
}

type recordingProjectionLister struct {
	mu       sync.Mutex
	delegate ropa.ActivityLister
	filters  []ropa.ListActivitiesFilter
}

func (l *recordingProjectionLister) ListActivities(ctx context.Context, filter ropa.ListActivitiesFilter) (ropa.ActivityPage, error) {
	l.mu.Lock()
	l.filters = append(l.filters, filter)
	l.mu.Unlock()
	return l.delegate.ListActivities(ctx, filter)
}

func (l *recordingProjectionLister) snapshot() []ropa.ListActivitiesFilter {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]ropa.ListActivitiesFilter(nil), l.filters...)
}

type recordingProjectionSummaryRepository struct {
	mu         sync.Mutex
	summary    ropa.RegisterSummary
	hasSummary bool
	writes     int
	replaceErr error
}

func (r *recordingProjectionSummaryRepository) LatestSummary(_ context.Context, _, _ string) (ropa.RegisterSummary, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.hasSummary {
		return ropa.RegisterSummary{}, ropa.ErrNotFound
	}
	return r.summary, nil
}

func (r *recordingProjectionSummaryRepository) ReplaceSummary(_ context.Context, summary ropa.RegisterSummary) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.writes++
	if r.replaceErr != nil {
		return r.replaceErr
	}
	r.summary = summary
	r.hasSummary = true
	return nil
}

func (r *recordingProjectionSummaryRepository) writeCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.writes
}

type blockingProjectionRepository struct {
	*ropa.MemoryRepository
	firstStarted chan struct{}
	releaseFirst chan struct{}
	mu           sync.Mutex
	calls        int
}

func (r *blockingProjectionRepository) ListActivities(ctx context.Context, filter ropa.ListActivitiesFilter) (ropa.ActivityPage, error) {
	r.mu.Lock()
	r.calls++
	call := r.calls
	r.mu.Unlock()
	if call == 1 {
		close(r.firstStarted)
		select {
		case <-r.releaseFirst:
		case <-ctx.Done():
			return ropa.ActivityPage{}, ctx.Err()
		}
	}
	return r.MemoryRepository.ListActivities(ctx, filter)
}

func assertNoProjectionSummary(t *testing.T, summaries ropa.SummaryRepository, scope ropa.Scope) {
	t.Helper()
	if _, err := summaries.LatestSummary(context.Background(), scope.TenantID, scope.LegalEntityID); !errors.Is(err, ropa.ErrNotFound) {
		t.Fatalf("summary was written after projection failure: %v", err)
	}
}

func TestMaintainStopsAtTheProjectionPageLimit(t *testing.T) {
	if ropa.MaxProjectionPages != expectedProjectionPageLimit {
		t.Fatalf("MaxProjectionPages = %d, want %d", ropa.MaxProjectionPages, expectedProjectionPageLimit)
	}
	repository := ropa.NewMemoryRepository()
	summaries := ropa.NewMemorySummaryRepository()
	lister := &advancingProjectionLister{maxCalls: expectedProjectionPageLimit + 1}
	service := ropa.NewService(repository, summaries)
	service.Now = func() time.Time { return projectionNow }
	service.SetLister(lister)
	maintainer := ropa.NewSummaryMaintainer(repository, summaries, service)
	scope := ropa.Scope{TenantID: "tenant-1", LegalEntityID: "entity-1"}

	if err := maintainer.Maintain(context.Background(), scope); !errors.Is(err, ropa.ErrInvalid) {
		t.Fatalf("page limit error = %v, want ErrInvalid", err)
	}
	if lister.calls > expectedProjectionPageLimit {
		t.Fatalf("lister calls = %d, want no more than %d", lister.calls, expectedProjectionPageLimit)
	}
	assertNoProjectionSummary(t, summaries, scope)
}

func TestMaintainRejectsMoreFlagOnAnEmptyPage(t *testing.T) {
	repository := ropa.NewMemoryRepository()
	summaries := ropa.NewMemorySummaryRepository()
	lister := &scriptedProjectionLister{pages: []ropa.ActivityPage{{HasMore: true, NextCursor: "next"}}}
	service := ropa.NewService(repository, summaries)
	service.Now = func() time.Time { return projectionNow }
	service.SetLister(lister)
	maintainer := ropa.NewSummaryMaintainer(repository, summaries, service)
	scope := ropa.Scope{TenantID: "tenant-1", LegalEntityID: "entity-1"}

	if err := maintainer.Maintain(context.Background(), scope); !errors.Is(err, ropa.ErrInvalid) {
		t.Fatalf("empty HasMore page error = %v, want ErrInvalid", err)
	}
	assertNoProjectionSummary(t, summaries, scope)
}

func TestMaintainRejectsACursorCycleAfterAnAdvance(t *testing.T) {
	repository := ropa.NewMemoryRepository()
	summaries := ropa.NewMemorySummaryRepository()
	lister := &scriptedProjectionLister{pages: []ropa.ActivityPage{
		{Rows: []ropa.ProcessingActivity{projectionTestActivity("cycle-a")}, HasMore: true, NextCursor: "A"},
		{Rows: []ropa.ProcessingActivity{projectionTestActivity("cycle-b")}, HasMore: true, NextCursor: "B"},
		{Rows: []ropa.ProcessingActivity{projectionTestActivity("cycle-a-again")}, HasMore: true, NextCursor: "A"},
	}}
	service := ropa.NewService(repository, summaries)
	service.Now = func() time.Time { return projectionNow }
	service.SetLister(lister)
	maintainer := ropa.NewSummaryMaintainer(repository, summaries, service)
	scope := ropa.Scope{TenantID: "tenant-1", LegalEntityID: "entity-1"}

	if err := maintainer.Maintain(context.Background(), scope); !errors.Is(err, ropa.ErrInvalid) {
		t.Fatalf("cursor cycle error = %v, want ErrInvalid", err)
	}
	if lister.calls != 3 {
		t.Fatalf("lister calls = %d, want 3 calls before detecting A,B,A", lister.calls)
	}
	assertNoProjectionSummary(t, summaries, scope)
}

func TestMaintainStopsWhenContextIsCancelledMidScan(t *testing.T) {
	repository := ropa.NewMemoryRepository()
	summaries := &recordingProjectionSummaryRepository{}
	ctx, cancel := context.WithCancel(context.Background())
	lister := &scriptedProjectionLister{
		pages: []ropa.ActivityPage{{Rows: []ropa.ProcessingActivity{projectionTestActivity("cancelled")}, HasMore: false}},
		before: func(int) {
			cancel()
		},
	}
	service := ropa.NewService(repository, summaries)
	service.Now = func() time.Time { return projectionNow }
	service.SetLister(lister)
	maintainer := ropa.NewSummaryMaintainer(repository, summaries, service)
	scope := ropa.Scope{TenantID: "tenant-1", LegalEntityID: "entity-1"}

	if err := maintainer.Maintain(ctx, scope); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled maintain error = %v, want context.Canceled", err)
	}
	if summaries.writeCount() != 0 {
		t.Fatal("cancelled maintain must not write a summary")
	}
}

func TestMaintainRejectsARowFromAnotherTenant(t *testing.T) {
	repository := ropa.NewMemoryRepository()
	summaries := ropa.NewMemorySummaryRepository()
	row := projectionTestActivity("other-tenant")
	row.TenantID = "tenant-2"
	lister := &scriptedProjectionLister{pages: []ropa.ActivityPage{{Rows: []ropa.ProcessingActivity{row}}}}
	service := ropa.NewService(repository, summaries)
	service.Now = func() time.Time { return projectionNow }
	service.SetLister(lister)
	maintainer := ropa.NewSummaryMaintainer(repository, summaries, service)
	scope := ropa.Scope{TenantID: "tenant-1", LegalEntityID: "entity-1"}

	if err := maintainer.Maintain(context.Background(), scope); !errors.Is(err, ropa.ErrScopeMismatch) {
		t.Fatalf("cross-tenant row error = %v, want ErrScopeMismatch", err)
	}
	assertNoProjectionSummary(t, summaries, scope)
}

func TestMaintainRejectsAnUnknownActivityStatus(t *testing.T) {
	repository := ropa.NewMemoryRepository()
	summaries := ropa.NewMemorySummaryRepository()
	row := projectionTestActivity("unknown-status")
	row.Status = ropa.Status("ARCHIVED")
	lister := &scriptedProjectionLister{pages: []ropa.ActivityPage{{Rows: []ropa.ProcessingActivity{row}}}}
	service := ropa.NewService(repository, summaries)
	service.Now = func() time.Time { return projectionNow }
	service.SetLister(lister)
	maintainer := ropa.NewSummaryMaintainer(repository, summaries, service)
	scope := ropa.Scope{TenantID: "tenant-1", LegalEntityID: "entity-1"}

	if err := maintainer.Maintain(context.Background(), scope); !errors.Is(err, ropa.ErrInvalid) {
		t.Fatalf("unknown status error = %v, want ErrInvalid", err)
	}
	assertNoProjectionSummary(t, summaries, scope)
}

func TestMemorySummaryRepositoryDoesNotReplaceNewerWithOlder(t *testing.T) {
	summaries := ropa.NewMemorySummaryRepository()
	scope := ropa.Scope{TenantID: "tenant-1", LegalEntityID: "entity-1"}
	newer := ropa.RegisterSummary{
		TenantID:          scope.TenantID,
		LegalEntityID:     scope.LegalEntityID,
		GeneratedAt:       projectionNow,
		ProjectionVersion: ropa.ProjectionVersion,
		Counts:            ropa.RegisterCounts{Total: 2, New: 2},
	}
	older := newer
	older.GeneratedAt = projectionNow.Add(-time.Minute)
	older.Counts = ropa.RegisterCounts{Total: 1, New: 1}

	if err := summaries.ReplaceSummary(context.Background(), newer); err != nil {
		t.Fatalf("store newer summary: %v", err)
	}
	if err := summaries.ReplaceSummary(context.Background(), older); err != nil {
		t.Fatalf("superseded older summary should be ignored: %v", err)
	}
	got, err := summaries.LatestSummary(context.Background(), scope.TenantID, scope.LegalEntityID)
	if err != nil {
		t.Fatalf("read summary: %v", err)
	}
	if !got.GeneratedAt.Equal(newer.GeneratedAt) || !reflect.DeepEqual(got.Counts, newer.Counts) {
		t.Fatalf("older summary replaced newer summary: got generated=%s counts=%#v", got.GeneratedAt, got.Counts)
	}
}

func TestMemorySummaryRepositoryAllowsEqualGeneratedAtRefresh(t *testing.T) {
	summaries := ropa.NewMemorySummaryRepository()
	scope := ropa.Scope{TenantID: "tenant-1", LegalEntityID: "entity-1"}
	first := ropa.RegisterSummary{
		TenantID:          scope.TenantID,
		LegalEntityID:     scope.LegalEntityID,
		GeneratedAt:       projectionNow,
		ProjectionVersion: ropa.ProjectionVersion,
		Counts:            ropa.RegisterCounts{Total: 1, New: 1},
	}
	refresh := first
	refresh.Counts = ropa.RegisterCounts{Total: 2, New: 2}

	if err := summaries.ReplaceSummary(context.Background(), first); err != nil {
		t.Fatalf("store first summary: %v", err)
	}
	if err := summaries.ReplaceSummary(context.Background(), refresh); err != nil {
		t.Fatalf("equal-time refresh: %v", err)
	}
	got, err := summaries.LatestSummary(context.Background(), scope.TenantID, scope.LegalEntityID)
	if err != nil {
		t.Fatalf("read refreshed summary: %v", err)
	}
	if !reflect.DeepEqual(got.Counts, refresh.Counts) {
		t.Fatalf("equal-time refresh was not applied: got counts=%#v", got.Counts)
	}
}

func TestMaintainOlderFinishingRunCannotSupersedeNewerRun(t *testing.T) {
	repository := &blockingProjectionRepository{
		MemoryRepository: ropa.NewMemoryRepository(),
		firstStarted:     make(chan struct{}),
		releaseFirst:     make(chan struct{}),
	}
	summaries := ropa.NewMemorySummaryRepository()
	service := ropa.NewService(repository, summaries)
	var clockMu sync.RWMutex
	clock := projectionNow.Add(-time.Hour)
	service.Now = func() time.Time {
		clockMu.RLock()
		defer clockMu.RUnlock()
		return clock
	}
	maintainer := ropa.NewSummaryMaintainer(repository, summaries, service)
	scope := ropa.Scope{TenantID: "tenant-1", LegalEntityID: "entity-1"}

	firstResult := make(chan error, 1)
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(repository.releaseFirst) }) }
	defer release()
	go func() {
		firstResult <- maintainer.Maintain(context.Background(), scope)
	}()
	select {
	case <-repository.firstStarted:
	case <-time.After(time.Second):
		t.Fatal("older maintain did not enter its lister call")
	}

	clockMu.Lock()
	clock = projectionNow
	clockMu.Unlock()
	if err := maintainer.Maintain(context.Background(), scope); err != nil {
		t.Fatalf("newer maintain: %v", err)
	}
	release()
	if err := <-firstResult; err != nil {
		t.Fatalf("older maintain: %v", err)
	}

	got, err := summaries.LatestSummary(context.Background(), scope.TenantID, scope.LegalEntityID)
	if err != nil {
		t.Fatalf("read summary: %v", err)
	}
	if !got.GeneratedAt.Equal(projectionNow) {
		t.Fatalf("summary generated at %s, want newer run time %s", got.GeneratedAt, projectionNow)
	}
}

func TestMaintainPropagatesListerError(t *testing.T) {
	repository := ropa.NewMemoryRepository()
	summaries := ropa.NewMemorySummaryRepository()
	expectedErr := errors.New("projection lister unavailable")
	lister := &scriptedProjectionLister{err: expectedErr}
	service := ropa.NewService(repository, summaries)
	service.Now = func() time.Time { return projectionNow }
	service.SetLister(lister)
	maintainer := ropa.NewSummaryMaintainer(repository, summaries, service)
	scope := ropa.Scope{TenantID: "tenant-1", LegalEntityID: "entity-1"}

	if err := maintainer.Maintain(context.Background(), scope); !errors.Is(err, expectedErr) {
		t.Fatalf("lister error = %v, want %v", err, expectedErr)
	}
	assertNoProjectionSummary(t, summaries, scope)
}

func TestMaintainPropagatesSummaryWriteError(t *testing.T) {
	repository := ropa.NewMemoryRepository()
	expectedErr := errors.New("summary write unavailable")
	summaries := &recordingProjectionSummaryRepository{replaceErr: expectedErr}
	service := ropa.NewService(repository, summaries)
	service.Now = func() time.Time { return projectionNow }
	maintainer := ropa.NewSummaryMaintainer(repository, summaries, service)
	scope := ropa.Scope{TenantID: "tenant-1", LegalEntityID: "entity-1"}

	if err := maintainer.Maintain(context.Background(), scope); !errors.Is(err, expectedErr) {
		t.Fatalf("summary write error = %v, want %v", err, expectedErr)
	}
	if summaries.writeCount() != 1 {
		t.Fatalf("summary write attempts = %d, want 1", summaries.writeCount())
	}
}

func TestMaintainClampsBatchSizeBeforeCallingLister(t *testing.T) {
	if ropa.MaxProjectionBatchSize != expectedProjectionBatchLimit {
		t.Fatalf("MaxProjectionBatchSize = %d, want %d", ropa.MaxProjectionBatchSize, expectedProjectionBatchLimit)
	}
	repository := ropa.NewMemoryRepository()
	summaries := ropa.NewMemorySummaryRepository()
	recordingLister := &recordingProjectionLister{delegate: repository}
	service := ropa.NewService(repository, summaries)
	service.Now = func() time.Time { return projectionNow }
	service.SetLister(recordingLister)
	maintainer := ropa.NewSummaryMaintainer(repository, summaries, service)
	maintainer.BatchSize = expectedProjectionBatchLimit + 1
	scope := ropa.Scope{TenantID: "tenant-1", LegalEntityID: "entity-1"}

	if err := maintainer.Maintain(context.Background(), scope); err != nil {
		t.Fatalf("clamped maintain: %v", err)
	}
	filters := recordingLister.snapshot()
	if len(filters) != 1 {
		t.Fatalf("lister calls = %d, want 1", len(filters))
	}
	if filters[0].Limit != ropa.MaxProjectionBatchSize {
		t.Fatalf("requested limit = %d, want clamped limit %d", filters[0].Limit, ropa.MaxProjectionBatchSize)
	}
}

func TestRegisterSummaryTreatsFutureAndPastTimestampsAsStale(t *testing.T) {
	service, _, summaries, _ := newProjectionService()
	service.Now = func() time.Time { return projectionNow }
	cases := []struct {
		name        string
		generatedAt time.Time
	}{
		{name: "past", generatedAt: projectionNow.Add(-16 * time.Minute)},
		{name: "future", generatedAt: projectionNow.Add(time.Minute)},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if err := summaries.ReplaceSummary(context.Background(), ropa.RegisterSummary{
				TenantID:          "tenant-1",
				LegalEntityID:     "entity-1",
				GeneratedAt:       test.generatedAt,
				ProjectionVersion: ropa.ProjectionVersion,
			}); err != nil {
				t.Fatalf("replace %s summary: %v", test.name, err)
			}
			summary, err := service.RegisterSummary(context.Background(), "tenant-1", "entity-1")
			if err != nil {
				t.Fatalf("read %s summary: %v", test.name, err)
			}
			if summary.Freshness != ropa.FreshnessStale {
				t.Fatalf("%s summary freshness = %s, want STALE", test.name, summary.Freshness)
			}
		})
	}
}

func TestMaintainUsesUTCForANonUTCInjectedClock(t *testing.T) {
	service, repository, summaries, _ := newProjectionService()
	location := time.FixedZone("UTC+02:00", 2*60*60)
	localNow := time.Date(2026, 6, 1, 14, 0, 0, 0, location)
	service.Now = func() time.Time { return localNow }
	maintainer := ropa.NewSummaryMaintainer(repository, summaries, service)
	scope := ropa.Scope{TenantID: "tenant-1", LegalEntityID: "entity-1"}

	if err := maintainer.Maintain(context.Background(), scope); err != nil {
		t.Fatalf("maintain with non-UTC clock: %v", err)
	}
	summary := readProjectionSummary(t, summaries, scope)
	if !summary.GeneratedAt.Equal(localNow) {
		t.Fatalf("generated instant = %s, want %s", summary.GeneratedAt, localNow)
	}
	if summary.GeneratedAt.Location() != time.UTC {
		t.Fatalf("generated location = %s, want UTC", summary.GeneratedAt.Location())
	}
}
