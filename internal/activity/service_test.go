package activity

import (
	"context"
	"testing"
	"time"
)

func TestServiceListsOnlyRequestedTenantWithStableCursor(t *testing.T) {
	now := time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC)
	repository := NewMemoryRepository(
		Event{TenantID: "bank-a", ID: "3", OccurredAt: now, EventType: "MATTER_STATE_CHANGED", ObjectType: "MATTER", ObjectID: "matter-3", ActorKind: "PERSON", ActorID: "user-1", ActorDisplayName: "Ada Bello"},
		Event{TenantID: "bank-a", ID: "2", OccurredAt: now.Add(-time.Minute), EventType: "FORM_DISTRIBUTION_CREATED", ObjectType: "FORM_DISTRIBUTION", ObjectID: "form-2"},
		Event{TenantID: "bank-b", ID: "hidden", OccurredAt: now.Add(-2 * time.Minute), EventType: "MATTER_CREATED", ObjectType: "MATTER", ObjectID: "matter-hidden"},
		Event{TenantID: "bank-a", ID: "1", OccurredAt: now.Add(-3 * time.Minute), EventType: "RoutingPolicyApproved", ObjectType: "ROUTING_POLICY", ObjectID: "policy-1"},
	)
	service := NewService(repository)

	page, err := service.List(context.Background(), Query{TenantID: "bank-a", Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 || page.Items[0].ID != "3" || page.Items[1].ID != "2" || page.NextCursor != "2" {
		t.Fatalf("unexpected first page: %#v", page)
	}
	if page.Items[0].ActorKind != ActorInternalUser || page.Items[0].Category != CategoryGRCWork || page.Items[0].Action != "Matter state changed" {
		t.Fatalf("event was not normalized: %#v", page.Items[0])
	}

	next, err := service.List(context.Background(), Query{TenantID: "bank-a", Limit: 2, Cursor: page.NextCursor})
	if err != nil {
		t.Fatal(err)
	}
	if len(next.Items) != 1 || next.Items[0].ID != "1" || next.Items[0].Category != CategoryConfiguration {
		t.Fatalf("unexpected second page: %#v", next)
	}
}

func TestServiceFiltersWithoutLeakingOtherTenants(t *testing.T) {
	now := time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC)
	repository := NewMemoryRepository(
		Event{TenantID: "bank-a", ID: "vendor", OccurredAt: now, EventType: "THIRD_PARTY_ASSESSMENT_COMPLETED", ObjectType: "THIRD_PARTY_ASSESSMENT", ObjectID: "assessment-1", ActorKind: "EXTERNAL_PARTY", ActorID: "vendor-user", LegalEntityID: "entity-a"},
		Event{TenantID: "bank-b", ID: "vendor-hidden", OccurredAt: now, EventType: "THIRD_PARTY_ASSESSMENT_COMPLETED", ObjectType: "THIRD_PARTY_ASSESSMENT", ObjectID: "assessment-2", ActorKind: "EXTERNAL_PARTY", ActorID: "vendor-user", LegalEntityID: "entity-a"},
	)
	service := NewService(repository)
	page, err := service.List(context.Background(), Query{TenantID: "bank-a", Category: CategoryVendor, ActorID: "vendor-user", LegalEntityID: "entity-a"})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "vendor" || page.Items[0].ActorKind != ActorExternalParticipant {
		t.Fatalf("unexpected filtered page: %#v", page)
	}
}

func TestServiceRejectsInvalidTimeRange(t *testing.T) {
	repository := NewMemoryRepository()
	service := NewService(repository)
	from := time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC)
	to := from.Add(-time.Hour)
	if _, err := service.List(context.Background(), Query{TenantID: "bank-a", From: &from, To: &to}); err != ErrInvalid {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}
