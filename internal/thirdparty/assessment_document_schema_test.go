package thirdparty

import (
	"os"
	"strings"
	"testing"
)

func TestAssessmentDocumentReviewMigrationAllowsOnlySafeMaterialEvents(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000040_third_party_document_review.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000040_third_party_document_review.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, eventType := range []string{"'AssessmentDocumentValidated'", "'AssessmentDocumentRejected'"} {
		if !strings.Contains(string(up), eventType) || strings.Contains(string(down), eventType) {
			t.Fatalf("document review event migration contract is inconsistent for %s", eventType)
		}
	}
	for _, protected := range []string{"recipient_email", "recipient_address", "invitation_token", "storage_key", "document_contents", "reviewer_notes"} {
		if strings.Contains(strings.ToLower(string(up)), protected) {
			t.Fatalf("migration contains protected field %q", protected)
		}
	}
}
