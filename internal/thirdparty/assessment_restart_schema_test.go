package thirdparty

import (
	"os"
	"strings"
	"testing"
)

func TestAssessmentRestartMigrationAllowsPurposeBoundOnboardingRestartKeys(t *testing.T) {
	content, err := os.ReadFile("../../migrations/000045_third_party_assessment_restart.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	schema := string(content)
	for _, required := range []string{
		"review_kind='ONBOARDING'",
		"source_trigger='INITIAL'",
		"source_trigger LIKE 'RESTART:%'",
	} {
		if !strings.Contains(schema, required) {
			t.Fatalf("assessment restart migration missing %q", required)
		}
	}
}
