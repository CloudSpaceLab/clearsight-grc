//go:build postgres

package evidence

import (
	"strings"
	"testing"
)

func TestExactDocumentQueryRequiresOccurrencePredicates(t *testing.T) {
	sql, ok := exactDocumentInventorySQL(documentInventorySQL())
	if !ok {
		t.Fatal("exact query could not be built")
	}
	for _, required := range []string{"submission.id=$13::uuid", "field->>'id'=$14", "artifact.id=$15::uuid", "req.legal_entity_id=$2::uuid"} {
		if !strings.Contains(sql, required) {
			t.Fatalf("missing exact or scope predicate %s", required)
		}
	}
	if strings.Contains(sql, "$13='' OR") || strings.Contains(sql, "$15='' OR") {
		t.Fatal("exact selectors must not be optional")
	}
	if _, ok := exactDocumentInventorySQL("unexpected query"); ok {
		t.Fatal("query drift must fail closed")
	}
}
