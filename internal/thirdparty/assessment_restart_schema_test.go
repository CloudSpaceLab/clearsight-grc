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

func TestAssessmentRestartRollbackRefusesToRewriteUsedRestartHistory(t *testing.T) {
	content, err := os.ReadFile("../../migrations/000045_third_party_assessment_restart.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	schema := string(content)
	guard := "IF EXISTS (SELECT 1 FROM third_party_assessments"
	drop := "DROP CONSTRAINT third_party_assessments_source_trigger_kind_check"
	if !strings.Contains(schema, guard) || !strings.Contains(schema, "source_trigger LIKE 'RESTART:%'") || !strings.Contains(schema, "RAISE EXCEPTION") {
		t.Fatal("assessment restart rollback must refuse to discard or relabel used restart history")
	}
	if strings.Index(schema, guard) > strings.Index(schema, drop) {
		t.Fatal("assessment restart rollback guard must run before changing the constraint")
	}
	for _, prohibited := range []string{"DELETE FROM third_party_assessments", "UPDATE third_party_assessments"} {
		if strings.Contains(schema, prohibited) {
			t.Fatalf("assessment restart rollback must not mutate assessment history with %q", prohibited)
		}
	}
}
