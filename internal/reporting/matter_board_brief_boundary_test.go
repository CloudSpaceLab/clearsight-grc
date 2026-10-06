//go:build postgres

package reporting

import (
	"strings"
	"testing"
)

func TestMatterBoardBriefBoundaryCoversEveryProjectedMutableSource(t *testing.T) {
	queries := make(map[string]string, len(matterBoardBriefBoundaryQueries))
	for _, source := range matterBoardBriefBoundaryQueries {
		if _, exists := queries[source.key]; exists {
			t.Fatalf("duplicate board brief boundary source %q", source.key)
		}
		queries[source.key] = source.query
	}
	for _, required := range []string{
		"matters",
		"matter_actions",
		"matter_decisions",
		"verification_contracts",
		"verification_results",
		"operational_losses",
		"operational_loss_recoveries",
		"form_distributions",
		"form_responses",
		"vendor_work",
		"matter_links",
		"programs",
		"monitoring_results",
		"organization_scopes",
	} {
		if strings.TrimSpace(queries[required]) == "" {
			t.Fatalf("board brief boundary is missing projected source %q", required)
		}
	}
	if !strings.Contains(queries["verification_contracts"], "updated_at") {
		t.Fatal("verification contract boundary is not version-sensitive")
	}
	if !strings.Contains(queries["organization_scopes"], "updated_at") {
		t.Fatal("organization scope boundary is not version-sensitive")
	}
}
