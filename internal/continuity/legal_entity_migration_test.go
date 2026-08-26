package continuity

import (
	"os"
	"strings"
	"testing"
)

func TestMatterLegalEntityMigrationFailsClosedAndPreservesReplay(t *testing.T) {
	payload, err := os.ReadFile("../../migrations/000035_matter_legal_entity_scope.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToUpper(string(payload))
	for _, required := range []string{
		"RAISE EXCEPTION",
		"UNRESOLVED",
		"ALTER COLUMN LEGAL_ENTITY_ID SET NOT NULL",
		"UPDATE CONTINUITY_EVENTS",
		"MATTER_CREATED",
		"CREATE CONSTRAINT TRIGGER",
		"MATTER_LINKS",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration is missing %q", required)
		}
	}
}

func TestMatterLegalEntityMigrationMakesParentEntityImmutable(t *testing.T) {
	upPayload, err := os.ReadFile("../../migrations/000035_matter_legal_entity_scope.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	downPayload, err := os.ReadFile("../../migrations/000035_matter_legal_entity_scope.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	upSQL := strings.ToUpper(string(upPayload))
	for _, required := range []string{
		"PREVENT_LEGAL_ENTITY_CHANGE",
		"BEFORE UPDATE OF LEGAL_ENTITY_ID ON PROGRAMS",
		"BEFORE UPDATE OF LEGAL_ENTITY_ID ON MATTERS",
		"OLD.LEGAL_ENTITY_ID IS DISTINCT FROM NEW.LEGAL_ENTITY_ID",
	} {
		if !strings.Contains(upSQL, required) {
			t.Fatalf("migration does not make parent entity immutable: missing %q", required)
		}
	}
	downSQL := strings.ToUpper(string(downPayload))
	for _, required := range []string{
		"DROP TRIGGER IF EXISTS PROGRAMS_LEGAL_ENTITY_IMMUTABLE ON PROGRAMS",
		"DROP TRIGGER IF EXISTS MATTERS_LEGAL_ENTITY_IMMUTABLE ON MATTERS",
		"DROP FUNCTION IF EXISTS PREVENT_LEGAL_ENTITY_CHANGE()",
	} {
		if !strings.Contains(downSQL, required) {
			t.Fatalf("down migration does not remove immutability boundary: missing %q", required)
		}
	}
}

func TestMatterLegalEntityMigrationBackfillsProgramsAndScopesTriggerDedupe(t *testing.T) {
	payload, err := os.ReadFile("../../migrations/000035_matter_legal_entity_scope.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToUpper(string(payload))
	for _, required := range []string{
		"ALTER TABLE PROGRAMS ALTER COLUMN LEGAL_ENTITY_ID SET NOT NULL",
		"PROGRAM LEGAL-ENTITY MIGRATION UNRESOLVED ROWS",
		"EVENT_TYPE='PROGRAM_CREATED'",
		"UPDATE OUTBOX_EVENTS",
		"UNIQUE (TENANT_ID,PROGRAM_ID,DEDUPE_KEY)",
		"DROP INDEX IF EXISTS MATTERS_OPEN_TRIGGER_IDX",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration is missing Program/dedupe safety %q", required)
		}
	}
}

func TestMatterLegalEntityDownMigrationPreflightsUniquenessBeforeDDL(t *testing.T) {
	payload, err := os.ReadFile("../../migrations/000035_matter_legal_entity_scope.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToUpper(string(payload))
	preflight := strings.Index(sql, "DO $$")
	firstDDL := strings.Index(sql, "DROP TRIGGER")
	if preflight < 0 || firstDDL < 0 || preflight > firstDDL {
		t.Fatal("down migration must run collision preflight before DDL")
	}
	for _, required := range []string{"GROUP BY TENANT_ID,CODE", "GROUP BY TENANT_ID,DEDUPE_KEY", "IRREVERSIBLE", "RAISE EXCEPTION"} {
		if !strings.Contains(sql, required) {
			t.Fatalf("down migration preflight missing %q", required)
		}
	}
}

func TestMatterLegalEntityMigrationOnlyEnrichesUnpublishedOutboxAndMatchesSummaryOrder(t *testing.T) {
	payload, err := os.ReadFile("../../migrations/000035_matter_legal_entity_scope.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToUpper(string(payload))
	if strings.Count(sql, "OE.PUBLISHED_AT IS NULL") < 2 {
		t.Fatal("both creation outbox enrichments must exclude published rows")
	}
	for _, required := range []string{
		"CASE STATUS WHEN 'ACTIVE' THEN 0 WHEN 'PAUSED' THEN 1 WHEN 'DRAFT' THEN 2 ELSE 3 END",
		"ON MATTERS(TENANT_ID,LEGAL_ENTITY_ID,PRIORITY DESC,UPDATED_AT DESC,ID DESC)",
		"WHERE STATUS NOT IN ('CLOSED','CANCELLED')",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("summary index structure missing %q", required)
		}
	}
}
