package thirdparty

import (
	"os"
	"strings"
	"testing"
)

func TestActivationPolicySchemaPersistsSimulationRollbackAndHistory(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000070_third_party_activation.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000070_third_party_activation.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	schema := string(up)
	for _, required := range []string{
		"BEGIN;", "CREATE TABLE third_party_activation_policy_simulations", "population_is_complete boolean NOT NULL",
		"evaluated_by uuid NOT NULL", "expires_at timestamptz NOT NULL", "rollback_of_policy_id uuid",
		"third_party_activation_policy_rollback_fk", "'REPLACED'",
	} {
		if !strings.Contains(schema, required) {
			t.Fatalf("activation policy schema missing %q", required)
		}
	}
	for _, existingEvent := range []string{"'VendorIdentityCreated'", "'VendorBrandApproved'", "'AssessmentSetupRetryQueued'", "'AssessmentDocumentExpired'"} {
		if !strings.Contains(schema, existingEvent) {
			t.Fatalf("activation migration removed supported event %s", existingEvent)
		}
	}
	rollback := string(down)
	if strings.Contains(rollback, "DROP TABLE third_party_activation") || strings.Contains(rollback, "DELETE FROM third_party_activation") || strings.Contains(rollback, "VendorRelationshipCreated','VendorRelationshipUpdated'") {
		t.Fatal("deployment rollback must preserve material activation policy, simulation, receipt and event history")
	}
}
