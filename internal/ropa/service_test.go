package ropa_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/ropa"
)

var task3Now = time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

func task3Service() (*ropa.Service, *ropa.MemoryRepository, *ropa.MemorySummaryRepository) {
	repository := ropa.NewMemoryRepository()
	summaries := ropa.NewMemorySummaryRepository()
	service := ropa.NewService(repository, summaries)
	service.Now = func() time.Time { return task3Now }
	return service, repository, summaries
}

func task3Input() ropa.CreateActivityInput {
	return ropa.CreateActivityInput{
		TenantID:              "tenant-1",
		LegalEntityID:         "entity-1",
		Code:                  "PA-001",
		Name:                  "Customer onboarding",
		Description:           "Collects identity data at onboarding.",
		Purpose:               "Onboard customers",
		LawfulBasis:           "Contract",
		Controller:            "Fidelity Bank",
		Processor:             "Internal operations",
		DataSubjectCategories: "Customers",
		OwnerPrincipalID:      "owner-1",
		ActorID:               "actor-1",
	}
}

func task3Create(t *testing.T, service *ropa.Service, input ropa.CreateActivityInput) ropa.ProcessingActivity {
	t.Helper()
	activity, err := service.CreateActivity(context.Background(), input)
	if err != nil {
		t.Fatalf("create activity: %v", err)
	}
	return activity
}

func TestCreateRejectsEachMissingMandatoryField(t *testing.T) {
	mutations := map[string]func(*ropa.CreateActivityInput){
		"tenant":       func(input *ropa.CreateActivityInput) { input.TenantID = "" },
		"code":         func(input *ropa.CreateActivityInput) { input.Code = "" },
		"name":         func(input *ropa.CreateActivityInput) { input.Name = "" },
		"legal entity": func(input *ropa.CreateActivityInput) { input.LegalEntityID = "" },
		"controller":   func(input *ropa.CreateActivityInput) { input.Controller = "" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			service, _, _ := task3Service()
			input := task3Input()
			mutate(&input)
			_, err := service.CreateActivity(context.Background(), input)
			if !errors.Is(err, ropa.ErrInvalid) {
				t.Fatalf("missing %s: expected ErrInvalid, got %v", name, err)
			}
		})
	}
}

func TestCreateStartsNewAndRecordsOneEvent(t *testing.T) {
	service, repository, _ := task3Service()
	input := task3Input()
	input.Code = "  PA-001  "
	input.Name = "  Customer onboarding  "
	activity := task3Create(t, service, input)

	if activity.Status != ropa.StatusNew || activity.Version != 1 {
		t.Fatalf("created activity status/version = %s/%d, want NEW/1", activity.Status, activity.Version)
	}
	if !activity.CreatedAt.Equal(task3Now) || !activity.UpdatedAt.Equal(task3Now) {
		t.Fatalf("created timestamps = %s/%s, want %s", activity.CreatedAt, activity.UpdatedAt, task3Now)
	}
	if activity.Code != "PA-001" || activity.Name != "Customer onboarding" {
		t.Fatalf("create did not trim strings: %#v", activity)
	}
	events, err := repository.ActivityEvents(context.Background(), activity.TenantID, activity.ID)
	if err != nil {
		t.Fatalf("read events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("created event count = %d, want 1", len(events))
	}
	event := events[0]
	if event.Type != "processing_activity.created" || event.AggregateVersion != 1 || event.AggregateID != activity.ID {
		t.Fatalf("unexpected created event: %#v", event)
	}
	if event.ActorType != "USER" || event.ActorID != "actor-1" {
		t.Fatalf("unexpected event actor: %#v", event)
	}
}

func TestCreateRejectsDuplicateCodeOnlyWithinTheLegalEntity(t *testing.T) {
	service, _, _ := task3Service()
	task3Create(t, service, task3Input())

	_, err := service.CreateActivity(context.Background(), task3Input())
	if !errors.Is(err, ropa.ErrDuplicate) {
		t.Fatalf("same-entity duplicate: expected ErrDuplicate, got %v", err)
	}

	otherEntity := task3Input()
	otherEntity.LegalEntityID = "entity-2"
	otherEntity.Name = "Payments processing"
	if _, err := service.CreateActivity(context.Background(), otherEntity); err != nil {
		t.Fatalf("same code in another legal entity should be allowed: %v", err)
	}
}

func TestClosureIsBlockedUntilRequiredFactsExist(t *testing.T) {
	service, _, _ := task3Service()
	input := task3Input()
	input.LawfulBasis = ""
	input.OwnerPrincipalID = ""
	input.DataSubjectCategories = ""
	activity := task3Create(t, service, input)

	_, err := service.TransitionActivity(context.Background(), ropa.TransitionActivityInput{
		TenantID:        activity.TenantID,
		ActivityID:      activity.ID,
		ExpectedVersion: activity.Version,
		To:              ropa.StatusClosed,
		ActorID:         "actor-1",
	})
	if !errors.Is(err, ropa.ErrClosureBlocked) {
		t.Fatalf("expected ErrClosureBlocked, got %v", err)
	}

	blockers, err := service.ClosureBlockers(context.Background(), activity.ID)
	if err != nil {
		t.Fatalf("read closure blockers: %v", err)
	}
	want := []string{"lawful basis", "named owner", "data subject category"}
	if !reflect.DeepEqual(blockers, want) {
		t.Fatalf("closure blockers = %#v, want %#v", blockers, want)
	}
}

func TestClosureSucceedsWhenRequiredFactsArePresent(t *testing.T) {
	service, _, _ := task3Service()
	activity := task3Create(t, service, task3Input())
	closed, err := service.TransitionActivity(context.Background(), ropa.TransitionActivityInput{
		TenantID:        activity.TenantID,
		ActivityID:      activity.ID,
		ExpectedVersion: activity.Version,
		To:              ropa.StatusClosed,
		ActorID:         "actor-1",
	})
	if err != nil {
		t.Fatalf("close activity: %v", err)
	}
	if closed.Status != ropa.StatusClosed || closed.Version != 2 {
		t.Fatalf("closed activity = %s/%d, want CLOSED/2", closed.Status, closed.Version)
	}
}

func TestClosedActivityCannotTransitionBackToOpen(t *testing.T) {
	service, _, _ := task3Service()
	activity := task3Create(t, service, task3Input())
	activity, err := service.TransitionActivity(context.Background(), ropa.TransitionActivityInput{
		TenantID:        activity.TenantID,
		ActivityID:      activity.ID,
		ExpectedVersion: activity.Version,
		To:              ropa.StatusClosed,
		ActorID:         "actor-1",
	})
	if err != nil {
		t.Fatalf("close activity: %v", err)
	}

	_, err = service.TransitionActivity(context.Background(), ropa.TransitionActivityInput{
		TenantID:        activity.TenantID,
		ActivityID:      activity.ID,
		ExpectedVersion: activity.Version,
		To:              ropa.StatusOpen,
		ActorID:         "actor-1",
	})
	if !errors.Is(err, ropa.ErrInvalid) {
		t.Fatalf("expected ErrInvalid for CLOSED -> OPEN, got %v", err)
	}
}

func TestStaleExpectedVersionReturnsVersionConflict(t *testing.T) {
	service, _, _ := task3Service()
	activity := task3Create(t, service, task3Input())
	_, err := service.TransitionActivity(context.Background(), ropa.TransitionActivityInput{
		TenantID:        activity.TenantID,
		ActivityID:      activity.ID,
		ExpectedVersion: activity.Version + 1,
		To:              ropa.StatusOpen,
		ActorID:         "actor-1",
	})
	if !errors.Is(err, ropa.ErrVersionConflict) {
		t.Fatalf("expected ErrVersionConflict, got %v", err)
	}

	_, err = service.UpdateActivity(context.Background(), ropa.UpdateActivityInput{
		TenantID:        activity.TenantID,
		ActivityID:      activity.ID,
		ExpectedVersion: activity.Version + 1,
		LawfulBasis:     stringPointer("Consent"),
		ActorID:         "actor-1",
	})
	if !errors.Is(err, ropa.ErrVersionConflict) {
		t.Fatalf("update: expected ErrVersionConflict, got %v", err)
	}
}

func TestUpdateChangesOnlySuppliedFields(t *testing.T) {
	service, _, _ := task3Service()
	input := task3Input()
	input.Description = "  Initial description  "
	input.RetentionPeriod = "  7 years  "
	activity := task3Create(t, service, input)
	newDescription := "  Updated description  "

	updated, err := service.UpdateActivity(context.Background(), ropa.UpdateActivityInput{
		TenantID:        activity.TenantID,
		ActivityID:      activity.ID,
		ExpectedVersion: activity.Version,
		Description:     &newDescription,
		ActorID:         "actor-1",
	})
	if err != nil {
		t.Fatalf("update activity: %v", err)
	}
	if updated.Description != "Updated description" {
		t.Fatalf("description = %q, want trimmed update", updated.Description)
	}
	if updated.LawfulBasis != activity.LawfulBasis || updated.OwnerPrincipalID != activity.OwnerPrincipalID || updated.RetentionPeriod != activity.RetentionPeriod {
		t.Fatalf("unsupplied fields changed: before=%#v after=%#v", activity, updated)
	}
	if updated.Version != 2 {
		t.Fatalf("updated version = %d, want 2", updated.Version)
	}

	empty := ""
	updated, err = service.UpdateActivity(context.Background(), ropa.UpdateActivityInput{
		TenantID:        updated.TenantID,
		ActivityID:      updated.ID,
		ExpectedVersion: updated.Version,
		LawfulBasis:     &empty,
		ActorID:         "actor-1",
	})
	if err != nil {
		t.Fatalf("clear lawful basis: %v", err)
	}
	if updated.LawfulBasis != "" {
		t.Fatalf("deliberate clear left lawful basis = %q", updated.LawfulBasis)
	}
}

func TestUpdateMaintainsTheLegalEntityCodeIndex(t *testing.T) {
	service, _, _ := task3Service()
	activity := task3Create(t, service, task3Input())
	newCode := "PA-002"
	updated, err := service.UpdateActivity(context.Background(), ropa.UpdateActivityInput{
		TenantID:        activity.TenantID,
		ActivityID:      activity.ID,
		ExpectedVersion: activity.Version,
		Code:            &newCode,
		ActorID:         "actor-1",
	})
	if err != nil {
		t.Fatalf("update code: %v", err)
	}
	if updated.Code != newCode {
		t.Fatalf("updated code = %q, want %q", updated.Code, newCode)
	}

	reused := task3Input()
	reused.Code = newCode
	if _, err := service.CreateActivity(context.Background(), reused); !errors.Is(err, ropa.ErrDuplicate) {
		t.Fatalf("new code should remain reserved, got %v", err)
	}

	oldCode := task3Input()
	oldCode.Code = "PA-001"
	oldCode.Name = "New activity using released code"
	if _, err := service.CreateActivity(context.Background(), oldCode); err != nil {
		t.Fatalf("old code should be released after update: %v", err)
	}
}

func TestRetiredActivityIsReadableButExcludedUnlessRequested(t *testing.T) {
	service, _, _ := task3Service()
	retiredAt := task3Now.AddDate(0, 0, -1)
	retiredInput := task3Input()
	retiredInput.EndDate = &retiredAt
	retired := task3Create(t, service, retiredInput)
	live := task3Create(t, service, func() ropa.CreateActivityInput {
		input := task3Input()
		input.Code = "PA-002"
		input.Name = "Live activity"
		return input
	}())

	if _, err := service.GetActivity(context.Background(), retired.TenantID, retired.ID); err != nil {
		t.Fatalf("retired activity should remain readable: %v", err)
	}
	page, err := service.ListActivities(context.Background(), ropa.ListActivitiesFilter{
		TenantID:       retired.TenantID,
		LegalEntityID:  retired.LegalEntityID,
		IncludeRetired: false,
		Limit:          50,
	})
	if err != nil {
		t.Fatalf("list live activities: %v", err)
	}
	if len(page.Rows) != 1 || page.Rows[0].ID != live.ID {
		t.Fatalf("live list = %#v, want only %s", page.Rows, live.ID)
	}

	page, err = service.ListActivities(context.Background(), ropa.ListActivitiesFilter{
		TenantID:       retired.TenantID,
		LegalEntityID:  retired.LegalEntityID,
		IncludeRetired: true,
		Limit:          50,
	})
	if err != nil {
		t.Fatalf("list including retired activities: %v", err)
	}
	if len(page.Rows) != 2 {
		t.Fatalf("history list count = %d, want 2", len(page.Rows))
	}
}

func TestEndDateBeforeStartDateIsRejected(t *testing.T) {
	service, _, _ := task3Service()
	start := task3Now
	end := start.AddDate(0, 0, -1)
	_, err := service.CreateActivity(context.Background(), ropa.CreateActivityInput{
		TenantID:      "tenant-1",
		LegalEntityID: "entity-1",
		Code:          "PA-009",
		Name:          "Bad dates",
		Controller:    "Fidelity Bank",
		StartDate:     &start,
		EndDate:       &end,
		ActorID:       "actor-1",
	})
	if !errors.Is(err, ropa.ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

func TestKeysetPaginationHasNoDuplicatesOrSkippedRowsAndRejectsInvalidCursor(t *testing.T) {
	service, _, _ := task3Service()
	created := make([]string, 0, 7)
	for index := 0; index < 7; index++ {
		input := task3Input()
		input.Code = "PA-" + string(rune('A'+index))
		input.Name = "Activity " + input.Code
		review := task3Now.AddDate(0, 0, index)
		input.NextReviewDate = &review
		created = append(created, task3Create(t, service, input).ID)
	}

	seen := make(map[string]bool)
	cursor := ""
	for pageNumber := 0; pageNumber < 10; pageNumber++ {
		page, err := service.ListActivities(context.Background(), ropa.ListActivitiesFilter{
			TenantID:      "tenant-1",
			LegalEntityID: "entity-1",
			Limit:         2,
			Cursor:        cursor,
		})
		if err != nil {
			t.Fatalf("list page %d: %v", pageNumber, err)
		}
		for _, row := range page.Rows {
			if seen[row.ID] {
				t.Fatalf("activity %s was repeated across pages", row.ID)
			}
			seen[row.ID] = true
		}
		if !page.HasMore {
			break
		}
		if page.NextCursor == "" {
			t.Fatal("HasMore was true without a next cursor")
		}
		cursor = page.NextCursor
	}
	if len(seen) != len(created) {
		t.Fatalf("saw %d activities across pages, want %d", len(seen), len(created))
	}
	for _, id := range created {
		if !seen[id] {
			t.Fatalf("activity %s was skipped", id)
		}
	}

	_, err := service.ListActivities(context.Background(), ropa.ListActivitiesFilter{
		TenantID:      "tenant-1",
		LegalEntityID: "entity-1",
		Limit:         2,
		Cursor:        "not-a-valid-cursor",
	})
	if !errors.Is(err, ropa.ErrInvalid) {
		t.Fatalf("invalid cursor: expected ErrInvalid, got %v", err)
	}
}

func TestRegisterSummaryMarksOldOrMismatchedSnapshotsStale(t *testing.T) {
	service, _, summaries := task3Service()
	old := task3Now.Add(-16 * time.Minute)
	if err := summaries.ReplaceSummary(context.Background(), ropa.RegisterSummary{
		TenantID:          "tenant-1",
		LegalEntityID:     "entity-1",
		GeneratedAt:       old,
		ProjectionVersion: ropa.ProjectionVersion,
	}); err != nil {
		t.Fatalf("replace old summary: %v", err)
	}
	summary, err := service.RegisterSummary(context.Background(), "tenant-1", "entity-1")
	if err != nil {
		t.Fatalf("read old summary: %v", err)
	}
	if summary.Freshness != ropa.FreshnessStale {
		t.Fatalf("old summary freshness = %s, want STALE", summary.Freshness)
	}

	if err := summaries.ReplaceSummary(context.Background(), ropa.RegisterSummary{
		TenantID:          "tenant-1",
		LegalEntityID:     "entity-1",
		GeneratedAt:       task3Now,
		ProjectionVersion: "ropa-v0",
	}); err != nil {
		t.Fatalf("replace mismatched summary: %v", err)
	}
	summary, err = service.RegisterSummary(context.Background(), "tenant-1", "entity-1")
	if err != nil {
		t.Fatalf("read mismatched summary: %v", err)
	}
	if summary.Freshness != ropa.FreshnessStale {
		t.Fatalf("mismatched summary freshness = %s, want STALE", summary.Freshness)
	}

	if err := summaries.ReplaceSummary(context.Background(), ropa.RegisterSummary{
		TenantID:          "tenant-1",
		LegalEntityID:     "entity-1",
		GeneratedAt:       task3Now,
		ProjectionVersion: ropa.ProjectionVersion,
	}); err != nil {
		t.Fatalf("replace fresh summary: %v", err)
	}
	summary, err = service.RegisterSummary(context.Background(), "tenant-1", "entity-1")
	if err != nil {
		t.Fatalf("read fresh summary: %v", err)
	}
	if summary.Freshness != ropa.FreshnessCurrent {
		t.Fatalf("fresh summary freshness = %s, want CURRENT", summary.Freshness)
	}
}

func TestEventsAccumulateWithAggregateVersionsAfterTransition(t *testing.T) {
	service, repository, _ := task3Service()
	activity := task3Create(t, service, task3Input())
	activity, err := service.TransitionActivity(context.Background(), ropa.TransitionActivityInput{
		TenantID:        activity.TenantID,
		ActivityID:      activity.ID,
		ExpectedVersion: activity.Version,
		To:              ropa.StatusOpen,
		ActorID:         "actor-1",
	})
	if err != nil {
		t.Fatalf("transition activity: %v", err)
	}
	events, err := repository.ActivityEvents(context.Background(), activity.TenantID, activity.ID)
	if err != nil {
		t.Fatalf("read events: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("event count = %d, want 2", len(events))
	}
	if events[0].Type != "processing_activity.created" || events[0].AggregateVersion != 1 {
		t.Fatalf("first event = %#v", events[0])
	}
	if events[1].Type != "processing_activity.transitioned" || events[1].AggregateVersion != 2 {
		t.Fatalf("second event = %#v", events[1])
	}
}

func stringPointer(value string) *string {
	return &value
}
