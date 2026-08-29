package thirdparty

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type relationshipTargetReaderStub struct {
	programs map[string]continuity.ProgramAggregate
	matters  map[string]continuity.MatterAggregate
}

func (s relationshipTargetReaderStub) GetProgram(_ context.Context, _, id string) (continuity.ProgramAggregate, error) {
	value, ok := s.programs[id]
	if !ok {
		return continuity.ProgramAggregate{}, errors.New("not found")
	}
	return value, nil
}

func (s relationshipTargetReaderStub) GetMatter(_ context.Context, _, id string) (continuity.MatterAggregate, error) {
	value, ok := s.matters[id]
	if !ok {
		return continuity.MatterAggregate{}, errors.New("not found")
	}
	return value, nil
}

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

func TestRelationshipLinkListUsesBoundedCursor(t *testing.T) {
	repo := NewMemoryRelationshipLinkRepository()
	repo.AllowRelationship("tenant-a", "entity-a", "relationship-1")
	service := NewRelationshipLinkService(repo)
	actor := Actor{TenantID: "tenant-a", LegalEntityID: "entity-a", PrincipalID: "owner-1"}
	base := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	for i, id := range []string{"link-1", "link-2", "link-3"} {
		repo.links[id] = RelationshipLink{ID: id, TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, RelationshipID: "relationship-1", TargetType: LinkTargetProgram, TargetID: "program-1", State: RelationshipLinkActive, UpdatedAt: base.Add(time.Duration(i) * time.Minute)}
	}
	first, err := service.List(context.Background(), actor, RelationshipLinkListInput{RelationshipID: "relationship-1", Limit: 2})
	if err != nil || len(first.Items) != 2 || first.NextCursor == "" {
		t.Fatalf("unexpected first page: %#v, %v", first, err)
	}
	second, err := service.List(context.Background(), actor, RelationshipLinkListInput{RelationshipID: "relationship-1", Cursor: first.NextCursor, Limit: 2})
	if err != nil || len(second.Items) != 1 || second.Items[0].ID != "link-1" || second.NextCursor != "" {
		t.Fatalf("unexpected second page: %#v, %v", second, err)
	}
}

func TestRelationshipLinksHideRestrictedMatterTargetsFromOtherActors(t *testing.T) {
	repo := NewMemoryRelationshipLinkRepository()
	repo.AllowRelationship("tenant-a", "entity-a", "relationship-1")
	repo.AllowTarget("tenant-a", "entity-a", LinkTargetMatter, "matter-restricted")
	service := NewRelationshipLinkService(repo)
	service.ConfigureTargetReader(relationshipTargetReaderStub{matters: map[string]continuity.MatterAggregate{
		"matter-restricted": {Matter: continuity.Matter{ID: "matter-restricted", TenantID: "tenant-a", Scope: json.RawMessage(`{"access":"RESTRICTED","allowed_principal_ids":["allowed-owner"]}`)}},
	}})
	service.newID = func() (string, error) { return "link-restricted", nil }
	allowed := Actor{TenantID: "tenant-a", LegalEntityID: "entity-a", PrincipalID: "allowed-owner"}
	denied := Actor{TenantID: "tenant-a", LegalEntityID: "entity-a", PrincipalID: "other-owner"}
	created, err := service.Link(context.Background(), allowed, "relationship-1", LinkRelationshipInput{TargetType: LinkTargetMatter, TargetID: "matter-restricted", PurposeCode: "DELIVERY_PARTY", PurposeLabel: "Delivery party"})
	if err != nil {
		t.Fatalf("link restricted matter as allowed actor: %v", err)
	}
	if page, err := service.List(context.Background(), denied, RelationshipLinkListInput{RelationshipID: "relationship-1", Limit: 20}); err != nil || len(page.Items) != 0 {
		t.Fatalf("restricted link leaked to denied actor: %#v, %v", page, err)
	}
	if _, err := service.End(context.Background(), denied, created.ID, EndRelationshipLinkInput{ExpectedVersion: created.Version, Reason: "No longer required."}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("denied actor end error = %v, want not found", err)
	}
	if _, err := service.Link(context.Background(), denied, "relationship-1", LinkRelationshipInput{TargetType: LinkTargetMatter, TargetID: "matter-restricted", PurposeCode: "SUPPORT", PurposeLabel: "Support"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("denied actor link error = %v, want not found", err)
	}
}

func TestRelationshipLinkListFillsPageAfterRestrictedTargetsAreRemoved(t *testing.T) {
	repo := NewMemoryRelationshipLinkRepository()
	base := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	repo.links["link-hidden"] = RelationshipLink{ID: "link-hidden", TenantID: "tenant-a", LegalEntityID: "entity-a", RelationshipID: "relationship-1", TargetType: LinkTargetMatter, TargetID: "matter-hidden", State: RelationshipLinkActive, UpdatedAt: base.Add(time.Minute)}
	repo.links["link-visible"] = RelationshipLink{ID: "link-visible", TenantID: "tenant-a", LegalEntityID: "entity-a", RelationshipID: "relationship-1", TargetType: LinkTargetMatter, TargetID: "matter-visible", State: RelationshipLinkActive, UpdatedAt: base}
	service := NewRelationshipLinkService(repo)
	service.ConfigureTargetReader(relationshipTargetReaderStub{matters: map[string]continuity.MatterAggregate{
		"matter-hidden":  {Matter: continuity.Matter{ID: "matter-hidden", TenantID: "tenant-a", Scope: json.RawMessage(`{"access":"RESTRICTED","allowed_principal_ids":["different-owner"]}`)}},
		"matter-visible": {Matter: continuity.Matter{ID: "matter-visible", TenantID: "tenant-a", Scope: json.RawMessage(`{"access":"INTERNAL"}`)}},
	}})
	actor := Actor{TenantID: "tenant-a", LegalEntityID: "entity-a", PrincipalID: "owner-1"}

	page, err := service.List(context.Background(), actor, RelationshipLinkListInput{RelationshipID: "relationship-1", Limit: 1})
	if err != nil || len(page.Items) != 1 || page.Items[0].ID != "link-visible" || page.NextCursor != "" {
		t.Fatalf("visible page = %#v, %v", page, err)
	}
}

func TestRelationshipLinksAcceptCanonicalTenantIDFromScopedTargetRead(t *testing.T) {
	repo := NewMemoryRelationshipLinkRepository()
	repo.links["link-canonical-tenant"] = RelationshipLink{
		ID: "link-canonical-tenant", TenantID: "tenant-a", LegalEntityID: "entity-a", RelationshipID: "relationship-1",
		TargetType: LinkTargetMatter, TargetID: "matter-1", State: RelationshipLinkActive, UpdatedAt: time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC),
	}
	service := NewRelationshipLinkService(repo)
	service.ConfigureTargetReader(relationshipTargetReaderStub{matters: map[string]continuity.MatterAggregate{
		"matter-1": {Matter: continuity.Matter{ID: "matter-1", TenantID: "33333333-3333-7333-8333-333333333331", LegalEntityID: "entity-a", Scope: json.RawMessage(`{"access":"INTERNAL"}`)}},
	}})
	actor := Actor{TenantID: "tenant-a", LegalEntityID: "entity-a", PrincipalID: "owner-1"}

	page, err := service.List(context.Background(), actor, RelationshipLinkListInput{TargetType: LinkTargetMatter, TargetID: "matter-1", Limit: 20})
	if err != nil || len(page.Items) != 1 || page.Items[0].ID != "link-canonical-tenant" {
		t.Fatalf("canonical tenant target page = %#v, %v", page, err)
	}
}

func TestRelationshipLinksRejectProgramsOutsideActorLegalEntity(t *testing.T) {
	repo := NewMemoryRelationshipLinkRepository()
	repo.AllowRelationship("tenant-a", "entity-a", "relationship-1")
	repo.AllowTarget("tenant-a", "entity-a", LinkTargetProgram, "program-other")
	service := NewRelationshipLinkService(repo)
	service.ConfigureTargetReader(relationshipTargetReaderStub{programs: map[string]continuity.ProgramAggregate{
		"program-other": {Program: continuity.Program{ID: "program-other", TenantID: "tenant-a", LegalEntityID: "entity-b"}},
	}})
	actor := Actor{TenantID: "tenant-a", LegalEntityID: "entity-a", PrincipalID: "owner-1"}
	if _, err := service.Link(context.Background(), actor, "relationship-1", LinkRelationshipInput{TargetType: LinkTargetProgram, TargetID: "program-other", PurposeCode: "SUPPORT", PurposeLabel: "Support"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-entity program link error = %v, want not found", err)
	}
}

type recordingRelationshipLinkGuard struct{ requests []commandauth.Request }

func (g *recordingRelationshipLinkGuard) Authorize(ctx context.Context, request commandauth.Request) (commandauth.Decision, error) {
	g.requests = append(g.requests, request)
	actor, _ := identity.FromContext(ctx)
	return commandauth.Decision{Allowed: true, Actor: actor}, nil
}

func TestRelationshipLinkRequiresRelationshipAndTargetOwnerAuthority(t *testing.T) {
	repo := NewMemoryRelationshipLinkRepository()
	repo.AllowRelationship("tenant-a", "entity-a", "relationship-1")
	repo.AllowTarget("tenant-a", "entity-a", LinkTargetProgram, "program-1")
	service := NewRelationshipLinkService(repo)
	guard := &recordingRelationshipLinkGuard{}
	service.ConfigureAuthority(guard)
	service.newID = func() (string, error) { return "link-1", nil }
	actor := Actor{TenantID: "tenant-a", LegalEntityID: "entity-a", PrincipalID: "owner-1"}
	ctx := identity.WithActor(context.Background(), identity.Actor{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PrincipalID: actor.PrincipalID, Kind: "PERSON", IssuedAt: time.Now().Add(-time.Minute), ExpiresAt: time.Now().Add(time.Hour)})

	created, err := service.Link(ctx, actor, "relationship-1", LinkRelationshipInput{TargetType: LinkTargetProgram, TargetID: "program-1", PurposeCode: "SUPPORT", PurposeLabel: "Support"})
	if err != nil {
		t.Fatal(err)
	}
	if len(guard.requests) != 2 || guard.requests[0].ObjectType != "VENDOR_RELATIONSHIP" || guard.requests[0].ObjectID != "relationship-1" || guard.requests[1].ObjectType != "PROGRAM" || guard.requests[1].ObjectID != "program-1" || guard.requests[1].Responsibility != authority.ResponsibilityOwner {
		t.Fatalf("authority requests = %#v", guard.requests)
	}
	if _, err := service.End(ctx, actor, created.ID, EndRelationshipLinkInput{ExpectedVersion: created.Version, Reason: "No longer required."}); err != nil {
		t.Fatal(err)
	}
	if len(guard.requests) != 4 || guard.requests[2].ObjectType != "VENDOR_RELATIONSHIP" || guard.requests[3].ObjectType != "PROGRAM" || guard.requests[3].ObjectID != "program-1" || guard.requests[3].DecisionType != "thirdparty.relationship.unlink" {
		t.Fatalf("end authority requests = %#v", guard.requests)
	}
}

func TestRelationshipLinkCoordinatorSerializesInvariantChecks(t *testing.T) {
	coordinator := &RelationshipLinkCoordinator{}
	coordinator.Lock()
	entered := make(chan struct{})
	done := make(chan struct{})
	go func() {
		coordinator.Lock()
		close(entered)
		coordinator.Unlock()
		close(done)
	}()
	select {
	case <-entered:
		t.Fatal("second relationship operation entered before the first completed")
	case <-time.After(20 * time.Millisecond):
	}
	coordinator.Unlock()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("second relationship operation did not resume")
	}
}
