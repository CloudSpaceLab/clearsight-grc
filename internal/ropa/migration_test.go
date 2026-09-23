package ropa_test

import (
	"os"
	"strings"
	"testing"
)

const upFile = "../../migrations/000092_ropa_register.up.sql"
const downFile = "../../migrations/000092_ropa_register.down.sql"

func readMigration(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func TestUpMigrationIsASingleTransaction(t *testing.T) {
	body := readMigration(t, upFile)
	if got := strings.Count(body, "BEGIN;"); got != 1 {
		t.Fatalf("expected exactly one BEGIN; got %d", got)
	}
	if got := strings.Count(body, "COMMIT;"); got != 1 {
		t.Fatalf("expected exactly one COMMIT; got %d", got)
	}
	trimmed := strings.TrimSpace(body)
	if !strings.HasPrefix(trimmed, "BEGIN;") {
		t.Fatal("migration must open with BEGIN;")
	}
	if !strings.HasSuffix(trimmed, "COMMIT;") {
		t.Fatal("migration must close with COMMIT;")
	}
}

func TestUpMigrationCreatesEveryOwnedTable(t *testing.T) {
	body := readMigration(t, upFile)
	for _, table := range []string{
		"ropa_processing_activities",
		"ropa_processing_activity_revisions",
		"ropa_processing_activity_data_categories",
		"ropa_processing_activity_recipients",
		"ropa_processing_activity_systems",
		"ropa_processing_activity_reviews",
		"ropa_events",
		"ropa_register_summary",
	} {
		if !strings.Contains(body, "CREATE TABLE "+table) {
			t.Errorf("migration must create %s", table)
		}
	}
}

func TestLegalEntityIsNotNullAndImmutable(t *testing.T) {
	body := readMigration(t, upFile)
	if !strings.Contains(body, "legal_entity_id uuid NOT NULL") {
		t.Error("legal_entity_id must be NOT NULL")
	}
	if !strings.Contains(body, "ropa_legal_entity_immutable") {
		t.Error("migration must install the legal-entity immutability trigger")
	}
}

func TestCurrentReadIndexesCoverTheKeyset(t *testing.T) {
	body := readMigration(t, upFile)
	for _, index := range []string{
		"ropa_register_keyset_idx",
		"ropa_lawful_basis_idx",
		"ropa_owner_idx",
	} {
		if !strings.Contains(body, index) {
			t.Errorf("migration must create index %s", index)
		}
	}
}

func TestActivityScopeKeySupportsCompositeChildReferences(t *testing.T) {
	body := readMigration(t, upFile)
	if !strings.Contains(body, "UNIQUE (id, tenant_id, legal_entity_id)") {
		t.Error("activity table must expose the scoped key required by child foreign keys")
	}
}

func TestDownMigrationRefusesWhenHistoryExists(t *testing.T) {
	body := readMigration(t, downFile)
	if !strings.Contains(body, "RAISE EXCEPTION") {
		t.Error("down migration must refuse to drop when revision history exists")
	}
}
