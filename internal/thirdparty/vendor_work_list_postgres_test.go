//go:build postgres

package thirdparty

import (
	"strings"
	"testing"
	"time"
)

func TestPostgresVendorWorkListQueryFiltersVisibilityBeforeKeysetLimit(t *testing.T) {
	query, args, err := postgresVendorWorkListQuery(
		Scope{TenantID: "bank", LegalEntityID: "entity"},
		VendorWorkListInput{TargetType: LinkTargetProgram, TargetID: "program-1", Cursor: encodeCursor(time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC), "work-2"), Limit: 1},
		"actor-1", time.Date(2026, 8, 26, 11, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	visibility := strings.Index(query, "WITH RECURSIVE route_defs")
	keyset := strings.Index(query, "ORDER BY w.updated_at DESC")
	limit := strings.LastIndex(query, "LIMIT $")
	if visibility < 0 || keyset < 0 || limit < 0 || visibility > keyset || keyset > limit || strings.Contains(query, "%!") || !strings.Contains(query, "jsonb_array_elements_text(m.scope->'allowed_principal_ids')") || !strings.Contains(query, "w.request_kind<>'ADDRESS_VERIFICATION'") {
		t.Fatalf("visibility was not applied before keyset pagination: %s", query)
	}
	if len(args) != 10 || args[7] != "actor-1" || args[9] != 2 {
		t.Fatalf("query arguments = %#v", args)
	}
}
