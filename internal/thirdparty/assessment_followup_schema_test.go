package thirdparty

import (
	"os"
	"strings"
	"testing"
)

func TestAssessmentFollowupMigrationAllowsOnlySafeDeficiencyEvent(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000039_third_party_assessment_followups.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000039_third_party_assessment_followups.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(up), "'AssessmentDeficiencyLinked'") || strings.Contains(string(down), "'AssessmentDeficiencyLinked'") {
		t.Fatal("deficiency event migration contract is inconsistent")
	}
	for _, protected := range []string{"recipient_email", "recipient_address", "invitation_token", "vendor_answers", "document_contents"} {
		if strings.Contains(strings.ToLower(string(up)), protected) {
			t.Fatalf("migration contains protected field %q", protected)
		}
	}
}
