//go:build postgres

package authority

import (
	"strings"
	"testing"
)

func TestReadRouteSQLBindsTheExactAssessmentAndReadPurpose(t *testing.T) {
	q := PostgresReadRouteSQL("assessment", "id", "THIRD_PARTY_ASSESSMENT", "THIRDPARTY.ASSESSMENT.REVIEW", "REVIEWER", 3, 4)
	for _, want := range []string{"ear.tenant_id=assessment.tenant_id", "ear.object_id=assessment.id::text", "upper(ear.object_type)='THIRD_PARTY_ASSESSMENT'", "e.principal_id=$3::uuid", "ear.valid_from<=$4", "count(DISTINCT candidates::text)=1", "sr.responsibility='REVIEWER'"} {
		if !strings.Contains(q, want) {
			t.Errorf("missing %s", want)
		}
	}
	if strings.Contains(q, "w.") || strings.Contains(q, "VENDOR_RELATIONSHIP") {
		t.Fatal("query retained wrong object context")
	}
}
