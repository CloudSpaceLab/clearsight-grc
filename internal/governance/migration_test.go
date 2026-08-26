package governance

import (
	"os"
	"strings"
	"testing"
)

func TestGovernanceLegalEntityMigrationIsFailClosedAndBounded(t *testing.T) {
	payload, err := os.ReadFile("../../migrations/000036_governance_legal_entity_scope.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(string(payload))
	for _, required := range []string{
		"routing_policies add column", "routing_policy_versions add column", "delegations add column",
		"unresolved governance legal-entity rows", "set not null",
		"routing_policies(tenant_id,legal_entity_id,code", "delegations(tenant_id,legal_entity_id,status",
		"jsonb_object_keys", "legal_entity_id",
		"published_at is null", "routing_policies_entity_code_uidx",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration lacks %q", required)
		}
	}
}

func TestGovernanceLegalEntityDownMigrationDropsScopeObjects(t *testing.T) {
	payload, err := os.ReadFile("../../migrations/000036_governance_legal_entity_scope.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(string(payload))
	for _, required := range []string{"drop index", "drop constraint", "drop column if exists legal_entity_id"} {
		if !strings.Contains(sql, required) {
			t.Fatalf("down migration lacks %q", required)
		}
	}
}
