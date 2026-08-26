package thirdparty

import (
	"os"
	"strings"
	"testing"
)

func TestAssessmentSetupRetryMigrationAddsOnlySafeEventType(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000038_third_party_assessment_setup_retry.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000038_third_party_assessment_setup_retry.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	upSchema, downSchema := string(up), string(down)
	if !strings.Contains(upSchema, "'AssessmentSetupRetryQueued'") {
		t.Fatal("setup retry migration does not allow its material event")
	}
	if strings.Contains(downSchema, "'AssessmentSetupRetryQueued'") {
		t.Fatal("setup retry rollback does not restore the prior event contract")
	}
	for _, protected := range []string{"recipient_email", "recipient_address", "invitation_token", "vendor_answers", "document_contents"} {
		if strings.Contains(strings.ToLower(upSchema), protected) {
			t.Fatalf("setup retry migration contains protected field %q", protected)
		}
	}
}
