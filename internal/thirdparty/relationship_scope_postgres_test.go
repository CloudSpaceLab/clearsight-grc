//go:build postgres

package thirdparty

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestRelationshipReadUsesVerifiedTenantScopeRepresentation(t *testing.T) {
	for _, tenant := range []string{"tenant-uuid", "bank-slug"} {
		value := Aggregate{Vendor: Vendor{ID: "vendor", TenantID: "bank-slug"}, Relationship: Relationship{ID: "relationship", VendorID: "vendor", TenantID: "bank-slug", LegalEntityID: "entity"}}
		got, err := relationshipWithScope(value, Scope{TenantID: tenant, LegalEntityID: "entity"})
		if err != nil || got.Vendor.TenantID != tenant || got.Relationship.TenantID != tenant {
			t.Fatalf("scope %q: %#v %v", tenant, got, err)
		}
	}
}

func TestRelationshipReadRejectsInconsistentResultBeforeApplyingScope(t *testing.T) {
	value := Aggregate{Vendor: Vendor{ID: "vendor", TenantID: "bank"}, Relationship: Relationship{ID: "relationship", VendorID: "vendor", TenantID: "bank", LegalEntityID: "other"}}
	if _, err := relationshipWithScope(value, Scope{TenantID: "bank", LegalEntityID: "entity"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected mismatched entity to fail closed, got %v", err)
	}
	value.Relationship.LegalEntityID = "entity"
	value.Relationship.TenantID = "another"
	if _, err := relationshipWithScope(value, Scope{TenantID: "bank", LegalEntityID: "entity"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected mismatched tenants to fail closed, got %v", err)
	}
}

func TestCreateReceiptReturnsVerifiedScopeRepresentation(t *testing.T) {
	value := Aggregate{Vendor: Vendor{ID: "vendor", TenantID: "original-slug"}, Relationship: Relationship{ID: "relationship", VendorID: "vendor", TenantID: "original-slug", LegalEntityID: "entity"}}
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	got, err := replayRelationshipReceipt(payload, Scope{TenantID: "tenant-uuid", LegalEntityID: "entity"}, "relationship")
	if err != nil || got.Vendor.TenantID != "tenant-uuid" || got.Relationship.TenantID != "tenant-uuid" || got.Relationship.ID != value.Relationship.ID {
		t.Fatalf("replay: %#v %v", got, err)
	}
	if _, err := replayRelationshipReceipt(payload, Scope{TenantID: "tenant-uuid", LegalEntityID: "entity"}, "different-relationship"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("wrong indexed relationship: %v", err)
	}
	if _, err := replayRelationshipReceipt(payload, Scope{TenantID: "tenant-uuid", LegalEntityID: "entity"}, ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing indexed relationship: %v", err)
	}
	if _, err := replayRelationshipReceipt(payload, Scope{TenantID: "tenant-uuid", LegalEntityID: "other"}, "relationship"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("wrong entity: %v", err)
	}
	value.Relationship.TenantID = "other-tenant"
	payload, err = json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := replayRelationshipReceipt(payload, Scope{TenantID: "tenant-uuid", LegalEntityID: "entity"}, "relationship"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("inconsistent receipt: %v", err)
	}
}
