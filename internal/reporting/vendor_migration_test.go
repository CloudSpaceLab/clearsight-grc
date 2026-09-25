package reporting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReportDomainMigrationExtendsEveryDatasetConstraint(t *testing.T) {
	up, err := os.ReadFile(filepath.Join("..", "..", "migrations", "000095_reporting_domains.up.sql"))
	if err != nil {
		t.Fatalf("read report domain migration: %v", err)
	}
	sql := string(up)
	for _, table := range []string{"report_definitions", "report_definition_revisions", "report_runs"} {
		if !strings.Contains(sql, "ALTER TABLE "+table) {
			t.Fatalf("report domain migration does not alter %s", table)
		}
	}
	if count := strings.Count(sql, "'VENDORS'"); count != 3 {
		t.Fatalf("vendor dataset must be enabled for definitions, revisions and runs; found %d occurrences", count)
	}
	if count := strings.Count(sql, "'MATTERS'"); count != 3 {
		t.Fatalf("full Work dataset must be enabled for definitions, revisions and runs; found %d occurrences", count)
	}
}

func TestReportDomainMigrationDownRefusesToEraseHistory(t *testing.T) {
	down, err := os.ReadFile(filepath.Join("..", "..", "migrations", "000095_reporting_domains.down.sql"))
	if err != nil {
		t.Fatalf("read report domain rollback: %v", err)
	}
	sql := string(down)
	for _, table := range []string{"report_definitions", "report_definition_revisions", "report_runs"} {
		if !strings.Contains(sql, "SELECT 1 FROM "+table+" WHERE dataset IN ('VENDORS','MATTERS')") {
			t.Fatalf("vendor/Work report rollback does not guard %s history", table)
		}
	}
}
