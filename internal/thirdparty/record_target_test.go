package thirdparty

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

type targetDocumentReaderStub struct {
	documents []AssessmentDocument
}

func (s targetDocumentReaderStub) CurrentRelationshipDocuments(context.Context, Scope, string, string) ([]AssessmentDocument, error) {
	return append([]AssessmentDocument(nil), s.documents...), nil
}

func TestRecordTargetResolverFreezesExactVendorVersion(t *testing.T) {
	now := time.Date(2026, 8, 29, 9, 0, 0, 0, time.UTC)
	aggregate := Aggregate{Vendor: Vendor{ID: "vendor-1", LegalName: "Cloud Operations Limited", Version: 7, UpdatedAt: now}, Relationship: Relationship{ID: "relationship-1", VendorID: "vendor-1"}}
	resolver := NewRecordTargetResolver(nil)

	baseline, err := resolver.Resolve(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity"}, aggregate, formcontract.RecordTarget{Key: "VENDOR.IDENTITY.LEGAL_NAME", RequiredSubjectType: "VENDOR_RELATIONSHIP"})
	if err != nil {
		t.Fatal(err)
	}
	if baseline.TargetKey != "VENDOR.IDENTITY.LEGAL_NAME" || baseline.SubjectID != "relationship-1" || baseline.RecordID != "vendor-1" || baseline.RecordVersion != 7 || baseline.DisplayValue != "Cloud Operations Limited" || !baseline.ObservedOrConfirmedAt.Equal(now) {
		t.Fatalf("unexpected baseline: %#v", baseline)
	}
}

func TestRecordTargetResolverRejectsWrongSubjectAndUnknownTarget(t *testing.T) {
	resolver := NewRecordTargetResolver(nil)
	aggregate := Aggregate{Vendor: Vendor{ID: "vendor-1", LegalName: "Vendor", Version: 1}, Relationship: Relationship{ID: "relationship-1"}}
	for _, target := range []formcontract.RecordTarget{
		{Key: "VENDOR.IDENTITY.LEGAL_NAME", RequiredSubjectType: "CONTROL"},
		{Key: "VENDOR.IDENTITY.ARBITRARY_COLUMN", RequiredSubjectType: "VENDOR_RELATIONSHIP"},
	} {
		if _, err := resolver.Resolve(context.Background(), Actor{}, aggregate, target); !errors.Is(err, ErrUnsupportedRecordTarget) {
			t.Fatalf("target %#v error = %v", target, err)
		}
	}
}

func TestRecordTargetResolverRequiresOneExactCurrentDocument(t *testing.T) {
	expires := time.Date(2027, 1, 31, 0, 0, 0, 0, time.UTC)
	document := AssessmentDocument{ID: "document-1", RelationshipID: "relationship-1", DocumentType: "CERTIFICATE_OF_OPERATION", Reference: "CAC-44", Status: AssessmentDocumentValidated, Version: 3, UpdatedAt: time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC), ExpiresOn: &expires}
	aggregate := Aggregate{Vendor: Vendor{ID: "vendor-1"}, Relationship: Relationship{ID: "relationship-1"}}
	target := formcontract.RecordTarget{Key: "VENDOR.DOCUMENT.CERTIFICATE_OF_OPERATION", RequiredSubjectType: "VENDOR_RELATIONSHIP"}

	baseline, err := NewRecordTargetResolver(targetDocumentReaderStub{documents: []AssessmentDocument{document}}).Resolve(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity"}, aggregate, target)
	if err != nil {
		t.Fatal(err)
	}
	if baseline.RecordID != "document-1" || baseline.RecordVersion != 3 || baseline.ExpiresAt == nil || baseline.DisplayValue != "Certificate of operation · CAC-44" {
		t.Fatalf("unexpected document baseline: %#v", baseline)
	}

	_, err = NewRecordTargetResolver(targetDocumentReaderStub{documents: []AssessmentDocument{document, {ID: "document-2", RelationshipID: "relationship-1", DocumentType: document.DocumentType, Status: AssessmentDocumentValidated, Version: 1}}}).Resolve(context.Background(), Actor{}, aggregate, target)
	if !errors.Is(err, ErrAmbiguousRecordTarget) {
		t.Fatalf("ambiguous document error = %v", err)
	}
}
