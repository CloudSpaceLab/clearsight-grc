package bankverticals

import (
	"context"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

func TestEnsureReferenceVendorIsIdempotentAndPreservesGovernedEdits(t *testing.T) {
	ctx := context.Background()
	config := normalizeSeedConfig(DemoSeedConfig())
	config.Now = time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	vendors := thirdparty.NewService(thirdparty.NewMemoryRepository())
	installer := &Service{}

	first, err := installer.EnsureReferenceVendor(ctx, config, vendors)
	if err != nil {
		t.Fatal(err)
	}
	if first.Vendor.SourceID != referenceVendorSourceID || first.Vendor.ExternalRef != referenceVendorExternalRef {
		t.Fatalf("vendor provenance=%q/%q, want %q/%q", first.Vendor.SourceID, first.Vendor.ExternalRef, referenceVendorSourceID, referenceVendorExternalRef)
	}
	if first.Relationship.SourceID != referenceVendorSourceID || first.Relationship.ExternalRef != referenceVendorExternalRef {
		t.Fatalf("relationship provenance=%q/%q, want %q/%q", first.Relationship.SourceID, first.Relationship.ExternalRef, referenceVendorSourceID, referenceVendorExternalRef)
	}

	edited, err := vendors.UpdateRelationship(ctx, thirdparty.Actor{
		TenantID: config.TenantID, LegalEntityID: config.LegalEntityID, PrincipalID: config.OwnerPrincipalID,
	}, first.Relationship.ID, thirdparty.UpdateRelationshipInput{
		ExpectedVersion: first.Relationship.Version,
		ServiceName:     "Managed infrastructure, backup and recovery services",
		Criticality:     first.Relationship.Criticality,
		PrivacyRole:     first.Relationship.PrivacyRole,
		EffectiveFrom:   first.Relationship.EffectiveFrom,
		RenewalAt:       first.Relationship.RenewalAt,
	})
	if err != nil {
		t.Fatal(err)
	}

	second, err := installer.EnsureReferenceVendor(ctx, config, vendors)
	if err != nil {
		t.Fatal(err)
	}
	if second.Relationship.ID != first.Relationship.ID || second.Vendor.ID != first.Vendor.ID {
		t.Fatalf("repeat install changed identity: first=%s/%s second=%s/%s", first.Vendor.ID, first.Relationship.ID, second.Vendor.ID, second.Relationship.ID)
	}
	if second.Relationship.ServiceName != edited.Relationship.ServiceName || second.Relationship.Version != edited.Relationship.Version {
		t.Fatalf("repeat install overwrote governed edit: got %#v want %#v", second.Relationship, edited.Relationship)
	}

	page, err := vendors.ListRelationships(ctx, thirdparty.Actor{
		TenantID: config.TenantID, LegalEntityID: config.LegalEntityID, PrincipalID: config.OwnerPrincipalID,
	}, thirdparty.ListInput{Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("reference relationships=%d, want 1", len(page.Items))
	}
}

func TestEnsureReferenceVendorDoesNotReuseUnrelatedVendor(t *testing.T) {
	ctx := context.Background()
	config := normalizeSeedConfig(DemoSeedConfig())
	config.Now = time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	vendors := thirdparty.NewService(thirdparty.NewMemoryRepository())
	actor := thirdparty.Actor{TenantID: config.TenantID, LegalEntityID: config.LegalEntityID, PrincipalID: config.OwnerPrincipalID}
	unrelated, err := vendors.CreateRelationship(ctx, actor, thirdparty.CreateRelationshipInput{
		LegalName: "Independent Reference Supplier Limited", ServiceName: "Independent service",
		Criticality: thirdparty.CriticalityStandard, PrivacyRole: thirdparty.PrivacyNone,
		SourceID: "external_registry", ExternalRef: "vendor:independent-supplier",
	})
	if err != nil {
		t.Fatal(err)
	}

	installed, err := (&Service{}).EnsureReferenceVendor(ctx, config, vendors)
	if err != nil {
		t.Fatal(err)
	}
	if installed.Vendor.ID == unrelated.Vendor.ID || installed.Relationship.ID == unrelated.Relationship.ID {
		t.Fatalf("reference installer reused unrelated vendor: unrelated=%#v installed=%#v", unrelated, installed)
	}
	page, err := vendors.ListRelationships(ctx, actor, thirdparty.ListInput{Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 {
		t.Fatalf("relationships=%d, want unrelated + reference", len(page.Items))
	}
}
