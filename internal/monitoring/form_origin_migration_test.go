package monitoring

import (
	"os"
	"strings"
	"testing"
)

func TestFormMatterOriginMigrationIsScopedAndImmutable(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000127_form_template_matter_origin.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(up)
	for _, required := range []string{
		"origin_type text",
		"origin_id uuid",
		"origin_type='MATTER'",
		"FOREIGN KEY (origin_id,tenant_id,legal_entity_id)",
		"REFERENCES matters(id,tenant_id,legal_entity_id)",
		"enforce_monitoring_form_origin_immutable",
		"existing.origin_type IS DISTINCT FROM NEW.origin_type",
		"existing.origin_id IS DISTINCT FROM NEW.origin_id",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("form origin migration is missing %q", required)
		}
	}
	if strings.Contains(sql, "CREATE UNIQUE INDEX matters_form_origin_scope_uq") {
		t.Fatal("form origin migration must reuse the existing Matter entity-scope uniqueness contract")
	}

	down, err := os.ReadFile("../../migrations/000127_form_template_matter_origin.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	rollback := string(down)
	if !strings.Contains(rollback, "DROP TRIGGER IF EXISTS monitoring_form_origin_immutable_trigger") ||
		!strings.Contains(rollback, "DROP COLUMN IF EXISTS origin_id") ||
		!strings.Contains(rollback, "DROP COLUMN IF EXISTS origin_type") {
		t.Fatal("form origin rollback does not remove origin-owned schema objects")
	}
}
