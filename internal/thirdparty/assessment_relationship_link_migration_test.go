package thirdparty

import (
	"os"
	"strings"
	"testing"
)

func TestAssessmentMatterLinkMigrationReferencesCanonicalRelationshipLink(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000043_assessment_relationship_link_canonical.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000043_assessment_relationship_link_canonical.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	schema := string(up)
	for _, required := range []string{
		"ADD COLUMN relationship_link_id uuid",
		"INSERT INTO third_party_relationship_matter_links",
		"UPDATE third_party_assessment_matter_links",
		"ALTER COLUMN relationship_link_id SET NOT NULL",
		"FOREIGN KEY (relationship_link_id,tenant_id,legal_entity_id)",
		"third_party_relationship_link_events",
		"outbox_events",
	} {
		if !strings.Contains(schema, required) {
			t.Fatalf("canonical assessment link migration missing %q", required)
		}
	}
	if !strings.Contains(string(down), "DROP COLUMN relationship_link_id") {
		t.Fatal("rollback does not restore the legacy assessment link shape")
	}
}
