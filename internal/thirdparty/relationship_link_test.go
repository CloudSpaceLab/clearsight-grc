package thirdparty

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRelationshipLinkServiceSupportsBankDefinedPurposeAndHistory(t *testing.T) {
	repo := NewMemoryRelationshipLinkRepository()
	repo.AllowRelationship("tenant-a", "entity-a", "relationship-1")
	repo.AllowTarget("tenant-a", "entity-a", LinkTargetProgram, "program-1")
	service := NewRelationshipLinkService(repo)
	service.now = func() time.Time { return time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC) }
	service.newID = func() (string, error) { return "link-1", nil }
	actor := Actor{TenantID: "tenant-a", LegalEntityID: "entity-a", PrincipalID: "owner-1"}

	created, err := service.Link(context.Background(), actor, "relationship-1", LinkRelationshipInput{
		TargetType: LinkTargetProgram, TargetID: "program-1", PurposeCode: "CORE_BANKING_RELEASE_SUPPORT", PurposeLabel: "Release support",
	})
	if err != nil {
		t.Fatalf("link relationship: %v", err)
	}
	if created.PurposeCode != "CORE_BANKING_RELEASE_SUPPORT" || created.State != RelationshipLinkActive || created.Version != 1 {
		t.Fatalf("unexpected link %#v", created)
	}

	ended, err := service.End(context.Background(), actor, created.ID, EndRelationshipLinkInput{ExpectedVersion: 1, Reason: "Application support ended."})
	if err != nil {
		t.Fatalf("end relationship link: %v", err)
	}
	if ended.State != RelationshipLinkEnded || ended.Version != 2 || ended.EndedAt == nil {
		t.Fatalf("unexpected ended link %#v", ended)
	}
	page, err := service.List(context.Background(), actor, RelationshipLinkListInput{RelationshipID: "relationship-1", IncludeEnded: true, Limit: 20})
	if err != nil || len(page.Items) != 1 || page.Items[0].State != RelationshipLinkEnded {
		t.Fatalf("history was not retained: %#v, %v", page, err)
	}
}

func TestRelationshipLinkServiceRejectsCrossScopeUnknownAndDuplicateTargets(t *testing.T) {
	repo := NewMemoryRelationshipLinkRepository()
	repo.AllowRelationship("tenant-a", "entity-a", "relationship-1")
	repo.AllowTarget("tenant-a", "entity-a", LinkTargetMatter, "matter-1")
	service := NewRelationshipLinkService(repo)
	ids := []string{"link-1", "link-2"}
	service.newID = func() (string, error) { value := ids[0]; ids = ids[1:]; return value, nil }
	actor := Actor{TenantID: "tenant-a", LegalEntityID: "entity-a", PrincipalID: "owner-1"}
	input := LinkRelationshipInput{TargetType: LinkTargetMatter, TargetID: "matter-1", PurposeCode: "DELIVERY_PARTY", PurposeLabel: "Delivery party"}
	if _, err := service.Link(context.Background(), actor, "relationship-1", input); err != nil {
		t.Fatalf("first link: %v", err)
	}
	if _, err := service.Link(context.Background(), actor, "relationship-1", input); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("expected duplicate conflict, got %v", err)
	}
	if _, err := service.Link(context.Background(), actor, "relationship-1", LinkRelationshipInput{TargetType: LinkTargetProgram, TargetID: "program-other", PurposeCode: "SUPPORT", PurposeLabel: "Support"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected unavailable target, got %v", err)
	}
}

func TestRelationshipLinkServiceValidatesPurposeWithoutFixedEnum(t *testing.T) {
	repo := NewMemoryRelationshipLinkRepository()
	repo.AllowRelationship("tenant-a", "entity-a", "relationship-1")
	repo.AllowTarget("tenant-a", "entity-a", LinkTargetProgram, "program-1")
	service := NewRelationshipLinkService(repo)
	actor := Actor{TenantID: "tenant-a", LegalEntityID: "entity-a", PrincipalID: "owner-1"}
	for _, input := range []LinkRelationshipInput{
		{TargetType: LinkTargetProgram, TargetID: "program-1", PurposeCode: "bad purpose", PurposeLabel: "Support"},
		{TargetType: LinkTargetProgram, TargetID: "program-1", PurposeCode: "SUPPORT", PurposeLabel: ""},
	} {
		if _, err := service.Link(context.Background(), actor, "relationship-1", input); !errors.Is(err, ErrInvalid) {
			t.Fatalf("expected invalid purpose for %#v, got %v", input, err)
		}
	}
}
