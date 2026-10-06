package reporting

import (
	"os"
	"strings"
	"testing"
)

func TestMatterBoardBriefReportingMigrationExtendsOnlyGovernedReportContracts(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000128_matter_board_brief_reporting.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(up)
	for _, required := range []string{
		"report_definitions_dataset_check",
		"report_definition_revisions_dataset_check",
		"report_runs_dataset_check",
		"MATTER_BOARD_BRIEF",
		"report_definitions_format_check",
		"report_definition_revisions_format_check",
		"report_runs_format_check",
		"'PDF'",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("board brief reporting migration missing %q", required)
		}
	}
	down, err := os.ReadFile("../../migrations/000128_matter_board_brief_reporting.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(down), "cannot remove Matter board brief reporting while governed definitions, history or runs exist") {
		t.Fatal("board brief reporting downgrade does not protect governed history")
	}
}
