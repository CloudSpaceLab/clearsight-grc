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

func TestActivityLegalEntityScopeUsesCompositeForeignKey(t *testing.T) {
	body := readMigration(t, upFile)
	sql := strings.ToLower(strings.Join(strings.Fields(body), " "))
	if !strings.Contains(sql, "foreign key (legal_entity_id, tenant_id) references legal_entities(id, tenant_id)") {
		t.Error("activity legal-entity scope must reference legal_entities with the tenant in the foreign key")
	}
}

func TestActivityChildReferencesUseTheRetainedScopedKey(t *testing.T) {
	body := readMigration(t, upFile)
	sql := strings.ToLower(strings.Join(strings.Fields(body), " "))
	required := "foreign key (activity_id, tenant_id, legal_entity_id) references ropa_processing_activities(id, tenant_id, legal_entity_id)"
	if got := strings.Count(sql, required); got != 5 {
		t.Errorf("expected five activity references to use UNIQUE (id, tenant_id, legal_entity_id); got %d", got)
	}
	if strings.Contains(sql, "references ropa_processing_activities(tenant_id, legal_entity_id, id)") {
		t.Error("activity references must not target an unavailable parent key order")
	}
}

func TestAllLegalEntityReferencesUseTheAvailableCompositeKey(t *testing.T) {
	body := readMigration(t, upFile)
	sql := strings.ToLower(strings.Join(strings.Fields(body), " "))
	required := "foreign key (legal_entity_id, tenant_id) references legal_entities(id, tenant_id)"
	if got := strings.Count(sql, required); got != 3 {
		t.Errorf("expected activity, event and summary legal-entity references to use the available key; got %d", got)
	}
	if strings.Contains(sql, "references legal_entities(tenant_id, id)") {
		t.Error("legal-entity references must not use unavailable reversed parent key order")
	}
}

func TestRopaMigrationDropsRedundantActivityUniqueKeys(t *testing.T) {
	body := readMigration(t, upFile)
	sql := strings.ToLower(strings.Join(strings.Fields(body), " "))
	for _, redundant := range []string{
		"unique (id, tenant_id)",
		"ropa_activities_legal_entity_uk",
	} {
		if strings.Contains(sql, redundant) {
			t.Errorf("migration must not retain redundant activity uniqueness key %q", redundant)
		}
	}
}

func TestDownMigrationRefusesWhenHistoryExists(t *testing.T) {
	body := readMigration(t, downFile)
	if !strings.Contains(body, "RAISE EXCEPTION") {
		t.Error("down migration must refuse to drop when revision history exists")
	}
}
