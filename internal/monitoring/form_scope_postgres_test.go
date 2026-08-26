//go:build postgres

package monitoring

import (
	"strings"
	"testing"
)

func TestPostgresFormListFiltersExactProgramAndEntityBeforeLimit(t *testing.T) {
	normalized := strings.ToUpper(strings.Join(strings.Fields(listFormRevisionsSQL), " "))
	entityFilter := strings.Index(normalized, "F.LEGAL_ENTITY_ID=$2::UUID")
	programFilter := strings.Index(normalized, "F.PROGRAM_ID=$3::UUID")
	limit := strings.Index(normalized, "LIMIT $4")
	if entityFilter < 0 || programFilter < 0 || limit < 0 || entityFilter > limit || programFilter > limit {
		t.Fatalf("form list must filter exact Program/entity before limit: %s", normalized)
	}
}

func TestPostgresFormReadRequiresExactProgramAndEntity(t *testing.T) {
	normalized := strings.ToUpper(strings.Join(strings.Fields(formRevisionSQL), " "))
	for _, required := range []string{
		"F.LEGAL_ENTITY_ID=$2::UUID",
		"F.PROGRAM_ID=$3::UUID",
		"F.ID=$4::UUID",
		"F.VERSION=$5",
	} {
		if !strings.Contains(normalized, required) {
			t.Fatalf("form read query is not exact-scoped; missing %q in %s", required, normalized)
		}
	}
}
