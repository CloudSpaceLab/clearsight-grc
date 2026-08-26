package evidence

import (
	"os"
	"strings"
	"testing"
)

func TestEvidenceRequestLegalEntityMigrationIsFailClosedAndReversible(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000037_evidence_request_legal_entity_scope.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	upSQL := string(up)
	for _, required := range []string{
		"ALTER TABLE capture_requests ADD COLUMN legal_entity_id uuid",
		"subject_type='PROGRAM'",
		"subject_type='MATTER'",
		"unresolved_count",
		"ALTER COLUMN legal_entity_id SET NOT NULL",
		"FOREIGN KEY (legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id)",
		"capture_requests_legal_entity_immutable",
	} {
		if !strings.Contains(upSQL, required) {
			t.Fatalf("up migration lacks %q", required)
		}
	}
	down, err := os.ReadFile("../../migrations/000037_evidence_request_legal_entity_scope.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	downSQL := string(down)
	for _, required := range []string{
		"DROP TRIGGER IF EXISTS capture_requests_legal_entity_immutable ON capture_requests",
		"DROP COLUMN IF EXISTS legal_entity_id",
	} {
		if !strings.Contains(downSQL, required) {
			t.Fatalf("down migration lacks %q", required)
		}
	}
}
