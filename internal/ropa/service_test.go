package ropa_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"strings"
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
	events, _, err := repository.ActivityEvents(context.Background(), ropa.ActivityScope{TenantID: activity.TenantID, LegalEntityID: activity.LegalEntityID}, activity.ID, 0, 100)
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
		LegalEntityID:   activity.LegalEntityID,
		ActivityID:      activity.ID,
		ExpectedVersion: activity.Version,
		To:              ropa.StatusClosed,
		ActorID:         "actor-1",
	})
	if !errors.Is(err, ropa.ErrClosureBlocked) {
		t.Fatalf("expected ErrClosureBlocked, got %v", err)
	}

	blockers, err := service.ClosureBlockers(context.Background(), ropa.ActivityScope{TenantID: activity.TenantID, LegalEntityID: activity.LegalEntityID}, activity.ID)
	if err != nil {
		t.Fatalf("read closure blockers: %v", err)
	}
	want := []string{"lawful basis", "named owner", "data subject category", "completed review"}
	if !reflect.DeepEqual(blockers, want) {
		t.Fatalf("closure blockers = %#v, want %#v", blockers, want)
	}
}

func TestClosureSucceedsWhenRequiredFactsArePresent(t *testing.T) {
	service, _, _ := task3Service()
	completed := task3Now.Add(-time.Hour)
	input := task3Input()
	input.Reviews = []ropa.Review{task3ReviewWithOutcome(&completed, "CONFIRMED")}
	activity := task3Create(t, service, input)
	closed, err := service.TransitionActivity(context.Background(), ropa.TransitionActivityInput{
		TenantID:        activity.TenantID,
		LegalEntityID:   activity.LegalEntityID,
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

func TestCompletedReviewIsRequiredForClosure(t *testing.T) {
	completed := task3Now.Add(-time.Hour)
	tests := []struct {
		name        string
		completedAt *time.Time
		outcome     string
		wantBlocked bool
	}{
		{name: "nil completion", completedAt: nil, outcome: "CONFIRMED", wantBlocked: true},
		{name: "empty outcome", completedAt: &completed, outcome: "", wantBlocked: true},
		{name: "withdrawn", completedAt: &completed, outcome: "WITHDRAWN", wantBlocked: true},
		{name: "confirmed", completedAt: &completed, outcome: "CONFIRMED", wantBlocked: false},
		{name: "revised", completedAt: &completed, outcome: "REVISED", wantBlocked: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service, _, _ := task3Service()
			input := task3Input()
			input.Reviews = []ropa.Review{task3ReviewWithOutcome(test.completedAt, test.outcome)}
			activity := task3Create(t, service, input)

			blockers, err := service.ClosureBlockers(context.Background(), ropa.ActivityScope{TenantID: activity.TenantID, LegalEntityID: activity.LegalEntityID}, activity.ID)
			if err != nil {
				t.Fatalf("read closure blockers: %v", err)
			}
			if test.wantBlocked {
				want := []string{"completed review"}
				if !reflect.DeepEqual(blockers, want) {
					t.Fatalf("closure blockers = %#v, want %#v", blockers, want)
				}
				_, err = service.TransitionActivity(context.Background(), ropa.TransitionActivityInput{
					TenantID:        activity.TenantID,
					LegalEntityID:   activity.LegalEntityID,
					ActivityID:      activity.ID,
					ExpectedVersion: activity.Version,
					To:              ropa.StatusClosed,
					ActorID:         "actor-1",
				})
				if !errors.Is(err, ropa.ErrClosureBlocked) {
					t.Fatalf("expected ErrClosureBlocked, got %v", err)
				}
				return
			}

			if len(blockers) != 0 {
				t.Fatalf("closure blockers = %#v, want none", blockers)
			}
			closed, err := service.TransitionActivity(context.Background(), ropa.TransitionActivityInput{
				TenantID:        activity.TenantID,
				LegalEntityID:   activity.LegalEntityID,
				ActivityID:      activity.ID,
				ExpectedVersion: activity.Version,
				To:              ropa.StatusClosed,
				ActorID:         "actor-1",
			})
			if err != nil {
				t.Fatalf("close activity: %v", err)
			}
			if closed.Status != ropa.StatusClosed {
				t.Fatalf("closed status = %s, want CLOSED", closed.Status)
			}
		})
	}
}

func TestClosedActivityCannotTransitionBackToOpen(t *testing.T) {
	service, _, _ := task3Service()
	completed := task3Now.Add(-time.Hour)
	input := task3Input()
	input.Reviews = []ropa.Review{task3ReviewWithOutcome(&completed, "CONFIRMED")}
	activity := task3Create(t, service, input)
	activity, err := service.TransitionActivity(context.Background(), ropa.TransitionActivityInput{
		TenantID:        activity.TenantID,
		LegalEntityID:   activity.LegalEntityID,
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
		LegalEntityID:   activity.LegalEntityID,
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
		LegalEntityID:   activity.LegalEntityID,
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
		LegalEntityID:   activity.LegalEntityID,
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
		LegalEntityID:   activity.LegalEntityID,
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
		LegalEntityID:   updated.LegalEntityID,
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
		LegalEntityID:   activity.LegalEntityID,
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

	if _, err := service.GetActivity(context.Background(), ropa.ActivityScope{TenantID: retired.TenantID, LegalEntityID: retired.LegalEntityID}, retired.ID); err != nil {
		t.Fatalf("retired activity should remain readable: %v", err)
	}
	page, err := service.ListActivities(context.Background(), ropa.ActivityScope{TenantID: retired.TenantID, LegalEntityID: retired.LegalEntityID}, ropa.ListActivitiesFilter{
		IncludeRetired: false,
		Limit:          50,
	})
	if err != nil {
		t.Fatalf("list live activities: %v", err)
	}
	if len(page.Rows) != 1 || page.Rows[0].ID != live.ID {
		t.Fatalf("live list = %#v, want only %s", page.Rows, live.ID)
	}

	page, err = service.ListActivities(context.Background(), ropa.ActivityScope{TenantID: retired.TenantID, LegalEntityID: retired.LegalEntityID}, ropa.ListActivitiesFilter{
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
		page, err := service.ListActivities(context.Background(), ropa.ActivityScope{TenantID: "tenant-1", LegalEntityID: "entity-1"}, ropa.ListActivitiesFilter{
			Limit:  2,
			Cursor: cursor,
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

	_, err := service.ListActivities(context.Background(), ropa.ActivityScope{TenantID: "tenant-1", LegalEntityID: "entity-1"}, ropa.ListActivitiesFilter{
		Limit:  2,
		Cursor: "not-a-valid-cursor",
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
		LegalEntityID:   activity.LegalEntityID,
		ActivityID:      activity.ID,
		ExpectedVersion: activity.Version,
		To:              ropa.StatusOpen,
		ActorID:         "actor-1",
	})
	if err != nil {
		t.Fatalf("transition activity: %v", err)
	}
	events, _, err := repository.ActivityEvents(context.Background(), ropa.ActivityScope{TenantID: activity.TenantID, LegalEntityID: activity.LegalEntityID}, activity.ID, 0, 100)
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

func task3DirectActivity() ropa.ProcessingActivity {
	return ropa.ProcessingActivity{
		ID:                           "activity-direct-1",
		TenantID:                     "tenant-1",
		LegalEntityID:                "entity-1",
		Code:                         "PA-DIRECT-1",
		Name:                         "Direct repository activity",
		Description:                  "A record seeded through the memory write boundary.",
		Status:                       ropa.StatusNew,
		Purpose:                      "Exercise repository validation.",
		LawfulBasis:                  "Contract",
		Controller:                   "Fidelity Bank",
		Processor:                    "Internal operations",
		DataSubjectCategories:        "Customers",
		OwnerPrincipalID:             "owner-1",
		RequiredAuthorityPrincipalID: "authority-1",
		Version:                      1,
		CreatedAt:                    task3Now,
		UpdatedAt:                    task3Now,
	}
}

func task3DirectEvent(t *testing.T, activity ropa.ProcessingActivity, eventType string, aggregateVersion int64) ropa.Event {
	t.Helper()
	payload, err := json.Marshal(activity)
	if err != nil {
		t.Fatalf("marshal direct activity: %v", err)
	}
	return ropa.Event{
		ID:               activity.ID + "-" + eventType,
		TenantID:         activity.TenantID,
		LegalEntityID:    activity.LegalEntityID,
		AggregateType:    "PROCESSING_ACTIVITY",
		AggregateID:      activity.ID,
		AggregateVersion: aggregateVersion,
		Type:             eventType,
		Payload:          payload,
		ActorType:        "SERVICE",
		OccurredAt:       activity.UpdatedAt,
	}
}

func task3DirectCreate(t *testing.T, repository *ropa.MemoryRepository, activity ropa.ProcessingActivity) (ropa.ProcessingActivity, error) {
	t.Helper()
	event := task3DirectEvent(t, activity, ropa.EventActivityCreated, activity.Version)
	return repository.CreateActivity(context.Background(), activity, event)
}

func task3ValidRecipient(name string) ropa.Recipient {
	return ropa.Recipient{
		Recipient:     name,
		RecipientKind: "EXTERNAL",
		CountryCode:   "US",
		IsCrossBorder: true,
		TransferBasis: ropa.TransferBasisStandardContractClauses,
	}
}

func task3ValidReview() ropa.Review {
	return ropa.Review{
		ID:        "review-1",
		CreatedAt: task3Now,
		DueDate:   task3Now.AddDate(0, 3, 0),
	}
}

func task3ReviewWithOutcome(completedAt *time.Time, outcome string) ropa.Review {
	review := task3ValidReview()
	review.CompletedAt = completedAt
	review.Outcome = outcome
	return review
}

func TestMemoryRepositoryRejectsNonNewStatusOnCreate(t *testing.T) {
	for _, test := range []struct {
		name   string
		status ropa.Status
	}{
		{name: "open", status: ropa.StatusOpen},
		{name: "closed", status: ropa.StatusClosed},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := ropa.NewMemoryRepository()
			activity := task3DirectActivity()
			activity.Status = test.status

			if _, err := task3DirectCreate(t, repository, activity); !errors.Is(err, ropa.ErrInvalid) {
				t.Fatalf("%s creation: expected ErrInvalid, got %v", test.name, err)
			}
		})
	}
}

func TestMemoryRepositoryRejectsNonPositiveVersionOnCreate(t *testing.T) {
	for _, test := range []struct {
		name    string
		version int64
	}{
		{name: "zero", version: 0},
		{name: "negative", version: -1},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := ropa.NewMemoryRepository()
			activity := task3DirectActivity()
			activity.Version = test.version

			if _, err := task3DirectCreate(t, repository, activity); !errors.Is(err, ropa.ErrInvalid) {
				t.Fatalf("version %d: expected ErrInvalid, got %v", test.version, err)
			}
		})
	}
}

func TestMemoryRepositoryRejectsClosureWithoutEachRequiredFact(t *testing.T) {
	completed := task3Now.Add(-time.Hour)
	cases := []struct {
		name   string
		mutate func(*ropa.ProcessingActivity)
	}{
		{name: "lawful basis", mutate: func(activity *ropa.ProcessingActivity) { activity.LawfulBasis = "" }},
		{name: "named owner", mutate: func(activity *ropa.ProcessingActivity) { activity.OwnerPrincipalID = "" }},
		{name: "data subject category", mutate: func(activity *ropa.ProcessingActivity) { activity.DataSubjectCategories = "" }},
		{name: "completed review", mutate: func(activity *ropa.ProcessingActivity) { activity.Reviews = nil }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			repository := ropa.NewMemoryRepository()
			seed := task3DirectActivity()
			seed.Reviews = []ropa.Review{task3ReviewWithOutcome(&completed, "CONFIRMED")}
			activity, err := task3DirectCreate(t, repository, seed)
			if err != nil {
				t.Fatalf("seed direct activity: %v", err)
			}
			next := activity
			next.Status = ropa.StatusClosed
			next.Version = activity.Version + 1
			next.UpdatedAt = task3Now.Add(time.Minute)
			test.mutate(&next)
			event := task3DirectEvent(t, next, ropa.EventActivityTransitioned, next.Version)

			if _, err := repository.ApplyActivityEvent(context.Background(), ropa.ActivityScope{TenantID: next.TenantID, LegalEntityID: next.LegalEntityID}, next.ID, activity.Version, event); !errors.Is(err, ropa.ErrClosureBlocked) {
				t.Fatalf("missing %s: expected ErrClosureBlocked, got %v", test.name, err)
			}
		})
	}
}

func TestMemoryRepositoryRejectsUnknownEventType(t *testing.T) {
	repository := ropa.NewMemoryRepository()
	activity, err := task3DirectCreate(t, repository, task3DirectActivity())
	if err != nil {
		t.Fatalf("seed direct activity: %v", err)
	}
	next := activity
	next.Status = ropa.StatusOpen
	next.Version = activity.Version + 1
	next.UpdatedAt = task3Now.Add(time.Minute)
	event := task3DirectEvent(t, next, "processing_activity.deleted", next.Version)

	if _, err := repository.ApplyActivityEvent(context.Background(), ropa.ActivityScope{TenantID: next.TenantID, LegalEntityID: next.LegalEntityID}, next.ID, activity.Version, event); !errors.Is(err, ropa.ErrInvalid) {
		t.Fatalf("unknown event type: expected ErrInvalid, got %v", err)
	}
}

func TestMemoryRepositoryRequiresExactEventVersion(t *testing.T) {
	for _, test := range []struct {
		name    string
		version int64
	}{
		{name: "wrong", version: 3},
		{name: "zero", version: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := ropa.NewMemoryRepository()
			activity, err := task3DirectCreate(t, repository, task3DirectActivity())
			if err != nil {
				t.Fatalf("seed direct activity: %v", err)
			}
			next := activity
			next.Status = ropa.StatusOpen
			next.Version = activity.Version + 1
			next.UpdatedAt = task3Now.Add(time.Minute)
			event := task3DirectEvent(t, next, ropa.EventActivityTransitioned, test.version)

			if _, err := repository.ApplyActivityEvent(context.Background(), ropa.ActivityScope{TenantID: next.TenantID, LegalEntityID: next.LegalEntityID}, next.ID, activity.Version, event); !errors.Is(err, ropa.ErrVersionConflict) {
				t.Fatalf("%s event version: expected ErrVersionConflict, got %v", test.name, err)
			}
		})
	}
}

func TestMemoryRepositoryRejectsMismatchedPayloadID(t *testing.T) {
	repository := ropa.NewMemoryRepository()
	activity, err := task3DirectCreate(t, repository, task3DirectActivity())
	if err != nil {
		t.Fatalf("seed direct activity: %v", err)
	}
	next := activity
	next.ID = "another-activity"
	next.Status = ropa.StatusOpen
	next.Version = activity.Version + 1
	next.UpdatedAt = task3Now.Add(time.Minute)
	event := task3DirectEvent(t, next, ropa.EventActivityTransitioned, next.Version)

	if _, err := repository.ApplyActivityEvent(context.Background(), ropa.ActivityScope{TenantID: activity.TenantID, LegalEntityID: activity.LegalEntityID}, activity.ID, activity.Version, event); !errors.Is(err, ropa.ErrInvalid) {
		t.Fatalf("mismatched payload ID: expected ErrInvalid, got %v", err)
	}
}

func TestActivityDomainValidationAgreesAcrossServiceAndRepository(t *testing.T) {
	cases := []struct {
		name           string
		mutateActivity func(*ropa.ProcessingActivity)
		mutateInput    func(*ropa.CreateActivityInput)
	}{
		{
			name:           "code exceeds bound",
			mutateActivity: func(activity *ropa.ProcessingActivity) { activity.Code = strings.Repeat("C", 201) },
			mutateInput:    func(input *ropa.CreateActivityInput) { input.Code = strings.Repeat("C", 201) },
		},
		{
			name:           "description exceeds bound",
			mutateActivity: func(activity *ropa.ProcessingActivity) { activity.Description = strings.Repeat("D", 4001) },
			mutateInput:    func(input *ropa.CreateActivityInput) { input.Description = strings.Repeat("D", 4001) },
		},
		{
			name:           "whitespace optional text",
			mutateActivity: func(activity *ropa.ProcessingActivity) { activity.Description = " \t" },
			mutateInput:    func(input *ropa.CreateActivityInput) { input.Description = " \t" },
		},
		{
			name:           "controller name exceeds bound",
			mutateActivity: func(activity *ropa.ProcessingActivity) { activity.Controller = strings.Repeat("C", 201) },
			mutateInput:    func(input *ropa.CreateActivityInput) { input.Controller = strings.Repeat("C", 201) },
		},
		{
			name: "recipient name exceeds bound",
			mutateActivity: func(activity *ropa.ProcessingActivity) {
				activity.Recipients = []ropa.Recipient{task3ValidRecipient(strings.Repeat("R", 201))}
			},
			mutateInput: func(input *ropa.CreateActivityInput) {
				input.Recipients = []ropa.Recipient{task3ValidRecipient(strings.Repeat("R", 201))}
			},
		},
		{
			name: "blank child key",
			mutateActivity: func(activity *ropa.ProcessingActivity) {
				activity.Recipients = []ropa.Recipient{{Recipient: " \t", RecipientKind: "EXTERNAL", CountryCode: "US", IsCrossBorder: true, TransferBasis: ropa.TransferBasisStandardContractClauses}}
			},
			mutateInput: func(input *ropa.CreateActivityInput) {
				input.Recipients = []ropa.Recipient{{Recipient: " \t", RecipientKind: "EXTERNAL", CountryCode: "US", IsCrossBorder: true, TransferBasis: ropa.TransferBasisStandardContractClauses}}
			},
		},
		{
			name: "invalid recipient kind",
			mutateActivity: func(activity *ropa.ProcessingActivity) {
				activity.Recipients = []ropa.Recipient{{Recipient: "Processor", RecipientKind: "UNKNOWN", CountryCode: "US", IsCrossBorder: true, TransferBasis: ropa.TransferBasisStandardContractClauses}}
			},
			mutateInput: func(input *ropa.CreateActivityInput) {
				input.Recipients = []ropa.Recipient{{Recipient: "Processor", RecipientKind: "UNKNOWN", CountryCode: "US", IsCrossBorder: true, TransferBasis: ropa.TransferBasisStandardContractClauses}}
			},
		},
		{
			name: "invalid sensitivity",
			mutateActivity: func(activity *ropa.ProcessingActivity) {
				activity.DataCategories = []ropa.DataCategory{{Category: "Identity", Sensitivity: "UNKNOWN"}}
			},
			mutateInput: func(input *ropa.CreateActivityInput) {
				input.DataCategories = []ropa.DataCategory{{Category: "Identity", Sensitivity: "UNKNOWN"}}
			},
		},
		{
			name: "invalid system kind",
			mutateActivity: func(activity *ropa.ProcessingActivity) {
				activity.Systems = []ropa.System{{SystemName: "Core", SystemKind: "UNKNOWN"}}
			},
			mutateInput: func(input *ropa.CreateActivityInput) {
				input.Systems = []ropa.System{{SystemName: "Core", SystemKind: "UNKNOWN"}}
			},
		},
		{
			name: "invalid transfer basis",
			mutateActivity: func(activity *ropa.ProcessingActivity) {
				activity.Recipients = []ropa.Recipient{{Recipient: "Processor", RecipientKind: "EXTERNAL", CountryCode: "US", IsCrossBorder: true, TransferBasis: ropa.TransferBasis("UNKNOWN")}}
			},
			mutateInput: func(input *ropa.CreateActivityInput) {
				input.Recipients = []ropa.Recipient{{Recipient: "Processor", RecipientKind: "EXTERNAL", CountryCode: "US", IsCrossBorder: true, TransferBasis: ropa.TransferBasis("UNKNOWN")}}
			},
		},
		{
			name: "cross-border without country",
			mutateActivity: func(activity *ropa.ProcessingActivity) {
				activity.Recipients = []ropa.Recipient{{Recipient: "Processor", RecipientKind: "EXTERNAL", IsCrossBorder: true, TransferBasis: ropa.TransferBasisStandardContractClauses}}
			},
			mutateInput: func(input *ropa.CreateActivityInput) {
				input.Recipients = []ropa.Recipient{{Recipient: "Processor", RecipientKind: "EXTERNAL", IsCrossBorder: true, TransferBasis: ropa.TransferBasisStandardContractClauses}}
			},
		},
		{
			name: "domestic recipient with country",
			mutateActivity: func(activity *ropa.ProcessingActivity) {
				activity.Recipients = []ropa.Recipient{{Recipient: "Internal team", RecipientKind: "INTERNAL", CountryCode: "US", TransferBasis: ropa.TransferBasisNotApplicable}}
			},
			mutateInput: func(input *ropa.CreateActivityInput) {
				input.Recipients = []ropa.Recipient{{Recipient: "Internal team", RecipientKind: "INTERNAL", CountryCode: "US", TransferBasis: ropa.TransferBasisNotApplicable}}
			},
		},
		{
			name: "domestic recipient with transfer basis",
			mutateActivity: func(activity *ropa.ProcessingActivity) {
				activity.Recipients = []ropa.Recipient{{Recipient: "Internal team", RecipientKind: "INTERNAL", TransferBasis: ropa.TransferBasisStandardContractClauses}}
			},
			mutateInput: func(input *ropa.CreateActivityInput) {
				input.Recipients = []ropa.Recipient{{Recipient: "Internal team", RecipientKind: "INTERNAL", TransferBasis: ropa.TransferBasisStandardContractClauses}}
			},
		},
		{
			name: "lowercase country code",
			mutateActivity: func(activity *ropa.ProcessingActivity) {
				activity.Recipients = []ropa.Recipient{{Recipient: "Processor", RecipientKind: "EXTERNAL", CountryCode: "us", IsCrossBorder: true, TransferBasis: ropa.TransferBasisStandardContractClauses}}
			},
			mutateInput: func(input *ropa.CreateActivityInput) {
				input.Recipients = []ropa.Recipient{{Recipient: "Processor", RecipientKind: "EXTERNAL", CountryCode: "us", IsCrossBorder: true, TransferBasis: ropa.TransferBasisStandardContractClauses}}
			},
		},
		{
			name: "duplicate data category",
			mutateActivity: func(activity *ropa.ProcessingActivity) {
				activity.DataCategories = []ropa.DataCategory{{Category: "Identity", Sensitivity: "DIRECT_PERSONAL"}, {Category: " Identity ", Sensitivity: "DIRECT_PERSONAL"}}
			},
			mutateInput: func(input *ropa.CreateActivityInput) {
				input.DataCategories = []ropa.DataCategory{{Category: "Identity", Sensitivity: "DIRECT_PERSONAL"}, {Category: " Identity ", Sensitivity: "DIRECT_PERSONAL"}}
			},
		},
		{
			name: "duplicate recipient",
			mutateActivity: func(activity *ropa.ProcessingActivity) {
				activity.Recipients = []ropa.Recipient{task3ValidRecipient("Processor"), task3ValidRecipient(" Processor ")}
			},
			mutateInput: func(input *ropa.CreateActivityInput) {
				input.Recipients = []ropa.Recipient{task3ValidRecipient("Processor"), task3ValidRecipient(" Processor ")}
			},
		},
		{
			name: "duplicate system",
			mutateActivity: func(activity *ropa.ProcessingActivity) {
				activity.Systems = []ropa.System{{SystemName: "Core", SystemKind: "APPLICATION"}, {SystemName: " Core ", SystemKind: "APPLICATION"}}
			},
			mutateInput: func(input *ropa.CreateActivityInput) {
				input.Systems = []ropa.System{{SystemName: "Core", SystemKind: "APPLICATION"}, {SystemName: " Core ", SystemKind: "APPLICATION"}}
			},
		},
		{
			name: "review without id",
			mutateActivity: func(activity *ropa.ProcessingActivity) {
				activity.Reviews = []ropa.Review{task3ValidReview()}
				activity.Reviews[0].ID = " \t"
			},
			mutateInput: func(input *ropa.CreateActivityInput) {
				input.Reviews = []ropa.Review{task3ValidReview()}
				input.Reviews[0].ID = " \t"
			},
		},
		{
			name: "recipient collection exceeds bound",
			mutateActivity: func(activity *ropa.ProcessingActivity) {
				activity.Recipients = make([]ropa.Recipient, 1001)
				for index := range activity.Recipients {
					activity.Recipients[index] = task3ValidRecipient("Recipient-" + string(rune(index+1000)))
				}
			},
			mutateInput: func(input *ropa.CreateActivityInput) {
				input.Recipients = make([]ropa.Recipient, 1001)
				for index := range input.Recipients {
					input.Recipients[index] = task3ValidRecipient("Recipient-" + string(rune(index+1000)))
				}
			},
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			service, _, _ := task3Service()
			input := task3Input()
			test.mutateInput(&input)
			if _, err := service.CreateActivity(context.Background(), input); !errors.Is(err, ropa.ErrInvalid) {
				t.Fatalf("service path: expected ErrInvalid, got %v", err)
			}

			repository := ropa.NewMemoryRepository()
			activity := task3DirectActivity()
			test.mutateActivity(&activity)
			if _, err := task3DirectCreate(t, repository, activity); !errors.Is(err, ropa.ErrInvalid) {
				t.Fatalf("repository path: expected ErrInvalid, got %v", err)
			}
		})
	}
}

func TestServiceRejectsUpdateTimestampBeforeCreation(t *testing.T) {
	service, _, _ := task3Service()
	activity := task3Create(t, service, task3Input())
	service.Now = func() time.Time { return task3Now.Add(-time.Hour) }
	description := "Updated after the clock moved backwards"

	_, err := service.UpdateActivity(context.Background(), ropa.UpdateActivityInput{
		TenantID:        activity.TenantID,
		LegalEntityID:   activity.LegalEntityID,
		ActivityID:      activity.ID,
		ExpectedVersion: activity.Version,
		Description:     &description,
		ActorID:         "actor-1",
	})
	if !errors.Is(err, ropa.ErrInvalid) {
		t.Fatalf("update before creation: expected ErrInvalid, got %v", err)
	}
}

func TestMemoryRepositoryRejectsTimestampBeforeCreation(t *testing.T) {
	repository := ropa.NewMemoryRepository()
	activity := task3DirectActivity()
	activity.UpdatedAt = task3Now.Add(-time.Hour)

	if _, err := task3DirectCreate(t, repository, activity); !errors.Is(err, ropa.ErrInvalid) {
		t.Fatalf("create timestamp order: expected ErrInvalid, got %v", err)
	}
}

func TestListActivitiesRejectsUnknownStatusThroughServiceAndRepository(t *testing.T) {
	service, repository, _ := task3Service()
	activity := task3Create(t, service, task3Input())
	filter := ropa.ListActivitiesFilter{
		Status: ropa.Status("ARCHIVED"),
		Limit:  1,
	}
	scope := ropa.ActivityScope{TenantID: activity.TenantID, LegalEntityID: activity.LegalEntityID}
	if _, err := service.ListActivities(context.Background(), scope, filter); !errors.Is(err, ropa.ErrInvalid) {
		t.Fatalf("service list: expected ErrInvalid, got %v", err)
	}
	if _, err := repository.ListActivities(context.Background(), scope, filter); !errors.Is(err, ropa.ErrInvalid) {
		t.Fatalf("repository list: expected ErrInvalid, got %v", err)
	}
}

func TestListActivitiesRejectsWhitespaceCursorID(t *testing.T) {
	service, _, _ := task3Service()
	cursor := base64.RawURLEncoding.EncodeToString([]byte(`{"s":"NEW","r":"0001-01-01T00:00:00Z","i":"   "}`))
	_, err := service.ListActivities(context.Background(), ropa.ActivityScope{TenantID: "tenant-1", LegalEntityID: "entity-1"}, ropa.ListActivitiesFilter{
		Limit:  1,
		Cursor: cursor,
	})
	if !errors.Is(err, ropa.ErrInvalid) {
		t.Fatalf("whitespace cursor ID: expected ErrInvalid, got %v", err)
	}
}

func TestRegisterSummaryMarksZeroGeneratedAtStale(t *testing.T) {
	service, _, summaries := task3Service()
	if err := summaries.ReplaceSummary(context.Background(), ropa.RegisterSummary{
		TenantID:          "tenant-1",
		LegalEntityID:     "entity-1",
		ProjectionVersion: ropa.ProjectionVersion,
	}); err != nil {
		t.Fatalf("replace zero-time summary: %v", err)
	}
	summary, err := service.RegisterSummary(context.Background(), "tenant-1", "entity-1")
	if err != nil {
		t.Fatalf("read zero-time summary: %v", err)
	}
	if summary.Freshness != ropa.FreshnessStale {
		t.Fatalf("zero-time summary freshness = %s, want STALE", summary.Freshness)
	}
}

func TestServiceFallsBackToServiceActorWhenActorIsAbsent(t *testing.T) {
	service, repository, _ := task3Service()
	input := task3Input()
	input.ActorID = ""
	activity := task3Create(t, service, input)
	events, _, err := repository.ActivityEvents(context.Background(), ropa.ActivityScope{TenantID: activity.TenantID, LegalEntityID: activity.LegalEntityID}, activity.ID, 0, 100)
	if err != nil {
		t.Fatalf("read created event: %v", err)
	}
	if len(events) != 1 || events[0].ActorType != "SERVICE" || events[0].ActorID != "" {
		t.Fatalf("service actor fallback event = %#v", events)
	}
}

func TestKeysetPaginationUsesIDAsStableTiebreaker(t *testing.T) {
	service, _, _ := task3Service()
	reviewDate := task3Now.AddDate(0, 2, 0)
	created := make([]string, 0, 3)
	for index := 0; index < 3; index++ {
		input := task3Input()
		input.Code = "PA-TIE-" + string(rune('A'+index))
		input.Name = "Tie activity " + input.Code
		input.NextReviewDate = &reviewDate
		created = append(created, task3Create(t, service, input).ID)
	}
	sort.Strings(created)

	first, err := service.ListActivities(context.Background(), ropa.ActivityScope{TenantID: "tenant-1", LegalEntityID: "entity-1"}, ropa.ListActivitiesFilter{Limit: 1})
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	firstAgain, err := service.ListActivities(context.Background(), ropa.ActivityScope{TenantID: "tenant-1", LegalEntityID: "entity-1"}, ropa.ListActivitiesFilter{Limit: 1})
	if err != nil {
		t.Fatalf("first page repeated: %v", err)
	}
	if len(first.Rows) != 1 || len(firstAgain.Rows) != 1 || first.Rows[0].ID != firstAgain.Rows[0].ID || first.Rows[0].ID != created[0] {
		t.Fatalf("first-page order is not stable: first=%#v repeated=%#v expected=%v", first.Rows, firstAgain.Rows, created)
	}

	seen := make([]string, 0, len(created))
	cursor := ""
	for pageNumber := 0; pageNumber < len(created)+1; pageNumber++ {
		page, err := service.ListActivities(context.Background(), ropa.ActivityScope{TenantID: "tenant-1", LegalEntityID: "entity-1"}, ropa.ListActivitiesFilter{Limit: 1, Cursor: cursor})
		if err != nil {
			t.Fatalf("page %d: %v", pageNumber, err)
		}
		if len(page.Rows) == 0 {
			break
		}
		seen = append(seen, page.Rows[0].ID)
		if !page.HasMore {
			break
		}
		cursor = page.NextCursor
	}
	if !reflect.DeepEqual(seen, created) {
		t.Fatalf("paged order = %v, want stable ID order %v", seen, created)
	}
}

func TestReturnedActivityCannotMutateStoredState(t *testing.T) {
	service, _, _ := task3Service()
	input := task3Input()
	completed := task3Now.Add(-time.Hour)
	input.DataCategories = []ropa.DataCategory{{Category: "Identity", Sensitivity: "DIRECT_PERSONAL"}}
	input.Recipients = []ropa.Recipient{task3ValidRecipient("Processor")}
	input.Systems = []ropa.System{{SystemName: "Core", SystemKind: "APPLICATION"}}
	input.Reviews = []ropa.Review{{ID: "review-1", CreatedAt: task3Now, DueDate: task3Now.AddDate(0, 3, 0), CompletedAt: &completed, Outcome: "CONFIRMED"}}
	activity := task3Create(t, service, input)

	returned, err := service.GetActivity(context.Background(), ropa.ActivityScope{TenantID: activity.TenantID, LegalEntityID: activity.LegalEntityID}, activity.ID)
	if err != nil {
		t.Fatalf("get activity: %v", err)
	}
	returned.Name = "Caller mutation"
	returned.DataCategories[0].Category = "Caller category"
	returned.Recipients[0].Recipient = "Caller recipient"
	returned.Systems[0].SystemName = "Caller system"
	returned.Reviews[0].ID = "Caller review"
	*returned.Reviews[0].CompletedAt = task3Now.Add(-48 * time.Hour)

	stored, err := service.GetActivity(context.Background(), ropa.ActivityScope{TenantID: activity.TenantID, LegalEntityID: activity.LegalEntityID}, activity.ID)
	if err != nil {
		t.Fatalf("get activity after mutation: %v", err)
	}
	if !reflect.DeepEqual(stored, activity) {
		t.Fatalf("stored activity was changed by caller:\nwant: %#v\ngot:  %#v", activity, stored)
	}
}

type closureBlockersRepository struct {
	activity ropa.ProcessingActivity
	err      error
}

func (r *closureBlockersRepository) CreateActivity(context.Context, ropa.ProcessingActivity, ropa.Event) (ropa.ProcessingActivity, error) {
	return ropa.ProcessingActivity{}, nil
}

func (r *closureBlockersRepository) GetActivity(context.Context, ropa.ActivityScope, string) (ropa.ProcessingActivity, error) {
	if r.err != nil {
		return ropa.ProcessingActivity{}, r.err
	}
	return r.activity, nil
}

func (r *closureBlockersRepository) ApplyActivityEvent(context.Context, ropa.ActivityScope, string, int64, ropa.Event) (int64, error) {
	return 0, nil
}

func (r *closureBlockersRepository) ActivityEvents(context.Context, ropa.ActivityScope, string, int64, int) ([]ropa.Event, bool, error) {
	return nil, false, nil
}

func (r *closureBlockersRepository) ActivityByCode(context.Context, ropa.ActivityScope, string) (ropa.ProcessingActivity, error) {
	return ropa.ProcessingActivity{}, ropa.ErrNotFound
}

func TestClosureBlockersRequiresTenantAndLegalEntityScope(t *testing.T) {
	service, _, _ := task3Service()
	completed := task3Now.Add(-time.Hour)
	input := task3Input()
	input.Reviews = []ropa.Review{task3ReviewWithOutcome(&completed, "CONFIRMED")}
	activity := task3Create(t, service, input)

	blockers, err := service.ClosureBlockers(context.Background(), ropa.ActivityScope{TenantID: activity.TenantID, LegalEntityID: activity.LegalEntityID}, activity.ID)
	if err != nil {
		t.Fatalf("read scoped blockers: %v", err)
	}
	if len(blockers) != 0 {
		t.Fatalf("complete activity blockers = %v, want none", blockers)
	}

	_, err = service.ClosureBlockers(context.Background(), ropa.ActivityScope{TenantID: "other-tenant", LegalEntityID: activity.LegalEntityID}, activity.ID)
	if !errors.Is(err, ropa.ErrNotFound) {
		t.Fatalf("wrong tenant: expected ErrNotFound, got %v", err)
	}
	_, err = service.ClosureBlockers(context.Background(), ropa.ActivityScope{TenantID: activity.TenantID, LegalEntityID: "other-entity"}, activity.ID)
	if !errors.Is(err, ropa.ErrNotFound) {
		t.Fatalf("wrong legal entity: expected ErrNotFound, got %v", err)
	}
	_, err = service.ClosureBlockers(context.Background(), ropa.ActivityScope{TenantID: "", LegalEntityID: activity.LegalEntityID}, activity.ID)
	if !errors.Is(err, ropa.ErrInvalid) {
		t.Fatalf("missing tenant: expected ErrInvalid, got %v", err)
	}
}

func TestClosureBlockersPropagatesRepositoryFailure(t *testing.T) {
	failure := errors.New("database unavailable")
	service := ropa.NewService(&closureBlockersRepository{err: failure}, nil)

	_, err := service.ClosureBlockers(context.Background(), ropa.ActivityScope{TenantID: "tenant-1", LegalEntityID: "entity-1"}, "activity-1")
	if err != failure {
		t.Fatalf("repository failure = %v, want unchanged %v", err, failure)
	}
}

func TestClosureBlockersRejectsMismatchedReturnedLegalEntity(t *testing.T) {
	service := ropa.NewService(&closureBlockersRepository{activity: ropa.ProcessingActivity{
		ID:            "activity-1",
		TenantID:      "tenant-1",
		LegalEntityID: "entity-other",
	}}, nil)

	_, err := service.ClosureBlockers(context.Background(), ropa.ActivityScope{TenantID: "tenant-1", LegalEntityID: "entity-1"}, "activity-1")
	if !errors.Is(err, ropa.ErrNotFound) {
		t.Fatalf("mismatched returned legal entity: expected ErrNotFound, got %v", err)
	}
}

func TestMemoryRepositoryRejectsNoOpTransitionEvent(t *testing.T) {
	repository := ropa.NewMemoryRepository()
	activity, err := task3DirectCreate(t, repository, task3DirectActivity())
	if err != nil {
		t.Fatalf("seed direct activity: %v", err)
	}
	next := activity
	next.Version = activity.Version + 1
	next.UpdatedAt = task3Now.Add(time.Minute)
	event := task3DirectEvent(t, next, ropa.EventActivityTransitioned, next.Version)

	if _, err := repository.ApplyActivityEvent(context.Background(), ropa.ActivityScope{TenantID: next.TenantID, LegalEntityID: next.LegalEntityID}, next.ID, activity.Version, event); !errors.Is(err, ropa.ErrInvalid) {
		t.Fatalf("no-op transition event: expected ErrInvalid, got %v", err)
	}
}

func stringPointer(value string) *string {
	return &value
}
