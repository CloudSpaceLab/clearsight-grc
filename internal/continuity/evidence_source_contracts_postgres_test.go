//go:build postgres

package continuity

import (
	"strings"
	"testing"
)

func TestPostgresContractSourceValidationBindsAggregateEntityInsideTransaction(t *testing.T) {
	for name, query := range map[string]string{"Program": programEvidenceSourceValidationSQL, "Matter": matterEvidenceSourceValidationSQL} {
		for _, required := range []string{"legal_entity_id=es.legal_entity_id", "es.status='ACTIVE'", "es.id=ANY($3::uuid[])", "FOR SHARE OF es"} {
			if !strings.Contains(query, required) {
				t.Fatalf("%s validation omitted %q: %s", name, required, query)
			}
		}
	}
}
