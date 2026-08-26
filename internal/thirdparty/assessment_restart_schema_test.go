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

func TestAssessmentRestartRollbackKeepsACompatibilityConstraintWithoutRewritingHistory(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000045_third_party_assessment_restart.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000045_third_party_assessment_restart.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	compatibility := "CHECK ((review_kind='ONBOARDING' AND (source_trigger='INITIAL' OR source_trigger LIKE 'RESTART:%'))"
	for name, schema := range map[string]string{"up": string(up), "down": string(down)} {
		if !strings.Contains(schema, compatibility) {
			t.Fatalf("assessment restart %s migration must admit preserved restart history", name)
		}
	}
	for _, prohibited := range []string{"RAISE EXCEPTION", "DELETE FROM third_party_assessments", "UPDATE third_party_assessments"} {
		if strings.Contains(string(down), prohibited) {
			t.Fatalf("assessment restart rollback must preserve rows without %q", prohibited)
		}
	}
}
