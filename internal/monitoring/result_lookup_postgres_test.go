//go:build postgres

package monitoring

import (
	"strings"
	"testing"
)

func TestPostgresResultLookupIsExactTenantScopedAndIndexedByID(t *testing.T) {
	normalized := strings.ToUpper(strings.Join(strings.Fields(resultByIDSQL), " "))
	for _, required := range []string{
		"R.TENANT_ID=(SELECT ID FROM TENANTS WHERE ID::TEXT=$1 OR SLUG=$1)",
		"R.ID=$2::UUID",
	} {
		if !strings.Contains(normalized, required) {
			t.Fatalf("result lookup must use exact tenant and primary-key ID predicates; missing %q in %s", required, normalized)
		}
	}
	if strings.Contains(normalized, "ORDER BY") || strings.Contains(normalized, "LIMIT") {
		t.Fatalf("result lookup should address one primary-key row, not scan a result population: %s", normalized)
	}
}
