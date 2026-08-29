package thirdparty

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestRefreshMaintenanceExpiresAtCalendarDateAndCreatesBoundedOwnerAttention(t *testing.T) {
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	repository := refreshMaintenanceFixture(now)
	maintainer := NewRefreshMaintainer(repository, RefreshMaintenancePolicy{BatchSize: 10, Lease: time.Minute, DocumentLead: 30 * 24 * time.Hour, FactConfirmationInterval: 365 * 24 * time.Hour})
	maintainer.now = func() time.Time { return now }

	receipt, err := maintainer.RunBatch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if receipt.RelationshipsExamined != 1 || receipt.DocumentsExpired != 1 || receipt.AttentionsCreated != 1 {
		t.Fatalf("unexpected maintenance receipt: %#v", receipt)
	}
	document := repository.assessmentDocuments["assessment-old"]["artifact-old"]
	if document.Status != AssessmentDocumentExpired || document.Version != 3 {
		t.Fatalf("document was not expired at its calendar date: %#v", document)
	}
	if len(repository.refreshAttentions) != 1 || len(repository.refreshEvents) != 1 || len(repository.refreshOutbox) != 1 {
		t.Fatalf("attention transaction incomplete: attentions=%d events=%d outbox=%d", len(repository.refreshAttentions), len(repository.refreshEvents), len(repository.refreshOutbox))
	}
	for _, attention := range repository.refreshAttentions {
		if attention.RelationshipID != "relationship-1" || attention.OwnerPrincipalID != "owner-1" || len(attention.TargetKeys) != 7 || attention.ObservedVersions["VENDOR.IDENTITY.LEGAL_NAME"] != 4 || attention.ObservedVersions["VENDOR.DOCUMENT.CERTIFICATE_OF_OPERATION"] != 2 {
			t.Fatalf("unexpected owner attention: %#v", attention)
		}
	}
}

func TestRefreshMaintenanceDeduplicatesRelationshipTargetVersionWindow(t *testing.T) {
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	repository := refreshMaintenanceFixture(now)
	maintainer := NewRefreshMaintainer(repository, RefreshMaintenancePolicy{BatchSize: 10, Lease: time.Minute, DocumentLead: 30 * 24 * time.Hour, FactConfirmationInterval: 365 * 24 * time.Hour})
	maintainer.now = func() time.Time { return now }
	if _, err := maintainer.RunBatch(context.Background()); err != nil {
		t.Fatal(err)
	}
	second, err := maintainer.RunBatch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if second.AttentionsCreated != 0 || second.DocumentsExpired != 0 || len(repository.refreshAttentions) != 1 {
		t.Fatalf("maintenance replay duplicated work: %#v attentions=%d", second, len(repository.refreshAttentions))
	}
}

func TestRefreshMaintenanceHonoursBatchLimitWithoutChoosingRecipientsOrSending(t *testing.T) {
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	repository := refreshMaintenanceFixture(now)
	for index := 2; index <= 4; index++ {
		vendorID := "vendor-" + string(rune('0'+index))
		relationshipID := "relationship-" + string(rune('0'+index))
		repository.vendors[vendorID] = Vendor{ID: vendorID, TenantID: "bank", LegalName: "Vendor", Version: 1, UpdatedAt: now.Add(-400 * 24 * time.Hour)}
		repository.relationships[relationshipID] = Relationship{ID: relationshipID, TenantID: "bank", LegalEntityID: "entity", VendorID: vendorID, BusinessOwnerPrincipalID: "owner", Version: 1}
	}
	maintainer := NewRefreshMaintainer(repository, RefreshMaintenancePolicy{BatchSize: 2, Lease: time.Minute, DocumentLead: 30 * 24 * time.Hour, FactConfirmationInterval: 365 * 24 * time.Hour})
	maintainer.now = func() time.Time { return now }
	receipt, err := maintainer.RunBatch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if receipt.RelationshipsExamined != 2 || receipt.AttentionsCreated != 2 || len(repository.refreshAttentions) != 2 {
		t.Fatalf("bounded maintenance = %#v attentions=%d", receipt, len(repository.refreshAttentions))
	}
	for _, attention := range repository.refreshAttentions {
		encoded, err := json.Marshal(attention)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(encoded), "audience") || strings.Contains(string(encoded), "distribution") || strings.Contains(string(encoded), "recipient") {
			t.Fatalf("maintenance selected or sent to a recipient: %s", encoded)
		}
	}
}

func refreshMaintenanceFixture(now time.Time) *MemoryAssessmentRepository {
	repository := NewMemoryAssessmentRepository()
	repository.vendors["vendor-1"] = Vendor{ID: "vendor-1", TenantID: "bank", LegalName: "Vendor Limited", Version: 4, UpdatedAt: now.Add(-365 * 24 * time.Hour)}
	repository.relationships["relationship-1"] = Relationship{ID: "relationship-1", TenantID: "bank", LegalEntityID: "entity", VendorID: "vendor-1", BusinessOwnerPrincipalID: "owner-1", Version: 2}
	repository.assessments["assessment-old"] = Assessment{ID: "assessment-old", TenantID: "bank", LegalEntityID: "entity", RelationshipID: "relationship-1", Status: AssessmentCompleted, Version: 8, UpdatedAt: now.Add(-30 * 24 * time.Hour)}
	expires := time.Date(2026, 8, 29, 0, 0, 0, 0, time.UTC)
	repository.assessmentDocuments["assessment-old"] = map[string]AssessmentDocument{"artifact-old": {ID: "document-old", Scope: Scope{TenantID: "bank", LegalEntityID: "entity"}, RelationshipID: "relationship-1", AssessmentID: "assessment-old", ArtifactID: "artifact-old", DocumentType: "CERTIFICATE_OF_OPERATION", Status: AssessmentDocumentValidated, Version: 2, ExpiresOn: &expires}}
	return repository
}
