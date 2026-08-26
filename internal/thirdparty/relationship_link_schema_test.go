package thirdparty

import (
	"os"
	"strings"
	"testing"
)

func TestRelationshipLinkSchemaKeepsTypedTargetsAndHistory(t *testing.T) {
	raw, err := os.ReadFile("../../migrations/000041_third_party_relationship_links.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	schema := string(raw)
	for _, required := range []string{
		"third_party_relationship_program_links", "third_party_relationship_matter_links",
		"REFERENCES programs(id,tenant_id)", "REFERENCES matters(id,tenant_id)",
		"WHERE state='ACTIVE'", "third_party_relationship_link_events",
	} {
		if !strings.Contains(schema, required) {
			t.Fatalf("schema missing %q", required)
		}
	}
}
