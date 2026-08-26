//go:build postgres

package evidence

import (
	"strings"
	"testing"
)

func TestActiveSessionQueryFiltersScopeAndUsabilityBeforeLimit(t *testing.T) {
	query := activeSessionMetadataQuery
	limitIndex := strings.LastIndex(query, "LIMIT")
	if limitIndex < 0 {
		t.Fatal("active session query is not bounded")
	}
	for _, predicate := range []string{
		"(t.id::text=$1 OR t.slug=$1)",
		"cs.request_id=$2::uuid",
		"cs.revoked_at IS NULL",
		"cs.expires_at>$3",
		"cr.status IN ('READY','IN_PROGRESS')",
		"cr.deadline>$3",
	} {
		index := strings.Index(query, predicate)
		if index < 0 || index > limitIndex {
			t.Fatalf("active-session filter %q does not run before LIMIT: %s", predicate, query)
		}
	}
	for _, protected := range []string{"token_hash", "audience_hash"} {
		if strings.Contains(strings.ToLower(query[:strings.Index(query, "FROM")]), protected) {
			t.Fatalf("active session SELECT exposes %q: %s", protected, query)
		}
	}
}
