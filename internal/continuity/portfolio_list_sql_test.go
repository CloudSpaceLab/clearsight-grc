//go:build postgres

package continuity

import (
	"strings"
	"testing"
)

func TestPostgresPortfolioSQLIsBoundedRelationalAndFullAggregate(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		required []string
	}{
		{
			name:  "Programs",
			query: currentProgramListSQL,
			required: []string{
				"WITH selected AS MATERIALIZED", "LIMIT $5", "ORDER BY s.rank,s.updated_at DESC,s.id",
				"'requirements'", "'control_implementations'", "'evidence_contracts'", "'current_state'",
			},
		},
		{
			name:  "Matters",
			query: currentMatterListSQL,
			required: []string{
				"WITH selected AS MATERIALIZED", "LIMIT $6", "ORDER BY s.priority DESC,s.due_at NULLS LAST,s.updated_at DESC,s.id",
				"jsonb_array_elements_text", "'links'", "'actions'", "'verification_contracts'", "'response_packages'",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, fragment := range test.required {
				if !strings.Contains(test.query, fragment) {
					t.Fatalf("portfolio SQL is missing %q", fragment)
				}
			}
			if strings.Contains(test.query, "continuity_events") {
				t.Fatal("ordinary portfolio SQL must not replay continuity history")
			}
		})
	}
}
