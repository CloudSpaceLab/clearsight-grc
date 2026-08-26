//go:build postgres

package evidence

import (
	"strings"
	"testing"
)

func TestInternalRecipientEligibilityPredicateKeepsEntityAndSubjectChecksTogether(t *testing.T) {
	predicate := internalRecipientEligibilityPredicate("p", "rs.tenant_id", "rs.legal_entity_id", "rs.subject_type", "rs.subject_id")
	for _, required := range []string{
		"p.status='ACTIVE'",
		"eligible_position.legal_entity_id=rs.legal_entity_id",
		"eligible_binding.legal_entity_id=rs.legal_entity_id",
		"WHEN 'PROGRAM'",
		"WHEN 'MATTER'",
		"btrim(allowed.value)=p.id::text",
	} {
		if !strings.Contains(predicate, required) {
			t.Fatalf("eligibility predicate omitted %q: %s", required, predicate)
		}
	}
	if strings.Contains(predicate, "%!") || strings.Contains(predicate, "%[") {
		t.Fatalf("eligibility predicate contains an unresolved format operand: %s", predicate)
	}
}
