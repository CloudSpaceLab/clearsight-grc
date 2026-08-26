//go:build postgres

package evidence

import (
	"strings"
	"testing"
)

func TestPostgresEntitySourceListFiltersScopeAndKeysetBeforeLimit(t *testing.T) {
	entity := strings.Index(listSourcesForEntitySQL, "es.legal_entity_id=$2::uuid")
	keyset := strings.Index(listSourcesForEntitySQL, "(es.name,es.id) >")
	limit := strings.LastIndex(listSourcesForEntitySQL, "LIMIT $5")
	if entity < 0 || keyset < entity || limit < keyset {
		t.Fatalf("entity/keyset filters must precede the bounded limit: %s", listSourcesForEntitySQL)
	}
}
