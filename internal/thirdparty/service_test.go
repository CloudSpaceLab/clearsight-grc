package thirdparty

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestCreateRelationshipRequiresVendorServiceAndLegalEntity(t *testing.T) {
	service := NewService(NewMemoryRepository())
	_, err := service.CreateRelationship(context.Background(), Actor{
		TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner",
	}, CreateRelationshipInput{})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestCreateRelationshipBindsOwnerToVerifiedActor(t *testing.T) {
	service := NewService(NewMemoryRepository())
	service.now = func() time.Time { return time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC) }

	got, err := service.CreateRelationship(context.Background(), Actor{
		TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner",
	}, validCreateInput())
	if err != nil {
		t.Fatal(err)
	}
	if got.Relationship.BusinessOwnerPrincipalID != "owner" {
		t.Fatalf("unexpected owner %q", got.Relationship.BusinessOwnerPrincipalID)
	}
	if got.Vendor.LegalName != "Acme Processing Limited" || got.Relationship.ServiceName != "Card transaction processing" {
		t.Fatalf("unexpected aggregate %#v", got)
	}
	if got.Vendor.Version != 1 || got.Relationship.Version != 1 {
		t.Fatalf("unexpected versions %#v", got)
	}
}

func TestListRelationshipsNeverCrossesLegalEntity(t *testing.T) {
	service := NewService(NewMemoryRepository())
	service.now = tickingClock()
	for _, entity := range []string{"entity-a", "entity-b"} {
		input := validCreateInput()
		input.ServiceName = "Service for " + entity
		if _, err := service.CreateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: entity, PrincipalID: "owner"}, input); err != nil {
			t.Fatal(err)
		}
	}

	page, err := service.ListRelationships(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "reader"}, ListInput{Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].Relationship.LegalEntityID != "entity-a" {
		t.Fatalf("unexpected scoped page %#v", page)
	}
}

func TestCreateRelationshipReusesExactExternalVendorIdentity(t *testing.T) {
	service := NewService(NewMemoryRepository())
	first, err := service.CreateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, validCreateInput())
	if err != nil {
		t.Fatal(err)
	}
	input := validCreateInput()
	input.ServiceName = "Settlement reporting"
	second, err := service.CreateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, input)
	if err != nil {
		t.Fatal(err)
	}
	if first.Vendor.ID != second.Vendor.ID || first.Relationship.ID == second.Relationship.ID {
		t.Fatalf("expected one exact vendor with two relationships: %#v %#v", first, second)
	}
}

func TestCreateRelationshipDoesNotFuzzyMergeVendorNames(t *testing.T) {
	service := NewService(NewMemoryRepository())
	first := validCreateInput()
	first.SourceID, first.ExternalRef = "", ""
	created, err := service.CreateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, first)
	if err != nil {
		t.Fatal(err)
	}
	second := first
	second.LegalName = "ACME PROCESSING LIMITED"
	second.ServiceName = "Another service"
	other, err := service.CreateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, second)
	if err != nil {
		t.Fatal(err)
	}
	if created.Vendor.ID == other.Vendor.ID {
		t.Fatal("vendor names must not be fuzzy-merged")
	}
}

func TestUpdateRelationshipRequiresCurrentVersion(t *testing.T) {
	service := NewService(NewMemoryRepository())
	created, err := service.CreateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, validCreateInput())
	if err != nil {
		t.Fatal(err)
	}
	updated, err := service.UpdateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, created.Relationship.ID, UpdateRelationshipInput{
		ExpectedVersion: created.Relationship.Version, ServiceName: "Card processing and settlement", Criticality: CriticalityCritical, PrivacyRole: PrivacyProcessor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Relationship.Version != 2 || updated.Relationship.ServiceName != "Card processing and settlement" {
		t.Fatalf("unexpected update %#v", updated)
	}
	if updated.Vendor.Version != created.Vendor.Version || updated.Vendor.LegalName != created.Vendor.LegalName {
		t.Fatalf("relationship update changed shared vendor identity: %#v", updated.Vendor)
	}
	_, err = service.UpdateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, created.Relationship.ID, UpdateRelationshipInput{
		ExpectedVersion: 1, ServiceName: "Stale change", Criticality: CriticalityStandard, PrivacyRole: PrivacyNone,
	})
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("expected version conflict, got %v", err)
	}
}

func TestListRelationshipsSearchesVendorAndService(t *testing.T) {
	service := NewService(NewMemoryRepository())
	service.now = tickingClock()
	first := validCreateInput()
	if _, err := service.CreateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, first); err != nil {
		t.Fatal(err)
	}
	second := validCreateInput()
	second.SourceID, second.ExternalRef = "procurement", "vendor-20002"
	second.LegalName, second.TradingName, second.ServiceName = "Beacon Hosting Limited", "Beacon", "Cloud hosting"
	if _, err := service.CreateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, second); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"beacon", "CLOUD HOST", "vendor-20002"} {
		page, err := service.ListRelationships(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "reader"}, ListInput{Search: query, Limit: 10})
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Items) != 1 || page.Items[0].Vendor.LegalName != "Beacon Hosting Limited" {
			t.Fatalf("unexpected search result for %q: %#v", query, page)
		}
	}
}

func TestCreateRelationshipCanReuseAnExistingVendorWithoutMergingRelationships(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewService(repo)
	service.now = tickingClock()
	ids := []string{"vendor-1", "relationship-1", "relationship-2"}
	service.newID = func() (string, error) { value := ids[0]; ids = ids[1:]; return value, nil }
	actor := Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}
	first, err := service.CreateRelationship(context.Background(), actor, validCreateInput())
	if err != nil {
		t.Fatal(err)
	}
	secondInput := validCreateInput()
	secondInput.ExistingRelationshipID = first.Relationship.ID
	secondInput.LegalName, secondInput.TradingName, secondInput.RegistrationRef = "", "", ""
	secondInput.ServiceName = "Settlement support"
	second, err := service.CreateRelationship(context.Background(), actor, secondInput)
	if err != nil {
		t.Fatalf("reuse vendor: %v", err)
	}
	if second.Vendor.ID != first.Vendor.ID || second.Relationship.ID == first.Relationship.ID || second.Relationship.VendorID != first.Vendor.ID {
		t.Fatalf("unexpected reused relationship: first=%#v second=%#v", first, second)
	}
	if len(repo.vendors) != 1 || len(repo.relationships) != 2 {
		t.Fatalf("repository counts vendors=%d relationships=%d", len(repo.vendors), len(repo.relationships))
	}
	if _, err := service.CreateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: "other-entity", PrincipalID: "owner"}, CreateRelationshipInput{ExistingRelationshipID: first.Relationship.ID, ServiceName: "Other", Criticality: CriticalityStandard, PrivacyRole: PrivacyNone}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-entity reuse error = %v, want not found", err)
	}
}

func TestGetRelationshipDoesNotLeakAcrossEntityOrTenant(t *testing.T) {
	service := NewService(NewMemoryRepository())
	created, err := service.CreateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "owner"}, validCreateInput())
	if err != nil {
		t.Fatal(err)
	}
	for _, actor := range []Actor{{TenantID: "bank", LegalEntityID: "entity-b", PrincipalID: "reader"}, {TenantID: "other-bank", LegalEntityID: "entity-a", PrincipalID: "reader"}} {
		if _, err := service.GetRelationship(context.Background(), actor, created.Relationship.ID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("expected not found for scope %#v, got %v", actor, err)
		}
	}
}

func TestCreateRelationshipWithWebsiteSchedulesBrandDiscoveryAtomically(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewService(repo)
	service.now = func() time.Time { return time.Date(2026, 8, 26, 8, 30, 0, 0, time.UTC) }
	ids := []string{"vendor-1", "relationship-1", "brand-job-1"}
	service.newID = func() (string, error) {
		value := ids[0]
		ids = ids[1:]
		return value, nil
	}
	input := validCreateInput()
	input.WebsiteDomain = "Vendor.Example"

	created, err := service.CreateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, input)
	if err != nil {
		t.Fatal(err)
	}
	if created.Vendor.WebsiteDomain != "vendor.example" {
		t.Fatalf("website domain = %q", created.Vendor.WebsiteDomain)
	}
	job, err := repo.GetVendorBrandJob(context.Background(), Scope{TenantID: "bank", LegalEntityID: "entity"}, created.Vendor.ID)
	if err != nil {
		t.Fatal(err)
	}
	if job.State != VendorBrandJobReady || job.WebsiteDomain != created.Vendor.WebsiteDomain || job.VendorVersion != created.Vendor.Version {
		t.Fatalf("brand job = %#v", job)
	}
	if len(repo.vendorIdentityEvents) != 1 || len(repo.vendorIdentityOutbox) != 1 {
		t.Fatalf("identity audit event/outbox counts = %d/%d", len(repo.vendorIdentityEvents), len(repo.vendorIdentityOutbox))
	}
	if repo.vendorIdentityEvents[0].EventType != VendorIdentityCreatedEvent || repo.vendorIdentityEvents[0].ActorPrincipalID != "owner" {
		t.Fatalf("identity event = %#v", repo.vendorIdentityEvents[0])
	}
}

func TestCreateRelationshipWithoutWebsiteRecordsReconstructableVendorIdentity(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewService(repo)
	service.now = func() time.Time { return time.Date(2026, 8, 26, 8, 45, 0, 0, time.UTC) }

	created, err := service.CreateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, validCreateInput())
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.vendorIdentityEvents) != 1 || len(repo.vendorIdentityOutbox) != 1 {
		t.Fatalf("identity audit event/outbox counts = %d/%d", len(repo.vendorIdentityEvents), len(repo.vendorIdentityOutbox))
	}
	for name, event := range map[string]VendorIdentityEvent{"event": repo.vendorIdentityEvents[0], "outbox": repo.vendorIdentityOutbox[0]} {
		if event.EventType != VendorIdentityCreatedEvent || event.VendorVersion != 1 || event.ActorPrincipalID != "owner" {
			t.Fatalf("%s identity envelope = %#v", name, event)
		}
		if event.LegalName != created.Vendor.LegalName || event.TradingName != created.Vendor.TradingName || event.RegistrationRef != created.Vendor.RegistrationRef || event.Jurisdiction != created.Vendor.Jurisdiction || event.WebsiteDomain != "" || event.Status != created.Vendor.Status {
			t.Fatalf("%s identity snapshot cannot reconstruct vendor: %#v", name, event)
		}
	}
	if _, err := repo.GetVendorBrandJob(context.Background(), Scope{TenantID: "bank", LegalEntityID: "entity"}, created.Vendor.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("vendor without a website scheduled discovery: %v", err)
	}
}

func TestRelationshipCreationCannotSilentlyChangeReusedVendorWebsite(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewService(repo)
	first := validCreateInput()
	first.WebsiteDomain = "first.example"
	created, err := service.CreateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, first)
	if err != nil {
		t.Fatal(err)
	}
	second := validCreateInput()
	second.ServiceName = "Settlement support"
	second.WebsiteDomain = "replacement.example"
	reused, err := service.CreateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, second)
	if err != nil {
		t.Fatal(err)
	}
	if reused.Vendor.ID != created.Vendor.ID || reused.Vendor.WebsiteDomain != "first.example" {
		t.Fatalf("relationship creation changed shared vendor identity: %#v", reused.Vendor)
	}
	if len(repo.vendorIdentityEvents) != 1 || len(repo.vendorIdentityOutbox) != 1 {
		t.Fatalf("reused vendor emitted identity audit: %d/%d", len(repo.vendorIdentityEvents), len(repo.vendorIdentityOutbox))
	}
}

func TestUpdateVendorIdentityUsesVerifiedAuthorityAndExpectedVersion(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewService(repo)
	service.now = func() time.Time { return time.Date(2026, 8, 26, 9, 0, 0, 0, time.UTC) }
	created, err := service.CreateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, validCreateInput())
	if err != nil {
		t.Fatal(err)
	}
	guard := &vendorIdentityGuardStub{}
	service.ConfigureIdentityAuthority(guard)
	ctx := vendorIdentityContext("bank", "entity", "verified-owner", service.now())

	updated, err := service.UpdateVendorIdentity(ctx, Actor{TenantID: "other", LegalEntityID: "other", PrincipalID: "body-actor"}, created.Vendor.ID, UpdateVendorIdentityInput{
		ExpectedVersion: created.Vendor.Version,
		LegalName:       "Acme Payments Limited",
		TradingName:     "Acme Payments",
		RegistrationRef: "RC-20002",
		Jurisdiction:    "Ghana",
		WebsiteDomain:   "BÜCHER.Example",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Version != created.Vendor.Version+1 || updated.WebsiteDomain != "xn--bcher-kva.example" {
		t.Fatalf("updated vendor = %#v", updated)
	}
	if len(guard.requests) != 1 || guard.requests[0].TenantID != "bank" || guard.requests[0].LegalEntityID != "entity" || guard.requests[0].ObjectID != created.Vendor.ID || guard.requests[0].DecisionType != VendorIdentityUpdateCommand {
		t.Fatalf("authority request = %#v", guard.requests)
	}
	if got := repo.vendorIdentityEvents[len(repo.vendorIdentityEvents)-1].ActorPrincipalID; got != "verified-owner" {
		t.Fatalf("event actor = %q", got)
	}
	updatedEvent := repo.vendorIdentityEvents[len(repo.vendorIdentityEvents)-1]
	if updatedEvent.VendorVersion != updated.Version || updatedEvent.LegalName != updated.LegalName || updatedEvent.TradingName != updated.TradingName || updatedEvent.RegistrationRef != updated.RegistrationRef || updatedEvent.Jurisdiction != updated.Jurisdiction || updatedEvent.WebsiteDomain != updated.WebsiteDomain || updatedEvent.Status != updated.Status {
		t.Fatalf("updated identity event cannot reconstruct vendor: event=%#v vendor=%#v", updatedEvent, updated)
	}
	updatedOutbox := repo.vendorIdentityOutbox[len(repo.vendorIdentityOutbox)-1]
	if updatedOutbox != updatedEvent {
		t.Fatalf("identity outbox snapshot differs from event: event=%#v outbox=%#v", updatedEvent, updatedOutbox)
	}
	job, err := repo.GetVendorBrandJob(ctx, Scope{TenantID: "bank", LegalEntityID: "entity"}, created.Vendor.ID)
	if err != nil {
		t.Fatal(err)
	}
	if job.State != VendorBrandJobReady || job.VendorVersion != updated.Version || job.WebsiteDomain != updated.WebsiteDomain {
		t.Fatalf("updated job = %#v", job)
	}
	_, err = service.UpdateVendorIdentity(ctx, Actor{}, created.Vendor.ID, UpdateVendorIdentityInput{
		ExpectedVersion: created.Vendor.Version,
		LegalName:       updated.LegalName,
		WebsiteDomain:   string(updated.WebsiteDomain),
	})
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale identity update error = %v", err)
	}
}

func TestUpdateVendorIdentityFailsClosedWithoutVerifiedAuthority(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewService(repo)
	created, err := service.CreateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, validCreateInput())
	if err != nil {
		t.Fatal(err)
	}
	input := UpdateVendorIdentityInput{ExpectedVersion: created.Vendor.Version, LegalName: created.Vendor.LegalName, WebsiteDomain: "vendor.example"}
	if _, err := service.UpdateVendorIdentity(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "body-actor"}, created.Vendor.ID, input); !errors.Is(err, identity.ErrMissingIdentity) {
		t.Fatalf("missing verified identity error = %v", err)
	}
	ctx := vendorIdentityContext("bank", "entity", "owner", time.Now().UTC())
	if _, err := service.UpdateVendorIdentity(ctx, Actor{}, created.Vendor.ID, input); !errors.Is(err, ErrVendorIdentityAuthorityUnavailable) {
		t.Fatalf("missing authority error = %v", err)
	}
}

type vendorIdentityGuardStub struct {
	requests []commandauth.Request
	err      error
}

func (g *vendorIdentityGuardStub) Authorize(ctx context.Context, request commandauth.Request) (commandauth.Decision, error) {
	g.requests = append(g.requests, request)
	if g.err != nil {
		return commandauth.Decision{}, g.err
	}
	actor, err := identity.Require(ctx)
	return commandauth.Decision{Allowed: err == nil, Enforced: true, Actor: actor}, err
}

func vendorIdentityContext(tenantID, legalEntityID, principalID string, now time.Time) context.Context {
	return identity.WithActor(context.Background(), identity.Actor{
		TenantID: tenantID, LegalEntityID: legalEntityID, PrincipalID: principalID,
		Kind: "human", AuthenticationMethod: "test", IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour),
	})
}

func tickingClock() func() time.Time {
	now := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	return func() time.Time {
		now = now.Add(time.Second)
		return now
	}
}

func validCreateInput() CreateRelationshipInput {
	return CreateRelationshipInput{
		LegalName:       "Acme Processing Limited",
		TradingName:     "Acme Processing",
		RegistrationRef: "RC-10001",
		Jurisdiction:    "Nigeria",
		SourceID:        "procurement",
		ExternalRef:     "vendor-10001",
		ServiceName:     "Card transaction processing",
		Criticality:     CriticalityImportant,
		PrivacyRole:     PrivacyProcessor,
	}
}
