package evidence

import (
	"os"
	"strings"
	"testing"
)

func TestCompletedResponseScoreIndexesMatchConcernSortNullOrdering(t *testing.T) {
	raw, err := os.ReadFile("../../migrations/000065_form_scoring_and_response_policies.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	migration := string(raw)
	for _, definition := range []string{
		"adverse_score DESC NULLS LAST,created_at DESC,id DESC",
		"raw_score DESC NULLS LAST,created_at DESC,id DESC",
	} {
		if !strings.Contains(migration, definition) {
			t.Fatalf("completed-response score index must match API sort ordering: missing %q", definition)
		}
	}
}
