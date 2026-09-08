//go:build postgres

package authority

import "testing"

func TestEffectiveOriginDedupPreservesDistinctRoleCodes(t *testing.T) {
	values := []effectiveCandidate{{Principal: Principal{ID: "delegate"}, OriginID: "reviewer", RoleCode: "CISO"}, {Principal: Principal{ID: "delegate"}, OriginID: "reviewer", RoleCode: "CRO"}, {Principal: Principal{ID: "delegate"}, OriginID: "reviewer", RoleCode: "CISO"}}
	origins := uniqueEffectiveOrigins(values)
	if len(origins) != 2 {
		t.Fatalf("role lineage lost: %+v", origins)
	}
}
