package thirdparty

import (
	"context"
	"errors"
	"testing"
	"time"
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
