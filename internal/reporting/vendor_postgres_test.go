//go:build postgres || load

package reporting

import (
	"strings"
	"testing"
)

func TestVendorReportPageSQLScopesAndBoundsBeforeLimit(t *testing.T) {
	query := VendorReportPageSQL("a.status = $6", 1)
	for _, required := range []string{
		"r.tenant_id=$1::uuid",
		"r.legal_entity_id=$2::uuid",
		"GREATEST(r.updated_at,v.updated_at)<=$5::timestamptz",
		"archive.record_type='VENDOR_RELATIONSHIP'",
		"archive.restored_at IS NULL",
		"(a.status = $6)",
		"a.updated_at<$8::timestamptz",
		"a.id<NULLIF($9,'')::text",
		"LIMIT $10",
	} {
		if !strings.Contains(query, required) {
			t.Fatalf("vendor report query is missing %q:\n%s", required, query)
		}
	}
	filterAt := strings.Index(query, "(a.status = $6)")
	limitAt := strings.Index(query, "LIMIT $10")
	if filterAt < 0 || limitAt < 0 || filterAt > limitAt {
		t.Fatalf("vendor filter must execute before the bounded limit: filter=%d limit=%d", filterAt, limitAt)
	}
}

func TestVendorReportPageSQLRejectsInvalidFilterMetadata(t *testing.T) {
	if query := VendorReportPageSQL("", 0); query != "" {
		t.Fatalf("empty filter fragment returned query %q", query)
	}
	if query := VendorReportPageSQL("TRUE", maxReportFilterNodes+1); query != "" {
		t.Fatalf("oversized filter metadata returned query %q", query)
	}
}
