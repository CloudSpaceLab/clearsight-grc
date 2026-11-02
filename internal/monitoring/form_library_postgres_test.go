//go:build postgres

package monitoring

import (
	"strings"
	"testing"
)

func TestPostgresFormLibraryFiltersExactEntityBeforeLimit(t *testing.T) {
	normalized := strings.ToUpper(strings.Join(strings.Fields(listFormLibrarySQL), " "))
	entityFilter := strings.Index(normalized, "F.LEGAL_ENTITY_ID=$2::UUID")
	originFilter := strings.Index(normalized, "F.ORIGIN_TYPE=$3 AND F.ORIGIN_ID=NULLIF($4,'')::UUID")
	limit := strings.Index(normalized, "LIMIT")
	for _, required := range []string{"DISTINCT ON", "F.TENANT_ID", "F.ID", "F.UPDATED_AT", "F.STATUS", "F.PROGRAM_ID", "F.OWNER_PRINCIPAL_ID", "F.APPROVED_USES", "F.TAGS", "F.ORIGIN_TYPE", "F.ORIGIN_ID"} {
		if !strings.Contains(normalized, required) {
			t.Fatalf("library query missing %q: %s", required, normalized)
		}
	}
	if entityFilter < 0 || originFilter < 0 || limit < 0 || entityFilter > limit || originFilter > limit {
		t.Fatalf("entity and origin filters must precede limit: %s", normalized)
	}
}
