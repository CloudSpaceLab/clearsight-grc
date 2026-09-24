package ropa_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/ropa"
)

func TestPostgresCreateRejectsANonOneVersionAtTheCommandBoundary(t *testing.T) {
	source := normalizeSQL(readPostgresSource(t, "postgres.go"))
	if !strings.Contains(source, "activity.status != statusnew || activity.version != 1") {
		t.Error("PostgreSQL create path must require status NEW and version 1")
	}
}

func TestMemoryCreateRejectsAVersionOtherThanOne(t *testing.T) {
	repository := ropa.NewMemoryRepository()
	activity := task3DirectActivity()
	activity.Version = 5
	event := task3DirectEvent(t, activity, ropa.EventActivityCreated, activity.Version)

	if _, err := repository.CreateActivity(context.Background(), activity, event); !errors.Is(err, ropa.ErrInvalid) {
		t.Fatalf("create at version 5: expected ErrInvalid, got %v", err)
	}
}

func TestMemoryCreateRejectsAnUpdateEventType(t *testing.T) {
	repository := ropa.NewMemoryRepository()
	activity := task3DirectActivity()
	event := task3DirectEvent(t, activity, ropa.EventActivityUpdated, activity.Version)

	if _, err := repository.CreateActivity(context.Background(), activity, event); !errors.Is(err, ropa.ErrInvalid) {
		t.Fatalf("create with updated event: expected ErrInvalid, got %v", err)
	}
}

func TestMemoryCreateRejectsPayloadIdentityDifferentFromEventRow(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*ropa.ProcessingActivity)
	}{
		{name: "id", mutate: func(activity *ropa.ProcessingActivity) { activity.ID = "other-id" }},
		{name: "tenant", mutate: func(activity *ropa.ProcessingActivity) { activity.TenantID = "other-tenant" }},
		{name: "legal entity", mutate: func(activity *ropa.ProcessingActivity) { activity.LegalEntityID = "other-entity" }},
		{name: "version", mutate: func(activity *ropa.ProcessingActivity) { activity.Version = activity.Version + 1 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := ropa.NewMemoryRepository()
			activity := task3DirectActivity()
			event := task3DirectEvent(t, activity, ropa.EventActivityCreated, activity.Version)
			payloadActivity := activity
			test.mutate(&payloadActivity)
			payload, err := json.Marshal(payloadActivity)
			if err != nil {
				t.Fatalf("marshal payload: %v", err)
			}
			event.Payload = payload
			if _, err := repository.CreateActivity(context.Background(), activity, event); !errors.Is(err, ropa.ErrInvalid) {
				t.Fatalf("mismatched payload %s: expected ErrInvalid, got %v", test.name, err)
			}
		})
	}
}

func TestMemoryApplyRejectsPayloadVersionDifferentFromEventVersion(t *testing.T) {
	repository := ropa.NewMemoryRepository()
	activity, err := task3DirectCreate(t, repository, task3DirectActivity())
	if err != nil {
		t.Fatalf("seed activity: %v", err)
	}
	next := activity
	next.Status = ropa.StatusOpen
	next.Version = activity.Version + 1
	next.UpdatedAt = task3Now.Add(time.Minute)
	payloadActivity := next
	payloadActivity.Version = activity.Version
	payload, err := json.Marshal(payloadActivity)
	if err != nil {
		t.Fatalf("marshal mismatched payload: %v", err)
	}
	event := task3DirectEvent(t, next, ropa.EventActivityTransitioned, next.Version)
	event.Payload = payload

	if _, err := repository.ApplyActivityEvent(context.Background(), ropa.ActivityScope{TenantID: activity.TenantID, LegalEntityID: activity.LegalEntityID}, activity.ID, activity.Version, event); !errors.Is(err, ropa.ErrInvalid) {
		t.Fatalf("mismatched payload version: expected ErrInvalid, got %v", err)
	}
}

func TestMemoryRejectsServiceActorWithActorID(t *testing.T) {
	for _, actorID := range []string{"not-a-service-actor", " \t"} {
		t.Run(strings.TrimSpace(actorID), func(t *testing.T) {
			repository := ropa.NewMemoryRepository()
			activity := task3DirectActivity()
			event := task3DirectEvent(t, activity, ropa.EventActivityCreated, activity.Version)
			event.ActorType = "SERVICE"
			event.ActorID = actorID

			if _, err := repository.CreateActivity(context.Background(), activity, event); !errors.Is(err, ropa.ErrInvalid) {
				t.Fatalf("service actor with actor ID %q: expected ErrInvalid, got %v", actorID, err)
			}
		})
	}
}

func TestCrossBorderRecipientCannotUseNotApplicableTransferBasis(t *testing.T) {
	input := task3Input()
	input.Recipients = []ropa.Recipient{{
		Recipient:     "External processor",
		RecipientKind: "EXTERNAL",
		CountryCode:   "GB",
		IsCrossBorder: true,
		TransferBasis: ropa.TransferBasisNotApplicable,
	}}
	service, _, _ := task3Service()
	if _, err := service.CreateActivity(context.Background(), input); !errors.Is(err, ropa.ErrInvalid) {
		t.Fatalf("service cross-border safeguard: expected ErrInvalid, got %v", err)
	}

	repository := ropa.NewMemoryRepository()
	activity := task3DirectActivity()
	activity.Recipients = input.Recipients
	event := task3DirectEvent(t, activity, ropa.EventActivityCreated, activity.Version)
	if _, err := repository.CreateActivity(context.Background(), activity, event); !errors.Is(err, ropa.ErrInvalid) {
		t.Fatalf("repository cross-border safeguard: expected ErrInvalid, got %v", err)
	}
}

func TestChildPersistenceDoesNotFlattenMultipleRowsIntoOneStatement(t *testing.T) {
	source := normalizeSQL(readPostgresSource(t, "postgres.go"))
	for _, fragment := range []string{
		"inserts[0].values = append(inserts[0].values,",
		"recipients.values = append(recipients.values,",
		"systems.values = append(systems.values,",
		"reviews.values = append(reviews.values,",
	} {
		if strings.Contains(source, fragment) {
			t.Errorf("child insert path still builds one flat argument list with %q", fragment)
		}
	}
}

func TestExactPostgresReadsAndHistoryRequireLegalEntityAndBound(t *testing.T) {
	source := normalizeSQL(readPostgresSource(t, "current_postgres.go"))
	for _, fragment := range []string{
		"where tenant_id = $1::uuid and legal_entity_id = $2::uuid and id = $3::uuid",
		"where tenant_id = $1::uuid and legal_entity_id = $2::uuid and id = $3::uuid for update",
		"where tenant_id = $1::uuid and legal_entity_id = $2::uuid and aggregate_type = 'processing_activity' and aggregate_id = $3::uuid",
		"where tenant_id = $22::uuid and legal_entity_id = $23::uuid and id = $24::uuid",
		"limit $5",
	} {
		if !strings.Contains(source, fragment) {
			t.Errorf("exact/history SQL must contain %q", fragment)
		}
	}
}

func TestRopaMigrationRejectsCrossBorderWithoutSafeguardAndScopesProgram(t *testing.T) {
	body := normalizeSQL(readMigration(t, upFile))
	for _, fragment := range []string{
		"check ((is_cross_border and country_code is not null and transfer_basis <> 'not_applicable') or",
		"foreign key (program_id, tenant_id, legal_entity_id) references programs(id, tenant_id, legal_entity_id)",
	} {
		if !strings.Contains(body, fragment) {
			t.Errorf("migration must contain %q", fragment)
		}
	}
}

func TestEventHistoryRejectsServiceActorWithAnActorIDInSQL(t *testing.T) {
	body := normalizeSQL(readMigration(t, upFile))
	if !strings.Contains(body, "new.actor_type = 'service'") ||
		!strings.Contains(body, "new.actor_id is not null") {
		t.Error("event actor trigger must reject a non-null actor ID for SERVICE events")
	}
}

func TestMemoryExactActivityPathsRejectWrongLegalEntity(t *testing.T) {
	service, repository, _ := task3Service()
	activity := task3Create(t, service, task3Input())
	correctScope := ropa.ActivityScope{TenantID: activity.TenantID, LegalEntityID: activity.LegalEntityID}
	wrongScope := ropa.ActivityScope{TenantID: activity.TenantID, LegalEntityID: "entity-other"}

	if _, err := repository.GetActivity(context.Background(), wrongScope, activity.ID); !errors.Is(err, ropa.ErrNotFound) {
		t.Fatalf("repository exact read with wrong entity: expected ErrNotFound, got %v", err)
	}
	if _, err := service.GetActivity(context.Background(), wrongScope, activity.ID); !errors.Is(err, ropa.ErrNotFound) {
		t.Fatalf("service exact read with wrong entity: expected ErrNotFound, got %v", err)
	}
	if _, err := repository.ActivityByCode(context.Background(), wrongScope, activity.Code); !errors.Is(err, ropa.ErrNotFound) {
		t.Fatalf("exact code read with wrong entity: expected ErrNotFound, got %v", err)
	}
	if _, _, err := repository.ActivityEvents(context.Background(), wrongScope, activity.ID, 0, 10); !errors.Is(err, ropa.ErrNotFound) {
		t.Fatalf("history with wrong entity: expected ErrNotFound, got %v", err)
	}
	if _, _, err := service.ActivityEvents(context.Background(), wrongScope, activity.ID, 0, 10); !errors.Is(err, ropa.ErrNotFound) {
		t.Fatalf("service history with wrong entity: expected ErrNotFound, got %v", err)
	}
	description := "wrong entity update"
	if _, err := service.UpdateActivity(context.Background(), ropa.UpdateActivityInput{
		TenantID:        wrongScope.TenantID,
		LegalEntityID:   wrongScope.LegalEntityID,
		ActivityID:      activity.ID,
		ExpectedVersion: activity.Version,
		Description:     &description,
	}); !errors.Is(err, ropa.ErrNotFound) {
		t.Fatalf("update with wrong entity: expected ErrNotFound, got %v", err)
	}
	if _, err := service.TransitionActivity(context.Background(), ropa.TransitionActivityInput{
		TenantID:        wrongScope.TenantID,
		LegalEntityID:   wrongScope.LegalEntityID,
		ActivityID:      activity.ID,
		ExpectedVersion: activity.Version,
		To:              ropa.StatusOpen,
	}); !errors.Is(err, ropa.ErrNotFound) {
		t.Fatalf("transition with wrong entity: expected ErrNotFound, got %v", err)
	}

	next := activity
	next.Status = ropa.StatusOpen
	next.Version++
	next.UpdatedAt = task3Now.Add(time.Minute)
	event := task3DirectEvent(t, next, ropa.EventActivityTransitioned, next.Version)
	if _, err := repository.ApplyActivityEvent(context.Background(), wrongScope, activity.ID, activity.Version, event); !errors.Is(err, ropa.ErrNotFound) {
		t.Fatalf("repository event write with wrong entity: expected ErrNotFound, got %v", err)
	}
	if _, err := service.GetActivity(context.Background(), correctScope, activity.ID); err != nil {
		t.Fatalf("correctly scoped read was damaged: %v", err)
	}
}

func TestUpdateAndTransitionRequireLegalEntityScope(t *testing.T) {
	service, _, _ := task3Service()
	activity := task3Create(t, service, task3Input())
	description := "no entity"
	if _, err := service.UpdateActivity(context.Background(), ropa.UpdateActivityInput{
		TenantID:        activity.TenantID,
		ActivityID:      activity.ID,
		ExpectedVersion: activity.Version,
		Description:     &description,
	}); !errors.Is(err, ropa.ErrInvalid) {
		t.Fatalf("update without legal entity: expected ErrInvalid, got %v", err)
	}
	if _, err := service.TransitionActivity(context.Background(), ropa.TransitionActivityInput{
		TenantID:        activity.TenantID,
		ActivityID:      activity.ID,
		ExpectedVersion: activity.Version,
		To:              ropa.StatusOpen,
	}); !errors.Is(err, ropa.ErrInvalid) {
		t.Fatalf("transition without legal entity: expected ErrInvalid, got %v", err)
	}
}

func TestMemoryRepositoryPersistsMultipleRowsInEveryChildCollection(t *testing.T) {
	service, _, _ := task3Service()
	input := task3Input()
	input.DataCategories = []ropa.DataCategory{
		{Category: "Identity", Sensitivity: "DIRECT_PERSONAL"},
		{Category: "Contact", Sensitivity: "INDIRECT_PERSONAL"},
	}
	input.Recipients = []ropa.Recipient{
		task3ValidRecipient("Processor A"),
		task3ValidRecipient("Processor B"),
	}
	input.Systems = []ropa.System{
		{SystemName: "Core banking", SystemKind: "APPLICATION"},
		{SystemName: "Fraud warehouse", SystemKind: "DATABASE"},
	}
	input.Reviews = []ropa.Review{
		{ID: "review-a", CreatedAt: task3Now, DueDate: task3Now.AddDate(0, 1, 0)},
		{ID: "review-b", CreatedAt: task3Now, DueDate: task3Now.AddDate(0, 2, 0)},
	}
	created := task3Create(t, service, input)
	got, err := service.GetActivity(context.Background(), ropa.ActivityScope{TenantID: created.TenantID, LegalEntityID: created.LegalEntityID}, created.ID)
	if err != nil {
		t.Fatalf("read activity with multiple children: %v", err)
	}
	if len(got.DataCategories) != 2 || len(got.Recipients) != 2 || len(got.Systems) != 2 || len(got.Reviews) != 2 {
		t.Fatalf("child counts = categories %d, recipients %d, systems %d, reviews %d; want 2 each", len(got.DataCategories), len(got.Recipients), len(got.Systems), len(got.Reviews))
	}
}

func TestReplayOfPagedHistoryRebuildsCurrentActivityWithContiguousVersions(t *testing.T) {
	service, _, _ := task3Service()
	completed := task3Now.Add(-time.Hour)
	input := task3Input()
	input.DataCategories = []ropa.DataCategory{
		{Category: "Identity", Sensitivity: "DIRECT_PERSONAL"},
		{Category: "Contact", Sensitivity: "INDIRECT_PERSONAL"},
	}
	input.Recipients = []ropa.Recipient{
		task3ValidRecipient("Processor A"),
		task3ValidRecipient("Processor B"),
	}
	input.Systems = []ropa.System{
		{SystemName: "Core banking", SystemKind: "APPLICATION"},
		{SystemName: "Fraud warehouse", SystemKind: "DATABASE"},
	}
	input.Reviews = []ropa.Review{task3ReviewWithOutcome(&completed, "CONFIRMED")}
	activity := task3Create(t, service, input)
	scope := ropa.ActivityScope{TenantID: activity.TenantID, LegalEntityID: activity.LegalEntityID}

	description := "Replayed description"
	activity, err := service.UpdateActivity(context.Background(), ropa.UpdateActivityInput{
		TenantID:        scope.TenantID,
		LegalEntityID:   scope.LegalEntityID,
		ActivityID:      activity.ID,
		ExpectedVersion: activity.Version,
		Description:     &description,
	})
	if err != nil {
		t.Fatalf("update before replay: %v", err)
	}
	activity, err = service.TransitionActivity(context.Background(), ropa.TransitionActivityInput{
		TenantID:        scope.TenantID,
		LegalEntityID:   scope.LegalEntityID,
		ActivityID:      activity.ID,
		ExpectedVersion: activity.Version,
		To:              ropa.StatusOpen,
	})
	if err != nil {
		t.Fatalf("first transition before replay: %v", err)
	}
	activity, err = service.UpdateActivity(context.Background(), ropa.UpdateActivityInput{
		TenantID:        scope.TenantID,
		LegalEntityID:   scope.LegalEntityID,
		ActivityID:      activity.ID,
		ExpectedVersion: activity.Version,
		RetentionPeriod: stringPointer("7 years"),
	})
	if err != nil {
		t.Fatalf("second update before replay: %v", err)
	}
	activity, err = service.TransitionActivity(context.Background(), ropa.TransitionActivityInput{
		TenantID:        scope.TenantID,
		LegalEntityID:   scope.LegalEntityID,
		ActivityID:      activity.ID,
		ExpectedVersion: activity.Version,
		To:              ropa.StatusClosed,
	})
	if err != nil {
		t.Fatalf("closing transition before replay: %v", err)
	}

	allEvents := make([]ropa.Event, 0)
	afterVersion := int64(0)
	for {
		page, hasMore, err := service.ActivityEvents(context.Background(), scope, activity.ID, afterVersion, 1)
		if err != nil {
			t.Fatalf("history page after version %d: %v", afterVersion, err)
		}
		if len(page) > 1 {
			t.Fatalf("history page returned %d rows for limit 1", len(page))
		}
		allEvents = append(allEvents, page...)
		if len(page) == 0 {
			break
		}
		afterVersion = page[len(page)-1].AggregateVersion
		if !hasMore {
			break
		}
	}
	if len(allEvents) != 5 {
		t.Fatalf("replayed event count = %d, want 5", len(allEvents))
	}
	for index, event := range allEvents {
		wantVersion := int64(index + 1)
		if event.AggregateVersion != wantVersion {
			t.Fatalf("event %d version = %d, want %d", index, event.AggregateVersion, wantVersion)
		}
	}
	replayed, err := ropa.ReplayActivityEvents(allEvents)
	if err != nil {
		t.Fatalf("replay history: %v", err)
	}
	current, err := service.GetActivity(context.Background(), scope, activity.ID)
	if err != nil {
		t.Fatalf("read current activity for replay comparison: %v", err)
	}
	if !reflect.DeepEqual(replayed, current) {
		t.Fatalf("replayed activity differs from current:\nreplayed=%#v\ncurrent=%#v", replayed, current)
	}
	if replayed.Version != int64(len(allEvents)) {
		t.Fatalf("replayed version = %d, event count = %d", replayed.Version, len(allEvents))
	}
}

func TestMemoryRepositoryRejectsOperationEventTypeMismatch(t *testing.T) {
	repository := ropa.NewMemoryRepository()
	activity, err := task3DirectCreate(t, repository, task3DirectActivity())
	if err != nil {
		t.Fatalf("seed activity: %v", err)
	}
	next := activity
	next.Status = ropa.StatusOpen
	next.Version++
	next.UpdatedAt = task3Now.Add(time.Minute)
	for _, eventType := range []string{ropa.EventActivityCreated, ropa.EventActivityUpdated} {
		event := task3DirectEvent(t, next, eventType, next.Version)
		if _, err := repository.ApplyActivityEvent(context.Background(), ropa.ActivityScope{TenantID: activity.TenantID, LegalEntityID: activity.LegalEntityID}, activity.ID, activity.Version, event); !errors.Is(err, ropa.ErrInvalid) {
			t.Fatalf("status-changing event labelled %q: expected ErrInvalid, got %v", eventType, err)
		}
	}
}
